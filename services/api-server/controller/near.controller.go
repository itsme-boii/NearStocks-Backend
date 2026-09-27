package controller

// NEAR onboarding and funding (Development.md §6.2, §7, Phase 2).

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/nearchain"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/gin-gonic/gin"
)

const (
	nearChallengeTTL     = 5 * time.Minute
	nearLoginClockSkew   = 5 * time.Minute
	nearSubaccountNumber = 1 // matches the existing frontend (`${broker}_${address}_1`)
)

type NearController struct{}

func RegisterNearController(r *gin.RouterGroup) {
	nc := NearController{}

	auth := r.Group("/auth/near")
	auth.GET("/challenge", nc.Challenge)
	auth.POST("", middleware.RequireSequencer, nc.Login)

	near := r.Group("/near")
	near.GET("/config", nc.Config)
	near.GET("/account", middleware.RequireAuth, nc.Account)
	near.GET("/transfers", middleware.RequireAuth, nc.Transfers)
	near.POST("/deposit/direct", middleware.RequireAuth, middleware.RequireSequencer, nc.DirectDeposit)
	near.POST("/withdraw", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, nc.Withdraw)

	intents := r.Group("/intents")
	intents.GET("/tokens", nc.Tokens)
	intents.POST("/quote", middleware.RequireAuth, middleware.RequireSequencer, nc.DepositQuote)
	intents.POST("/withdraw-quote", middleware.RequireAuth, middleware.RequireSequencer, nc.WithdrawQuote)
	intents.GET("/status/:depositAddress", middleware.RequireAuth, nc.Status)
}

func nearConfigOrAbort(ctx *gin.Context) *services.NearConfig {
	c, err := services.NearConfigFromEnv()
	if err != nil {
		xlog.Errorf("NEAR config: %v", err)
		cutils.ApiAbort(ctx, http.StatusServiceUnavailable, "NEAR is not configured")
		return nil
	}
	return c
}

func challengeKey(nonce []byte) string { return "near:auth:nonce:" + hex.EncodeToString(nonce) }

// ---------- sign-in ----------

func (nc *NearController) Challenge(ctx *gin.Context) {
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "could not create challenge")
		return
	}
	if err := xredis.GetRedisClient().Set(ctx, challengeKey(nonce), "1", nearChallengeTTL).Err(); err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "could not store challenge")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"nonce":     base64.StdEncoding.EncodeToString(nonce),
		"recipient": c.ContractAccount,
		"chainId":   c.ChainId,
		"brokerId":  c.BrokerId,
		"expiresAt": time.Now().Add(nearChallengeTTL).UnixMilli(),
	})
}

type NearLoginBody struct {
	AccountId        string  `json:"accountId" binding:"required"`
	PublicKey        string  `json:"publicKey" binding:"required"`
	Signature        string  `json:"signature" binding:"required"` // base64, as returned by signMessage
	Message          string  `json:"message" binding:"required"`   // exact signed string
	Nonce            string  `json:"nonce" binding:"required"`     // base64 32 bytes from /challenge
	CallbackUrl      *string `json:"callbackUrl"`
	SessionKey       string  `json:"sessionKey" binding:"required"`
	SessionSignature string  `json:"sessionSignature" binding:"required"` // EIP-712 Register by the session key
	ExpiryTs         int64   `json:"expiryTs" binding:"required"`
}

type nearLoginMessage struct {
	Action     string `json:"action"`
	SessionKey string `json:"sessionKey"`
	Expiry     int64  `json:"expiry"`
	Ts         int64  `json:"ts"`
}

