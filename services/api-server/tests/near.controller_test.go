package tests

// End-to-end Phase 2 flow through the real handlers: NEP-413 sign-in, direct USDC deposit,
// and treasury withdrawal, against Postgres (testcontainers), miniredis, a fake NEAR RPC and a
// fake balance-server. Attack cases are asserted alongside the happy path.

import (
	"github/eugenix-io/logx-inf-backend/contract"

	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/nearchain"
	"github/eugenix-io/logx-inf-backend/services/api-server/controller"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"github/eugenix-io/logx-inf-backend/testutils"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUser     = "alice.near"
	testTreasury = "treasury.near-stocks.near"
	depositTx    = "4zvGftYD56iVPGgSMK9ccq3Aka41SWnd5FbAkoJzNQFz"
)

type fakeBalance struct {
	sync.Mutex
	bal map[string]*big.Int // subHex -> x18 quote balance
}

func (f *fakeBalance) get(sub string) *big.Int {
	f.Lock()
	defer f.Unlock()
	if v := f.bal[sub]; v != nil {
		return new(big.Int).Set(v)
	}
	return big.NewInt(0)
}

func (f *fakeBalance) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/balance/update-token-balance":
			var req struct {
				SubaccountID string `json:"subaccount_id"`
				TokenBalance string `json:"token_balance"`
			}
			raw := new(bytes.Buffer)
			_, _ = raw.ReadFrom(r.Body)
			var m map[string]any
			_ = json.Unmarshal(raw.Bytes(), &m)
			for k, v := range m { // tolerate either json casing
				switch strings.ToLower(strings.ReplaceAll(k, "_", "")) {
				case "subaccountid":
					req.SubaccountID = fmt.Sprint(v)
				case "tokenbalance":
					req.TokenBalance = fmt.Sprint(v)
				}
			}
			d, _ := new(big.Int).SetString(req.TokenBalance, 10)
			f.Lock()
			if f.bal[req.SubaccountID] == nil {
				f.bal[req.SubaccountID] = big.NewInt(0)
			}
			f.bal[req.SubaccountID].Add(f.bal[req.SubaccountID], d)
			f.Unlock()
			w.WriteHeader(http.StatusOK)
		case "/balance/withdrawableTokenBalance":
			sub := r.URL.Query().Get("subaccountID")
			_ = json.NewEncoder(w).Encode(map[string]any{"withdrawableBalance": map[string]string{"4": f.get(sub).String()}})
		default:
			http.NotFound(w, r)
		}
	}))
}

type fakeNear struct {
	sessionExpiry uint64 // session_key_expiry view (NEAR settlement)
	sync.Mutex
	userPK, treasuryPK string
	fcOnlyPK           string
	sent               [][]byte
}

func bytesJSON(s string) string {
	parts := make([]string, len(s))
	for i := range s {
		parts[i] = fmt.Sprint(int(s[i]))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func (f *fakeNear) server(t *testing.T) *httptest.Server {
	block := sha256.Sum256([]byte("b"))
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := func(body string) { _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":` + body + `}`)) }
		switch req.Method {
		case "query":
			switch req.Params["request_type"] {
			case "view_access_key":
				switch req.Params["public_key"] {
				case f.userPK, f.treasuryPK:
					res(`{"block_hash":"x","block_height":1,"nonce":10,"permission":"FullAccess"}`)
				case f.fcOnlyPK:
					res(`{"block_hash":"x","block_height":1,"nonce":0,"permission":{"FunctionCall":{"allowance":null,"method_names":[],"receiver_id":"x"}}}`)
				default:
					res(`{"block_hash":"x","block_height":1,"error":"access key ` + fmt.Sprint(req.Params["public_key"]) + ` does not exist while viewing","logs":[]}`)
				}
			case "call_function":
				out := map[string]string{
					"storage_balance_of":     `{"total":"1250000000000000000000","available":"0"}`,
					"storage_balance_bounds": `{"min":"1250000000000000000000","max":"1250000000000000000000"}`,
					"session_key_expiry":     fmt.Sprint(f.sessionExpiry),
					// the contract's USDC withdraw fee (1.5) differs from the static table's on purpose
					"get_products": `{"spots":[{"product_id":4,"withdraw_fee_x18":"1500000000000000000"}],"perps":[]}`,
				}[fmt.Sprint(req.Params["method_name"])]
				res(`{"block_hash":"x","block_height":1,"logs":[],"result":` + bytesJSON(out) + `}`)
			}
		case "block":
			res(`{"header":{"hash":"` + base58.Encode(block[:]) + `","height":1}}`)
		case "tx":
			if req.Params["tx_hash"] != depositTx || req.Params["sender_account_id"] != testUser {
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"name":"HANDLER_ERROR","cause":{"name":"UNKNOWN_TRANSACTION"},"code":-32000,"message":"Server error"}}`))
				return
			}
			ev := `EVENT_JSON:{\"standard\":\"nep141\",\"version\":\"1.0.0\",\"event\":\"ft_transfer\",\"data\":[{\"old_owner_id\":\"` + testUser + `\",\"new_owner_id\":\"` + testTreasury + `\",\"amount\":\"5000000\"}]}`
			res(`{"final_execution_status":"FINAL","status":{"SuccessValue":""},
			"transaction":{"signer_id":"` + testUser + `","receiver_id":"` + nearchain.USDCMainnet + `","hash":"` + depositTx + `","actions":[{"FunctionCall":{"method_name":"ft_transfer","args":"e30=","gas":1,"deposit":"1"}}]},
			"transaction_outcome":{"id":"t","outcome":{"executor_id":"` + testUser + `","logs":[],"status":{"SuccessReceiptId":"r"}}},
			"receipts_outcome":[{"id":"r","outcome":{"executor_id":"` + nearchain.USDCMainnet + `","logs":["` + ev + `"],"status":{"SuccessValue":""}}}]}`)
		case "send_tx":
			raw, err := base64.StdEncoding.DecodeString(fmt.Sprint(req.Params["signed_tx_base64"]))
			require.NoError(t, err)
			f.Lock()
			f.sent = append(f.sent, raw)
			f.Unlock()
			res(`{"final_execution_status":"FINAL","status":{"SuccessValue":""},"transaction":{"signer_id":"` + testTreasury + `","receiver_id":"` + nearchain.USDCMainnet + `","hash":"h","actions":[]},"transaction_outcome":{"id":"t","outcome":{"executor_id":"x","logs":[],"status":{"SuccessValue":""}}},"receipts_outcome":[]}`)
		}
	}))
}

