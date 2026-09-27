package main

// flows drives the user flows through the running local stack (api-server, engine, balance-server,
// cron-server batcher, indexer) against near-stocks.testnet, the way the frontend does, and checks
// after each step that the contract and the backend ledgers agree (Development.md §15, gate G4).
// Run it with ~/.near-stocks-testnet/common.env loaded (it reads Redis like the reconciler does).
//
//	nstestnet flows login                  both test users: 409 before registration, register, sign in
//	nstestnet flows deposit <user> <usdc>  ft_transfer_call USDC into near-stocks, wait for the indexer
//	nstestnet flows fund-user2 <usdc>      user1 sends user2 wallet USDC (if only user1 used the faucet)
//	nstestnet flows amm-setup <usdc>       DAO registers the AMM session key and funds it ({"system":"amm"})
//	nstestnet flows trade <user> buy|sell <amountEth>  the user takes an AMM quote on ETH-PERP
//	nstestnet flows withdraw <user> <usdc> withdraw USDC to the user's wallet through /near/withdraw
//	nstestnet flows check                  chain vs backend for both users

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/nearchain"
	"github/eugenix-io/logx-inf-backend/services/cron-server/nearreconcile"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const apiURL = "http://localhost:8080/api/v1"

var flowUsers = []string{"near-stocks-user1.testnet", "near-stocks-user2.testnet"}

// session is one test user's browser state: the session key and API credentials (test only,
// stored 0600 next to the stack, never printed).
type session struct {
	Account      string `json:"account"`
	SessionKey   string `json:"session_key"` // hex secp256k1
	Expiry       int64  `json:"expiry"`
	BrokerKey    string `json:"broker_key"`
	BrokerSecret string `json:"broker_secret"`
	SubaccountId string `json:"subaccount_id"`
	BrokerId     uint   `json:"broker_id,omitempty"` // 0 = the NEAR broker
}

func sessionPath(account string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".near-stocks-testnet", "sessions", account+".json")
}

func loadSession(account string) (*session, error) {
	raw, err := os.ReadFile(sessionPath(account))
	if err != nil {
		return nil, err
	}
	var s session
	return &s, json.Unmarshal(raw, &s)
}

func (s *session) save() error {
	if err := os.MkdirAll(filepath.Dir(sessionPath(s.Account)), 0o700); err != nil {
		return err
	}
	raw, _ := json.Marshal(s)
	return os.WriteFile(sessionPath(s.Account), raw, 0o600)
}

func (s *session) key() *ecdsa.PrivateKey {
	k, err := crypto.HexToECDSA(s.SessionKey)
	must(err)
	return k
}

func (s *session) addr() string { return crypto.PubkeyToAddress(s.key().PublicKey).Hex() }

func (s *session) subHex() string {
	h, err := cutils.SubaccountIdToHex(s.SubaccountId)
	must(err)
	return strings.ToLower(h)
}

func (s *session) sign(h []byte) string {
	sig, err := crypto.Sign(h, s.key())
	must(err)
	sig[64] += 27
	return hexutil.Encode(sig)
}

// api calls the api-server; the session's credentials are sent when s != nil.
func api(method, path string, s *session, body any, extra map[string]string) (int, map[string]any) {
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, apiURL+path, rd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Broker-Id", fmt.Sprint(brokerId))
	if s != nil && s.BrokerId != 0 {
		req.Header.Set("Broker-Id", fmt.Sprint(s.BrokerId))
	}
	if s != nil {
		req.Header.Set("Broker-Key", s.BrokerKey)
		req.Header.Set("Broker-Secret", s.BrokerSecret)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil {
		out = map[string]any{"raw": string(raw)}
	}
	return resp.StatusCode, out
}

func expect(ok bool, format string, a ...any) {
	if !ok {
		fmt.Printf("  FAIL "+format+"\n", a...)
		os.Exit(1)
	}
	fmt.Printf("  ok   "+format+"\n", a...)
}

func onChainExpiry(ctx context.Context, rpc *nearchain.Client, subHex, key string) uint64 {
	raw, err := rpc.CallView(ctx, coreAccount, "session_key_expiry", map[string]any{"subaccount": subHex, "session_key": strings.ToLower(key)})
	must(err)
	var v uint64
	must(json.Unmarshal(raw, &v))
	return v
}

func nearLogin(s *session, userKey *nearchain.LocalSigner) (int, map[string]any) {
	code, ch := api("GET", "/auth/near/challenge", nil, nil, nil)
	if code != 200 {
		return code, ch
	}
	nonce, _ := base64.StdEncoding.DecodeString(ch["nonce"].(string))
	var n [32]byte
	copy(n[:], nonce)
	msg := fmt.Sprintf(`{"action":"login","sessionKey":"%s","expiry":%d,"ts":%d}`, s.addr(), s.Expiry, time.Now().UnixMilli())
	h := contractUtils.NEP413Hash(contractUtils.NEP413Payload{Message: msg, Nonce: n, Recipient: coreAccount})
	sig, _ := userKey.Sign(h)
	addr20, _ := cutils.NearAccountToAddr20(s.Account)
	regHash, err := contractUtils.NearRegisterHash(coreAccount, chainIdTestnet, contractUtils.NearRegister{SubaccountId: s.SubaccountId, UserAddress: addr20, SessionKey: s.addr(), ExpiryTs: s.Expiry})
	must(err)
	return api("POST", "/auth/near", nil, map[string]any{
		"accountId": s.Account, "publicKey": nearchain.PublicKeyString(userKey.PublicKey()),
		"signature": base64.StdEncoding.EncodeToString(sig), "message": msg, "nonce": ch["nonce"],
		"sessionKey": s.addr(), "sessionSignature": s.sign(regHash), "expiryTs": s.Expiry,
	}, nil)
}

func flowLogin(ctx context.Context, rpc *nearchain.Client) {
	for _, u := range flowUsers {
		fmt.Printf("login %s\n", u)
		k, _ := crypto.GenerateKey()
		addr20, _ := cutils.NearAccountToAddr20(u)
		s := &session{Account: u, SessionKey: hex.EncodeToString(crypto.FromECDSA(k)), Expiry: time.Now().Add(6 * 24 * time.Hour).UnixMilli(),
			SubaccountId: cutils.CreateSubaccountId(brokerId, addr20, 1)}
		userKey, err := signer(u)
		must(err)

		code, body := nearLogin(s, userKey)
		expect(code == 409 && body["needsRegistration"] == true, "sign-in before on-chain registration is refused with needsRegistration (%d)", code)

		deposit := new(big.Int).Mul(big.NewInt(5), new(big.Int).Exp(big.NewInt(10), big.NewInt(22), nil))
		res, err := send(ctx, rpc, u, coreAccount, call("register_session_key",
			map[string]any{"subaccount_number": 1, "session_key": strings.ToLower(s.addr()), "expiry_ms": s.Expiry}, 30, deposit))
		must(err)
		fmt.Printf("  register_session_key tx %s\n", res.Transaction.Hash)
		expect(onChainExpiry(ctx, rpc, s.subHex(), s.addr()) == uint64(s.Expiry), "session key registered on-chain until %d", s.Expiry)

		code, body = nearLogin(s, userKey)
		data, _ := body["body"].(map[string]any)
		expect(code == 200 && data != nil && data["subaccountId"] == s.SubaccountId, "signed in as %s (%d)", s.SubaccountId, code)
		s.BrokerKey, _ = data["broker_key"].(string)
		s.BrokerSecret, _ = data["broker_secret"].(string)
		must(s.save())

		code, acct := api("GET", "/near/account", s, nil, nil)
		expect(code == 200 && acct["accountId"] == u, "/near/account links %s", u)
	}
}

func usdcUnits(usdc string) *big.Int {
	f, ok := new(big.Float).SetString(usdc)
	if !ok {
		panic("bad amount " + usdc)
	}
	v, _ := new(big.Float).Mul(f, big.NewFloat(1e6)).Int(nil)
	return v
}

// ledgers returns (chain, backend) snapshots of a subaccount.
func ledgers(ctx context.Context, rpc *nearchain.Client, subHex string) (nearreconcile.Snapshot, nearreconcile.Snapshot) {
	raw, err := rpc.CallView(ctx, coreAccount, "get_subaccount", map[string]any{"subaccount": subHex})
	must(err)
	chain, err := nearreconcile.FromChain(raw)
	must(err)
	b := subaccount.NewSubaccountBalanceImpl().MustGetSubaccountBalancesFromIds([]string{subHex})
	return chain, nearreconcile.FromBackend(b[0])
}

func fmtX18(v *big.Int) string {
	if v == nil {
		return "0"
	}
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(v), new(big.Float).SetInt(e18)).Float64()
	return fmt.Sprintf("%.6f", f)
}