// Login verifies a NEP-413 signature from a full-access key, proves possession of the browser
// session key, then creates (or reuses) the subaccount and returns API credentials.
func (nc *NearController) Login(ctx *gin.Context) {
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	var body NearLoginBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid request")
		return
	}
	if !cutils.IsValidNearAccountId(body.AccountId) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid NEAR account id")
		return
	}
	nonce, err := base64.StdEncoding.DecodeString(body.Nonce)
	if err != nil || len(nonce) != 32 {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid nonce")
		return
	}

	// 1. The challenge is single-use: GETDEL removes it even if a later check fails.
	if n, err := xredis.GetRedisClient().GetDel(ctx, challengeKey(nonce)).Result(); err != nil || n != "1" {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "challenge expired or already used")
		return
	}

	// 2. The signed message must bind this session key and expiry, and be fresh.
	var msg nearLoginMessage
	if err := json.Unmarshal([]byte(body.Message), &msg); err != nil || msg.Action != "login" {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid login message")
		return
	}
	now := time.Now()
	if !strings.EqualFold(msg.SessionKey, body.SessionKey) || msg.Expiry != body.ExpiryTs {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "login message does not match request")
		return
	}
	if d := now.Sub(time.UnixMilli(msg.Ts)); d > nearLoginClockSkew || d < -nearLoginClockSkew {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "login message timestamp out of range")
		return
	}
	if body.ExpiryTs <= now.UnixMilli() || body.ExpiryTs-now.UnixMilli() > int64(EXPIRY_DURATION) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid session expiry")
		return
	}

	// 3. NEP-413 signature over {message, nonce, recipient, callbackUrl}.
	var nonceArr [32]byte
	copy(nonceArr[:], nonce)
	payload := contractUtils.NEP413Payload{Message: body.Message, Nonce: nonceArr, Recipient: c.ContractAccount, CallbackUrl: body.CallbackUrl}
	if err := contractUtils.VerifyNEP413(body.PublicKey, body.Signature, payload); err != nil {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "NEAR signature verification failed")
		return
	}

	// 4. The key must be a FullAccess key of this account right now (NEP-413 requirement).
	rpcCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	key, err := c.RPC.ViewAccessKey(rpcCtx, body.AccountId, body.PublicKey)
	if errors.Is(err, nearchain.ErrAccessKeyNotFound) {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "public key is not a key of this account")
		return
	}
	if err != nil {
		xlog.Errorf("NEAR login view_access_key: %v", err)
		cutils.ApiAbort(ctx, http.StatusServiceUnavailable, "NEAR RPC unavailable")
		return
	}
	if !key.IsFullAccess() {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "a full-access key is required to sign in")
		return
	}

	// 5. Derive the subaccount and prove the browser holds the session key.
	addr20, _ := cutils.NearAccountToAddr20(body.AccountId)
	subaccountId := cutils.CreateSubaccountId(c.BrokerId, addr20, nearSubaccountNumber)
	reg := contractUtils.NearRegister{SubaccountId: subaccountId, UserAddress: addr20, SessionKey: body.SessionKey, ExpiryTs: body.ExpiryTs, Nonce: 0}
	if err := contractUtils.VerifyNearRegister(c.ContractAccount, c.ChainId, reg, body.SessionSignature); err != nil {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "session key signature verification failed")
		return
	}

	if (&db.BrokerDB{}).GetById(c.BrokerId) == nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "broker not found")
		return
	}
	subHex, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "invalid subaccount")
		return
	}
	if cutils.IsSubAccountPaused(strings.ToLower(subHex)) {
		cutils.ApiAbort(ctx, http.StatusForbidden, "account is paused: please contact support")
		return
	}

	// NEAR settlement: the contract only accepts orders from session keys the user's account
	// registered on-chain (register_session_key, §6.3). Check before issuing credentials, so the
	// frontend registers first instead of the user's first trade being refused on-chain.
	if contractUtils.NearSettlement() {
		raw, err := c.RPC.CallView(ctx, c.ContractAccount, "session_key_expiry", map[string]any{"subaccount": strings.ToLower(subHex), "session_key": strings.ToLower(body.SessionKey)})
		var onChain uint64
		if err == nil {
			err = json.Unmarshal(raw, &onChain)
		}
		if err != nil {
			xlog.Errorf("NEAR login session_key_expiry: %v", err)
			cutils.ApiAbort(ctx, http.StatusServiceUnavailable, "NEAR RPC unavailable")
			return
		}
		if onChain <= uint64(time.Now().UnixMilli()) {
			ctx.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "register the session key on near-stocks first", "needsRegistration": true, "contract": c.ContractAccount})
			return
		}
		if uint64(body.ExpiryTs) > onChain {
			body.ExpiryTs = int64(onChain) // credentials never outlive the on-chain key
		}
	}

	authData, err := GenerateAuthData()
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "unable to generate auth data")
		return
	}

	resp, err := services.WithNonceRedisLock(subHex, func(_ string) (*Response, error) {
		sub := (&db.SubaccountDB{}).GetByIdBrokerId(c.BrokerId, subaccountId)
		if sub == nil {
			var errSub error
			if sub, errSub = (&db.SubaccountDB{}).Create(addr20, c.BrokerId, subaccountId); errSub != nil {
				return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "error creating subaccount"}}, errSub
			}
			if _, err := (&db.ReferralDB{}).CreateReferralUser(db.ReferralUserTable{UserAddress: addr20}); err != nil {
				xlog.Errorf("NEAR login: referral user: %v", err)
			}
		} else if sub.EthAddress != addr20 {
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "subaccount address mismatch"}}, fmt.Errorf("address mismatch")
		}
		if _, err := (&db.NearAccountDB{}).GetOrCreate(body.AccountId, c.BrokerId, subaccountId, addr20); err != nil {
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "error linking NEAR account"}}, err
		}

		if existing := (&db.SigningKeyDB{}).GetBySigningAddress(body.SessionKey); existing != nil {
			if existing.SubaccountID != sub.ID {
				return &Response{status: http.StatusBadRequest, data: gin.H{"error": "session key already in use"}}, fmt.Errorf("session key in use")
			}
		} else if err := (&db.SigningKeyDB{}).Create(body.SessionKey, sub.ID, c.BrokerId, uint64(body.ExpiryTs)); err != nil {
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "error creating session key"}}, err
		}
		// Phase 2: the DB is the source of truth. With NEAR settlement the key was checked on-chain above.

		if err := (&db.AuthDB{}).Create(authData.LogxKey, authData.logxSecretHash, sub.ID, uint64(body.ExpiryTs)); err != nil {
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "error creating auth key"}}, err
		}
		return &Response{status: http.StatusOK}, nil
	})
	if err != nil {
		ctx.AbortWithStatusJSON(resp.status, resp.data)
		return
	}
	if c.BrokerId == 2 { // same response shape as RegisterAuth for broker 2
		authData.LogxKey, authData.LogxSecret = "", ""
	}
	xlog.Infof("NEAR login: account=%s subaccount=%s", body.AccountId, subaccountId)
	// Same envelope and credential fields as RegisterAuth, so the frontend stores it identically.
	cutils.ApiSuccess(ctx, gin.H{
		"logx_key":      authData.LogxKey,
		"logx_secret":   authData.LogxSecret,
		"broker_key":    authData.BrokerKey,
		"broker_secret": authData.BrokerSecret,
		"subaccountId":  subaccountId,
		"addr20":        addr20,
		"accountId":     body.AccountId,
	}, "NEAR account signed in")
}

