package services

// NEAR funding on the current stack (Development.md Phase 2): users fund and withdraw through a
// platform treasury NEAR account while the DB stays the source of truth. Phase 4 moves custody
// into near-stocks.near (ft_on_transfer) and this service becomes the indexer's mirror.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/nearchain"
	"github/eugenix-io/logx-inf-backend/xclient"
)

type NearConfig struct {
	Network         string // mainnet | testnet
	ContractAccount string // NEP-413 recipient and EIP-712 verifyingContract source
	ChainId         int64
	Treasury        string
	USDC            string
	BrokerId        uint
	WithdrawMaxUSDC *big.Int // per-withdrawal cap in 6-decimal units (hot-wallet guard)
	RPC             *nearchain.Client
	OneClick        *nearchain.OneClick
	Signer          nearchain.Signer // nil when withdrawals are disabled
}

var nearCfg *NearConfig

// NearConfigFromEnv loads and validates NEAR settings once. Mainnet defaults are the verified
// values (near-stocks.near, chain 397, native USDC); testnet requires NEAR_USDC_CONTRACT.
func NearConfigFromEnv() (*NearConfig, error) {
	if nearCfg != nil {
		return nearCfg, nil
	}
	c := &NearConfig{Network: envOr("NEAR_NETWORK", "mainnet")}
	switch c.Network {
	case "mainnet":
		c.ContractAccount = envOr("NEAR_STOCKS_ACCOUNT", contractUtils.NEAR_STOCKS_MAINNET_ACCOUNT)
		c.ChainId = contractUtils.NEAR_STOCKS_MAINNET_CHAIN_ID
		c.USDC = envOr("NEAR_USDC_CONTRACT", nearchain.USDCMainnet)
	case "testnet":
		c.ContractAccount = envOr("NEAR_STOCKS_ACCOUNT", contractUtils.NEAR_STOCKS_TESTNET_ACCOUNT)
		c.ChainId = contractUtils.NEAR_STOCKS_TESTNET_CHAIN_ID
		c.USDC = os.Getenv("NEAR_USDC_CONTRACT")
		if c.USDC == "" {
			return nil, fmt.Errorf("NEAR_USDC_CONTRACT is required on testnet")
		}
	default:
		return nil, fmt.Errorf("NEAR_NETWORK must be mainnet or testnet")
	}
	c.Treasury = os.Getenv("NEAR_TREASURY_ACCOUNT")
	if c.Treasury == "" || !cutils.IsValidNearAccountId(c.Treasury) {
		return nil, fmt.Errorf("NEAR_TREASURY_ACCOUNT must be a valid NEAR account id")
	}
	// Default broker 2: the live frontend registers every user on broker 2 (auth.utils.ts:158).
	broker, err := strconv.ParseUint(envOr("NEAR_BROKER_ID", "2"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("NEAR_BROKER_ID: %w", err)
	}
	c.BrokerId = uint(broker)
	maxUSDC, err := strconv.ParseInt(envOr("NEAR_WITHDRAW_MAX_USDC", "1000"), 10, 64)
	if err != nil || maxUSDC <= 0 {
		return nil, fmt.Errorf("NEAR_WITHDRAW_MAX_USDC must be a positive whole number")
	}
	c.WithdrawMaxUSDC = new(big.Int).Mul(big.NewInt(maxUSDC), big.NewInt(1_000_000))
	rpcURL := os.Getenv("NEAR_RPC_URL")
	if rpcURL == "" {
		return nil, fmt.Errorf("NEAR_RPC_URL is required")
	}
	c.RPC = nearchain.NewClient(rpcURL)
	c.OneClick = nearchain.NewOneClick(os.Getenv("ONECLICK_BASE_URL"), os.Getenv("ONECLICK_API_KEY"))
	if pk := os.Getenv("NEAR_TREASURY_PRIVATE_KEY"); pk != "" {
		s, err := nearchain.ParseNearPrivateKey(pk)
		if err != nil {
			return nil, fmt.Errorf("NEAR_TREASURY_PRIVATE_KEY: %w", err)
		}
		c.Signer = s
	}
	nearCfg = c
	return c, nil
}

// SetNearConfigForTest replaces the loaded config (tests only).
func SetNearConfigForTest(c *NearConfig) { nearCfg = c }

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// USDCAssetId is the 1Click asset id for the configured USDC token.
func (c *NearConfig) USDCAssetId() string { return "nep141:" + c.USDC }

// DepositRecipient is where 1Click delivers deposits: the treasury in Phase 2, the core contract
// with NEAR settlement (it credits in ft_on_transfer and the indexer mirrors it, §7.2).
func (c *NearConfig) DepositRecipient() string {
	if contractUtils.NearSettlement() {
		return c.ContractAccount
	}
	return c.Treasury
}

// DepositRecipientMsg is the ft_transfer_call msg 1Click passes to ft_on_transfer (NEAR settlement
// only; the Phase 2 treasury is a plain account). It names the user's trading subaccount.
func DepositRecipientMsg(nearAccountId string) string {
	if !contractUtils.NearSettlement() {
		return ""
	}
	b, _ := json.Marshal(map[string]any{"account_id": nearAccountId, "subaccount_number": 1, "source": "1click"})
	return string(b)
}

func discord(msg string) {
	xlog.Errorf("%s", msg)
	if dc := xclient.GetGlobalDiscordClient(); dc != nil {
		_ = dc.SendWebhookMessage(msg)
	}
}

// ---------- crediting ----------

var ErrAlreadyCredited = errors.New("already credited")

// CreditNearDeposit credits `amountUSDC` (6 decimals) of quote to the subaccount exactly once
// for `key`. The DB claim happens before the balance update, so a crash can never double
// credit; a balance-server failure is recorded and alerted for manual resolution.
func CreditNearDeposit(key, subaccountId, amountUSDC, nearTxHash string) error {
	amt, ok := new(big.Int).SetString(amountUSDC, 10)
	if !ok || amt.Sign() <= 0 {
		return fmt.Errorf("invalid deposit amount %q", amountUSDC)
	}
	won, err := (&db.IntentsTransferDB{}).ClaimCredit(key, amountUSDC, nearTxHash)
	if err != nil {
		return err
	}
	if !won {
		return ErrAlreadyCredited
	}
	subHex, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		_ = (&db.IntentsTransferDB{}).MarkCreditFailed(key, err.Error())
		return err
	}
	x18 := nearchain.ScaleUSDCToX18(amt).String()
	okBal, errBal := xclient.GlobalBalanceClient.UpdateTokenBalance(subHex, contractUtils.QUOTE_TOKEN_PRODUCT_ID, x18)
	if errBal != nil || !okBal {
		_ = (&db.IntentsTransferDB{}).MarkCreditFailed(key, fmt.Sprint(errBal))
		discord(fmt.Sprintf("NEAR DEPOSIT CREDIT FAILED (manual fix needed): key=%s sub=%s usdc=%s err=%v", key, subHex, amountUSDC, errBal))
		return fmt.Errorf("balance update failed: %v", errBal)
	}
	if err := (&db.DepositWithdrawDB{}).InsertNearTransfer(key, nearTxHash, subHex, x18, contractUtils.QUOTE_TOKEN_PRODUCT_ID, true, uint64(nearCfgChain())); err != nil {
		xlog.Errorf("NEAR deposit history insert failed for %s: %v", key, err) // history only; balance is already correct
	}
	xlog.Infof("NEAR deposit credited: key=%s sub=%s usdc=%s", key, subHex, amountUSDC)
	return nil
}

