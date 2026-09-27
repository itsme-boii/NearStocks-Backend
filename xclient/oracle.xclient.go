package xclient

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"

	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"net/http"
	"os"
	"strings"
)

type OracleClientImpl struct {
	baseUrl string
}

type OracleClient interface {
	XClient
	GetAllPrices() (map[string]TokenPrice, error)
	GetAllCollateralTokenPrices() (map[string]TokenPrice, error)
	GetFunctionCallCount(functionName string) int // This is just for testing purpose
}

// Validate OracleClientImpl implements OracleClient
var _ OracleClient = &OracleClientImpl{}

type OraclePrice struct {
	Price struct {
		Type string `json:"type" binding:"required"`
		Hex  string `json:"hex" binding:"required"`
	} `json:"price" binding:"required"`
	Expo        *int64 `json:"expo" binding:"required"`
	PublishTime int64  `json:"publishTime" binding:"required"` // Assume this won't be 0
}

type OraclePriceResponse struct {
	Data    map[string]OraclePrice `json:"data" binding:"required"`
	Tokens  []string               `json:"tokens" binding:"required"`
	Message string                 `json:"message"`
}

var GlobalOracleClient OracleClient

func NewOracleClient() *OracleClientImpl {
	baseUrl := os.Getenv("ORACLE_SERVER_URL")
	if baseUrl == "" {
		panic("ORACLE_SERVER_URL is not set")
	}
	return &OracleClientImpl{baseUrl: baseUrl}
}

func InitOracleClient() {
	xlog.Infof("Initialising Oracle Client")
	GlobalOracleClient = NewOracleClient()
}

func (oc *OracleClientImpl) GetBaseUrl() string {
	return oc.baseUrl
}

type TokenPrice struct {
	Token  string   `json:"token" binding:"required"`
	PriceX *big.Int `json:"price" binding:"required"`
	Expo   *int64   `json:"expo" binding:"required"`
}

func convertOraclePriceToTokenPrice(tokenSymbol string, oraclePrice OraclePrice) TokenPrice {
	priceInt := new(big.Int)
	priceInt.SetString(oraclePrice.Price.Hex[2:], 16)
	return TokenPrice{
		Token:  tokenSymbol,
		PriceX: priceInt,
		Expo:   oraclePrice.Expo,
	}
}

func (oc *OracleClientImpl) GetAllCollateralTokenPrices() (map[string]TokenPrice, error) {
	return oc.getAllPrices(contractUtils.ALL_COLLATERAL_TOKEN_SYMBOLS)
}

// Error cases:
// 1. Oracle server is down,
// 2. Oracle server returns an error
// 3. Oracle server returns invalid data or requested tokens do not exist
func (oc *OracleClientImpl) getAllPrices(tokens []string) (tokenPriceMap map[string]TokenPrice, err error) {
	// Join the tokens into a comma separated string

	queryParam := strings.Join(tokens, ",")
	var url string
	// If no tokens are provided, get all prices
	if len(tokens) == 0 {
		// xlog.Infof("Oracle Client - sending GET request to fetch prices of all tokens")
		url = fmt.Sprintf("%s/allprices", oc.baseUrl)
	} else {
		// xlog.Infof("Oracle Client - sending GET request to fetch prices of %d tokens", len(tokens))
		url = fmt.Sprintf("%s/prices?tokens=%s", oc.baseUrl, queryParam)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("oracle client - error getting prices from Oracle: %v", err)
	}
	defer resp.Body.Close()

	var oraclePriceResp OraclePriceResponse
	err = json.NewDecoder(resp.Body).Decode(&oraclePriceResp)
	if err != nil {
		return nil, fmt.Errorf("oracle client - error decoding Oracle response: %v | Response body: %+v | Response status: %+v", err, resp.Body, resp.Status)
	}

	tokenPriceMap = make(map[string]TokenPrice)
	for _, token := range oraclePriceResp.Tokens {
		tokenPriceMap[token] = convertOraclePriceToTokenPrice(token, oraclePriceResp.Data[token])
		// Add ostrich markets in the map as well
		if ostrichToken, ok := contractUtils.SYMBOL_TO_OSTRICH_SYMBOL[token]; ok {
			tokenPriceMap[ostrichToken] = convertOraclePriceToTokenPrice(ostrichToken, oraclePriceResp.Data[token])
		}
	}

	return tokenPriceMap, nil
}

func (oc *OracleClientImpl) GetAllPrices() (map[string]TokenPrice, error) {
	return oc.getAllPrices([]string{})
}

func (oc *OracleClientImpl) GetFunctionCallCount(functionName string) int {
	return 0
}