// ---------- read endpoints ----------

func (nc *NearController) Config(ctx *gin.Context) {
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"network": c.Network, "contractAccount": c.ContractAccount, "chainId": c.ChainId,
		"treasury": c.Treasury, "usdc": c.USDC, "usdcAssetId": c.USDCAssetId(), "brokerId": c.BrokerId,
		"settlement":         contractUtils.NearSettlement(),
		// with NEAR settlement the contract pays withdrawals out; the treasury signer is not needed
		"withdrawalsEnabled": c.Signer != nil || contractUtils.NearSettlement(), "withdrawMaxUsdc": c.WithdrawMaxUSDC.String(),
		"withdrawFeeX18": nearWithdrawFee(ctx, c, contractUtils.QUOTE_TOKEN_PRODUCT_ID).String(),
	})
}

func currentNearAccount(ctx *gin.Context) (*db.SubaccountTable, *db.NearAccountTable, bool) {
	sub, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "subaccount not found")
		return nil, nil, false
	}
	link := (&db.NearAccountDB{}).GetBySubaccountId(sub.ID)
	if link == nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "this subaccount is not linked to a NEAR account")
		return nil, nil, false
	}
	return sub, link, true
}

func (nc *NearController) Account(ctx *gin.Context) {
	sub, link, ok := currentNearAccount(ctx)
	if !ok {
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"accountId": link.AccountId, "subaccountId": sub.ID, "addr20": link.Addr20, "brokerId": link.BrokerId})
}

func (nc *NearController) Transfers(ctx *gin.Context) {
	sub, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "subaccount not found")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"transfers": (&db.IntentsTransferDB{}).ListBySubaccount(sub.ID, 50)})
}

// ---------- deposits ----------

type directDepositBody struct {
	TxHash string `json:"txHash" binding:"required"`
}