func nearCfgChain() int64 {
	if nearCfg != nil {
		return nearCfg.ChainId
	}
	return contractUtils.NEAR_STOCKS_MAINNET_CHAIN_ID
}

// ---------- 1Click deposits ----------

// SyncOneClickDeposit refreshes one 1Click deposit and credits it on SUCCESS. The status
// response is cross-checked against our own treasury and asset, so a quote pointing anywhere
// else can never be credited.
func SyncOneClickDeposit(ctx context.Context, row *db.IntentsTransferTable) (string, error) {
	c, err := NearConfigFromEnv()
	if err != nil {
		return "", err
	}
	st, err := c.OneClick.Status(ctx, row.DepositAddress, row.DepositMemo)
	if err != nil {
		return row.Status, err
	}
	qr := st.QuoteResponse.QuoteRequest
	if qr.Recipient != c.DepositRecipient() || qr.DestinationAsset != c.USDCAssetId() || qr.RecipientType != "DESTINATION_CHAIN" {
		msg := fmt.Sprintf("1Click status mismatch for %s: recipient=%s asset=%s type=%s", row.Key, qr.Recipient, qr.DestinationAsset, qr.RecipientType)
		_ = (&db.IntentsTransferDB{}).UpdateStatus(row.Key, "MISMATCH", msg)
		discord(msg)
		return "MISMATCH", errors.New(msg)
	}
	switch st.Status {
	case nearchain.StatusSuccess:
		if contractUtils.NearSettlement() {
			// the contract credited it in ft_on_transfer; the NEAR indexer mirrors the deposit event
			_ = (&db.IntentsTransferDB{}).UpdateStatus(row.Key, nearchain.StatusSuccess, "")
			return nearchain.StatusSuccess, nil
		}
		tx := ""
		if len(st.SwapDetails.DestinationChainTxHashes) > 0 {
			tx = st.SwapDetails.DestinationChainTxHashes[0].Hash
		}
		if err := CreditNearDeposit(row.Key, row.SubaccountId, st.SwapDetails.AmountOut, tx); err != nil && !errors.Is(err, ErrAlreadyCredited) {
			return st.Status, err
		}
		return "SUCCESS_CREDITED", nil
	default:
		_ = (&db.IntentsTransferDB{}).UpdateStatus(row.Key, st.Status, st.SwapDetails.RefundReason)
		return st.Status, nil
	}
}