// waitAgree polls until the chain and backend agree and cond(chain) holds.
func waitAgree(ctx context.Context, rpc *nearchain.Client, s *session, what string, cond func(nearreconcile.Snapshot) bool) nearreconcile.Snapshot {
	deadline := time.Now().Add(90 * time.Second)
	for {
		chain, backend := ledgers(ctx, rpc, s.subHex())
		d := nearreconcile.Diff(chain, backend)
		if len(d) == 0 && cond(chain) {
			expect(true, "%s: chain == backend (USDC %s)", what, fmtX18(chain.Spots[4]))
			return chain
		}
		if time.Now().After(deadline) {
			expect(false, "%s: chain and backend differ or condition not met after 90s: %v", what, d)
		}
		time.Sleep(3 * time.Second)
	}
}

func mustSession(u string) *session {
	s, err := loadSession(u)
	if err != nil {
		fmt.Printf("no session for %s: run `flows login` first\n", u)
		os.Exit(1)
	}
	return s
}

func flowDeposit(ctx context.Context, rpc *nearchain.Client, u, usdc string) {
	s := mustSession(u)
	amount := usdcUnits(usdc)
	fmt.Printf("deposit %s USDC from %s (wallet has %s units)\n", usdc, u, ftBalance(ctx, rpc, usdcTestnet, u))
	before, _ := ledgers(ctx, rpc, s.subHex())
	res, err := send(ctx, rpc, u, usdcTestnet, call("ft_transfer_call", map[string]any{"receiver_id": coreAccount, "amount": amount.String(), "msg": ""}, 100, big.NewInt(1)))
	must(err)
	fmt.Printf("  ft_transfer_call tx %s\n", res.Transaction.Hash)
	want := new(big.Int).Add(nz(before.Spots[4]), new(big.Int).Mul(amount, big.NewInt(1e12)))
	waitAgree(ctx, rpc, s, "deposit mirrored by the indexer", func(c nearreconcile.Snapshot) bool { return nz(c.Spots[4]).Cmp(want) == 0 })
	code, tr := api("GET", "/near/transfers", s, nil, nil)
	expect(code == 200, "/near/transfers lists the deposit: %v", short(tr))
}

func nz(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return v
}

func short(v any) string {
	raw, _ := json.Marshal(v)
	if len(raw) > 300 {
		return string(raw[:300]) + "..."
	}
	return string(raw)
}