// DirectDeposit credits a USDC ft_transfer that the user's linked NEAR account sent to the
// treasury. The transaction is fetched from the chain and verified; each hash credits once.
func (nc *NearController) DirectDeposit(ctx *gin.Context) {
	if contractUtils.NearSettlement() {
		// deposits go to the contract (ft_transfer_call) and are credited from its event (§7.1)
		cutils.ApiAbort(ctx, http.StatusConflict, "deposits are credited automatically once they reach the contract")
		return
	}
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	sub, link, ok := currentNearAccount(ctx)
	if !ok {
		return
	}
	var body directDepositBody
	if err := ctx.ShouldBindJSON(&body); err != nil || len(body.TxHash) < 32 || len(body.TxHash) > 64 {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid transaction hash")
		return
	}
	key := "neartx:" + body.TxHash
	if row := (&db.IntentsTransferDB{}).GetByKey(key); row != nil && row.CreditedAt != nil {
		ctx.JSON(http.StatusOK, gin.H{"status": row.Status, "amountUsdc": row.AmountUSDC})
		return
	}
	rpcCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx, err := c.RPC.TxStatus(rpcCtx, body.TxHash, link.AccountId)
	if errors.Is(err, nearchain.ErrUnknownTx) {
		cutils.ApiAbort(ctx, http.StatusNotFound, "transaction not found yet: retry in a few seconds")
		return
	}
	if err != nil {
		xlog.Errorf("NEAR direct deposit tx status %s: %v", body.TxHash, err)
		cutils.ApiAbort(ctx, http.StatusServiceUnavailable, "NEAR RPC unavailable")
		return
	}
	amount, err := nearchain.VerifyDirectDeposit(tx, link.AccountId, c.Treasury, c.USDC)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}
	row := &db.IntentsTransferTable{Key: key, Direction: "deposit", Kind: "direct", SubaccountId: sub.ID, NearAccountId: link.AccountId, Status: "VERIFIED", AmountUSDC: amount.String(), NearTxHash: body.TxHash}
	if err := (&db.IntentsTransferDB{}).Create(row); err != nil && !errors.Is(err, db.ErrIntentsTransferExists) {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "could not record deposit")
		return
	}
	if err := services.CreditNearDeposit(key, sub.ID, amount.String(), body.TxHash); err != nil && !errors.Is(err, services.ErrAlreadyCredited) {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "deposit verified but crediting failed: support has been alerted")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "SUCCESS_CREDITED", "amountUsdc": amount.String()})
}

// ---------- 1Click ----------

var tokensCache struct {
	sync.Mutex
	at   time.Time
	data []nearchain.OneClickToken
}

func (nc *NearController) Tokens(ctx *gin.Context) {
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	tokensCache.Lock()
	defer tokensCache.Unlock()
	if time.Since(tokensCache.at) > 5*time.Minute || tokensCache.data == nil {
		t, err := c.OneClick.Tokens(ctx)
		if err != nil {
			xlog.Errorf("1Click tokens: %v", err)
			if tokensCache.data == nil {
				cutils.ApiAbort(ctx, http.StatusBadGateway, "token list unavailable")
				return
			}
		} else {
			tokensCache.data, tokensCache.at = t, time.Now()
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"tokens": tokensCache.data})
}

type depositQuoteBody struct {
	OriginAsset     string `json:"originAsset" binding:"required"`
	Amount          string `json:"amount" binding:"required"` // smallest units of originAsset
	RefundTo        string `json:"refundTo" binding:"required"`
	Confidentiality string `json:"confidentiality"` // public | basic | advanced
	SlippageBps     int    `json:"slippageBps"`
}

func validConfidentiality(s string) (string, bool) {
	switch s {
	case "", "public":
		return "public", true
	case "basic", "advanced":
		return s, true
	}
	return "", false
}

func validAmount(s string) bool {
	v, ok := new(big.Int).SetString(s, 10)
	return ok && v.Sign() > 0
}

func slippage(bps int) (int, bool) {
	if bps == 0 {
		return 100, true
	}
	return bps, bps > 0 && bps <= 500
}