func do(t *testing.T, r http.Handler, method, path string, body any, headers map[string]string) (int, map[string]any) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestNearPhase2Flow(t *testing.T) {
	testutils.SetMainnetEnv()
	defer testutils.ResetEnv()
	testutils.SetupDBEnv(t)
	db.Init()
	(&db.BrokerDB{}).Create("broker-1")
	(&db.BrokerDB{}).Create("broker-2")
	require.NotNil(t, (&db.BrokerDB{}).GetById(2))

	userSeed := sha256.Sum256([]byte("user"))
	userKey := ed25519.NewKeyFromSeed(userSeed[:])
	treasurySeed := sha256.Sum256([]byte("treasury"))
	treasuryKey := ed25519.NewKeyFromSeed(treasurySeed[:])
	fcSeed := sha256.Sum256([]byte("fc"))
	fcKey := ed25519.NewKeyFromSeed(fcSeed[:])

	near := &fakeNear{
		userPK:     nearchain.PublicKeyString(userKey.Public().(ed25519.PublicKey)),
		treasuryPK: nearchain.PublicKeyString(treasuryKey.Public().(ed25519.PublicKey)),
		fcOnlyPK:   nearchain.PublicKeyString(fcKey.Public().(ed25519.PublicKey)),
	}
	rpc := near.server(t)
	defer rpc.Close()
	bal := &fakeBalance{bal: map[string]*big.Int{}}
	bs := bal.server()
	defer bs.Close()

	for k, v := range map[string]string{
		"NEAR_NETWORK": "mainnet", "NEAR_TREASURY_ACCOUNT": testTreasury, "NEAR_RPC_URL": rpc.URL,
		"NEAR_TREASURY_PRIVATE_KEY": "ed25519:" + base58.Encode(treasuryKey), "BALANCE_SERVER_URL": bs.URL,
		"NEAR_WITHDRAW_MAX_USDC": "100",
	} {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}
	services.SetNearConfigForTest(nil)
	defer services.SetNearConfigForTest(nil)
	xclient.InitBalanceClient()

	testutils.WithSetupMockRedis(t, func() {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		controller.RegisterNearController(r.Group("/api/v1"))

		sessionSK, _ := crypto.GenerateKey()
		sessionAddr := crypto.PubkeyToAddress(sessionSK.PublicKey).Hex()
		expiry := time.Now().Add(6 * 24 * time.Hour).UnixMilli()
		addr20, _ := cutils.NearAccountToAddr20(testUser)
		subId := cutils.CreateSubaccountId(2, addr20, 1)
		subHex, _ := cutils.SubaccountIdToHex(subId)

		signEIP712 := func(h []byte) string {
			sig, err := crypto.Sign(h, sessionSK)
			require.NoError(t, err)
			sig[64] += 27
			return hexutil.Encode(sig)
		}
		regHash, err := contractUtils.NearRegisterHash("near-stocks.near", 397, contractUtils.NearRegister{SubaccountId: subId, UserAddress: addr20, SessionKey: sessionAddr, ExpiryTs: expiry})
		require.NoError(t, err)
		sessionSig := signEIP712(regHash)

		loginBody := func(key ed25519.PrivateKey, recipient string) map[string]any {
			code, ch := do(t, r, "GET", "/api/v1/auth/near/challenge", nil, nil)
			require.Equal(t, 200, code)
			require.Equal(t, "near-stocks.near", ch["recipient"])
			nonce, _ := base64.StdEncoding.DecodeString(ch["nonce"].(string))
			msg := fmt.Sprintf(`{"action":"login","sessionKey":"%s","expiry":%d,"ts":%d}`, sessionAddr, expiry, time.Now().UnixMilli())
			var n [32]byte
			copy(n[:], nonce)
			h := contractUtils.NEP413Hash(contractUtils.NEP413Payload{Message: msg, Nonce: n, Recipient: recipient})
			return map[string]any{
				"accountId": testUser, "publicKey": nearchain.PublicKeyString(key.Public().(ed25519.PublicKey)),
				"signature": base64.StdEncoding.EncodeToString(ed25519.Sign(key, h[:])), "message": msg,
				"nonce": ch["nonce"], "sessionKey": sessionAddr, "sessionSignature": sessionSig, "expiryTs": expiry,
			}
		}

		// --- sign-in attacks ---
		code, _ := do(t, r, "POST", "/api/v1/auth/near", loginBody(fcKey, "near-stocks.near"), nil)
		assert.Equal(t, 401, code, "function-call key must not sign in")
		code, _ = do(t, r, "POST", "/api/v1/auth/near", loginBody(userKey, "evil.near"), nil)
		assert.Equal(t, 401, code, "signature for another recipient must fail")
		bad := loginBody(userKey, "near-stocks.near")
		bad["sessionSignature"] = "0x" + strings.Repeat("11", 65)
		code, _ = do(t, r, "POST", "/api/v1/auth/near", bad, nil)
		assert.Equal(t, 401, code, "session key possession proof required")
		short := loginBody(userKey, "near-stocks.near")
		short["sessionSignature"] = "0x1234"
		code, _ = do(t, r, "POST", "/api/v1/auth/near", short, nil)
		assert.Equal(t, 401, code, "short signature must be rejected, not panic")

		// --- sign-in ---
		body := loginBody(userKey, "near-stocks.near")
		code, out := do(t, r, "POST", "/api/v1/auth/near", body, nil)
		require.Equal(t, 200, code, "%v", out)
		creds := out["body"].(map[string]any)
		assert.Equal(t, subId, creds["subaccountId"])
		code, _ = do(t, r, "POST", "/api/v1/auth/near", body, nil)
		assert.Equal(t, 401, code, "replayed challenge must fail")

		auth := map[string]string{"Broker-Key": creds["broker_key"].(string), "Broker-Secret": creds["broker_secret"].(string), "Broker-Id": "2"}
		code, acct := do(t, r, "GET", "/api/v1/near/account", nil, auth)
		require.Equal(t, 200, code)
		assert.Equal(t, testUser, acct["accountId"])

		// --- direct deposit: credited once ---
		code, out = do(t, r, "POST", "/api/v1/near/deposit/direct", map[string]string{"txHash": depositTx}, auth)
		require.Equal(t, 200, code, "%v", out)
		want := nearchain.ScaleUSDCToX18(big.NewInt(5_000_000))
		assert.Equal(t, want.String(), bal.get(subHex).String())
		code, _ = do(t, r, "POST", "/api/v1/near/deposit/direct", map[string]string{"txHash": depositTx}, auth)
		assert.Equal(t, 200, code)
		assert.Equal(t, want.String(), bal.get(subHex).String(), "same tx must not credit twice")
		// Racing requests bypass the controller pre-check; the credit guard itself must hold.
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := services.CreditNearDeposit("neartx:"+depositTx, subId, "5000000", depositTx)
				assert.ErrorIs(t, err, services.ErrAlreadyCredited)
			}()
		}
		wg.Wait()
		assert.Equal(t, want.String(), bal.get(subHex).String(), "concurrent re-credits must all be refused")

		code, _ = do(t, r, "POST", "/api/v1/near/deposit/direct", map[string]string{"txHash": strings.Repeat("A", 44)}, auth)
		assert.Equal(t, 404, code, "unknown tx")

		// --- withdraw ---
		withdraw := func(receiver string, nonce int64, amountX18 string) (int, map[string]any) {
			h, err := contractUtils.NearWithdrawHash("near-stocks.near", 397, contractUtils.NearWithdraw{SubaccountId: subId, SessionKey: sessionAddr, ProductId: 4, Amount: amountX18, Nonce: nonce, Receiver: receiver})
			require.NoError(t, err)
			hdr := map[string]string{"Logx-Signer-Address": sessionAddr, "Logx-User-Address": addr20, "Logx-Signature": signEIP712(h)}
			for k, v := range auth {
				hdr[k] = v
			}
			return do(t, r, "POST", "/api/v1/near/withdraw", map[string]any{"subAccountId": subId, "productId": 4, "amount": amountX18, "nonce": nonce, "receiver": receiver}, hdr)
		}
		two := cutils.FloatStrToX18("2")
		code, _ = withdraw("mallory.near", 0, two.String())
		assert.Equal(t, 403, code, "receiver must be the user's own account or their quote")
		code, _ = withdraw(testUser, 0, cutils.FloatStrToX18("500").String())
		assert.Equal(t, 400, code, "above cap / above balance must fail")

		code, out = withdraw(testUser, 0, two.String())
		require.Equal(t, 200, code, "%v", out)
		assert.Equal(t, services.PayoutSent, out["status"])
		fee := cutils.FloatStrToX18(contractUtils.WITHDRAWAL_FEE_MAP[4])
		assert.Equal(t, new(big.Int).Sub(two, fee).String(), nearchain.ScaleUSDCToX18(mustBig(out["amountUsdc"].(string))).String(), "user receives amount minus fee")
		assert.Equal(t, new(big.Int).Sub(want, two).String(), bal.get(subHex).String(), "ledger debited by the full amount")

		require.Len(t, near.sent, 1)
		sent := string(near.sent[0])
		assert.Contains(t, sent, "ft_transfer")
		assert.Contains(t, sent, `"receiver_id":"`+testUser+`"`)
		assert.Contains(t, sent, `"amount":"`+out["amountUsdc"].(string)+`"`)
		assert.NotContains(t, sent, "storage_deposit", "registered receiver needs no storage deposit")

		code, _ = withdraw(testUser, 0, two.String())
		assert.Equal(t, 400, code, "replayed nonce must fail")
		assert.Len(t, near.sent, 1, "no second payout")

		// --- NEAR settlement (Phase 4): the contract holds funds and pays out ---
		os.Setenv("NEAR_SETTLEMENT", "1")
		defer os.Unsetenv("NEAR_SETTLEMENT")
		contract.GlobalContracts.EndpointContract = *contract.NewEndpointContract()
		var queued []contract.QueuedTx
		defer contract.SetNearQueueForTest(func(q contract.QueuedTx) error { queued = append(queued, q); return nil })()

		code, _ = do(t, r, "POST", "/api/v1/near/deposit/direct", map[string]string{"txHash": depositTx}, auth)
		assert.Equal(t, 409, code, "deposits are credited from the contract's event, not by tx hash")

		// login needs the session key registered on-chain first
		code, out = do(t, r, "POST", "/api/v1/auth/near", loginBody(userKey, "near-stocks.near"), nil)
		assert.Equal(t, 409, code)
		assert.Equal(t, true, out["needsRegistration"])
		near.sessionExpiry = uint64(time.Now().Add(time.Hour).UnixMilli())
		code, out = do(t, r, "POST", "/api/v1/auth/near", loginBody(userKey, "near-stocks.near"), nil)
		assert.Equal(t, 200, code, "%v", out)

		code, cfg := do(t, r, "GET", "/api/v1/near/config", nil, nil)
		require.Equal(t, 200, code)
		assert.Equal(t, true, cfg["withdrawalsEnabled"], "the contract pays withdrawals: no treasury signer needed")
		assert.Equal(t, "1500000000000000000", cfg["withdrawFeeX18"], "the fee is the contract's, not the static table's")

		before := new(big.Int).Set(bal.get(subHex))
		code, out = withdraw(testUser, 1, two.String())
		require.Equal(t, 200, code, "%v", out)
		assert.Equal(t, "QUEUED", out["status"])
		wantPayout := new(big.Int).Div(new(big.Int).Sub(two, mustBig("1500000000000000000")), big.NewInt(1e12))
		assert.Equal(t, wantPayout.String(), out["amountUsdc"], "payout shown net of the on-chain fee")
		require.Len(t, queued, 1)
		subB, _ := cutils.SubaccountIdToBytes32(subId)
		want3 := nearchain.NearWithdraw{Subaccount: subB, SessionKey: common.HexToAddress(sessionAddr), ProductId: 4, Amount: two, Nonce: big.NewInt(1), Receiver: testUser}
		assert.Equal(t, want3.Encode(), queued[0].Payload, "WITHDRAW_COLLATERAL with the signed receiver")
		assert.Equal(t, 65, len(queued[0].Sig1))
		assert.Equal(t, new(big.Int).Sub(before, two).String(), bal.get(subHex).String(), "ledger debited when queued")
		assert.Len(t, near.sent, 1, "the treasury pays nothing with NEAR settlement")
	})
}

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}
