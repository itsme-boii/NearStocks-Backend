package nearchain

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
)

// Response bodies below reproduce the shapes returned by a live mainnet node
// (free.rpc.fastnear.com, 2026-09-26).

func rpcServer(t *testing.T, handler func(method string, params map[string]any) string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_, _ = w.Write([]byte(handler(req.Method, req.Params)))
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL)
}

func TestViewAccessKey(t *testing.T) {
	c := rpcServer(t, func(_ string, p map[string]any) string {
		switch p["public_key"] {
		case "ed25519:full":
			return `{"jsonrpc":"2.0","result":{"block_hash":"x","block_height":1,"nonce":7,"permission":"FullAccess"},"id":1}`
		case "ed25519:fc":
			return `{"jsonrpc":"2.0","result":{"block_hash":"x","block_height":1,"nonce":0,"permission":{"FunctionCall":{"allowance":"1","method_names":["claim"],"receiver_id":"near"}}},"id":1}`
		default: // real node behavior: success envelope with result.error
			return `{"jsonrpc":"2.0","result":{"block_hash":"x","block_height":1,"error":"access key ed25519:zz does not exist while viewing","logs":[]},"id":1}`
		}
	})
	ctx := context.Background()
	k, err := c.ViewAccessKey(ctx, "alice.near", "ed25519:full")
	if err != nil || !k.IsFullAccess() || k.Nonce != 7 {
		t.Fatalf("full access: %+v %v", k, err)
	}
	k, err = c.ViewAccessKey(ctx, "alice.near", "ed25519:fc")
	if err != nil || k.IsFullAccess() {
		t.Fatalf("function-call key must not count as full access: %+v %v", k, err)
	}
	if _, err = c.ViewAccessKey(ctx, "alice.near", "ed25519:missing"); !errors.Is(err, ErrAccessKeyNotFound) {
		t.Fatalf("missing key: got %v", err)
	}
}

func TestCallViewAndBalances(t *testing.T) {
	enc := func(s string) string {
		ints := make([]string, len(s))
		for i := range s {
			ints[i] = itoa(int(s[i]))
		}
		return "[" + strings.Join(ints, ",") + "]"
	}
	c := rpcServer(t, func(_ string, p map[string]any) string {
		var body string
		switch p["method_name"] {
		case "ft_balance_of":
			body = `"123456789"`
		case "storage_balance_of":
			body = `null`
		case "storage_balance_bounds":
			body = `{"min":"1250000000000000000000","max":"1250000000000000000000"}`
		}
		return `{"jsonrpc":"2.0","result":{"block_hash":"x","block_height":1,"logs":[],"result":` + enc(body) + `},"id":1}`
	})
	ctx := context.Background()
	bal, err := c.FtBalanceOf(ctx, USDCMainnet, "t.near")
	if err != nil || bal.String() != "123456789" {
		t.Fatalf("balance %v %v", bal, err)
	}
	reg, err := c.StorageRegistered(ctx, USDCMainnet, "t.near")
	if err != nil || reg {
		t.Fatalf("registered %v %v", reg, err)
	}
	min, err := c.StorageMinimum(ctx, USDCMainnet)
	if err != nil || min.String() != "1250000000000000000000" {
		t.Fatalf("min %v %v", min, err)
	}
}

func itoa(i int) string { return big.NewInt(int64(i)).String() }

// tx builds a TxResult JSON modeled on the live USDC transaction
// 4zvGftYD56iVPGgSMK9ccq3Aka41SWnd5FbAkoJzNQFz (plain transfer variant).
func txJSON(signer, method, final, finalStatus string, logs ...string) *TxResult {
	l, _ := json.Marshal(logs)
	raw := `{"final_execution_status":"` + final + `","status":{"` + finalStatus + `":""},
	"transaction":{"signer_id":"` + signer + `","receiver_id":"` + USDCMainnet + `","hash":"h","actions":[{"FunctionCall":{"method_name":"` + method + `","args":"e30=","gas":30000000000000,"deposit":"1"}}]},
	"transaction_outcome":{"id":"t","outcome":{"executor_id":"` + signer + `","logs":[],"status":{"SuccessReceiptId":"r"}}},
	"receipts_outcome":[{"id":"r","outcome":{"executor_id":"` + USDCMainnet + `","logs":` + string(l) + `,"status":{"SuccessValue":""}}}]}`
	var tx TxResult
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		panic(err)
	}
	return &tx
}