// DepositQuote asks 1Click for a deposit address that swaps any asset into USDC delivered to
// the treasury with a plain ft_transfer. customRecipientMsg is deliberately NOT set: the 1Click
// spec warns funds are lost if the recipient doesn't implement ft_on_transfer, and the
// treasury is a plain account until Phase 4.
func (nc *NearController) DepositQuote(ctx *gin.Context) {
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	sub, link, ok := currentNearAccount(ctx)
	if !ok {
		return
	}
	var body depositQuoteBody
	if err := ctx.ShouldBindJSON(&body); err != nil || !validAmount(body.Amount) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid request")
		return
	}
	conf, okC := validConfidentiality(body.Confidentiality)
	bps, okS := slippage(body.SlippageBps)
	if !okC || !okS {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid confidentiality or slippage")
		return
	}
	req := nearchain.QuoteRequest{
		Dry: false, SwapType: "EXACT_INPUT", SlippageTolerance: bps,
		OriginAsset: body.OriginAsset, DepositType: "ORIGIN_CHAIN",
		DestinationAsset: c.USDCAssetId(), Amount: body.Amount,
		RefundTo: body.RefundTo, RefundType: "ORIGIN_CHAIN",
		Recipient: c.DepositRecipient(), RecipientType: "DESTINATION_CHAIN",
		Deadline: time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339), Confidentiality: conf, Referral: "near-stocks",
		CustomRecipientMsg: services.DepositRecipientMsg(link.AccountId),
	}
	q, err := c.OneClick.Quote(ctx, req)
	if err != nil {
		var oe *nearchain.OneClickError
		if errors.As(err, &oe) && oe.Status == http.StatusBadRequest {
			cutils.ApiAbort(ctx, http.StatusBadRequest, "quote rejected: "+oe.Body)
			return
		}
		xlog.Errorf("1Click deposit quote: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadGateway, "quote unavailable")
		return
	}
	if q.Quote.DepositAddress == "" || q.QuoteRequest.Recipient != c.DepositRecipient() || q.QuoteRequest.DestinationAsset != c.USDCAssetId() {
		cutils.ApiAbort(ctx, http.StatusBadGateway, "unexpected quote from 1Click")
		return
	}
	raw, _ := json.Marshal(q)
	row := &db.IntentsTransferTable{
		Key: "1click:" + q.Quote.DepositAddress + ":" + q.Quote.DepositMemo, Direction: "deposit", Kind: "1click",
		SubaccountId: sub.ID, NearAccountId: link.AccountId, DepositAddress: q.Quote.DepositAddress,
		DepositMemo: q.Quote.DepositMemo, Confidential: conf, QuoteJson: string(raw), Status: nearchain.StatusPendingDeposit,
	}
	if err := (&db.IntentsTransferDB{}).Create(row); err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "could not record quote")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"quote": q.Quote, "correlationId": q.CorrelationId, "confidentiality": conf})
}

type withdrawQuoteBody struct {
	DestinationAsset string `json:"destinationAsset" binding:"required"`
	Recipient        string `json:"recipient" binding:"required"`  // user's address on the destination chain
	AmountUsdc       string `json:"amountUsdc" binding:"required"` // exact USDC (6 decimals) to send in
	Confidentiality  string `json:"confidentiality"`
	SlippageBps      int    `json:"slippageBps"`
}

// WithdrawQuote creates a NEAR-origin 1Click quote whose deposit address the user then signs as
// the NearWithdraw receiver. Refunds go back to the user's own NEAR account.
func (nc *NearController) WithdrawQuote(ctx *gin.Context) {
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	sub, link, ok := currentNearAccount(ctx)
	if !ok {
		return
	}
	var body withdrawQuoteBody
	if err := ctx.ShouldBindJSON(&body); err != nil || !validAmount(body.AmountUsdc) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid request")
		return
	}
	conf, okC := validConfidentiality(body.Confidentiality)
	bps, okS := slippage(body.SlippageBps)
	if !okC || !okS {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid confidentiality or slippage")
		return
	}
	req := nearchain.QuoteRequest{
		Dry: false, SwapType: "EXACT_INPUT", SlippageTolerance: bps,
		OriginAsset: c.USDCAssetId(), DepositType: "ORIGIN_CHAIN",
		DestinationAsset: body.DestinationAsset, Amount: body.AmountUsdc,
		RefundTo: link.AccountId, RefundType: "ORIGIN_CHAIN",
		Recipient: body.Recipient, RecipientType: "DESTINATION_CHAIN",
		Deadline: time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339), Confidentiality: conf, Referral: "near-stocks",
	}
	q, err := c.OneClick.Quote(ctx, req)
	if err != nil {
		var oe *nearchain.OneClickError
		if errors.As(err, &oe) && oe.Status == http.StatusBadRequest {
			cutils.ApiAbort(ctx, http.StatusBadRequest, "quote rejected: "+oe.Body)
			return
		}
		cutils.ApiAbort(ctx, http.StatusBadGateway, "quote unavailable")
		return
	}
	if q.Quote.DepositAddress == "" || !cutils.IsValidNearAccountId(q.Quote.DepositAddress) || q.Quote.DepositMemo != "" {
		cutils.ApiAbort(ctx, http.StatusBadGateway, "unexpected NEAR deposit address from 1Click")
		return
	}
	raw, _ := json.Marshal(q)
	row := &db.IntentsTransferTable{
		Key: "1click-out:" + q.Quote.DepositAddress, Direction: "withdraw-quote", Kind: "1click",
		SubaccountId: sub.ID, NearAccountId: link.AccountId, DepositAddress: q.Quote.DepositAddress,
		Confidential: conf, QuoteJson: string(raw), Status: "QUOTED", AmountUSDC: q.Quote.AmountIn,
	}
	if err := (&db.IntentsTransferDB{}).Create(row); err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "could not record quote")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"quote": q.Quote, "correlationId": q.CorrelationId, "confidentiality": conf})
}

