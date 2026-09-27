package xclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	balanceTypes "github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
)

// This file will contain logic to interact with the balance service
// It will contain functions to update subaccount balances, perp balances etc
type BalanceClient struct {
	baseUrl string
}

// Verify that the BalanceClient implements the XClient interface
var _ XClient = &BalanceClient{}

type BalanceLockResponse struct {
	// Body will be nil
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type UnlockBalanceResponse struct {
	// Body will be nil
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type UpdateTokenBalanceRequest struct {
	SubaccountID string `json:"subaccountID"`
	ProductID    uint32 `json:"productId"`
	TokenBalance string `json:"tokenBalance"`
}

type UpdateTokenBalanceErrorResponse struct {
	Error string `json:"error"`
}

type PerpBalanceRequest struct {
	TakerSubaccountID  string `json:"takerSubaccountID" binding:"required"`
	MakerSubaccountID  string `json:"makerSubaccountID" binding:"required"`
	ProductId          uint32 `json:"productId" binding:"required"`
	TakerAmount        string `json:"takerAmount" binding:"required"`
	MakerAmount        string `json:"makerAmount" binding:"required"`
	TakerVQuoteBalance string `json:"takerVQuoteBalance" binding:"required"`
	MakerVQuoteBalance string `json:"makerVQuoteBalance" binding:"required"`
}

type BalancePerpResponse struct {
	Body struct {
		TakerSuccess     bool   `json:"taker_success"`
		MakerSuccess     bool   `json:"maker_success"`
		MakerRealizedPnl string `json:"maker_realized_pnl"`
		TakerRealizedPnl string `json:"taker_realized_pnl"`
		MakerFundingFees string `json:"maker_funding_fees"`
		TakerFundingFees string `json:"taker_funding_fees"`
	} `json:"body"`
	Status   int    `json:"status"`
	Messsage string `json:"message"`
}

type UpdateSubaccountsForLiquidationResponse struct {
	Body struct {
		MakerRealizedPnlx18 string `json:"maker_realized_pnl"`
		TakerRealizedPnlx18 string `json:"taker_realized_pnl"`
		MakerFundingFeesx18 string `json:"maker_funding_fees"`
		TakerFundingFeesx18 string `json:"taker_funding_fees"`
	} `json:"body"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type GetHealthResponse struct {
	Body    *balanceTypes.Health `json:"body"`
	Status  int                  `json:"status"`
	Message string               `json:"message"`
}

var GlobalBalanceClient *BalanceClient

func InitBalanceClient() {
	xlog.Infof("Initialising Balance Client")
	GlobalBalanceClient = NewBalanceClient()
}

func NewBalanceClient() *BalanceClient {
	baseUrl := os.Getenv("BALANCE_SERVER_URL")
	if baseUrl == "" {
		panic("BC - BALANCE_SERVER_URL is not set")
	}
	return &BalanceClient{baseUrl: baseUrl}
}

func (bcl *BalanceClient) GetBaseUrl() string {
	return bcl.baseUrl
}

// This will update the perp position for the given subaccounts as well as change the available balance
// Return success if both taker and maker subaccounts are updated successfully
// If taker subaccount is not updated, return false with error
func (bcl *BalanceClient) LockBalance(requestPayload balanceTypes.LockBalanceRequest) (success bool, err error) {
	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return false, err
	}

	xlog.Infof("BC - %v -  sending LockBalance request", requestPayload.EntityId)
	// Make the POST request to the balance service
	resp, err := http.Post(bcl.baseUrl+"/balance/lock-balance", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("BC - error locking balance: %s", resp.Status)
	}

	xlog.Infof("BC - balance server response for LockBalance request : %d", resp.StatusCode)

	var balanceLockResponse BalanceLockResponse
	err = json.NewDecoder(resp.Body).Decode(&balanceLockResponse)
	if err != nil {
		return false, err
	}
	if balanceLockResponse.Status != http.StatusOK {
		return false, fmt.Errorf("BC - error locking balance: %v | error msg: %v", balanceLockResponse.Status, balanceLockResponse.Message)
	}

	return true, nil
}

func (bcl *BalanceClient) UnlockFullBalance(requestPayload balanceTypes.UnlockBalanceRequest) (success bool, err error) {
	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return false, err
	}

	xlog.Infof("BC - sending UnlockFullBalance request for subAccount: %s, entityId: %v", requestPayload.SubaccountId, requestPayload.EntityId)
	// Make the POST request to the balance service
	resp, err := http.Post(bcl.baseUrl+"/balance/unlock-balance", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("BC - error updating subaccounts for match: %s", resp.Status)
	}
	xlog.Infof("BC - balance server response for UnlockFullBalance request : %d", resp.StatusCode)

	var unlockBalanceResponse UnlockBalanceResponse
	err = json.NewDecoder(resp.Body).Decode(&unlockBalanceResponse)
	if err != nil {
		return false, err
	}
	if unlockBalanceResponse.Status != http.StatusOK {
		return false, fmt.Errorf("BC - error updating subaccounts for match: %v | error msg: %v", unlockBalanceResponse.Status, unlockBalanceResponse.Message)
	}

	return true, nil
}

func (bcl *BalanceClient) GetBuyingPower(subaccountID string) (string, error) {
	url := fmt.Sprintf("%s/balance/getTotalBuyingPower?subaccountID=%s", bcl.baseUrl, subaccountID)
	resp, err := http.Get(url)
	if err != nil {
		return "0", err
	}

	if resp.StatusCode != http.StatusOK {
		return "0", fmt.Errorf("BC - error fetching buyingpower: %s", resp.Status)
	}

	var result struct {
		TotalBuyingPower string `json:"totalBuyingPower"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "0", err
	}
	return result.TotalBuyingPower, nil

}

func (bcl *BalanceClient) GetWithdrawableTokenBalance(subaccountID string) (map[uint32]string, error) {
	var result struct {
		WithdrawableBalance map[uint32]string `json:"withdrawableBalance"`
	}
	result.WithdrawableBalance = make(map[uint32]string)
	url := fmt.Sprintf("%s/balance/withdrawableTokenBalance?subaccountID=%s", bcl.baseUrl, subaccountID)
	resp, err := http.Get(url)
	if err != nil {
		return result.WithdrawableBalance, err
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result.WithdrawableBalance, err
	}
	return result.WithdrawableBalance, nil

}

// Fetch n or all perp positions for a subaccount.
// If productIds is empty, fetch all positions
func (bcl *BalanceClient) GetPerpPositions(subaccountID string, productIds []uint) ([]map[string]interface{}, error) {
	var url string
	baseUrl := os.Getenv("BALANCE_SERVER_URL")
	if len(productIds) == 0 {
		url = fmt.Sprintf("%s/balance/getPerpPositions?subaccountID=%s", baseUrl, subaccountID)
	} else {
		productIdsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(productIds)), ","), "[]")
		url = fmt.Sprintf("%s/balance/getPerpPositions?subaccountID=%s&productIds=%s", bcl.baseUrl, subaccountID, productIdsStr)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BC - error fetching perpetual positions: %s", resp.Status)
	}

	var result struct {
		Positions []map[string]interface{} `json:"PerpPositions"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	xlog.Infof("BC - fetched %d positions for subAccount %s", len(result.Positions), subaccountID)
	return result.Positions, nil
}

func (bcl *BalanceClient) GetFullBalance(subaccountID string, productIDs []uint32) (map[string]interface{}, error) {
	baseUrl := os.Getenv("BALANCE_SERVER_URL")
	url := fmt.Sprintf("%s/balance/getFullBalance/%s", baseUrl, subaccountID)
	fmt.Printf("URL: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-OK HTTP status: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading HTTP response body: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshalling HTTP response: %v", err)
	}

	bodyData, ok := response["body"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing 'body' field in response")
	}

	// web2Perp, ok := bodyData["web2Perp"].(map[string]interface{})
	// if !ok {
	// 	return nil, fmt.Errorf("missing 'web2Perp' field in response")
	// }

	// perpData, ok := web2Perp["PerpPositions"].([]interface{})
	// if !ok {
	// 	return nil, fmt.Errorf("missing or invalid 'PerpPositions' field in 'web2Perp'")
	// }

	// filteredWeb2Positions := []map[string]interface{}{}
	// for _, position := range perpData {
	// 	pos, ok := position.(map[string]interface{})
	// 	if !ok {
	// 		continue
	// 	}
	// 	productIDFloat, ok := pos["productID"].(float64)
	// 	if !ok {
	// 		continue
	// 	}
	// 	productID := uint32(productIDFloat)
	// 	for _, id := range productIDs {
	// 		if productID == id {
	// 			if tokenName, exists := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[productID]; exists {
	// 				pos["tokenName"] = tokenName
	// 				delete(pos, "productID")
	// 			}
	// 			filteredWeb2Positions = append(filteredWeb2Positions, pos)
	// 			break
	// 		}
	// 	}
	// }

	web3Perp, ok := bodyData["web3Perp"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing 'web3Perp' field in response")
	}

	web3PositionsData, ok := web3Perp["positions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'positions' field in 'web3Perp'")
	}

	filteredWeb3Positions := []map[string]interface{}{}
	for _, position := range web3PositionsData {
		pos, ok := position.(map[string]interface{})
		if !ok {
			continue
		}
		productIDFloat, ok := pos["productID"].(float64)
		if !ok {
			continue
		}
		productID := uint32(productIDFloat)
		for _, id := range productIDs {
			if productID == id {
				if tokenName, exists := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[productID]; exists {
					pos["tokenName"] = tokenName
					delete(pos, "productID")
				}
				filteredWeb3Positions = append(filteredWeb3Positions, pos)
				break
			}
		}
	}

	result := map[string]interface{}{
		"web3Perp": filteredWeb3Positions,
	}
	return result, nil
}

// func (bcl *BalanceClient) GetSubaccountContractData(subaccountID string) (map[uint]string, []map[string]interface{}, error) {
// 	baseUrl := os.Getenv("BALANCE_SERVER_URL")
// 	url := fmt.Sprintf("%s/balance/GetSubaccountContractData?subaccountID=%s", baseUrl, subaccountID)
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("error making HTTP request: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
// 	}

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("error reading HTTP response body: %v", err)
// 	}

// 	var response struct {
// 		PnLPerToken map[uint]string          `json:"pnlPerToken"`
// 		Positions   []map[string]interface{} `json:"positions"`
// 	}
// 	if err := json.Unmarshal(body, &response); err != nil {
// 		return nil, nil, fmt.Errorf("error unmarshalling HTTP response: %v", err)
// 	}
// 	return response.PnLPerToken, response.Positions, nil
// }

// func (bcl *BalanceClient) GetSubAccountUSDCBalance(subaccountID string) (float64, error) {

// 	type SubaccountBalanceResponse struct {
// 		ContractNonce string    `json:"contract_nonce"`
// 		Positions     []float64 `json:"positions"`
// 		RedisNonce    string    `json:"redis_nonce"`
// 	}

// 	baseUrl := os.Getenv("BALANCE_SERVER_URL")
// 	// Construct the URL
// 	url := fmt.Sprintf("%s/balance/GetSubaccountSpotContractData?subaccountID=%s", baseUrl, subaccountID)

// 	// Make the HTTP GET request
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to make request: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	// Check for non-200 status code
// 	if resp.StatusCode != http.StatusOK {
// 		return 0, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
// 	}

// 	// Read the response body
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to read response body: %v", err)
// 	}

// 	// Parse the JSON response
// 	var balanceResponse SubaccountBalanceResponse
// 	if err := json.Unmarshal(body, &balanceResponse); err != nil {
// 		return 0, fmt.Errorf("failed to parse JSON: %v", err)
// 	}

// 	// Check if the positions array has at least one element
// 	if len(balanceResponse.Positions) == 0 {
// 		return 0, fmt.Errorf("positions array is empty")
// 	}

// 	// Return the 0th index of the positions array
// 	return balanceResponse.Positions[0], nil
// }

// func (bcl *BalanceClient) SyncSubaccountBalances(subaccountIDs []string) (string, error) {
// 	url := fmt.Sprintf("%s/balance/syncBalances", bcl.baseUrl)

// 	requestBody := map[string]interface{}{
// 		"subaccountIDs": subaccountIDs,
// 	}
// 	jsonBody, err := json.Marshal(requestBody)
// 	if err != nil {
// 		return "", fmt.Errorf("BC - error marshaling request body: %v", err)
// 	}

// 	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
// 	if err != nil {
// 		return "", fmt.Errorf("BC - error making POST request to sync subaccount balances: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		bodyBytes, _ := io.ReadAll(resp.Body)
// 		return "", fmt.Errorf("BC - syncSubaccountBalances returned non-OK status (%d): %s",
// 			resp.StatusCode, string(bodyBytes))
// 	}

// 	var result struct {
// 		Message string `json:"message,omitempty"`
// 		Error   string `json:"error,omitempty"`
// 	}
// 	err = json.NewDecoder(resp.Body).Decode(&result)
// 	if err != nil {
// 		return "", fmt.Errorf("BC - error decoding syncSubaccountBalances response: %v", err)
// 	}

// 	if result.Error != "" {
// 		return "", fmt.Errorf("BC - syncSubaccountBalances server error: %s", result.Error)
// 	}

// 	return result.Message, nil
// }

func (bcl *BalanceClient) GetPerpSinglePosition(subaccountID string, productId uint) (*balanceTypes.PerpBalance, error) {
	perpBalances, err := bcl.GetPerpPositions(subaccountID, []uint{productId})
	if err != nil {
		return nil, err
	}
	if len(perpBalances) == 0 {
		return nil, fmt.Errorf("BC - no position found for subaccount %s and product %d", subaccountID, productId)
	}

	perpBalance := perpBalances[0]
	var ret balanceTypes.PerpBalance
	cutils.ConvertMapToStructViaJson(perpBalance, &ret)

	return &ret, nil
}

func (bcl *BalanceClient) GetSpotBalance(subaccountID string) ([]ctypes.SpotBalance, string, string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/balance/getSpotBalance?subaccountID=%s", bcl.baseUrl, subaccountID))
	if err != nil {
		return nil, "0", "0", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "0", "0", fmt.Errorf("BC - error fetching spot balance: %s", resp.Status)
	}

	var result struct {
		FreeBalance string               `json:"freeBalance"`
		UsedBalance string               `json:"usedBalance"`
		SpotBalance []ctypes.SpotBalance `json:"spotBalance"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, "0", "0", err
	}

	xlog.Infof("BC - for subAccount %s Free Balance is %s and Used Balance is %s", subaccountID, result.FreeBalance, result.UsedBalance)
	return result.SpotBalance, result.FreeBalance, result.UsedBalance, nil
}

