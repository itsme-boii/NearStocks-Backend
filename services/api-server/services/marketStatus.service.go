package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/xcache"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
)

type MarketStatusResponse struct {
	Exchange string  `json:"exchange"`
	Holiday  *string `json:"holiday"`
	IsOpen   bool    `json:"isOpen"`
	Session  string  `json:"session"`
	T        int64   `json:"t"`
	Timezone string  `json:"timezone"`
}

type MarketStatusService struct {
	cache          xcache.Cache[*MarketStatusResponse]
	apiKey         string
	fallbackAPIKey string
}

var (
	marketStatusService *MarketStatusService
	once                sync.Once
)

func NewMarketStatusService() *MarketStatusService {
	once.Do(func() {
		marketStatusService = &MarketStatusService{
			cache:          xcache.NewCache[*MarketStatusResponse](nil, time.Now().Add(5*time.Minute)),
			apiKey:         os.Getenv("FINNHUB_API_KEY"),
			fallbackAPIKey: "d137rb1r01qv1k0p1pm0d137rb1r01qv1k0p1pmg",
		}
	})
	return marketStatusService
}

func (mss *MarketStatusService) GetMarketStatus() (*MarketStatusResponse, error) {
	if cached := mss.cache.Get(5 * time.Minute); cached != nil {
		return *cached, nil
	}

	response, err := mss.fetchFromAPI()
	if err != nil {
		return nil, err
	}

	mss.cache.Set(response, 5*time.Minute)

	return response, nil
}

func (mss *MarketStatusService) fetchFromAPI() (*MarketStatusResponse, error) {
	apiKey := mss.apiKey
	if apiKey == "" {
		apiKey = mss.fallbackAPIKey
		xlog.Warnf("Using fallback Finnhub API key as FINNHUB_API_KEY is not configured")
	}

	url := fmt.Sprintf("https://finnhub.io/api/v1/stock/market-status?exchange=US&token=%s", apiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var marketStatus MarketStatusResponse
	if err := json.Unmarshal(body, &marketStatus); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &marketStatus, nil
}

func (mss *MarketStatusService) IsMarketOpen() bool {
	status, err := mss.GetMarketStatus()
	if err != nil {
		xlog.Warnf("Failed to check market status, blocking trading as fallback: %v", err)
		return false // Block trading if we can't check market status
	}
	return status.IsOpen
}