func ev(from, to, amount string) string {
	return `EVENT_JSON:{"standard":"nep141","version":"1.0.0","event":"ft_transfer","data":[{"old_owner_id":"` + from + `","new_owner_id":"` + to + `","amount":"` + amount + `"}]}`
}

func TestVerifyDirectDeposit(t *testing.T) {
	const user, treasury = "alice.near", "treasury.near-stocks.near"
	ok := txJSON(user, "ft_transfer", "FINAL", "SuccessValue", ev(user, treasury, "5000000"))
	amt, err := VerifyDirectDeposit(ok, user, treasury, USDCMainnet)
	if err != nil || amt.String() != "5000000" {
		t.Fatalf("valid deposit: %v %v", amt, err)
	}

	cases := map[string]*TxResult{
		"not final":        txJSON(user, "ft_transfer", "EXECUTED_OPTIMISTIC", "SuccessValue", ev(user, treasury, "5")),
		"failed tx":        txJSON(user, "ft_transfer", "FINAL", "Failure", ev(user, treasury, "5")),
		"other signer":     txJSON("mallory.near", "ft_transfer", "FINAL", "SuccessValue", ev("mallory.near", treasury, "5")),
		"ft_transfer_call": txJSON(user, "ft_transfer_call", "FINAL", "SuccessValue", ev(user, treasury, "5")),
		"to someone else":  txJSON(user, "ft_transfer", "FINAL", "SuccessValue", ev(user, "bob.near", "5")),
		"refunded in full": txJSON(user, "ft_transfer", "FINAL", "SuccessValue", ev(user, treasury, "5"), ev(treasury, user, "5")),
		"forged non-event": txJSON(user, "ft_transfer", "FINAL", "SuccessValue", `ft_transfer alice.near -> treasury 5`),
	}
	for name, tx := range cases {
		if _, err := VerifyDirectDeposit(tx, user, treasury, USDCMainnet); !errors.Is(err, ErrNotADeposit) {
			t.Errorf("%s: expected ErrNotADeposit, got %v", name, err)
		}
	}

	// Events from a contract other than USDC are ignored even if they claim a transfer.
	spoof := txJSON(user, "ft_transfer", "FINAL", "SuccessValue", ev(user, treasury, "5"))
	spoof.ReceiptsOutcome[0].Outcome.ExecutorId = "evil-token.near"
	if _, err := VerifyDirectDeposit(spoof, user, treasury, USDCMainnet); !errors.Is(err, ErrNotADeposit) {
		t.Errorf("spoofed token: expected rejection, got %v", err)
	}
}

func TestScaleUSDC(t *testing.T) {
	x18 := ScaleUSDCToX18(big.NewInt(1_500_000)) // 1.5 USDC
	if x18.String() != "1500000000000000000" {
		t.Fatalf("scale up %s", x18)
	}
	usdc, dust := ScaleX18ToUSDC(new(big.Int).Add(x18, big.NewInt(999)))
	if usdc.String() != "1500000" || dust.String() != "999" {
		t.Fatalf("scale down %s %s", usdc, dust)
	}
}

func TestParseNearPrivateKeyRejectsMismatch(t *testing.T) {
	if _, err := ParseNearPrivateKey("secp256k1:abc"); err == nil {
		t.Fatal("expected rejection of non-ed25519 key")
	}
	bad := make([]byte, 64) // zero seed with zero "public" half: halves don't match
	if _, err := ParseNearPrivateKey("ed25519:" + b58(bad)); err == nil {
		t.Fatal("expected rejection of mismatched key halves")
	}
}

func b58(b []byte) string { return base58.Encode(b) }