// Status returns (and advances) a 1Click flow owned by the caller.
func (nc *NearController) Status(ctx *gin.Context) {
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	sub, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "subaccount not found")
		return
	}
	addr := ctx.Param("depositAddress")
	memo := ctx.Query("depositMemo")
	row := (&db.IntentsTransferDB{}).GetByKey("1click:" + addr + ":" + memo)
	if row == nil {
		row = (&db.IntentsTransferDB{}).GetByKey("1click-out:" + addr)
	}
	if row == nil || row.SubaccountId != sub.ID {
		cutils.ApiAbort(ctx, http.StatusNotFound, "transfer not found")
		return
	}
	if row.Direction == "deposit" && row.CreditedAt == nil {
		status, err := services.SyncOneClickDeposit(ctx, row)
		if err != nil {
			xlog.Errorf("1Click status %s: %v", row.Key, err)
		}
		row.Status = status
	} else if row.Direction == "withdraw-quote" {
		if st, err := c.OneClick.Status(ctx, addr, ""); err == nil {
			row.Status = st.Status
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"status": row.Status, "transfer": row})
}

// ---------- withdrawals ----------

type nearWithdrawBody struct {
	SubAccountId string `json:"subAccountId" binding:"required"`
	ProductId    uint32 `json:"productId"`
	Amount       string `json:"amount" binding:"required"` // x18, total debited (fee included)
	Nonce        int64  `json:"nonce"`
	Receiver     string `json:"receiver" binding:"required"`
}