func (bcl *BalanceClient) GetSpotBalanceInternal(subaccountID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/balance/getSpotBalance?subaccountID=%s", bcl.baseUrl, subaccountID)
	xlog.Infof("BC - sending request to get spot balance for subAccount %s", subaccountID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("BC - error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BC - received non-OK HTTP status: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("BC - error reading HTTP response body: %v", err)
	}

	var response struct {
		FreeBalance string               `json:"freeBalance"`
		UsedBalance string               `json:"usedBalance"`
		SpotBalance []ctypes.SpotBalance `json:"spotBalance"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("BC - error unmarshalling HTTP response: %v", err)
	}

	result := map[string]interface{}{
		"freeBalance": response.FreeBalance,
		"usedBalance": response.UsedBalance,
		"spotBalance": response.SpotBalance,
	}

	return result, nil
}

func (bcl *BalanceClient) GetPreMarketBalance(subaccountID string, productId uint32) (string, error) {
	url := fmt.Sprintf("%s/balance/get-pre-market-balance?subaccountID=%s&productID=%d", bcl.baseUrl, subaccountID, productId)

	xlog.Infof("BC - sending request to get pre-market balance of product ID %d in subAccount %s", productId, subaccountID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("BC - error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("BC - error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		xlog.Infof("BC - balance server response for GetPreMarketBalance request: %d", resp.StatusCode)

		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return "", fmt.Errorf("BC - error decoding error response: %v", err)
		}
		return "", fmt.Errorf("BC - error response from server: %s", errorResponse.Error)
	}

	var response struct {
		Balance string `json:"balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("BC - error decoding response: %v", err)
	}

	return response.Balance, nil
}

func (bcl *BalanceClient) GetPreMarketBalances(subaccountID string) (map[uint32]string, error) {
	url := fmt.Sprintf("%s/balance/get-pre-market-balances?subaccountID=%s", bcl.baseUrl, subaccountID)

	xlog.Infof("BC - sending request to get pre-market balances for subAccount %s", subaccountID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("BC - error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BC - error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		xlog.Infof("BC - balance server response for GetPreMarketBalances request: %d", resp.StatusCode)

		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return nil, fmt.Errorf("BC - error decoding error response: %v", err)
		}
		return nil, fmt.Errorf("BC - error response from server: %s", errorResponse.Error)
	}

	var response struct {
		Balances map[uint32]string `json:"balances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("BC - error decoding response: %v", err)
	}

	return response.Balances, nil
}

func (bcl *BalanceClient) GetSyntheticSpotBalance(subaccountID string, productId uint32) (string, error) {
	url := fmt.Sprintf("%s/balance/get-synthetic-spot-balance?subaccountID=%s&productID=%d", bcl.baseUrl, subaccountID, productId)

	xlog.Infof("BC - sending request to get synthetic spot balance of product ID %d in subAccount %s", productId, subaccountID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("BC - error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("BC - error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		xlog.Infof("BC - balance server response for GetSyntheticSpotBalance request: %d", resp.StatusCode)

		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return "", fmt.Errorf("BC - error decoding error response: %v", err)
		}
		return "", fmt.Errorf("BC - error response from server: %s", errorResponse.Error)
	}

	var response struct {
		Balance string `json:"balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("BC - error decoding response: %v", err)
	}

	return response.Balance, nil
}

func (bcl *BalanceClient) GetSyntheticSpotBalances(subaccountID string) (map[uint32]string, error) {
	url := fmt.Sprintf("%s/balance/get-synthetic-spot-balances?subaccountID=%s", bcl.baseUrl, subaccountID)

	xlog.Infof("BC - sending request to get synthetic spot balances for subAccount %s", subaccountID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("BC - error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BC - error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		xlog.Infof("BC - balance server response for GetSyntheticSpotBalances request: %d", resp.StatusCode)

		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return nil, fmt.Errorf("BC - error decoding error response: %v", err)
		}
		return nil, fmt.Errorf("BC - error response from server: %s", errorResponse.Error)
	}

	var response struct {
		Balances map[uint32]string `json:"balances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("BC - error decoding response: %v", err)
	}

	return response.Balances, nil
}

func (bcl *BalanceClient) UpdateTokenBalance(subaccountID string, productId uint32, tokenBalance string) (bool, error) {
	url := fmt.Sprintf("%s/balance/update-token-balance", bcl.baseUrl)

	requestBody := UpdateTokenBalanceRequest{
		SubaccountID: subaccountID,
		ProductID:    productId,
		TokenBalance: tokenBalance,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("BC - error marshaling request body: %v", err)
	}

	xlog.Infof("BC - sending request to UpdateTokenBalance of product ID %d in subAccount %s to %s", productId, subaccountID, tokenBalance)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, fmt.Errorf("BC - error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("BC - error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	xlog.Infof("BC - balance server response for UpdateTokenBalance request : %d", resp.StatusCode)

	var errorResponse UpdateTokenBalanceErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		return false, fmt.Errorf("BC - error decoding error response: %v", err)
	}

	return false, fmt.Errorf("BC - error response from server: %s", errorResponse.Error)
}

func (bcl *BalanceClient) UpdatePreMarketBalance(subaccountID string, productId uint32, preMarketBalance string) (bool, error) {
	url := fmt.Sprintf("%s/balance/update-pre-market-balance", bcl.baseUrl)

	requestBody := UpdateTokenBalanceRequest{
		SubaccountID: subaccountID,
		ProductID:    productId,
		TokenBalance: preMarketBalance,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("BC - error marshaling request body: %v", err)
	}

	xlog.Infof("BC - sending request to UpdatePreMarketBalance of product ID %d in subAccount %s to %s", productId, subaccountID, preMarketBalance)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, fmt.Errorf("BC - error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("BC - error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	xlog.Infof("BC - balance server response for UpdatePreMarketBalance request : %d", resp.StatusCode)

	var errorResponse UpdateTokenBalanceErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		return false, fmt.Errorf("BC - error decoding error response: %v", err)
	}

	return false, fmt.Errorf("BC - error response from server: %s", errorResponse.Error)
}

func (bcl *BalanceClient) UpdateSyntheticSpotBalance(subaccountID string, productId uint32, syntheticSpotBalance string) (bool, error) {
	url := fmt.Sprintf("%s/balance/update-synthetic-spot-balance", bcl.baseUrl)

	requestBody := UpdateTokenBalanceRequest{
		SubaccountID: subaccountID,
		ProductID:    productId,
		TokenBalance: syntheticSpotBalance,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("BC - error marshaling request body: %v", err)
	}

	xlog.Infof("BC - sending request to UpdateSyntheticSpotBalance of product ID %d in subAccount %s to %s", productId, subaccountID, syntheticSpotBalance)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, fmt.Errorf("BC - error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("BC - error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	xlog.Infof("BC - balance server response for UpdateSyntheticSpotBalance request : %d", resp.StatusCode)

	var errorResponse UpdateTokenBalanceErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		return false, fmt.Errorf("BC - error decoding error response: %v", err)
	}

	return false, fmt.Errorf("BC - error response from server: %s", errorResponse.Error)
}

func (bcl *BalanceClient) UpdateMultiTokenBalance(requestPayload *balanceTypes.MultiTokenBalanceRequest) error {
	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return err
	}

	// Make the POST request to the balance service
	resp, err := http.Post(bcl.baseUrl+"/balance/update-multi-token-balance", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errorResponse UpdateTokenBalanceErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
		return fmt.Errorf("BC - error decoding error response: %v", err)
	}

	return fmt.Errorf("BC - error response from server: %s", errorResponse.Error)
}

// This will update the perp position for the given subaccounts as well as change the available balance
// Return success if both taker and maker subaccounts are updated successfully
// If taker subaccount is not updated, return false with error
func (bcl *BalanceClient) UpdateSubaccountsForMatch(requestPayload balanceTypes.UpdateSubaccountForMatchRequest) (takerSuccess, makerSuccess bool, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees *big.Int, err error) {

	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return false, false, nil, nil, nil, nil, err
	}

	// Make the POST request to the balance service
	resp, err := http.Post(bcl.baseUrl+"/balance/subaccount-match", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return false, false, nil, nil, nil, nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, false, nil, nil, nil, nil, fmt.Errorf("error updating subaccounts for match: %s", resp.Status)
	}

	var balancePerpResponse BalancePerpResponse
	err = json.NewDecoder(resp.Body).Decode(&balancePerpResponse)
	if err != nil {
		return false, false, nil, nil, nil, nil, err
	}
	if balancePerpResponse.Status != http.StatusOK {
		return false, false, nil, nil, nil, nil, fmt.Errorf("error updating subaccounts for match: %s", resp.Status)
	}

	if balancePerpResponse.Messsage != "" {
		xlog.Warnf("There was some issue while updating subaccounts for match: %s | But this is non blocking.: %+v", balancePerpResponse.Messsage, string(jsonPayload))
	}

	takerSuccess = balancePerpResponse.Body.TakerSuccess
	makerSuccess = balancePerpResponse.Body.MakerSuccess

	// Convert all values from strings to *big.Int
	makerRealizedPnl = new(big.Int)
	takerRealizedPnl = new(big.Int)
	makerFundingFees = new(big.Int)
	takerFundingFees = new(big.Int)

	if _, ok := makerRealizedPnl.SetString(balancePerpResponse.Body.MakerRealizedPnl, 10); !ok {
		xlog.Infof("Error converting makerRealizedPnl to *big.Int: %s", balancePerpResponse.Body.MakerRealizedPnl)
	}

	if _, ok := takerRealizedPnl.SetString(balancePerpResponse.Body.TakerRealizedPnl, 10); !ok {
		xlog.Infof("Error converting takerRealizedPnl to *big.Int: %s", balancePerpResponse.Body.TakerRealizedPnl)
	}

	if _, ok := makerFundingFees.SetString(balancePerpResponse.Body.MakerFundingFees, 10); !ok {
		xlog.Infof("Error converting makerFundingFees to *big.Int: %s", balancePerpResponse.Body.MakerFundingFees)
	}

	if _, ok := takerFundingFees.SetString(balancePerpResponse.Body.TakerFundingFees, 10); !ok {
		xlog.Infof("Error converting takerFundingFees to *big.Int: %s", balancePerpResponse.Body.TakerFundingFees)
	}

	return takerSuccess, makerSuccess, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees, nil
}

// TODO: Define handling in failure cases
func (bcl *BalanceClient) UpdateSubaccountsForLiquidation(requestPayload balanceTypes.FinaliseLiquidationRequest) (makerRealizedPnlx18 *big.Int, takerRealizedPnlx18 *big.Int, makerFundingFeesx18 *big.Int, takerFundingFeesx18 *big.Int, err error) {
	makerRealizedPnlx18 = big.NewInt(0)
	takerRealizedPnlx18 = big.NewInt(0)
	makerFundingFeesx18 = big.NewInt(0)
	takerFundingFeesx18 = big.NewInt(0)
	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Make the POST request to the balance service
	resp, err := http.Post(bcl.baseUrl+"/balance/subaccount-liquidation-match", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return makerRealizedPnlx18, takerRealizedPnlx18, makerFundingFeesx18, takerFundingFeesx18, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return makerRealizedPnlx18, takerRealizedPnlx18, makerFundingFeesx18, takerFundingFeesx18, fmt.Errorf("error updating subaccounts for liquidation: %s", resp.Status)
	}

	var updateSubaccountsForLiquidationResponse UpdateSubaccountsForLiquidationResponse
	err = json.NewDecoder(resp.Body).Decode(&updateSubaccountsForLiquidationResponse)
	if err != nil {
		return makerRealizedPnlx18, takerRealizedPnlx18, makerFundingFeesx18, takerFundingFeesx18, err
	}
	if updateSubaccountsForLiquidationResponse.Status != http.StatusOK {
		return makerRealizedPnlx18, takerRealizedPnlx18, makerFundingFeesx18, takerFundingFeesx18, fmt.Errorf("error updating subaccounts balances for liquidation: %s", resp.Status)
	}

	if _, ok := makerRealizedPnlx18.SetString(updateSubaccountsForLiquidationResponse.Body.MakerRealizedPnlx18, 10); !ok {
		xlog.Infof("Error converting makerRealizedPnl to *big.Int: %s", updateSubaccountsForLiquidationResponse.Body.MakerRealizedPnlx18)
	}

	if _, ok := takerRealizedPnlx18.SetString(updateSubaccountsForLiquidationResponse.Body.TakerRealizedPnlx18, 10); !ok {
		xlog.Infof("Error converting takerRealizedPnl to *big.Int: %s", updateSubaccountsForLiquidationResponse.Body.TakerRealizedPnlx18)
	}

	if _, ok := makerFundingFeesx18.SetString(updateSubaccountsForLiquidationResponse.Body.MakerFundingFeesx18, 10); !ok {
		xlog.Infof("Error converting makerFundingFees to *big.Int: %s", updateSubaccountsForLiquidationResponse.Body.MakerFundingFeesx18)
	}

	if _, ok := takerFundingFeesx18.SetString(updateSubaccountsForLiquidationResponse.Body.TakerFundingFeesx18, 10); !ok {
		xlog.Infof("Error converting takerFundingFees to *big.Int: %s", updateSubaccountsForLiquidationResponse.Body.TakerFundingFeesx18)
	}

	return makerRealizedPnlx18, takerRealizedPnlx18, makerFundingFeesx18, takerFundingFeesx18, nil
}

func (bcl *BalanceClient) SettleUsingInsuranceFunds(requestPayload balanceTypes.SettleWithInsuranceRequest) error {
	// Convert the payload to JSON
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return err
	}

	// Make the POST request to the balance service
	resp, err := http.Post(bcl.baseUrl+"/balance/settle-with-insurance", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error settling with insurance: %s", resp.Status)
	}

	var balanceLockResponse BalanceLockResponse
	err = json.NewDecoder(resp.Body).Decode(&balanceLockResponse)
	if err != nil {
		return err
	}
	if balanceLockResponse.Status != http.StatusOK {
		return fmt.Errorf("error settling with insurance: %s", balanceLockResponse.Message)
	}

	return nil
}

func (bcl *BalanceClient) SettlePnLForSubaccounts(subaccountIDs []string) (map[uint32]*big.Int, map[string]string, error) {
	// Construct the API URL
	url := fmt.Sprintf("%s/balance/settlePnl", bcl.baseUrl)

	// Prepare the request body
	requestBody := map[string]interface{}{
		"subaccountIDs": subaccountIDs,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, nil, fmt.Errorf("BC - error marshaling request body: %v", err)
	}

	// Create the POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, nil, fmt.Errorf("BC - error making POST request to settle PnL: %v", err)
	}
	defer resp.Body.Close()

	// Check if the status code is OK
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("BC - error settling PnL: %s", resp.Status)
	}

	// Decode the response
	var result struct {
		TokenPrices       map[uint32]*big.Int `json:"tokenPrices"`
		FailedSubAccounts map[string]string   `json:"failedSubAccounts"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, nil, fmt.Errorf("BC - error decoding response: %v", err)
	}

	// Return the token prices map
	return result.TokenPrices, result.FailedSubAccounts, nil
}

func (bcl *BalanceClient) GetHealth(subaccountHex string) (*balanceTypes.Health, error) {
	url := fmt.Sprintf("%s/balance/health/%s", bcl.baseUrl, subaccountHex)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BC - error fetching health: %s", resp.Status)
	}

	var result GetHealthResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

// func (bcl *BalanceClient) GetSubAccountContractSpotBalance(subaccountID string, productID int) (*big.Int, error) {
// 	type SubaccountBalanceResponse struct {
// 		ContractNonce string     `json:"contract_nonce"`
// 		Positions     []*big.Int `json:"positions"` // Assume positions are float64 for conversion
// 		RedisNonce    string     `json:"redis_nonce"`
// 	}

// 	baseUrl := os.Getenv("BALANCE_SERVER_URL")
// 	// Construct the URL
// 	url := fmt.Sprintf("%s/balance/GetSubaccountSpotContractData?subaccountID=%s", baseUrl, subaccountID)

// 	// Make the HTTP GET request
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to make request: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	// Check for non-200 status code
// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
// 	}

// 	// Read the response body
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read response body: %v", err)
// 	}

// 	// Parse the JSON response
// 	var balanceResponse SubaccountBalanceResponse
// 	if err := json.Unmarshal(body, &balanceResponse); err != nil {
// 		return nil, fmt.Errorf("failed to parse JSON: %v", err)
// 	}

// 	// Check if the positions array has the required product ID
// 	if productID < 0 || productID >= len(balanceResponse.Positions) {
// 		return nil, fmt.Errorf("no position found for productID: %d", productID)
// 	}
// 	positionBigInt := new(big.Int)

// 	productIndex := getIndexFromSpotProductID(uint32(productID))

// 	positionBigInt = balanceResponse.Positions[productIndex]
// 	// Return the balance as *big.Int
// 	return positionBigInt, nil
// }

func getIndexFromSpotProductID(productID uint32) int {
	switch productID {
	case 4:
		return 0
	case 0:
		return 1
	case 2:
		return 2
	default:
		if productID%2 == 0 {
			return int(productID / 2)
		}
		// Return -1 for invalid productID
		return -1
	}
}

func (bcl *BalanceClient) GetAvailableMargin(subaccountID string) (*big.Int, error) {
	if subaccountID == "" {
		return nil, fmt.Errorf("subaccountID is required")
	}

	// Build the URL with query parameters
	url := fmt.Sprintf("%s/balance/getAvailableMargin?subaccountID=%s", bcl.baseUrl, subaccountID)
	xlog.Infof("BC - Sending request to get available margin for subaccountID: %s", subaccountID)

	// Make the GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("BC - error making request to get available margin: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BC - error response from server: %s", resp.Status)
	}

	// Decode the response
	var response map[string]string
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("BC - error decoding response: %v", err)
	}

	// Parse the available margin
	availableMarginStr, exists := response["availableMargin"]
	if !exists {
		return nil, fmt.Errorf("BC - missing availableMargin in response")
	}

	availableMargin := new(big.Int)
	_, success := availableMargin.SetString(availableMarginStr, 10)
	if !success {
		return nil, fmt.Errorf("BC - error converting availableMargin to big.Int")
	}

	xlog.Infof("BC - Successfully retrieved available margin: %s for subaccountID: %s", availableMargin.String(), subaccountID)
	return availableMargin, nil
}

