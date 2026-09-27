package nearchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// OneClick is a client for the NEAR Intents 1Click API. Field names follow the official
// OpenAPI spec (https://1click.chaindefuser.com/docs/v0/openapi.yaml). Auth uses the
// recommended X-API-Key header; the key never leaves the backend.
type OneClick struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewOneClick(baseURL, apiKey string) *OneClick {
	if baseURL == "" {
		baseURL = "https://1click.chaindefuser.com"
	}
	return &OneClick{BaseURL: baseURL, APIKey: apiKey, HTTP: NewHTTPClient(30 * time.Second)}
}

type OneClickToken struct {
	AssetId         string  `json:"assetId"`
	Decimals        int     `json:"decimals"`
	Blockchain      string  `json:"blockchain"`
	Symbol          string  `json:"symbol"`
	Price           float64 `json:"price"`
	PriceUpdatedAt  string  `json:"priceUpdatedAt"`
	ContractAddress string  `json:"contractAddress,omitempty"`
}

type QuoteRequest struct {
	Dry                bool   `json:"dry"`
	SwapType           string `json:"swapType"`
	SlippageTolerance  int    `json:"slippageTolerance"`
	OriginAsset        string `json:"originAsset"`
	DepositType        string `json:"depositType"`
	DestinationAsset   string `json:"destinationAsset"`
	Amount             string `json:"amount"`
	RefundTo           string `json:"refundTo"`
	RefundType         string `json:"refundType"`
	Recipient          string `json:"recipient"`
	RecipientType      string `json:"recipientType"`
	Deadline           string `json:"deadline"`
	Confidentiality    string `json:"confidentiality,omitempty"`
	Referral           string `json:"referral,omitempty"`
	CustomRecipientMsg string `json:"customRecipientMsg,omitempty"`
}

type Quote struct {
	DepositAddress     string `json:"depositAddress"`
	DepositMemo        string `json:"depositMemo,omitempty"`
	AmountIn           string `json:"amountIn"`
	AmountInFormatted  string `json:"amountInFormatted"`
	AmountInUsd        string `json:"amountInUsd"`
	MinAmountIn        string `json:"minAmountIn"`
	AmountOut          string `json:"amountOut"`
	AmountOutFormatted string `json:"amountOutFormatted"`
	AmountOutUsd       string `json:"amountOutUsd"`
	MinAmountOut       string `json:"minAmountOut"`
	Deadline           string `json:"deadline"`
	TimeWhenInactive   string `json:"timeWhenInactive"`
	TimeEstimate       int    `json:"timeEstimate"`
	WithdrawFee        string `json:"withdrawFee,omitempty"`
	RefundFee          string `json:"refundFee,omitempty"`
}

type QuoteResponse struct {
	CorrelationId string       `json:"correlationId"`
	Timestamp     string       `json:"timestamp"`
	Signature     string       `json:"signature"`
	QuoteRequest  QuoteRequest `json:"quoteRequest"`
	Quote         Quote        `json:"quote"`
}

type TxDetails struct {
	Hash        string `json:"hash"`
	ExplorerUrl string `json:"explorerUrl"`
}

type SwapDetails struct {
	IntentHashes             []string    `json:"intentHashes"`
	NearTxHashes             []string    `json:"nearTxHashes"`
	AmountIn                 string      `json:"amountIn"`
	AmountOut                string      `json:"amountOut"`
	AmountOutFormatted       string      `json:"amountOutFormatted"`
	OriginChainTxHashes      []TxDetails `json:"originChainTxHashes"`
	DestinationChainTxHashes []TxDetails `json:"destinationChainTxHashes"`
	RefundedAmount           string      `json:"refundedAmount"`
	RefundReason             string      `json:"refundReason"`
	DepositedAmount          string      `json:"depositedAmount"`
}

// Status values from GetExecutionStatusResponse.status.
const (
	StatusKnownDepositTx = "KNOWN_DEPOSIT_TX"
	StatusPendingDeposit = "PENDING_DEPOSIT"
	StatusIncomplete     = "INCOMPLETE_DEPOSIT"
	StatusProcessing     = "PROCESSING"
	StatusSuccess        = "SUCCESS"
	StatusRefunded       = "REFUNDED"
	StatusFailed         = "FAILED"
)

func IsTerminalStatus(s string) bool {
	return s == StatusSuccess || s == StatusRefunded || s == StatusFailed
}

type StatusResponse struct {
	CorrelationId string        `json:"correlationId"`
	QuoteResponse QuoteResponse `json:"quoteResponse"`
	Status        string        `json:"status"`
	UpdatedAt     string        `json:"updatedAt"`
	SwapDetails   SwapDetails   `json:"swapDetails"`
}

type OneClickError struct {
	Status int
	Body   string
}

func (e *OneClickError) Error() string { return fmt.Sprintf("1click http %d: %s", e.Status, e.Body) }

func (o *OneClick) do(ctx context.Context, method, path string, q url.Values, body any, out any) error {
	u := o.BaseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if o.APIKey != "" {
		req.Header.Set("X-API-Key", o.APIKey)
	}
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		b := string(raw)
		if len(b) > 500 {
			b = b[:500]
		}
		return &OneClickError{resp.StatusCode, b}
	}
	return json.Unmarshal(raw, out)
}

func (o *OneClick) Tokens(ctx context.Context) ([]OneClickToken, error) {
	var out []OneClickToken
	return out, o.do(ctx, http.MethodGet, "/v0/tokens", nil, nil, &out)
}

func (o *OneClick) Quote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	var out QuoteResponse
	if err := o.do(ctx, http.MethodPost, "/v0/quote", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (o *OneClick) Status(ctx context.Context, depositAddress, depositMemo string) (*StatusResponse, error) {
	q := url.Values{"depositAddress": {depositAddress}}
	if depositMemo != "" {
		q.Set("depositMemo", depositMemo)
	}
	var out StatusResponse
	if err := o.do(ctx, http.MethodGet, "/v0/status", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (o *OneClick) SubmitDeposit(ctx context.Context, txHash, depositAddress, nearSender string) error {
	body := map[string]string{"txHash": txHash, "depositAddress": depositAddress}
	if nearSender != "" {
		body["nearSenderAccount"] = nearSender
	}
	var out json.RawMessage
	return o.do(ctx, http.MethodPost, "/v0/deposit/submit", nil, body, &out)
}