// oraclePrice reads the stack's oracle (ORACLE_SERVER_URL), the price the backend and the price tick use.
func oraclePrice(symbol string) *big.Int {
	resp, err := http.Get(os.Getenv("ORACLE_SERVER_URL") + "/prices?tokens=" + symbol)
	must(err)
	defer resp.Body.Close()
	var v struct {
		Data map[string]struct {
			Price struct{ Hex string } `json:"price"`
			Expo  int64                `json:"expo"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&v))
	e, ok := v.Data[symbol]
	if !ok {
		panic("oracle has no price for " + symbol)
	}
	p, _ := new(big.Int).SetString(strings.TrimPrefix(e.Price.Hex, "0x"), 16)
	return new(big.Int).Mul(p, new(big.Int).Exp(big.NewInt(10), big.NewInt(18+e.Expo), nil))
}

func placeOrder(s *session, marketId uint32, isBuy bool, amount, price string, orderType string, party string) (int, map[string]any) {
	expiry := uint64(time.Now().Add(time.Hour).UnixMilli())
	amt := cutils.FloatStrToX18(amount)
	px := cutils.FloatStrToX18(price)
	signed := new(big.Int).Set(amt)
	if !isBuy {
		signed.Neg(signed)
	}
	sub, _ := cutils.SubaccountIdToBytes32(s.SubaccountId)
	h, err := contractUtils.NearOrderDigest(coreAccount, chainIdTestnet, sub, px, signed, expiry, false, s.addr(), marketId)
	must(err)
	return api("POST", "/order", s, map[string]any{
		"marketId": marketId, "isBuy": isBuy, "orderType": orderType, "amount": amount, "price": price,
		"expiryTs": expiry, "party": party, "isReduce": false,
	}, map[string]string{"Broker-Signature": s.sign(h), "Broker-Signer-Address": s.addr()})
}

// flowTrade: the AMM (the only SOLVER, AMM_SUBACCOUNT_ID_1) rests a limit order at the oracle price
// and the user takes it as a TRADER — the book only matches TRADER against SOLVER.
func flowTrade(ctx context.Context, rpc *nearchain.Client, u string, userBuys bool, amountEth string) {
	flowTradeMarket(ctx, rpc, 1, "ETH", u, userBuys, amountEth)
}

// flowTradeMarket takes an AMM quote on any listed perp (marketId), e.g. a stock like TSLA (103).
func flowTradeMarket(ctx context.Context, rpc *nearchain.Client, marketId uint32, symbol, u string, userBuys bool, amount string) {
	s, amm := mustSession(u), mustSession("amm")
	p := oraclePrice(symbol)
	price := new(big.Int).Quo(p, e18).String() // whole dollars: a multiple of every price quantum
	fmt.Printf("trade: %s %s %s %s-PERP at %s against the AMM\n", u, map[bool]string{true: "buys", false: "sells"}[userBuys], amount, symbol, price)
	bu, _ := ledgers(ctx, rpc, s.subHex())
	ba, _ := ledgers(ctx, rpc, amm.subHex())

	code, body := placeOrder(amm, marketId, !userBuys, amount, price, "LIMIT", "SOLVER")
	expect(code == 200, "AMM maker order accepted (%d %s)", code, short(body))
	code, body = placeOrder(s, marketId, userBuys, amount, price, "LIMIT", "TRADER")
	expect(code == 200, "user taker order accepted (%d %s)", code, short(body))

	amt := cutils.FloatStrToX18(amount)
	sign := int64(1)
	if !userBuys {
		sign = -1
	}
	want := func(before nearreconcile.Snapshot, sg int64) *big.Int {
		return new(big.Int).Add(nz(before.Perps[marketId][0]), new(big.Int).Mul(amt, big.NewInt(sg)))
	}
	cu := waitAgree(ctx, rpc, s, "user fill settled on-chain", func(c nearreconcile.Snapshot) bool { return nz(c.Perps[marketId][0]).Cmp(want(bu, sign)) == 0 })
	ca := waitAgree(ctx, rpc, amm, "AMM fill settled on-chain", func(c nearreconcile.Snapshot) bool { return nz(c.Perps[marketId][0]).Cmp(want(ba, -sign)) == 0 })
	fmt.Printf("  user %s-PERP %s vQuote %s | AMM %s-PERP %s vQuote %s\n", symbol, fmtX18(cu.Perps[marketId][0]), fmtX18(cu.Perps[marketId][1]), symbol, fmtX18(ca.Perps[marketId][0]), fmtX18(ca.Perps[marketId][1]))
	fee, _ := ledgers(ctx, rpc, feeSub)
	fmt.Printf("  fee account USDC %s\n", fmtX18(fee.Spots[4]))
}

// flowTakeLiveQuote places only a TRADER order (unlike flowTradeMarket, it does not also place a
// paired SOLVER order) to verify an independently-running amm service is quoting live: the order
// must fill against whatever the amm is already resting in the book on its own.
func flowTakeLiveQuote(ctx context.Context, rpc *nearchain.Client, marketId uint32, symbol, u string, userBuys bool, amount string) {
	s := mustSession(u)
	p := oraclePrice(symbol)
	adj := new(big.Int).Div(p, big.NewInt(100)) // cross by 1% to guarantee a marketable price
	if userBuys {
		p = new(big.Int).Add(p, adj)
	} else {
		p = new(big.Int).Sub(p, adj)
	}
	price := new(big.Int).Quo(p, e18).String()
	fmt.Printf("take-live-quote: %s %s %s %s-PERP at %s (crossing the live amm's resting quote)\n", u, map[bool]string{true: "buys", false: "sells"}[userBuys], amount, symbol, price)
	code, body := placeOrder(s, marketId, userBuys, amount, price, "LIMIT", "TRADER")
	fmt.Printf("  order result: %d %s\n", code, short(body))
}

// stockPerp is a US stock/ETF this driver can list on near-stocks as a perpetual. Product ids and
// symbols match contractUtils.PRODUCT_ID_SYMBOL_TO_MAP; only extend nsoracle's stockSymbols (Yahoo
// Finance) or point ORACLE_SERVER_URL at a real feed before listing a new one, or its price tick
// will go stale immediately.
type stockPerp struct {
	id uint32
	// symbol is the internal oracle-lookup key (must match contractUtils.PRODUCT_ID_SYMBOL_TO_MAP);
	// display is the clean ticker for the market row and UI (they differ for "Ostrich"-suffixed ids).
	symbol, display, name string
}

var usStocks = []stockPerp{
	{103, "TSLA", "TSLA", "Tesla"},
	{105, "NVDA", "NVDA", "Nvidia"},
	{107, "AAPL", "AAPL", "Apple"},
	{109, "QQQ", "QQQ", "Nasdaq-100 ETF"},
	{165, "SPY_OSTRICH", "SPY", "S&P 500 ETF"},
	{125, "COIN_OSTRICH", "COIN", "Coinbase"},
	{127, "GOOGL_OSTRICH", "GOOGL", "Alphabet"},
	{129, "MSFT_OSTRICH", "MSFT", "Microsoft"},
	{131, "AMZN_OSTRICH", "AMZN", "Amazon"},
	{133, "META_OSTRICH", "META", "Meta Platforms"},
	{135, "MSTR_OSTRICH", "MSTR", "Strategy (MicroStrategy)"},
	{139, "PLTR_OSTRICH", "PLTR", "Palantir"},
	{141, "HOOD_OSTRICH", "HOOD", "Robinhood"},
	{151, "CRCL_OSTRICH", "CRCL", "Circle Internet Group"},
}

// flowListStocks lists usStocks on-chain (DAO upsert_perp, same margins as ETH/BTC/SOL) and mirrors
// each as a MarketTable row + OI caps in the backend, exactly like the crypto perps' original setup.
func flowListStocks(ctx context.Context, rpc *nearchain.Client) {
	contractUtils.Init()
	db.Init()
	x := func(num, den int64) string {
		return new(big.Int).Div(new(big.Int).Mul(big.NewInt(num), e18), big.NewInt(den)).String()
	}
	var actions []nearchain.Action
	for _, m := range usStocks {
		price := oraclePrice(m.symbol)
		actions = append(actions, call("upsert_perp", map[string]any{"product_id": m.id, "config": map[string]any{
			"imf_x18": x(1, 10), "mmf_x18": x(1, 20), "liq_frac_x18": x(15, 1000), "price_x18": price.String(),
			"max_deviation_bps": 1000, "amm_max_position_x18": "0"}}, 20, new(big.Int)))
		fmt.Printf("  %s (%s): listing at $%s\n", m.display, m.name, new(big.Int).Quo(price, e18))
	}
	res, err := send(ctx, rpc, daoAccount, coreAccount, actions...)
	must(err)
	fmt.Printf("listed %d stocks on-chain, tx %s\n", len(usStocks), res.Transaction.Hash)

	r := xredis.GetRedisClient()
	for _, m := range usStocks {
		if (&db.MarketDB{}).GetById(uint(m.id)) == nil {
			mkt := (&db.MarketDB{}).Create(&db.MarketTable{
				BaseTable: db.BaseTable{ID: uint(m.id)}, Symbol: m.display + "-USD", Type: ctypes.PERPETUAL,
				AmtToQtmConversionExpo: 9, PriceToQtmConversionExpo: 6,
				MaxPositionValuex18: ctypes.NewBigIntFromString(new(big.Int).Mul(big.NewInt(1_000_000), e18).String()),
				MinAmountx18:        ctypes.NewBigIntFromString("1000000000000000"), // 0.001 shares
				BaseAsset:           m.display, QuoteAsset: "USDC", IsActive: true,
				MakerFeeFractionx18:          ctypes.NewBigIntFromString(x(2, 10000)),
				TakerFeeFractionx18:          ctypes.NewBigIntFromString(x(5, 10000)),
				InitialMarginFractionx18:     ctypes.NewBigIntFromString(x(1, 10)),
				MaintenanceMarginFractionx18: ctypes.NewBigIntFromString(x(1, 20)),
			})
			fmt.Printf("  backend market row created: id %d %s\n", mkt.ID, mkt.Symbol)
		}
		pid := fmt.Sprint(m.id)
		must(r.Set(ctx, "MARKET_LEVEL_LONG_OI_CAP_"+pid, "1000000", 0).Err())
		must(r.Set(ctx, "MARKET_LEVEL_SHORT_OI_CAP_"+pid, "1000000", 0).Err())
		must(r.Set(ctx, "SUBACCOUNT_MARKET_CAP_"+pid, "100000", 0).Err())
	}
	fmt.Println("done: trade with `flows trade-market <productId> <symbol> <user> buy|sell <amount>`")
}

// flowAMMSetup onboards the AMM on NEAR: a DAO-registered session key (register_system_session_key),
// DAO-funded collateral (ft_transfer_call msg {"system":"amm"}), and backend API credentials.
func flowAMMSetup(ctx context.Context, rpc *nearchain.Client, usdc string) {
	amm, err := loadSession("amm")
	if err != nil {
		k, _ := crypto.GenerateKey()
		amm = &session{Account: "amm", SessionKey: hex.EncodeToString(crypto.FromECDSA(k)), Expiry: time.Now().Add(6 * 24 * time.Hour).UnixMilli(),
			SubaccountId: contractUtils.AMM_SUBACCOUNT_ID_1, BrokerId: 1}
		res, err := send(ctx, rpc, daoAccount, coreAccount, call("register_system_session_key",
			map[string]any{"subaccount": amm.subHex(), "session_key": strings.ToLower(amm.addr()), "expiry_ms": amm.Expiry}, 30, new(big.Int)))
		must(err)
		fmt.Printf("  register_system_session_key tx %s\n", res.Transaction.Hash)

		db.Init()
		if (&db.SubaccountDB{}).GetByIdBrokerId(1, amm.SubaccountId) == nil {
			_, err := (&db.SubaccountDB{}).Create("0x0000000000000000000000000000000000000001", 1, amm.SubaccountId)
			must(err)
		}
		must((&db.SigningKeyDB{}).Create(amm.addr(), amm.SubaccountId, 1, uint64(amm.Expiry)))
		// the API secret is a secp256k1 private key and its stored hash the public key (GenerateApiSecretAndHash)
		sk, _ := crypto.GenerateKey()
		amm.BrokerKey = fmt.Sprintf("%x", crypto.Keccak256([]byte(amm.SessionKey)))[:32] // the API key is a 32-hex id
		amm.BrokerSecret = hexutil.Encode(crypto.FromECDSA(sk))[2:]
		must((&db.AuthDB{}).Create(amm.BrokerKey, hexutil.Encode(crypto.FromECDSAPub(&sk.PublicKey))[2:], amm.SubaccountId, uint64(amm.Expiry)))
		must(amm.save())
	}
	expect(onChainExpiry(ctx, rpc, amm.subHex(), amm.addr()) == uint64(amm.Expiry), "AMM session key registered on-chain")

	units := usdcUnits(usdc)
	storage := new(big.Int).Mul(big.NewInt(125), new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil))
	_, err = send(ctx, rpc, daoAccount, usdcTestnet, call("storage_deposit", map[string]any{"account_id": daoAccount, "registration_only": true}, 30, storage))
	must(err)
	half := new(big.Int).Quo(units, big.NewInt(2))
	for i, u := range flowUsers { // the faucet only pays test wallets: they lend the DAO the AMM's capital
		amt := half
		if i == 1 {
			amt = new(big.Int).Sub(units, half)
		}
		_, err := send(ctx, rpc, u, usdcTestnet, call("ft_transfer", map[string]any{"receiver_id": daoAccount, "amount": amt.String()}, 30, big.NewInt(1)))
		must(err)
	}
	before, _ := ledgers(ctx, rpc, amm.subHex())
	res, err := send(ctx, rpc, daoAccount, usdcTestnet, call("ft_transfer_call", map[string]any{"receiver_id": coreAccount, "amount": units.String(), "msg": `{"system":"amm"}`}, 100, big.NewInt(1)))
	must(err)
	fmt.Printf("  DAO funded the AMM with %s USDC, tx %s\n", usdc, res.Transaction.Hash)
	want := new(big.Int).Add(nz(before.Spots[4]), new(big.Int).Mul(units, big.NewInt(1e12)))
	waitAgree(ctx, rpc, amm, "AMM deposit mirrored", func(c nearreconcile.Snapshot) bool { return nz(c.Spots[4]).Cmp(want) == 0 })

	// a non-owner cannot fund a system subaccount (the contract panics, the FT refunds)
	_, err = send(ctx, rpc, flowUsers[0], usdcTestnet, call("ft_transfer_call", map[string]any{"receiver_id": coreAccount, "amount": "1000000", "msg": `{"system":"amm"}`}, 100, big.NewInt(1)))
	after, _ := ledgers(ctx, rpc, amm.subHex())
	expect(nz(after.Spots[4]).Cmp(want) == 0, "a user's {\"system\":\"amm\"} deposit is refused and refunded (%v)", err)
}

type token struct {
	name     string
	pid      uint32
	contract string
	decimals int64
	fee      *big.Int // x18, deducted from the payout
}

var (
	usdcToken = token{"USDC", 4, usdcTestnet, 6, new(big.Int).Div(e18, big.NewInt(2))}
	logxToken = token{"LogX", 0, logxAccount, 18, new(big.Int).Mul(big.NewInt(25), e18)}
)

func flowWithdraw(ctx context.Context, rpc *nearchain.Client, u, usdc string) {
	flowWithdrawToken(ctx, rpc, usdcToken, u, usdc)
}

// flowWithdrawToken withdraws through /near/withdraw and checks the wallet got amount - fee.
func flowWithdrawToken(ctx context.Context, rpc *nearchain.Client, t token, u, amountStr string) {
	s := mustSession(u)
	amount := cutils.FloatStrToX18(amountStr)
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(18-t.decimals), nil)
	walletBefore, _ := new(big.Int).SetString(ftBalance(ctx, rpc, t.contract, u), 10)
	before, _ := ledgers(ctx, rpc, s.subHex())
	nonce := apiNonce(s)
	fmt.Printf("withdraw %s %s to %s\n", amountStr, t.name, u)
	w := contractUtils.NearWithdraw{SubaccountId: s.SubaccountId, SessionKey: s.addr(), ProductId: t.pid, Amount: amount.String(), Nonce: nonce, Receiver: u}
	h, err := contractUtils.NearWithdrawHash(coreAccount, chainIdTestnet, w)
	must(err)
	addr20, _ := cutils.NearAccountToAddr20(u)
	hdr := map[string]string{"Broker-Signature": s.sign(h), "Broker-Signer-Address": s.addr(), "Broker-User-Address": addr20}
	body := map[string]any{"subAccountId": s.SubaccountId, "productId": t.pid, "amount": amount.String(), "nonce": nonce, "receiver": u}

	bad := map[string]any{"subAccountId": s.SubaccountId, "productId": t.pid, "amount": amount.String(), "nonce": nonce, "receiver": "near-stocks-user2.testnet"}
	if u == flowUsers[1] {
		bad["receiver"] = flowUsers[0]
	}
	code, rb := api("POST", "/near/withdraw", s, bad, hdr)
	expect(code == 403, "withdraw to someone else's account is refused (%d %s)", code, short(rb))

	code, rb = api("POST", "/near/withdraw", s, body, hdr)
	expect(code == 200 && rb["status"] == "QUEUED", "withdraw queued (%d %s)", code, short(rb))
	code, rb = api("POST", "/near/withdraw", s, body, hdr)
	expect(code != 200, "replaying the same withdraw is refused (%d)", code)

	want := new(big.Int).Sub(nz(before.Spots[t.pid]), amount)
	waitAgree(ctx, rpc, s, "withdraw settled on-chain", func(c nearreconcile.Snapshot) bool { return nz(c.Spots[t.pid]).Cmp(want) == 0 })
	payout := new(big.Int).Div(new(big.Int).Sub(amount, t.fee), scale)
	deadline := time.Now().Add(60 * time.Second)
	for {
		now, _ := new(big.Int).SetString(ftBalance(ctx, rpc, t.contract, u), 10)
		if now != nil && new(big.Int).Sub(now, walletBefore).Cmp(payout) == 0 {
			expect(true, "wallet received %s %s units (amount minus the %s fee)", payout, t.name, fmtX18(t.fee))
			break
		}
		if time.Now().After(deadline) {
			expect(false, "wallet did not receive %s units (before %s, now %s)", payout, walletBefore, now)
		}
		time.Sleep(3 * time.Second)
	}
}

var optionBetType = []apitypes.Type{
	{Name: "subAccountId", Type: "bytes32"}, {Name: "productId", Type: "uint32"}, {Name: "amount", Type: "int128"},
	{Name: "interval", Type: "uint32"}, {Name: "nonce", Type: "uint128"}, {Name: "sessionKey", Type: "address"}, {Name: "chainId", Type: "uint256"},
}

func apiNonce(s *session) int64 {
	code, nb := api("GET", "/subaccount/nonce/"+s.subHex(), nil, nil, nil)
	expect(code == 200, "nonce %v", nb["body"])
	var n int64
	fmt.Sscan(fmt.Sprint(nb["body"].(map[string]any)["nonce"]), &n)
	return n
}

// flowOption places a 1-minute ETH option (amount > 0 bets up) and waits for the close cron to settle it.
func flowOption(ctx context.Context, rpc *nearchain.Client, u, amountEth string) {
	s := mustSession(u)
	fmt.Printf("option: %s bets %s ETH (1 minute)\n", u, amountEth)
	before, _ := ledgers(ctx, rpc, s.subHex())
	optX, _ := ledgers(ctx, rpc, optionsX)
	n := apiNonce(s)
	amt := cutils.FloatStrToX18(amountEth)
	sub, _ := cutils.SubaccountIdToBytes32(s.SubaccountId)
	h, err := contractUtils.NearTypedHash(coreAccount, chainIdTestnet, "UserOptionBet", optionBetType, apitypes.TypedDataMessage{
		"subAccountId": sub[:], "productId": math.NewHexOrDecimal256(1), "amount": amt.String(), "interval": math.NewHexOrDecimal256(1),
		"nonce": math.NewHexOrDecimal256(n), "sessionKey": s.addr(), "chainId": math.NewHexOrDecimal256(chainIdTestnet)})
	must(err)
	addr20, _ := cutils.NearAccountToAddr20(u)
	sig := s.sign(h)
	code, body := api("POST", "/options/placeBet", s, map[string]any{"subAccountId": s.SubaccountId, "productId": 1, "amount": amt.String(),
		"interval": 1, "nonce": n, "sessionKey": s.addr(), "chainId": chainIdTestnet},
		map[string]string{"Logx-Signature": sig, "Logx-Signer-Address": s.addr(), "Broker-Signer-Address": s.addr(), "Broker-User-Address": addr20})
	expect(code == 200, "bet accepted (%d %s)", code, short(body))
	placed := waitAgree(ctx, rpc, s, "stake debited on-chain", func(c nearreconcile.Snapshot) bool { return nz(c.Spots[4]).Cmp(nz(before.Spots[4])) < 0 })
	stake := new(big.Int).Sub(nz(before.Spots[4]), nz(placed.Spots[4]))
	fmt.Printf("  stake %s USDC\n", fmtX18(stake))
	x, xb := ledgers(ctx, rpc, optionsX)
	expect(len(nearreconcile.Diff(x, xb)) == 0 && new(big.Int).Sub(nz(x.Spots[4]), nz(optX.Spots[4])).Cmp(stake) == 0, "options house received the stake, chain == backend")
	fmt.Println("  waiting for expiry and the close cron...")
	deadline := time.Now().Add(4 * time.Minute)
	for {
		c, b := ledgers(ctx, rpc, s.subHex())
		x, xb := ledgers(ctx, rpc, optionsX)
		if nz(c.Spots[4]).Cmp(nz(placed.Spots[4])) != 0 && len(nearreconcile.Diff(c, b)) == 0 && len(nearreconcile.Diff(x, xb)) == 0 {
			expect(true, "bet WON and settled: user USDC %s (payout %s), chain == backend", fmtX18(c.Spots[4]), fmtX18(new(big.Int).Sub(nz(c.Spots[4]), nz(placed.Spots[4]))))
			return
		}
		if time.Now().After(deadline) {
			// a lost bet leaves the balance unchanged: confirm it is gone from the contract
			raw, err := rpc.CallView(ctx, coreAccount, "get_subaccount", map[string]any{"subaccount": s.subHex()})
			must(err)
			expect(len(nearreconcile.Diff(c, b)) == 0 && len(nearreconcile.Diff(x, xb)) == 0, "bet closed as LOST (or not yet closed): chain == backend %s", short(json.RawMessage(raw)))
			return
		}
		time.Sleep(5 * time.Second)
	}
}

// flowStake stakes then unstakes LogX through /token/stake and /token/unstake (D-9: staking lives in
// the core ledger as LogX -> stLogX), then checks R-7: unstaked LogX is not withdrawable during cooldown.
func flowStake(ctx context.Context, rpc *nearchain.Client, u, stake, unstake string) {
	contractUtils.Init()
	s := mustSession(u)
	addr20, _ := cutils.NearAccountToAddr20(u)
	sub, _ := cutils.SubaccountIdToBytes32(s.SubaccountId)
	staker := contractUtils.STAKER_CONTRACT_ADDRESS
	req := func(path, primary string, fields []apitypes.Type, amountField, amount string) (int, map[string]any) {
		n := apiNonce(s)
		amt := new(big.Int).Mul(cutils.FloatStrToX18(amount), big.NewInt(1))
		h, err := contractUtils.NearTypedHash(coreAccount, chainIdTestnet, primary, fields, apitypes.TypedDataMessage{
			"subAccountId": sub[:], "productId": math.NewHexOrDecimal256(int64(contractUtils.ST_LOGX)), amountField: amt.String(), "stakerContract": staker,
			"sessionKey": s.addr(), "nonce": math.NewHexOrDecimal256(n), "chainId": math.NewHexOrDecimal256(chainIdTestnet)})
		must(err)
		sig := s.sign(h)
		return api("POST", path, s, map[string]any{"subAccountId": s.SubaccountId, "productId": contractUtils.ST_LOGX, amountField: amt.String(),
			"stakerContract": staker, "sessionKey": s.addr(), "nonce": n, "chainId": chainIdTestnet},
			map[string]string{"Logx-Signature": sig, "Logx-Signer-Address": s.addr(), "Broker-Signer-Address": s.addr(), "Broker-User-Address": addr20})
	}
	before, _ := ledgers(ctx, rpc, s.subHex())
	fmt.Printf("stake: %s stakes %s LogX (has LogX %s, stLogX %s)\n", u, stake, fmtX18(before.Spots[0]), fmtX18(before.Spots[2]))
	code, body := req("/token/stake", "StakeLogXRequest", contractUtils.NearStakeType, "tokenAmount", stake)
	expect(code == 200, "stake accepted (%d %s)", code, short(body))
	amt := cutils.FloatStrToX18(stake)
	st := waitAgree(ctx, rpc, s, "stake settled on-chain", func(c nearreconcile.Snapshot) bool {
		return nz(c.Spots[2]).Cmp(new(big.Int).Add(nz(before.Spots[2]), amt)) == 0
	})
	fmt.Printf("  LogX %s stLogX %s\n", fmtX18(st.Spots[0]), fmtX18(st.Spots[2]))

	code, body = req("/token/unstake", "UnstakeLogXRequest", contractUtils.NearUnstakeType, "amount", unstake)
	expect(code == 200, "unstake accepted (%d %s)", code, short(body))
	un := cutils.FloatStrToX18(unstake)
	us := waitAgree(ctx, rpc, s, "unstake settled on-chain", func(c nearreconcile.Snapshot) bool {
		return nz(c.Spots[2]).Cmp(new(big.Int).Sub(nz(st.Spots[2]), un)) == 0
	})
	fmt.Printf("  LogX %s stLogX %s\n", fmtX18(us.Spots[0]), fmtX18(us.Spots[2]))

	// R-7: everything except the just-unstaked amount is withdrawable; asking for all of it is refused
	n := apiNonce(s)
	all := nz(us.Spots[0])
	w := contractUtils.NearWithdraw{SubaccountId: s.SubaccountId, SessionKey: s.addr(), ProductId: 0, Amount: all.String(), Nonce: n, Receiver: u}
	h, err := contractUtils.NearWithdrawHash(coreAccount, chainIdTestnet, w)
	must(err)
	code, body = api("POST", "/near/withdraw", s, map[string]any{"subAccountId": s.SubaccountId, "productId": 0, "amount": all.String(), "nonce": n, "receiver": u},
		map[string]string{"Broker-Signature": s.sign(h), "Broker-Signer-Address": s.addr(), "Broker-User-Address": addr20})
	expect(code == 400, "withdrawing unstaked LogX during the cooldown is refused (%d %s)", code, short(body))
}

// flowPreMarketList lists a pre-market product the way ops would: the backend admin API (pool and
// DB row) plus the DAO's set_side_product and add_pool_supply, so both ledgers start with the supply.
func flowPreMarketList(ctx context.Context, rpc *nearchain.Client, pid uint32, supply, price string) {
	r := xredis.GetRedisClient()
	must(r.Set(ctx, xredis.GetPreMarketAdminKey(), "1", 0).Err())
	must(r.HSet(ctx, xredis.GetPreMarketFeesKey(), xredis.GetPreMarketPricingField(pid), "100").Err()) // 1%
	sup, px := cutils.FloatStrToX18(supply), cutils.FloatStrToX18(price)
	code, body := api("POST", "/pre-markets/admin/addProduct", nil, map[string]any{"product_id": pid, "max_supply": sup.String(),
		"closing_timestamp": time.Now().Add(30 * 24 * time.Hour).Unix(), "starting_price": px.String(), "is_enabled": true, "details": "near-stocks testnet"}, nil)
	expect(code == 200, "backend listed pre-market %d (%d %s)", pid, code, short(body))
	res, err := send(ctx, rpc, daoAccount, coreAccount,
		call("set_side_product", map[string]any{"kind": 1, "product_id": pid, "config": map[string]any{"enabled": true, "max_fee": 200, "max_payout_pct": 0}}, 20, new(big.Int)),
		call("add_pool_supply", map[string]any{"kind": 1, "product_id": pid, "amount_x18": sup.String()}, 20, new(big.Int)))
	must(err)
	fmt.Printf("  DAO listed it on-chain and minted %s to the house, tx %s\n", supply, res.Transaction.Hash)
	expect(poolChain(ctx, rpc, 1, preX, pid).Cmp(sup) == 0 && poolBackend(1, preX, pid).Cmp(sup) == 0, "house supply chain == backend (%s)", supply)
}

func poolChain(ctx context.Context, rpc *nearchain.Client, kind uint8, sub string, pid uint32) *big.Int {
	raw, err := rpc.CallView(ctx, coreAccount, "get_pool_balance", map[string]any{"kind": kind, "subaccount": sub, "product_id": pid})
	must(err)
	var v string
	must(json.Unmarshal(raw, &v))
	b, _ := new(big.Int).SetString(v, 10)
	return b
}

func poolBackend(kind uint8, sub string, pid uint32) *big.Int {
	var m map[uint32]string
	var err error
	if kind == 1 {
		m, err = xclient.GlobalBalanceClient.GetPreMarketBalances(sub)
	} else {
		m, err = xclient.GlobalBalanceClient.GetSyntheticSpotBalances(sub)
	}
	must(err)
	b, ok := new(big.Int).SetString(m[pid], 10)
	if !ok {
		return new(big.Int)
	}
	return b
}

// flowPreMarket buys (amount = USDC) or sells (amount = tokens) a pre-market product.
type poolKind struct {
	kind            uint8
	name, typeName  string
	path, x, feeAcc string
}

var (
	preMarketKind = poolKind{1, "pre-market", "PlacePreMarketOrderRequest", "/pre-markets/placeOrder", preX, preFees}
	syntheticKind = poolKind{2, "synthetic spot", "PlaceSyntheticSpotOrderRequest", "/syn-spot/placeOrder", synX, synFees}
)

// flowSynthList enables a synthetic spot product (backend Redis config + DAO set_side_product).
// Synthetic tokens are minted by the pool, so there is no supply to add.
func flowSynthList(ctx context.Context, rpc *nearchain.Client, pid uint32) {
	r := xredis.GetRedisClient()
	f := xredis.GetPreMarketPricingField(pid)
	must(r.HSet(ctx, xredis.GetSyntheticSpotsEnabledKey(), fmt.Sprint(pid), "1").Err())
	must(r.HSet(ctx, xredis.GetSyntheticSpotsFeesKey(), f, "100").Err())    // 1%
	must(r.HSet(ctx, xredis.GetSyntheticSpotsSlippageKey(), f, "50").Err()) // 0.5% (bps added to/taken from the price)
	res, err := send(ctx, rpc, daoAccount, coreAccount, call("set_side_product", map[string]any{"kind": 2, "product_id": pid,
		"config": map[string]any{"enabled": true, "max_fee": 200, "max_payout_pct": 0}}, 20, new(big.Int)))
	must(err)
	fmt.Printf("synthetic spot %d enabled (backend + on-chain), tx %s\n", pid, res.Transaction.Hash)
}

func flowPreMarket(ctx context.Context, rpc *nearchain.Client, u string, pid uint32, isBuy bool, amount string) {
	flowPool(ctx, rpc, preMarketKind, u, pid, isBuy, amount)
}

// flowPool buys (amount = USDC) or sells (amount = tokens) a pre-market or synthetic spot product.
func flowPool(ctx context.Context, rpc *nearchain.Client, k poolKind, u string, pid uint32, isBuy bool, amount string) {
	s := mustSession(u)
	addr20, _ := cutils.NearAccountToAddr20(u)
	sub, _ := cutils.SubaccountIdToBytes32(s.SubaccountId)
	fmt.Printf("%s: %s %s %s of product %d\n", k.name, u, map[bool]string{true: "buys with USDC", false: "sells tokens"}[isBuy], amount, pid)
	before, _ := ledgers(ctx, rpc, s.subHex())
	tokBefore := poolChain(ctx, rpc, k.kind, s.subHex(), pid)
	n := apiNonce(s)
	amt := cutils.FloatStrToX18(amount)
	h, err := contractUtils.NearTypedHash(coreAccount, chainIdTestnet, k.typeName, contractUtils.NearPoolOrderType, apitypes.TypedDataMessage{
		"subAccountId": sub[:], "productId": math.NewHexOrDecimal256(int64(pid)), "amount": amt.String(), "isBuy": isBuy,
		"nonce": math.NewHexOrDecimal256(n), "sessionKey": s.addr(), "chainId": math.NewHexOrDecimal256(chainIdTestnet)})
	must(err)
	sig := s.sign(h)
	code, body := api("POST", k.path, s, map[string]any{"subAccountId": s.SubaccountId, "productId": pid, "amount": amt.String(),
		"isBuy": isBuy, "nonce": n, "sessionKey": s.addr(), "chainId": chainIdTestnet},
		map[string]string{"Logx-Signature": sig, "Logx-Signer-Address": s.addr(), "Broker-Signer-Address": s.addr(), "Broker-User-Address": addr20})
	expect(code == 200, "order accepted (%d %s)", code, short(body))
	c := waitAgree(ctx, rpc, s, "USDC settled on-chain", func(c nearreconcile.Snapshot) bool { return nz(c.Spots[4]).Cmp(nz(before.Spots[4])) != 0 })
	var tc, tb, hc, hb *big.Int
	for deadline := time.Now().Add(60 * time.Second); ; time.Sleep(3 * time.Second) { // token and USDC updates land separately
		tc, tb = poolChain(ctx, rpc, k.kind, s.subHex(), pid), poolBackend(k.kind, s.subHex(), pid)
		hc, hb = poolChain(ctx, rpc, k.kind, k.x, pid), poolBackend(k.kind, k.x, pid)
		if (tc.Cmp(tb) == 0 && hc.Cmp(hb) == 0 && tc.Cmp(tokBefore) != 0) || time.Now().After(deadline) {
			break
		}
	}
	expect(tc.Cmp(tb) == 0 && tc.Cmp(tokBefore) != 0, "user tokens chain == backend (%s)", fmtX18(tc))
	expect(hc.Cmp(hb) == 0, "house tokens chain == backend (%s)", fmtX18(hc))
	x, xb := ledgers(ctx, rpc, k.x)
	f, fb := ledgers(ctx, rpc, k.feeAcc)
	expect(len(nearreconcile.Diff(x, xb)) == 0 && len(nearreconcile.Diff(f, fb)) == 0, "house USDC %s and fees %s chain == backend", fmtX18(x.Spots[4]), fmtX18(f.Spots[4]))
	fmt.Printf("  user USDC %s, tokens %s\n", fmtX18(c.Spots[4]), fmtX18(tc))
}

var rewardsPool = strings.ToLower(contractUtils.LOGX_REWARDS_SUBACCOUNT_ID)

// flowFundRewards: the DAO deposits LogX into the rewards pool claims are paid from.
func flowFundRewards(ctx context.Context, rpc *nearchain.Client, amount string) {
	before, _ := ledgers(ctx, rpc, rewardsPool)
	amt := cutils.FloatStrToX18(amount)
	res, err := send(ctx, rpc, daoAccount, logxAccount, call("ft_transfer_call", map[string]any{"receiver_id": coreAccount, "amount": amt.String(), "msg": `{"system":"rewards"}`}, 100, big.NewInt(1)))
	must(err)
	fmt.Printf("DAO funded the LogX rewards pool with %s, tx %s\n", amount, res.Transaction.Hash)
	want := new(big.Int).Add(nz(before.Spots[0]), amt)
	for deadline := time.Now().Add(10 * time.Minute); ; time.Sleep(10 * time.Second) {
		c, b := ledgers(ctx, rpc, rewardsPool)
		if nz(c.Spots[0]).Cmp(want) == 0 && len(nearreconcile.Diff(c, b)) == 0 {
			expect(true, "rewards pool chain == backend (LogX %s)", fmtX18(c.Spots[0]))
			return
		}
		if time.Now().After(deadline) {
			expect(false, "rewards pool not mirrored: chain %s backend %s", fmtX18(c.Spots[0]), fmtX18(b.Spots[0]))
		}
	}
}

// flowClaimRewards claims a user's staking rewards through /token/claimrewards.
func flowClaimRewards(ctx context.Context, rpc *nearchain.Client, u string) {
	contractUtils.Init()
	s := mustSession(u)
	addr20, _ := cutils.NearAccountToAddr20(u)
	sub, _ := cutils.SubaccountIdToBytes32(s.SubaccountId)
	before, _ := ledgers(ctx, rpc, s.subHex())
	poolBefore, _ := ledgers(ctx, rpc, rewardsPool)
	n := apiNonce(s)
	h, err := contractUtils.NearTypedHash(coreAccount, chainIdTestnet, "ClaimRewards", contractUtils.NearClaimRewardsType, apitypes.TypedDataMessage{
		"subAccountId": sub[:], "sessionKey": s.addr(), "stakerContract": contractUtils.STAKER_CONTRACT_ADDRESS,
		"productId": math.NewHexOrDecimal256(0), "nonce": math.NewHexOrDecimal256(n), "chainId": math.NewHexOrDecimal256(chainIdTestnet)})
	must(err)
	sig := s.sign(h)
	fmt.Printf("claim rewards: %s (LogX %s, stLogX %s)\n", u, fmtX18(before.Spots[0]), fmtX18(before.Spots[2]))
	code, body := api("POST", "/token/claimrewards", s, map[string]any{"subAccountId": s.SubaccountId, "sessionKey": s.addr(),
		"stakerContract": contractUtils.STAKER_CONTRACT_ADDRESS, "productId": 0, "nonce": n, "chainId": chainIdTestnet},
		map[string]string{"Logx-Signature": sig, "Logx-Signer-Address": s.addr(), "Broker-Signer-Address": s.addr(), "Broker-User-Address": addr20})
	expect(code == 200, "claim accepted (%d %s)", code, short(body))
	c := waitAgree(ctx, rpc, s, "claim settled on-chain", func(c nearreconcile.Snapshot) bool { return nz(c.Spots[0]).Cmp(nz(before.Spots[0])) > 0 })
	got := new(big.Int).Sub(nz(c.Spots[0]), nz(before.Spots[0]))
	pc, pb := ledgers(ctx, rpc, rewardsPool)
	expect(len(nearreconcile.Diff(pc, pb)) == 0 && new(big.Int).Sub(nz(poolBefore.Spots[0]), nz(pc.Spots[0])).Cmp(got) == 0,
		"claimed %s LogX, paid from the rewards pool (now %s), chain == backend", fmtX18(got), fmtX18(pc.Spots[0]))
}

// flowClaimLogX claims an airdrop allocation through /token/claim (paid from the rewards pool).
func flowClaimLogX(ctx context.Context, rpc *nearchain.Client, u, amount string) {
	s := mustSession(u)
	addr20, _ := cutils.NearAccountToAddr20(u)
	sub, _ := cutils.SubaccountIdToBytes32(s.SubaccountId)
	before, _ := ledgers(ctx, rpc, s.subHex())
	poolBefore, _ := ledgers(ctx, rpc, rewardsPool)
	n := apiNonce(s)
	amt := cutils.FloatStrToX18(amount)
	h, err := contractUtils.NearTypedHash(coreAccount, chainIdTestnet, "ClaimLogX", contractUtils.NearClaimLogXType, apitypes.TypedDataMessage{
		"subAccountId": sub[:], "tokenAmount": amt.String(), "sessionKey": s.addr(), "nonce": math.NewHexOrDecimal256(n), "chainId": math.NewHexOrDecimal256(chainIdTestnet)})
	must(err)
	sig := s.sign(h)
	fmt.Printf("claim airdrop: %s claims %s LogX\n", u, amount)
	code, body := api("POST", "/token/claim", s, map[string]any{"subAccountId": s.SubaccountId, "tokenAmount": amt.String(), "sessionKey": s.addr(), "nonce": n, "chainId": chainIdTestnet},
		map[string]string{"Logx-Signature": sig, "Logx-Signer-Address": s.addr(), "Broker-Signer-Address": s.addr(), "Broker-User-Address": addr20})
	expect(code == 200, "airdrop claim accepted (%d %s)", code, short(body))
	want := new(big.Int).Add(nz(before.Spots[0]), amt)
	waitAgree(ctx, rpc, s, "airdrop settled on-chain", func(c nearreconcile.Snapshot) bool { return nz(c.Spots[0]).Cmp(want) == 0 })
	pc, pb := ledgers(ctx, rpc, rewardsPool)
	expect(len(nearreconcile.Diff(pc, pb)) == 0 && new(big.Int).Sub(nz(poolBefore.Spots[0]), nz(pc.Spots[0])).Cmp(amt) == 0,
		"paid from the rewards pool (now %s), chain == backend", fmtX18(pc.Spots[0]))
	code, body = api("POST", "/token/claim", s, map[string]any{"subAccountId": s.SubaccountId, "tokenAmount": amt.String(), "sessionKey": s.addr(), "nonce": n + 1, "chainId": chainIdTestnet},
		map[string]string{"Logx-Signature": sig, "Logx-Signer-Address": s.addr(), "Broker-Signer-Address": s.addr(), "Broker-User-Address": addr20})
	expect(code != 200, "claiming the same allocation again is refused (%d)", code)
}

// flowSystemDeposit: the DAO funds a system subaccount with USDC (ft_transfer_call msg
// {"system": target}); on testnet the DAO first borrows the USDC from a test wallet.
func flowSystemDeposit(ctx context.Context, rpc *nearchain.Client, target, sub, usdc, lender string) {
	units := usdcUnits(usdc)
	_, err := send(ctx, rpc, lender, usdcTestnet, call("ft_transfer", map[string]any{"receiver_id": daoAccount, "amount": units.String()}, 30, big.NewInt(1)))
	must(err)
	before, _ := ledgers(ctx, rpc, sub)
	res, err := send(ctx, rpc, daoAccount, usdcTestnet, call("ft_transfer_call", map[string]any{"receiver_id": coreAccount, "amount": units.String(), "msg": fmt.Sprintf(`{"system":%q}`, target)}, 100, big.NewInt(1)))
	must(err)
	fmt.Printf("DAO funded %s with %s USDC, tx %s\n", target, usdc, res.Transaction.Hash)
	want := new(big.Int).Add(nz(before.Spots[4]), new(big.Int).Mul(units, big.NewInt(1e12)))
	for deadline := time.Now().Add(10 * time.Minute); ; time.Sleep(10 * time.Second) {
		c, b := ledgers(ctx, rpc, sub)
		if nz(c.Spots[4]).Cmp(want) == 0 && len(nearreconcile.Diff(c, b)) == 0 {
			expect(true, "%s chain == backend (USDC %s)", target, fmtX18(c.Spots[4]))
			return
		}
		if time.Now().After(deadline) {
			expect(false, "%s deposit not mirrored", target)
		}
	}
}

func flowCheck(ctx context.Context, rpc *nearchain.Client) {
	for _, u := range append(flowUsers, "amm") {
		s, err := loadSession(u)
		if err != nil {
			continue
		}
		chain, backend := ledgers(ctx, rpc, s.subHex())
		d := nearreconcile.Diff(chain, backend)
		expect(len(d) == 0, "%s chain == backend (USDC %s, ETH-PERP %s)%v", u, fmtX18(chain.Spots[4]), fmtX18(chain.Perps[1][0]), d)
	}
	chain, backend := ledgers(ctx, rpc, feeSub) // trading and withdrawal fees (D-7)
	d := nearreconcile.Diff(chain, backend)
	expect(len(d) == 0, "fee account chain == backend (USDC %s)%v", fmtX18(chain.Spots[4]), d)
}

func flows(ctx context.Context, rpc *nearchain.Client, args []string) {
	_ = xredis.GetRedisClient()
	xclient.InitBalanceClient()
	if len(args) == 0 {
		fmt.Println("usage: flows login|deposit|fund-user2|amm-setup|trade|withdraw|check")
		os.Exit(2)
	}
	switch args[0] {
	case "login":
		flowLogin(ctx, rpc)
	case "deposit":
		flowDeposit(ctx, rpc, args[1], args[2])
	case "fund-user2":
		_, err := send(ctx, rpc, flowUsers[0], usdcTestnet, call("ft_transfer", map[string]any{"receiver_id": flowUsers[1], "amount": usdcUnits(args[1]).String()}, 30, big.NewInt(1)))
		must(err)
		fmt.Printf("user2 wallet USDC units: %s\n", ftBalance(ctx, rpc, usdcTestnet, flowUsers[1]))
	case "amm-setup":
		flowAMMSetup(ctx, rpc, args[1])
	case "amm-quote": // amm-quote buy|sell <amountEth> <price>: rest a SOLVER order (what the AMM service does)
		code, body := placeOrder(mustSession("amm"), 1, args[1] == "buy", args[2], args[3], "LIMIT", "SOLVER")
		expect(code == 200, "AMM quote resting (%d %s)", code, short(body))
	case "trade": // trade <user> buy|sell <amountEth>
		flowTrade(ctx, rpc, args[1], args[2] == "buy", args[3])
	case "list-stocks": // lists TSLA, NVDA, AAPL, QQQ as perps on-chain + backend (see usStocks)
		flowListStocks(ctx, rpc)
	case "trade-market": // trade-market <productId> <symbol> <user> buy|sell <amount>
		var pid uint32
		fmt.Sscan(args[1], &pid)
		flowTradeMarket(ctx, rpc, pid, args[2], args[3], args[4] == "buy", args[5])
	case "take-live-quote": // take-live-quote <productId> <symbol> <user> buy|sell <amount>: TRADER-only order, to prove a live amm service (not this CLI) is quoting
		var pid uint32
		fmt.Sscan(args[1], &pid)
		flowTakeLiveQuote(ctx, rpc, pid, args[2], args[3], args[4] == "buy", args[5])
	case "withdraw":
		flowWithdraw(ctx, rpc, args[1], args[2])
	case "enable": // enable <kind 0=options 1=pre-market 2=synthetic> <productId> <maxFee> <maxPayoutPct>: DAO set_side_product
		var kind, pid, fee, payout int
		fmt.Sscan(args[1], &kind)
		fmt.Sscan(args[2], &pid)
		fmt.Sscan(args[3], &fee)
		fmt.Sscan(args[4], &payout)
		res, err := send(ctx, rpc, daoAccount, coreAccount, call("set_side_product", map[string]any{"kind": kind, "product_id": pid,
			"config": map[string]any{"enabled": true, "max_fee": fee, "max_payout_pct": payout}}, 20, new(big.Int)))
		must(err)
		fmt.Printf("side product %d/%d enabled, tx %s\n", kind, pid, res.Transaction.Hash)
	case "option": // option <user> <amountEth>  (positive = up)
		flowOption(ctx, rpc, args[1], args[2])
	case "stake": // stake <user> <stakeLogX> <unstakeLogX>
		flowStake(ctx, rpc, args[1], args[2], args[3])
	case "premarket-list": // premarket-list <productId> <maxSupply> <startingPrice>
		var pid uint32
		fmt.Sscan(args[1], &pid)
		flowPreMarketList(ctx, rpc, pid, args[2], args[3])
	case "premarket": // premarket <user> <productId> buy|sell <amount>
		var pid uint32
		fmt.Sscan(args[2], &pid)
		flowPreMarket(ctx, rpc, args[1], pid, args[3] == "buy", args[4])
	case "revoke": // revoke <user>: the user revokes their session key directly on-chain (wallet tx)
		s := mustSession(args[1])
		res, err := send(ctx, rpc, args[1], coreAccount, call("revoke_session_key", map[string]any{"subaccount_number": 1, "session_key": strings.ToLower(s.addr())}, 30, new(big.Int)))
		must(err)
		fmt.Printf("revoked %s on-chain, tx %s (on-chain expiry now %d)\n", s.addr(), res.Transaction.Hash, onChainExpiry(ctx, rpc, s.subHex(), s.addr()))
	case "order-refused": // order-refused <user>: an order signed with a revoked key must be refused by the API
		code, body := placeOrder(mustSession(args[1]), 1, true, "0.01", new(big.Int).Quo(oraclePrice("ETH"), e18).String(), "LIMIT", "TRADER")
		expect(code != 200, "order with the revoked session key refused by the API (%d %s)", code, short(body))
	case "synth-list": // synth-list <productId>
		var pid uint32
		fmt.Sscan(args[1], &pid)
		flowSynthList(ctx, rpc, pid)
	case "synth": // synth <user> <productId> buy|sell <amount>
		var pid uint32
		fmt.Sscan(args[2], &pid)
		flowPool(ctx, rpc, syntheticKind, args[1], pid, args[3] == "buy", args[4])
	case "withdraw-logx": // withdraw-logx <user> <amount>
		flowWithdrawToken(ctx, rpc, logxToken, args[1], args[2])
	case "fund-rewards": // fund-rewards <logx>
		flowFundRewards(ctx, rpc, args[1])
	case "claim-rewards": // claim-rewards <user>
		flowClaimRewards(ctx, rpc, args[1])
	case "claim-logx-steal": // claim-logx-steal <attacker> <victim> <amount>: header spoofing must not work
		s := mustSession(args[1])
		victim, _ := cutils.NearAccountToAddr20(args[2])
		own, _ := cutils.NearAccountToAddr20(args[1])
		sub, _ := cutils.SubaccountIdToBytes32(s.SubaccountId)
		n := apiNonce(s)
		amt := cutils.FloatStrToX18(args[3])
		h, err := contractUtils.NearTypedHash(coreAccount, chainIdTestnet, "ClaimLogX", contractUtils.NearClaimLogXType, apitypes.TypedDataMessage{
			"subAccountId": sub[:], "tokenAmount": amt.String(), "sessionKey": s.addr(), "nonce": math.NewHexOrDecimal256(n), "chainId": math.NewHexOrDecimal256(chainIdTestnet)})
		must(err)
		sig := s.sign(h)
		code, body := api("POST", "/token/claim", s, map[string]any{"subAccountId": s.SubaccountId, "tokenAmount": amt.String(), "sessionKey": s.addr(), "nonce": n, "chainId": chainIdTestnet},
			map[string]string{"Logx-Signature": sig, "Logx-Signer-Address": s.addr(), "Broker-Signer-Address": s.addr(), "Broker-User-Address": own, "Logx-User-Address": victim})
		expect(code != 200, "claiming another user's airdrop via Logx-User-Address is refused (%d %s)", code, short(body))
	case "claim-logx": // claim-logx <user> <amount>
		flowClaimLogX(ctx, rpc, args[1], args[2])
	case "fund-insurance": // fund-insurance <usdc> <lenderWallet>
		flowSystemDeposit(ctx, rpc, "insurance", strings.ToLower(contractUtils.INSURANCE_SUBACCOUNT_ID), args[1], args[2])
	case "check":
		flowCheck(ctx, rpc)
	default:
		fmt.Println("unknown flow", args[0])
		os.Exit(2)
	}
}
