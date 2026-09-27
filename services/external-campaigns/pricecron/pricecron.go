package pricecron

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/cerrors"
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"github.com/robfig/cron/v3"
)

// PriceData represents the structure of price data from the API
type PriceData struct {
	Timestamp        int64  `json:"timestamp"`
	AssetID          string `json:"asset_id"`
	SignatureType    string `json:"signature_type"`
	Trigger          string `json:"trigger"`
	Price            string `json:"price"`
	StorkSignedPrice struct {
		PublicKey            string `json:"public_key"`
		EncodedAssetID       string `json:"encoded_asset_id"`
		Price                string `json:"price"`
		TimestampedSignature struct {
			Signature struct {
				R string `json:"r"`
				S string `json:"s"`
				V string `json:"v"`
			} `json:"signature"`
			Timestamp int64  `json:"timestamp"`
			MsgHash   string `json:"msg_hash"`
		} `json:"timestamped_signature"`
	} `json:"stork_signed_price"`
	PublisherMerkleRoot string `json:"publisher_merkle_root"`
	CalculationAlg      struct {
		Type     string `json:"type"`
		Version  string `json:"version"`
		Checksum string `json:"checksum"`
	} `json:"calculation_alg"`
	SignedPrices []struct {
		PublisherKey         string `json:"publisher_key"`
		ExternalAssetID      string `json:"external_asset_id"`
		SignatureType        string `json:"signature_type"`
		Price                string `json:"price"`
		TimestampedSignature struct {
			Signature struct {
				R string `json:"r"`
				S string `json:"s"`
				V string `json:"v"`
			} `json:"signature"`
			Timestamp int64  `json:"timestamp"`
			MsgHash   string `json:"msg_hash"`
		} `json:"timestamped_signature"`
		Metadata struct {
			Markets []struct {
				Mic    string `json:"mic"`
				Status string `json:"status"`
			} `json:"markets"`
		} `json:"metadata"`
	} `json:"signed_prices"`
}

// PriceResponse represents the API response structure
type PriceResponse struct {
	Data map[string]PriceData `json:"data"`
}

// StorkOracleClient handles API communication
type StorkOracleClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewStorkOracleClient creates a new client instance
func NewStorkOracleClient() *StorkOracleClient {
	baseURL := os.Getenv("STORK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://rest.jp.stork-oracle.network/v1"
	}

	apiKey := os.Getenv("STORK_API_KEY")
	if apiKey == "" {
		apiKey = "b3N0cmljaDpjb252ZW5lLWxlYXAtYXN0b25pc2hpbmctc3VydmV5"
	}

	return &StorkOracleClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchPrices fetches price data from the Stork Oracle API
func (c *StorkOracleClient) FetchPrices(assets string) (*PriceResponse, error) {
	url := fmt.Sprintf("%s/prices/latest", c.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Authorization", "Basic "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Add query parameters
	q := req.URL.Query()
	q.Add("assets", assets)
	req.URL.RawQuery = q.Encode()

	// Make the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response
	var priceResponse PriceResponse
	if err := json.Unmarshal(body, &priceResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &priceResponse, nil
}

// ProcessPrices processes the fetched price data
func ProcessPrices() {
	client := NewStorkOracleClient()

	// Default assets to fetch
	assets := os.Getenv("STORK_ASSETS")
	if assets == "" {
		assets = "TSLA,NVDA,META"
	}

	// Fetch prices
	priceResponse, err := client.FetchPrices(assets)
	if err != nil {
		xlog.Errorf("Failed to fetch prices: %v", err)
		return
	}

	// Log successful price response
	xlog.Infof("Successfully fetched price data for assets: %s", assets)

	// Process the response
	_ = priceResponse
}

// StartPriceCron starts the price fetching cron job
func StartPriceCron() {
	xlog.Infof("Starting price cron job")

	// Create cron scheduler with seconds precision
	c := cron.New(cron.WithSeconds())

	// Add cron job to run every second
	c.AddFunc("* * * * * *", func() {
		cerrors.WithPanicRecover(ProcessPrices)
	})

	// Start the cron scheduler
	c.Start()
	xlog.Infof("Price cron job started successfully")

	// Keep the scheduler running
	select {}
}