// Withdraw mirrors TokenController.WithdrawCollateral's checks, then pays out from the treasury.
// The receiver must be the user's own linked NEAR account or a 1Click withdraw quote created
// for this subaccount, so a leaked session key cannot redirect funds.
func (nc *NearController) Withdraw(ctx *gin.Context) {
	c := nearConfigOrAbort(ctx)
	if c == nil {
		return
	}
	settlement := contractUtils.NearSettlement()
	if c.Signer == nil && !settlement {
		cutils.ApiAbort(ctx, http.StatusServiceUnavailable, "NEAR withdrawals are not enabled")
		return
	}
	sub, link, ok := currentNearAccount(ctx)
	if !ok {
		return
	}
	var body nearWithdrawBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid request")
		return
	}
	if body.SubAccountId != sub.ID {
		cutils.ApiAbort(ctx, http.StatusForbidden, "subaccount does not match credentials")
		return
	}
	// the contract also pays out LogX (product 0, WITHDRAW_LOGX); the treasury only held USDC
	if body.ProductId != contractUtils.QUOTE_TOKEN_PRODUCT_ID && !(settlement && body.ProductId == contractUtils.LOGX) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "only USDC (and LogX with NEAR settlement) withdrawals are supported")
		return
	}
	subHex, err := cutils.SubaccountIdToHex(sub.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "invalid subaccount")
		return
	}
	if flagged, err := cutils.IsSubaccountFlagged(ctx, sub.ID); err != nil || flagged {
		cutils.ApiAbort(ctx, http.StatusForbidden, "withdrawal blocked: contact support")
		return
	}
	if internal, err := cutils.IsInternalSubaccount(subHex); err != nil || internal {
		cutils.ApiAbort(ctx, http.StatusForbidden, "withdrawal blocked")
		return
	}

	// Session key: must be registered to this subaccount and unexpired.
	sessionKey := ctx.GetHeader("Logx-Signer-Address")
	if sessionKey == "" {
		sessionKey = ctx.GetHeader("Broker-Signer-Address")
	}
	signature := ctx.GetHeader("Logx-Signature")
	if signature == "" {
		signature = ctx.GetHeader("Broker-Signature")
	}
	sk := (&db.SigningKeyDB{}).GetBySigningAddress(sessionKey)
	if sk == nil || sk.SubaccountID != sub.ID || sk.ExpiryTs <= uint64(time.Now().UnixMilli()) {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "session key missing or expired")
		return
	}

	amountX18, okA := new(big.Int).SetString(body.Amount, 10)
	fee := nearWithdrawFee(ctx, c, body.ProductId)
	if !okA || amountX18.Cmp(fee) <= 0 {
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("amount must be greater than the withdrawal fee (%s)", fee))
		return
	}
	payoutUSDC, _ := nearchain.ScaleX18ToUSDC(new(big.Int).Sub(amountX18, fee))
	if body.ProductId == contractUtils.LOGX {
		payoutUSDC = new(big.Int).Sub(amountX18, fee) // LogX has 18 decimals: the payout is the x18 amount; no USDC cap
	} else if payoutUSDC.Sign() <= 0 || payoutUSDC.Cmp(c.WithdrawMaxUSDC) > 0 {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "amount outside the allowed withdrawal range")
		return
	}

	// Receiver policy.
	if !cutils.IsValidNearAccountId(body.Receiver) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid receiver")
		return
	}
	var quoteRow *db.IntentsTransferTable
	if body.Receiver != link.AccountId {
		quoteRow = (&db.IntentsTransferDB{}).GetByKey("1click-out:" + body.Receiver)
		if quoteRow == nil || quoteRow.SubaccountId != sub.ID || quoteRow.Status != "QUOTED" {
			cutils.ApiAbort(ctx, http.StatusForbidden, "receiver must be your NEAR account or a withdraw quote you created")
			return
		}
		if quoteRow.AmountUSDC != payoutUSDC.String() {
			cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("amount after fee (%s USDC units) must equal the quote amount (%s)", payoutUSDC, quoteRow.AmountUSDC))
			return
		}
	}

	w := contractUtils.NearWithdraw{SubaccountId: sub.ID, SessionKey: sessionKey, ProductId: body.ProductId, Amount: body.Amount, Nonce: body.Nonce, Receiver: body.Receiver}
	if err := contractUtils.VerifyNearWithdraw(c.ContractAccount, c.ChainId, w, signature); err != nil {
		cutils.ApiAbort(ctx, http.StatusUnauthorized, "withdraw signature verification failed")
		return
	}

	key := fmt.Sprintf("withdraw:%s:%d", subHex, body.Nonce)
	resp, err := services.WithNonceRedisLock(subHex, func(currentNonce string) (*Response, error) {
		if !cutils.CheckNonce(body.Nonce, currentNonce) {
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "nonce check failed"}}, fmt.Errorf("nonce")
		}
		avail, err := nearAvailable(subHex, sub.ID, body.ProductId)
		if err != nil {
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "failed to fetch available balance"}}, err
		}
		if avail.Cmp(amountX18) < 0 {
			return &Response{status: http.StatusBadRequest, data: gin.H{"error": "insufficient withdrawable balance"}}, fmt.Errorf("insufficient")
		}
		row := &db.IntentsTransferTable{Key: key, Direction: "withdraw", Kind: map[bool]string{true: "1click", false: "direct"}[quoteRow != nil],
			SubaccountId: sub.ID, NearAccountId: link.AccountId, DepositAddress: body.Receiver, Status: services.PayoutPending, AmountUSDC: payoutUSDC.String()}
		if err := (&db.IntentsTransferDB{}).Create(row); err != nil {
			return &Response{status: http.StatusConflict, data: gin.H{"error": "withdrawal already submitted"}}, err
		}
		if err := services.IncrementNonce(subHex, currentNonce); err != nil {
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "nonce update failed"}}, err
		}
		okBal, err := xclient.GlobalBalanceClient.UpdateTokenBalance(subHex, body.ProductId, new(big.Int).Neg(amountX18).String())
		if err != nil || !okBal {
			services.IncrementNonce(subHex, currentNonce, true)
			_ = (&db.IntentsTransferDB{}).SetWithdrawResult(key, "DEBIT_FAILED", "", fmt.Sprint(err))
			return &Response{status: http.StatusInternalServerError, data: gin.H{"error": "balance update failed"}}, fmt.Errorf("debit: %v", err)
		}
		return &Response{status: http.StatusOK}, nil
	})
	if err != nil {
		ctx.AbortWithStatusJSON(resp.status, resp.data)
		return
	}
	if quoteRow != nil {
		_ = (&db.IntentsTransferDB{}).SetWithdrawResult(quoteRow.Key, "FUNDING", "", "")
	}

	if settlement {
		// the contract pays out (WITHDRAW_COLLATERAL / WITHDRAW_LOGX) once the batcher sends it; a
		// failed transfer is re-credited on-chain and mirrored by the indexer (withdraw_failed)
		counter, err := transaction.IncrementCounter(1)
		if err == nil {
			err = contract.GlobalContracts.EndpointContract.NearWithdraw(w, signature, counter)
		}
		if err != nil {
			_, _ = xclient.GlobalBalanceClient.UpdateTokenBalance(subHex, body.ProductId, amountX18.String())
			_ = (&db.IntentsTransferDB{}).SetWithdrawResult(key, services.PayoutFailed, "", err.Error())
			xlog.Errorf("NEAR withdraw %s not queued (debit reversed): %v", key, err)
			cutils.ApiAbort(ctx, http.StatusInternalServerError, "withdrawal could not be queued; your balance was restored")
			return
		}
		_ = (&db.IntentsTransferDB{}).SetWithdrawResult(key, "QUEUED", "", "")
		ctx.JSON(http.StatusOK, gin.H{"status": "QUEUED", "amountUsdc": payoutUSDC.String()})
		return
	}

	status, hash, perr := services.SendTreasuryPayout(key, subHex, amountX18, body.Receiver, payoutUSDC)
	if status == services.PayoutSent {
		_ = (&db.DepositWithdrawDB{}).InsertNearTransfer(key, hash, subHex, body.Amount, body.ProductId, false, uint64(c.ChainId))
		if quoteRow != nil {
			_ = c.OneClick.SubmitDeposit(ctx, hash, body.Receiver, c.Treasury) // optional speed-up
		}
		ctx.JSON(http.StatusOK, gin.H{"status": status, "txHash": hash, "amountUsdc": payoutUSDC.String()})
		return
	}
	xlog.Errorf("NEAR payout %s: %s %v", key, status, perr)
	code := http.StatusBadGateway
	if status == services.PayoutFailed {
		code = http.StatusInternalServerError
	}
	ctx.JSON(code, gin.H{"status": status, "txHash": hash, "error": "payout did not complete; your balance is safe and support has been alerted"})
}

