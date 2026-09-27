package xclient

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"net/http"
	"os"
)

type LiquidationClient struct {
	baseUrl string
}

// Verify that the LiquidationClient implements the XClient interface
var _ XClient = &LiquidationClient{}

var GlobalLiquidationClient *LiquidationClient

type GetStats struct {
	Body    map[string]interface{} `json:"body"`
	Message string                 `json:"message"`
	Status  int                    `json:"status"`
}

func InitLiquidationClient() {
	xlog.Debugf("Initializing Liquidation Client")
	baseUrl := os.Getenv("LIQUIDATION_URL")
	if baseUrl == "" {
		panic("LIQUIDATION_URL is not set")
	}
	GlobalLiquidationClient = &LiquidationClient{baseUrl: baseUrl}
}

func (lc *LiquidationClient) GetBaseUrl() string {
	return lc.baseUrl
}

func (lc *LiquidationClient) GetStats() (map[string]interface{}, error) {
	resp, err := http.Get(lc.GetBaseUrl() + "/stats")
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting liquidation stats: %s", resp.Status)
	}

	var getStatsResponse GetStats
	err = json.NewDecoder(resp.Body).Decode(&getStatsResponse)
	if err != nil {
		return nil, err
	}

	return getStatsResponse.Body, nil
}

func (lc *LiquidationClient) GetWithdrawableFundsStats() (map[string]interface{}, error) {
	resp, err := http.Get(lc.GetBaseUrl() + "/withdrawableFundsStats")
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting liquidation stats: %s", resp.Status)
	}

	var getStatsResponse GetStats
	err = json.NewDecoder(resp.Body).Decode(&getStatsResponse)
	if err != nil {
		return nil, err
	}

	return getStatsResponse.Body, nil
}
