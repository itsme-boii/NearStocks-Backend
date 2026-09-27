package xclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	apiserverTypes "github/eugenix-io/logx-inf-backend/services/api-server/types"
	"net/http"
	"os"
)

type ApiServerClient struct {
	baseUrl string
}

// Verify that the ApiServerClient implements the XClient interface
var _ XClient = &ApiServerClient{}

var GlobalApiServerClient *ApiServerClient

type PlaceLiquidationOrderResponse struct {
	Body    apiserverTypes.PlaceOrderResponse `json:"body"`
	Message string                            `json:"message"`
	Status  int                               `json:"status"`
}

type PlaceConditionalOrderResponse struct {
	Body    apiserverTypes.PlaceOrderResponse `json:"body"`
	Message string                            `json:"message"`
	Status  int                               `json:"status"`
}

// Quote represents a single quote with asks and bids
type Quote struct {
	Asks []struct {
		Price    string `json:"price"`
		Quantity string `json:"quantity"`
	} `json:"asks"`
	Bids []struct {
		Price    string `json:"price"`
		Quantity string `json:"quantity"`
	} `json:"bids"`
	Timestamp int64 `json:"timestamp"`
	MarketID  int   `json:"marketId"`
}

type CombinedOrderBook struct {
	Asks      []OrderbookLevel `json:"asks"`      // List of ask levels
	Bids      []OrderbookLevel `json:"bids"`      // List of bid levels
	Timestamp uint64           `json:"timestamp"` // Timestamp of the order book snapshot
	MarketId  uint             `json:"marketId"`  // Market identifier
}

type OrderbookLevel struct {
	Price    string `json:"price"`    // Decode as string
	Quantity string `json:"quantity"` // Decode as string
}

func InitApiServerClient() {
	xlog.Infof("Initializing Api Server Client")
	GlobalApiServerClient = NewApiServerClient()
}

func NewApiServerClient() *ApiServerClient {
	baseUrl := os.Getenv("API_SERVER_URL")
	if baseUrl == "" {
		panic("API_SERVER_URL is not set")
	}
	return &ApiServerClient{baseUrl: baseUrl}
}

func (c *ApiServerClient) GetBaseUrl() string {
	return c.baseUrl
}

// Error cases:
// - If request payload has some issues
// - There is some issue on the api server side or matching engine while matching the order
func (c *ApiServerClient) PlaceLiquidationOrder(requestPayload apiserverTypes.LiquidationOrderRequest) (*apiserverTypes.PlaceOrderResponse, error) {
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.GetBaseUrl()+"/api/v1/order/liquidation", "application/json", bytes.NewBuffer(jsonPayload))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error placing liquidation order: %s", resp.Status)
	}

	var placeOrderResponse PlaceLiquidationOrderResponse
	err = json.NewDecoder(resp.Body).Decode(&placeOrderResponse)
	if err != nil {
		return nil, err
	}

	return &placeOrderResponse.Body, nil
}

// Error cases:
// - If request payload has some issues
// - There is some issue on the api server side or matching engine while matching the order
func (c *ApiServerClient) PlaceConditionalOrder(requestPayload apiserverTypes.ConditionalOrderRequest) (*apiserverTypes.PlaceOrderResponse, int, error) {
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	resp, err := http.Post(c.GetBaseUrl()+"/api/v1/order/conditional", "application/json", bytes.NewBuffer(jsonPayload))

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("error placing conditional order: %s", resp.Status)
	}

	var placeOrderResponse PlaceConditionalOrderResponse
	err = json.NewDecoder(resp.Body).Decode(&placeOrderResponse)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return &placeOrderResponse.Body, resp.StatusCode, nil
}

func (c *ApiServerClient) SettleWithInsurance(requestPayload apiserverTypes.SettleWithInsuranceRequest) error {
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return err
	}
	resp, err := http.Post(c.GetBaseUrl()+"/api/v1/insurance/settle", "application/json", bytes.NewBuffer(jsonPayload))

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error settling with insurance: %s", resp.Status)
	}

	return nil
}

func (c *ApiServerClient) FinaliseDeposits() error {
	resp, err := http.Post(c.GetBaseUrl()+"/api/v1/token/finalisePendingDeposits", "application/json", nil)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error finalising deposits: %s", resp.Status)
	}

	return nil
}

func (c *ApiServerClient) CloseOptionBet(requestPayload contractUtils.PlaceOptionBetRedis) error {
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return err
	}
	resp, err := http.Post(c.GetBaseUrl()+"/api/v1/options/closeBetCronOnly", "application/json", bytes.NewBuffer(jsonPayload))

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error closing option bet: %s", resp.Status)
	}

	return nil
}

func (c *ApiServerClient) GetAllQuotes() ([]Quote, error) {
	if c == nil || c.baseUrl == "" {
		return nil, fmt.Errorf("internal server error")
	}

	resp, err := http.Get(c.baseUrl + "/api/v1/order/internalQuotes")
	if err != nil {
		return nil, fmt.Errorf("error making request to internal quotes API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching quotes: %s", resp.Status)
	}

	var result struct {
		Body []Quote `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	return result.Body, nil
}
func (c *ApiServerClient) GetAllCombinedQuotes() ([]CombinedOrderBook, error) {
	if c == nil || c.baseUrl == "" {
		return nil, fmt.Errorf("internal server error")
	}

	resp, err := http.Get(c.baseUrl + "/api/v1/order/internalCombinedQuotes")
	if err != nil {
		return nil, fmt.Errorf("error making request to internal combined quotes API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching combined quotes: %s", resp.Status)
	}

	var result struct {
		Body []CombinedOrderBook `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	return result.Body, nil
}

func (c *ApiServerClient) BurnKromaBalance(subaccountId string) error {
	if c == nil || c.baseUrl == "" {
		return fmt.Errorf("internal server error")
	}

	jsonPayload, err := json.Marshal(contractUtils.BurnBalanceKroma{SubAccountId: subaccountId})
	if err != nil {
		return fmt.Errorf("error marshalling JSON payload: %w", err)
	}

	resp, err := http.Post(c.baseUrl+"/api/v1/token/burnBalanceKroma", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error making request to burn KROMA balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error burning KROMA balance: %s", resp.Status)
	}

	return nil
}