func (bcl *BalanceClient) SettleBroker2PnL(subaccountID string) error {
	// Construct the API URL
	url := fmt.Sprintf("%s/balance/settleBroker2Pnl/%s", bcl.baseUrl, subaccountID)

	// Create the POST request
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("BC - error making POST request to settle Ostrich PnL: %v", err)
	}
	defer resp.Body.Close()

	// Check if the status code is OK
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("BC - settle Ostrich PnL returned non-OK status (%d): %s",
			resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (bcl *BalanceClient) ShiftKromaFunds(subaccountID string) error {
	// Construct the API URL
	url := fmt.Sprintf("%s/balance/shiftKromaFunds/%s", bcl.baseUrl, subaccountID)

	// Create the POST request
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("BC - error making POST request to shift Kroma funds: %v", err)
	}
	defer resp.Body.Close()

	// Check if the status code is OK
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("BC - shift Kroma funds returned non-OK status (%d): %s",
			resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (bcl *BalanceClient) BurnKromaFunds(subaccountID string) error {
	// Construct the API URL
	url := fmt.Sprintf("%s/balance/burnKromaFunds/%s", bcl.baseUrl, subaccountID)

	// Create the POST request
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("BC - error making POST request to burn Kroma funds: %v", err)
	}
	defer resp.Body.Close()

	// Check if the status code is OK
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("BC - burn Kroma funds returned non-OK status (%d): %s",
			resp.StatusCode, string(bodyBytes))
	}

	return nil
}
