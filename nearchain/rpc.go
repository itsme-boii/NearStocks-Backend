package nearchain

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mr-tron/base58"
)

// Client is a minimal NEAR JSON-RPC client. Response shapes were checked against a live
// mainnet node (Development.md §16.1): note that a missing access key comes back as a
// *successful* JSON-RPC response carrying result.error, not as a JSON-RPC error.
type Client struct {
	URL  string
	HTTP *http.Client
	id   atomic.Int64
}

func NewClient(url string) *Client {
	return &Client{URL: url, HTTP: NewHTTPClient(30 * time.Second)}
}

// NewHTTPClient is the client every long-running NEAR service uses. HTTP/2 health pings detect a
// dead pooled connection: without them Go keeps reusing it and every request times out, which
// froze the batcher (all settlement) until a restart.
func NewHTTPClient(timeout time.Duration) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.HTTP2 = &http.HTTP2Config{SendPingTimeout: 15 * time.Second, PingTimeout: 5 * time.Second}
	return &http.Client{Timeout: timeout, Transport: t}
}

var (
	ErrAccessKeyNotFound = errors.New("access key does not exist")
	ErrUnknownTx         = errors.New("transaction not found")
)

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Name    string          `json:"name"`
	Cause   json.RawMessage `json:"cause"`
	Data    json.RawMessage `json:"data"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("near rpc %s: %s %s %s", e.Name, e.Message, string(e.Cause), string(e.Data))
}

func (c *Client) call(ctx context.Context, method string, params any, out any) error {
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": c.id.Add(1), "method": method, "params": params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	var env struct {
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("near rpc %s: http %d: invalid json: %w", method, resp.StatusCode, err)
	}
	if env.Error != nil {
		if strings.Contains(string(env.Error.Cause), "UNKNOWN_TRANSACTION") {
			return ErrUnknownTx
		}
		return env.Error
	}
	// Query-style methods report failures inside result.error.
	var qe struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(env.Result, &qe) == nil && qe.Error != "" {
		if strings.Contains(qe.Error, "does not exist while viewing") && strings.Contains(qe.Error, "access key") {
			return ErrAccessKeyNotFound
		}
		return fmt.Errorf("near rpc %s: %s", method, qe.Error)
	}
	return json.Unmarshal(env.Result, out)
}

// ---- access keys ----

type AccessKeyView struct {
	Nonce       uint64          `json:"nonce"`
	Permission  json.RawMessage `json:"permission"`
	BlockHash   string          `json:"block_hash"`
	BlockHeight uint64          `json:"block_height"`
}

// IsFullAccess is true only for the plain "FullAccess" permission (nearcore
// AccessKeyPermissionView serializes it as a bare string; FunctionCall keys are objects).
func (a *AccessKeyView) IsFullAccess() bool {
	var s string
	return json.Unmarshal(a.Permission, &s) == nil && s == "FullAccess"
}

func (c *Client) ViewAccessKey(ctx context.Context, accountId, publicKey string) (*AccessKeyView, error) {
	var out AccessKeyView
	err := c.call(ctx, "query", map[string]any{
		"request_type": "view_access_key", "finality": "final", "account_id": accountId, "public_key": publicKey,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- view calls ----

// CallView runs a view function with JSON args and returns the raw result bytes.
func (c *Client) CallView(ctx context.Context, contract, method string, args any) ([]byte, error) {
	a, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	var out struct {
		Result []byte `json:"-"`
		Raw    []int  `json:"result"`
	}
	err = c.call(ctx, "query", map[string]any{
		"request_type": "call_function", "finality": "final", "account_id": contract,
		"method_name": method, "args_base64": base64.StdEncoding.EncodeToString(a),
	}, &out)
	if err != nil {
		return nil, err
	}
	b := make([]byte, len(out.Raw))
	for i, v := range out.Raw {
		b[i] = byte(v)
	}
	return b, nil
}

// ---- blocks ----

func (c *Client) FinalBlockHash(ctx context.Context) ([32]byte, error) {
	var out struct {
		Header struct {
			Hash string `json:"hash"`
		} `json:"header"`
	}
	var h [32]byte
	if err := c.call(ctx, "block", map[string]any{"finality": "final"}, &out); err != nil {
		return h, err
	}
	raw, err := base58.Decode(out.Header.Hash)
	if err != nil || len(raw) != 32 {
		return h, fmt.Errorf("invalid block hash %q", out.Header.Hash)
	}
	copy(h[:], raw)
	return h, nil
}

// ---- transactions ----

type ExecutionOutcome struct {
	Id      string `json:"id"`
	Outcome struct {
		ExecutorId string                     `json:"executor_id"`
		Logs       []string                   `json:"logs"`
		Status     map[string]json.RawMessage `json:"status"`
	} `json:"outcome"`
}

// Succeeded is true for SuccessValue or SuccessReceiptId.
func (o *ExecutionOutcome) Succeeded() bool {
	_, v := o.Outcome.Status["SuccessValue"]
	_, r := o.Outcome.Status["SuccessReceiptId"]
	return v || r
}

type TxResult struct {
	FinalExecutionStatus string                     `json:"final_execution_status"`
	Status               map[string]json.RawMessage `json:"status"`
	Transaction          struct {
		SignerId   string                       `json:"signer_id"`
		ReceiverId string                       `json:"receiver_id"`
		Hash       string                       `json:"hash"`
		Actions    []map[string]json.RawMessage `json:"actions"`
	} `json:"transaction"`
	TransactionOutcome ExecutionOutcome   `json:"transaction_outcome"`
	ReceiptsOutcome    []ExecutionOutcome `json:"receipts_outcome"`
}

// Succeeded is true when the whole transaction resolved with SuccessValue.
func (t *TxResult) Succeeded() bool {
	_, ok := t.Status["SuccessValue"]
	return ok
}

// FunctionCallMethods lists the method names of the transaction's FunctionCall actions.
func (t *TxResult) FunctionCallMethods() []string {
	var names []string
	for _, a := range t.Transaction.Actions {
		if fc, ok := a["FunctionCall"]; ok {
			var v struct {
				MethodName string `json:"method_name"`
			}
			if json.Unmarshal(fc, &v) == nil {
				names = append(names, v.MethodName)
			}
		}
	}
	return names
}

func (c *Client) TxStatus(ctx context.Context, txHash, senderId string) (*TxResult, error) {
	var out TxResult
	if err := c.call(ctx, "tx", map[string]any{"tx_hash": txHash, "sender_account_id": senderId, "wait_until": "FINAL"}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SendTx broadcasts a signed transaction and waits for finality.
func (c *Client) SendTx(ctx context.Context, signed []byte) (*TxResult, error) {
	var out TxResult
	err := c.call(ctx, "send_tx", map[string]any{"signed_tx_base64": base64.StdEncoding.EncodeToString(signed), "wait_until": "FINAL"}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AccountView is view_account: balances in yoctoNEAR and state size in bytes.
type AccountView struct {
	Amount       string `json:"amount"`
	Locked       string `json:"locked"`
	StorageUsage uint64 `json:"storage_usage"`
}

func (c *Client) ViewAccount(ctx context.Context, accountId string) (*AccountView, error) {
	var out AccountView
	if err := c.call(ctx, "query", map[string]any{"request_type": "view_account", "finality": "final", "account_id": accountId}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