// StartOneClickPoller polls pending 1Click deposits. Crediting is idempotent, so running more
// than one poller is safe; the Redis lock just avoids duplicate API traffic.
func StartOneClickPoller() {
	go func() {
		for range time.Tick(15 * time.Second) {
			_, _ = xredis.WithRedisLock("near:oneclick:poller", func() (*xredis.NOOP, error) {
				rows := (&db.IntentsTransferDB{}).Pending("1click", "deposit", time.Now().Add(-10*time.Second), 100)
				for i := range rows {
					ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
					if _, err := SyncOneClickDeposit(ctx, &rows[i]); err != nil {
						xlog.Errorf("1Click sync %s: %v", rows[i].Key, err)
					}
					cancel()
				}
				return nil, nil
			})
		}
	}()
}

// ---------- payouts ----------

// PayoutStatus values stored on withdraw rows.
const (
	PayoutSent     = "PAYOUT_SENT"
	PayoutFailed   = "PAYOUT_FAILED_REFUNDED"
	PayoutUnknown  = "PAYOUT_UNKNOWN"
	PayoutPending  = "PAYOUT_PENDING"
	treasuryLockID = "near:treasury:payout"
)

// SendTreasuryPayout pays `usdc` (6 decimals) to `receiver`, serialized on the treasury key.
// On a definite on-chain failure it re-credits the user's ledger; on an ambiguous error
// (timeout, RPC down) it does NOT refund, because the transaction may still land.
func SendTreasuryPayout(key, subHex string, debitX18 *big.Int, receiver string, usdc *big.Int) (string, string, error) {
	c, err := NearConfigFromEnv()
	if err != nil {
		return "", "", err
	}
	if c.Signer == nil {
		return "", "", fmt.Errorf("treasury signer not configured")
	}
	type result struct {
		status, hash string
		err          error
	}
	res, lockErr := xredis.WithRedisLock(treasuryLockID, func() (*result, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		tx, hash, err := c.RPC.Payout(ctx, c.Signer, c.Treasury, c.USDC, receiver, usdc)
		hashStr := nearchain.HashString(hash)
		switch {
		case err == nil && tx.Succeeded():
			return &result{PayoutSent, hashStr, nil}, nil
		case err == nil:
			return &result{PayoutFailed, hashStr, fmt.Errorf("payout transaction failed: %s", statusString(tx))}, nil
		case hash == [32]byte{}:
			return &result{PayoutFailed, "", err}, nil // failed before signing: nothing was broadcast
		case strings.Contains(err.Error(), "INVALID_TRANSACTION"):
			return &result{PayoutFailed, hashStr, err}, nil // node rejected it before execution: safe to refund
		default:
			return &result{PayoutUnknown, hashStr, err}, nil
		}
	})
	if lockErr != nil {
		return PayoutUnknown, "", lockErr
	}
	switch res.status {
	case PayoutSent:
		_ = (&db.IntentsTransferDB{}).SetWithdrawResult(key, PayoutSent, res.hash, "")
		// D-7: the withdrawal fee plus sub-unit dust (debit - payout*1e12) stays in the ledger,
		// in the fee account, exactly as the NEAR contract's withdraw callback does.
		if kept := new(big.Int).Sub(debitX18, new(big.Int).Mul(usdc, big.NewInt(1_000_000_000_000))); kept.Sign() > 0 {
			ok, err := xclient.GlobalBalanceClient.UpdateTokenBalance(contractUtils.TRADING_FEES_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, kept.String())
			if err != nil || !ok {
				discord(fmt.Sprintf("NEAR withdrawal fee not credited to the fee account (manual fix): key=%s x18=%s err=%v", key, kept, err))
			}
		}
	case PayoutFailed:
		ok, err := xclient.GlobalBalanceClient.UpdateTokenBalance(subHex, contractUtils.QUOTE_TOKEN_PRODUCT_ID, debitX18.String())
		if err != nil || !ok {
			discord(fmt.Sprintf("NEAR PAYOUT FAILED AND REFUND FAILED (manual fix): key=%s sub=%s x18=%s err=%v", key, subHex, debitX18, err))
		}
		_ = (&db.IntentsTransferDB{}).SetWithdrawResult(key, PayoutFailed, res.hash, fmt.Sprint(res.err))
	case PayoutUnknown:
		_ = (&db.IntentsTransferDB{}).SetWithdrawResult(key, PayoutUnknown, res.hash, fmt.Sprint(res.err))
		discord(fmt.Sprintf("NEAR PAYOUT OUTCOME UNKNOWN (check tx %s before any refund): key=%s sub=%s", res.hash, key, subHex))
	}
	return res.status, res.hash, res.err
}

func statusString(tx *nearchain.TxResult) string {
	b, _ := json.Marshal(tx.Status)
	return string(b)
}