// nearAvailable is what a NEAR withdrawal may take: the withdrawable collateral for USDC, and for
// LogX the balance minus unstakes still in cooldown (TokenController.isValidWithdrawLogX, R-7).
func nearAvailable(subHex, subId string, productId uint32) (*big.Int, error) {
	if productId != contractUtils.LOGX {
		withdrawable, err := xclient.GlobalBalanceClient.GetWithdrawableTokenBalance(subHex)
		if err != nil {
			return nil, err
		}
		v, ok := new(big.Int).SetString(withdrawable[productId], 10)
		if !ok {
			return new(big.Int), nil
		}
		return v, nil
	}
	spots, _, _, err := xclient.GlobalBalanceClient.GetSpotBalance(subHex)
	if err != nil {
		return nil, err
	}
	pending, err := (&db.StakingDB{}).GetUnstakeSumOfPendingUnstake(subId)
	if err != nil {
		return nil, err
	}
	for _, sp := range spots {
		if sp.ProductId == contractUtils.LOGX {
			v, ok := new(big.Int).SetString(sp.TokenBalance, 10)
			if !ok {
				return nil, fmt.Errorf("invalid LogX balance %q", sp.TokenBalance)
			}
			return v.Sub(v, pending), nil
		}
	}
	return new(big.Int), nil
}

var (
	nearFeeMu    sync.Mutex
	nearFees     map[uint32]*big.Int
	nearFeesTime time.Time
)

// nearWithdrawFee is what a withdrawal of productId is charged. With NEAR settlement the contract
// charges its spot's withdraw_fee_x18, so that is read (cached for a minute); the static
// WITHDRAWAL_FEE_MAP drifted from it (LogX: 0.2 there, 25 on-chain).
func nearWithdrawFee(ctx context.Context, c *services.NearConfig, productId uint32) *big.Int {
	if !contractUtils.NearSettlement() {
		return getWithdrawalFee(productId)
	}
	nearFeeMu.Lock()
	defer nearFeeMu.Unlock()
	if nearFees == nil || time.Since(nearFeesTime) > time.Minute {
		raw, err := c.RPC.CallView(ctx, c.ContractAccount, "get_products", map[string]any{})
		var v struct {
			Spots []struct {
				ProductId      uint32 `json:"product_id"`
				WithdrawFeeX18 string `json:"withdraw_fee_x18"`
			} `json:"spots"`
		}
		if err == nil {
			err = json.Unmarshal(raw, &v)
		}
		if err != nil {
			xlog.Errorf("NEAR withdraw fee: get_products: %v", err)
		} else {
			fees := map[uint32]*big.Int{}
			for _, sp := range v.Spots {
				if f, ok := new(big.Int).SetString(sp.WithdrawFeeX18, 10); ok {
					fees[sp.ProductId] = f
				}
			}
			nearFees, nearFeesTime = fees, time.Now()
		}
	}
	if f, ok := nearFees[productId]; ok {
		return new(big.Int).Set(f)
	}
	return getWithdrawalFee(productId) // RPC down on first use: the static fee, as before
}
