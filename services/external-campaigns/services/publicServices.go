package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"

	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/redis/go-redis/v9"
)

// MarketContract represents the structure of a market contract
type MarketContract struct {
	TickerID                 string   `json:"ticker_id"`
	BaseCurrency             string   `json:"base_currency"`
	QuoteCurrency            string   `json:"quote_currency"`
	LastPrice                float64  `json:"last_price"`
	BaseVolume               float64  `json:"base_volume"`
	QuoteVolume              float64  `json:"quote_volume"`
	USDVolume                float64  `json:"usd_volume,omitempty"`
	Bid                      float64  `json:"bid"`
	Ask                      float64  `json:"ask"`
	High                     float64  `json:"high"`
	Low                      float64  `json:"low"`
	ProductType              string   `json:"product_type"`
	OpenInterest             float64  `json:"open_interest"`
	OpenInterestUSD          float64  `json:"open_interest_usd"`
	IndexPrice               float64  `json:"index_price"`
	CreationTimestamp        *int64   `json:"creation_timestamp"`
	ExpiryTimestamp          *int64   `json:"expiry_timestamp"`
	FundingRate              float64  `json:"funding_rate"`
	NextFundingRate          *float64 `json:"next_funding_rate"`
	NextFundingRateTimestamp *int64   `json:"next_funding_rate_timestamp"`
	ContractType             string   `json:"contract_type"`
	ContractPrice            float64  `json:"contract_price"`
	ContractPriceCurrency    string   `json:"contract_price_currency"`
	MakerFee                 *float64 `json:"maker_fee"`
	TakerFee                 *float64 `json:"taker_fee"`
}

// PublicApi manages interactions with the app state
type PublicApi struct {
	appState    appstate.AppState
	redisClient *redis.Client
}

type OrderbookLevel struct {
	Price    string `json:"price"`    // Price as a string
	Quantity string `json:"quantity"` // Quantity as a string
}

// UserDashboardData represents user dashboard statistics
type UserDashboardData struct {
	WalletAddressShortened string  `json:"wallet_address_shortened"`
	WalletAddressFull      string  `json:"wallet_address_full"`
	TotalNetPnLUSD         float64 `json:"total_net_pnl_usd"`
	WinRate                float64 `json:"win_rate"`
	TotalTrades            int64   `json:"total_trades"`
	VolumeUSD              float64 `json:"volume_usd"`
	AvgTradeSize           float64 `json:"avg_trade_size"`
	FavoritePair           string  `json:"favorite_pair"`
	ActiveDays             int64   `json:"active_days"`
	LastTrade              string  `json:"last_trade"`
	AccountAgeDays         float64 `json:"account_age_days"`
}

// PaginatedUserDashboardResponse represents paginated user dashboard data
type PaginatedUserDashboardResponse struct {
	Data       []UserDashboardData `json:"data"`
	TotalCount int64               `json:"total_count"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

// DepositHistoryResponse represents a deposit transaction
type DepositHistoryResponse struct {
	TxnHash            *string `json:"txn_hash"`
	Amount             string  `json:"amount"`
	SourceChainID      uint64  `json:"source_chain_id"`
	DestinationChainID uint64  `json:"destination_chain_id"`
	CreatedAt          string  `json:"createdAt"`
	IsDeposit          bool    `json:"is_deposit"` // Add this field to distinguish deposits from withdrawals
}

// TradeHistoryResponse represents a trade transaction
type TradeHistoryResponse struct {
	TxnHash       *string `json:"txn_hash"`
	Amount        string  `json:"amount"`
	ProductID     uint64  `json:"product_id"`
	RealisedPnl   string  `json:"realised_pnl"`
	CreatedAt     string  `json:"createdAt"`
	Type          string  `json:"type"`
	Price         string  `json:"price"`
	Fee           string  `json:"fee"`
	IsLiquidation bool    `json:"is_liquidation"`
}

// PaginatedTradeHistoryResponse represents paginated trade history data
type PaginatedTradeHistoryResponse struct {
	Data       []TradeHistoryResponse `json:"data"`
	TotalCount int64                  `json:"total_count"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
}

// UserDashboardSubAccountIdData represents dashboard data for a specific subaccount
type UserDashboardSubAccountIdData struct {
	TotalNetUnrealisedPnLUSD        *big.Int                       `json:"total_net_unrealised_pnl_usd"`
	DateOfJoining                   string                         `json:"date_of_joining"`
	FundingFeeUSD                   string                         `json:"funding_fee_usd"`
	DepositHistory                  []DepositHistoryResponse       `json:"deposit_history"`
	DailyPnlGraph                   []map[string]interface{}       `json:"daily_pnl_graph"`
	OpenPositions                   []*ctypes.PositionSummary      `json:"open_positions"`
	TradeHistory                    *PaginatedTradeHistoryResponse `json:"trade_history"`
	TotalNetUnrealisedFundingFeeUSD *big.Int                       `json:"total_net_unrealised_fundingFee_usd"`
	SpotBalance                     []ctypes.SpotBalance           `json:"spot_balance"`
	FreeBalance                     string                         `json:"free_balance"`
	UsedBalance                     string                         `json:"used_balance"`
}

// NewPublicApi initializes and returns a new PublicApi instance
func NewPublicApi() *PublicApi {
	return &PublicApi{
		appState:    appstate.NewAppState(),
		redisClient: xredis.GetRedisClient(),
	}
}

// FetchFundingRatesAndTimestamps retrieves funding rate and calculates the next funding rate timestamp
func (api *PublicApi) FetchFundingRatesAndTimestamps(ctx context.Context, productID uint32) (float64, int64, error) {
	// Fetch symbol for the product ID
	symbol, exists := marketutils.GetFundingSymbolForProduct(productID)
	if !exists {
		xlog.Warnf("Symbol not found for productID: %d", productID)
		return 0, 0, fmt.Errorf("symbol not found for productID: %d", productID)
	}
	symbol += "-USD" // Append "-USD" to the symbol
	// Prepare Redis key for funding rate
	fundingRateKey := xredis.GetFundingRateKey(symbol)

	// Fetch funding rate data from Redis
	fundingRateStr, err := api.redisClient.Get(ctx, fundingRateKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, 0, fmt.Errorf("funding rate not found for symbol %s", symbol)
		}
		return 0, 0, fmt.Errorf("error fetching funding rate for symbol %s: %v", symbol, err)
	}

	if fundingRateStr == "" {
		return 0, 0, fmt.Errorf("funding rate not found for symbol %s", symbol)
	}

	// Parse and unmarshal funding rate data
	var fundingRateData xredis.FundingRateData
	err = json.Unmarshal([]byte(fundingRateStr), &fundingRateData)
	if err != nil {
		return 0, 0, fmt.Errorf("error unmarshalling funding rate JSON for symbol %s: %v", symbol, err)
	}

	// Parse funding rate
	fundingRate, err := strconv.ParseInt(fundingRateData.FundingRate, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("error converting funding rate string to int for symbol %s: %v", symbol, err)
	}

	// Calculate the next funding rate timestamp (aligned with even 2-minute mark)
	nextFundingRateTimestamp := getNextEven2MinuteTimestamp()

	return float64(fundingRate), nextFundingRateTimestamp, nil
}

// Helper function to calculate the next even 2-minute timestamp
func getNextEven2MinuteTimestamp() int64 {
	now := time.Now()
	currentMinutes := now.Minute()
	nextEvenMinute := (currentMinutes/2 + 1) * 2 // Round up to the next multiple of 2

	// Set the timestamp to the next even minute mark
	nextTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		now.Hour(), nextEvenMinute, 0, 0, now.Location(),
	)

	return nextTime.UnixMilli()
}

// FetchOraclePrice fetches the oracle price for a given market ID
func (api *PublicApi) FetchOraclePrice(marketID uint32) (*big.Int, error) {
	// Set a fetch expiration time for the Oracle prices
	fetchExpiration := time.Second * 1

	// Retrieve all Oracle prices from the app state
	oraclePricesMap, err := api.appState.GetAllOraclePrices(fetchExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Oracle prices: %w", err)
	}

	// Retrieve the symbol for the given market ID
	symbol, exists := marketutils.GetBaseSymbolForProduct(marketID)
	if !exists {
		return nil, fmt.Errorf("invalid market ID: %d", marketID)
	}

	// Get the price from the Oracle prices map
	price, found := oraclePricesMap[symbol]
	if !found {
		return nil, fmt.Errorf("price not found for symbol: %s", symbol)
	}

	return price.Pricex18, nil
}

// getOIMultiplier retrieves the OI multiplier from an environment variable or returns the default value
func getOIMultiplier() float64 {
	const defaultMultiplier = 1.4

	// Retrieve the OI multiplier from the environment variable
	oiMultiplierStr := os.Getenv("OI_MULTIPLIER_CONSTANT")
	if oiMultiplierStr == "" {
		xlog.Infof("OI_MULTIPLIER_CONSTANT not set, using default value: %.2f", defaultMultiplier)
		return defaultMultiplier
	}

	// Parse the multiplier value
	oiMultiplier, err := strconv.ParseFloat(oiMultiplierStr, 64)
	if err != nil {
		xlog.Errorf("Error parsing OI_MULTIPLIER_CONSTANT: %v. Using default value: %.2f", err, defaultMultiplier)
		return defaultMultiplier
	}

	return oiMultiplier
}
func roundToThreeDecimals(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func (api *PublicApi) FetchOrderBookByTicker(ctx context.Context, ticker string) (map[string]interface{}, error) {
	// Validate and extract the base symbol from the ticker (e.g., "btc" from "btc_usd")
	tickerParts := strings.Split(ticker, "_")
	if len(tickerParts) < 2 {
		return nil, fmt.Errorf("invalid ticker format: %s. Expected format is 'base_quote'", ticker)
	}
	baseSymbol := strings.ToLower(tickerParts[0]) // Normalize the symbol to lowercase

	// Reverse map to get product ID from the symbol
	symbolToProductID := reverseProductIDSymbolMap(contractUtils.PRODUCT_ID_SYMBOL_TO_MAP)
	productID, exists := symbolToProductID[baseSymbol]
	if !exists {
		return nil, fmt.Errorf("no matching product ID found for base symbol: %s", baseSymbol)
	}

	// Fetch combined quotes from the API
	combinedQuotes, err := xclient.GlobalApiServerClient.GetAllCombinedQuotes()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch combined quotes from API: %w", err)
	}
	// Match the product ID with the returned market ID in the combined quotes
	for _, orderBook := range combinedQuotes {
		if orderBook.MarketId == uint(productID) {
			// Format and return the order book response
			return map[string]interface{}{
				"s": ticker,
				"t": orderBook.Timestamp,
				"b": formatOrderbookLevels(orderBook.Bids, 50), // Limit to 50 levels
				"a": formatOrderbookLevels(orderBook.Asks, 50), // Limit to 50 levels
			}, nil
		}
	}

	return nil, fmt.Errorf("order book not found for ticker: %s", ticker)
}

func (api *PublicApi) GetPositionsByMarketId(marketId uint, ctx context.Context) ([]*ctypes.PositionSummary, error) {
	positionMapKey := xredis.GetPositionsMapKey()
	val, err := api.redisClient.Get(ctx, positionMapKey).Result()
	if err != nil {
		xlog.Errorf("Failed to get positions map from Redis: %v", err)
		return nil, err
	}

	var positionMap map[string]*ctypes.PositionSummary
	if err := json.Unmarshal([]byte(val), &positionMap); err != nil {
		xlog.Errorf("Failed to unmarshal positions map from Redis: %v", err)
		return nil, err
	}

	var openPositions []*ctypes.PositionSummary
	for _, pos := range positionMap {
		if pos == nil {
			continue
		}
		if pos.MarketID == marketId && pos.TotalAmount != nil && pos.TotalAmount.Cmp(big.NewInt(0)) != 0 {
			openPositions = append(openPositions, pos)
		}
	}
	return openPositions, nil
}

func formatOrderbookLevels(levels []xclient.OrderbookLevel, maxDepth int) [][]string {
	// Limit the depth to the specified maxDepth
	depth := len(levels)
	if depth > maxDepth {
		depth = maxDepth
	}

	formatted := make([][]string, depth)
	for i := 0; i < depth; i++ {
		// Parse Price and Quantity from string to float64
		price, err1 := strconv.ParseFloat(levels[i].Price, 64)
		quantity, err2 := strconv.ParseFloat(levels[i].Quantity, 64)

		// Handle potential parsing errors gracefully
		if err1 != nil || err2 != nil {
			formatted[i] = []string{"0.000", "0.000"} // Default fallback values
			continue
		}

		// Format to 3 decimal places
		formatted[i] = []string{
			fmt.Sprintf("%.3f", price),
			fmt.Sprintf("%.3f", quantity),
		}
	}

	return formatted
}

func reverseProductIDSymbolMap(original map[uint32]string) map[string]uint32 {
	if original == nil {
		return make(map[string]uint32) // Return an empty map to avoid nil map usage
	}

	reversed := make(map[string]uint32, len(original))
	for id, symbol := range original {
		if symbol == "" {
			continue
		}
		reversed[strings.ToLower(symbol)] = id // Convert symbol to lowercase before storing
	}
	return reversed
}

// FetchUserDashboardData retrieves paginated user dashboard data
func (api *PublicApi) FetchUserDashboardData(page, limit int) (*PaginatedUserDashboardResponse, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20 // Default limit
	}

	offset := (page - 1) * limit

	// Fetch data from database
	results, totalCount, err := (&db.FillDB{}).GetUserDashboardData(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user dashboard data: %w", err)
	}

	// Convert results to UserDashboardData structs using the helper function
	var dashboardData []UserDashboardData
	for _, result := range results {
		data := convertToUserDashboardData(result)
		dashboardData = append(dashboardData, data)
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	return &PaginatedUserDashboardResponse{
		Data:       dashboardData,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// FetchUserDashboardByAddress retrieves dashboard data for a specific user address
func (api *PublicApi) FetchUserDashboardByAddress(userAddress string) (*UserDashboardData, error) {
	// Validate user address format
	if len(userAddress) < 10 || !strings.HasPrefix(userAddress, "0x") {
		return nil, fmt.Errorf("invalid user address format")
	}

	// Fetch data from database
	result, err := (&db.FillDB{}).GetUserDashboardByAddress(userAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user dashboard data: %w", err)
	}

	data := convertToUserDashboardData(result)

	return &data, nil
}

func convertToUserDashboardData(result map[string]interface{}) UserDashboardData {
	data := UserDashboardData{}

	if userAddress, exists := cutils.GetStringValue(result, "user_address"); exists && len(userAddress) > 10 {
		data.WalletAddressShortened = userAddress[:6] + "..." + userAddress[len(userAddress)-4:]
		data.WalletAddressFull = userAddress
	}

	if productID := cutils.GetInt64Value(result, "favorite_product_id"); productID > 0 {
		if symbol, exists := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(productID)]; exists {
			data.FavoritePair = symbol + "-USD"
		} else {
			data.FavoritePair = fmt.Sprintf("Product-%d", productID)
		}
	}

	if lastTrade := cutils.GetTimeValue(result, "last_trade"); !lastTrade.IsZero() {
		data.LastTrade = lastTrade.Format("2006-01-02 15:04:05")
	}

	data.TotalNetPnLUSD = cutils.GetFloat64Value(result, "total_net_pnl_usd")
	data.WinRate = cutils.GetFloat64Value(result, "win_rate")
	data.TotalTrades = cutils.GetInt64Value(result, "total_trades")
	data.VolumeUSD = cutils.GetFloat64Value(result, "total_volume_usd")
	data.AvgTradeSize = cutils.GetFloat64Value(result, "avg_trade_size")
	data.ActiveDays = cutils.GetInt64Value(result, "active_days")
	data.AccountAgeDays = cutils.GetFloat64Value(result, "account_age_days")

	return data
}

func (api *PublicApi) FetchUserDashBoardDetailsBySubAccountId(subaccountId string, page, limit int) (*UserDashboardSubAccountIdData, error) {
	if len(subaccountId) < 10 {
		return nil, fmt.Errorf("invalid user address format")
	}

	// Convert subaccount ID to hex format once at the beginning
	hexSubaccountId, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		xlog.Errorf("Failed to convert string subaccount ID to hex: %v, subaccountId: %s", err, subaccountId)
		return nil, fmt.Errorf("failed to convert string subaccount ID to hex format: %w", err)
	}

	ctx := context.Background()
	fundingFee, err := fetchFundingFee(hexSubaccountId)
	if err != nil {
		xlog.Errorf("Failed to fetch funding fee for subaccount %s: %v", hexSubaccountId, err)
		return nil, err
	}

	dateOfJoining, err := fetchDateOfJoining(subaccountId)
	if err != nil {
		xlog.Errorf("Failed to fetch date of joining for subaccount %s: %v", subaccountId, err)
		return nil, err
	}

	depositHistory, err := fetchDepositHistory(hexSubaccountId)
	if err != nil {
		xlog.Errorf("Failed to fetch deposit history for subaccount %s: %v", hexSubaccountId, err)
		return nil, err
	}

	pnlGraph, err := fetchDailyPnlGraph(hexSubaccountId)
	if err != nil {
		xlog.Errorf("Failed to fetch PnL graph for subaccount %s: %v", hexSubaccountId, err)
		return nil, err
	}

	unrealizedPnlx36, unrealizedfundingx36, err := fetchUnrealizedPnl(subaccountId)
	if err != nil {
		xlog.Errorf("Failed to fetch unrealized PnL or unrealized Funding Fee for subaccount %s: %v", subaccountId, err)
		return nil, err
	}

	openPositions, err := fetchOpenPositions(subaccountId, api.redisClient, ctx)
	if err != nil {
		xlog.Errorf("Failed to fetch open positions for subaccount %s: %v", subaccountId, err)
		return nil, err
	}

	tradeHistory, err := fetchTradeHistory(subaccountId, page, limit)
	if err != nil {
		xlog.Errorf("Failed to fetch trade history for subaccount %s: %v", subaccountId, err)
		return nil, err
	}

	// Fetch spot balances
	spotBalance, freeBalance, usedBalance, err := xclient.GlobalBalanceClient.GetSpotBalance(hexSubaccountId)
	if err != nil {
		xlog.Errorf("Failed to fetch spot balance for subaccount %s: %v", hexSubaccountId, err)
		spotBalance = []ctypes.SpotBalance{}
		freeBalance = "0"
		usedBalance = "0"
	}

	result := &UserDashboardSubAccountIdData{
		TotalNetUnrealisedPnLUSD:        unrealizedPnlx36,
		DateOfJoining:                   dateOfJoining,
		FundingFeeUSD:                   fundingFee,
		DepositHistory:                  depositHistory,
		DailyPnlGraph:                   pnlGraph,
		OpenPositions:                   openPositions,
		TradeHistory:                    tradeHistory,
		TotalNetUnrealisedFundingFeeUSD: unrealizedfundingx36,
		SpotBalance:                     spotBalance,
		FreeBalance:                     freeBalance,
		UsedBalance:                     usedBalance,
	}

	return result, nil
}

// FetchUnrealizedPnl returns a map[marketID]openPositionAmount for the given subaccountId
func fetchUnrealizedPnl(subaccountId string) (*big.Int, *big.Int, error) {
	// Use subaccountId directly (provided by frontend)
	subaccountImpl := subaccount.NewSubaccountBalanceImpl()
	subAccountIds := []string{subaccountId}
	subaccountBalance := subaccountImpl.MustGetSubaccountBalancesFromIds(subAccountIds)[0]
	appState := appstate.NewAppState()
	oraclePrices, err := appState.GetAllOraclePrices()
	if err != nil {
		xlog.Errorf("Error fetching oracle prices: %v", err)
		return nil, nil, err
	}
	cumulativeFundingRateMap := appState.GetAllFundingRates()
	upnlx36, ufundingx36 := subaccount.GetUnrealisedSubaccountValue(subaccountBalance, oraclePrices, cumulativeFundingRateMap)
	// Collect open position amount per market for this subaccount
	return upnlx36, ufundingx36, nil
}

// FetchFundingFee returns the total funding fee for the given subaccountId
func fetchFundingFee(subaccountId string) (string, error) {

	// Extract broker ID from the subaccount ID
	brokerId := cutils.ExtractBrokerIdFromSubaccountHex(subaccountId)
	if brokerId == 0 {
		xlog.Errorf("Failed to extract broker ID from subaccount ID: %s", subaccountId)
		return "", fmt.Errorf("failed to extract broker ID from subaccount ID")
	}

	// Get funding fee for the subaccount
	fundingFee, err := (&db.FillDB{}).GetFundingFeeBySubaccountId(subaccountId, brokerId)
	if err != nil {
		xlog.Errorf("Failed to get funding fee for subaccount: %v, subaccountId: %s, brokerId: %d", err, subaccountId, brokerId)
		return "", err
	}

	return fundingFee, nil
}

// FetchDateOfJoining returns the date of joining (created_at) for the given subaccountId
func fetchDateOfJoining(subaccountId string) (string, error) {
	subaccount := (&db.SubaccountDB{}).GetById(subaccountId)
	if subaccount == nil {
		xlog.Errorf("Subaccount not found: %s", subaccountId)
		return "", fmt.Errorf("subaccount not found")
	}
	return subaccount.CreatedAt.Format("2006-01-02"), nil
}

// FetchDepositHistory returns all deposit transactions for the given subaccountId
func fetchDepositHistory(subaccountId string) ([]DepositHistoryResponse, error) {
	deposits := (&db.DepositWithdrawDB{}).GetDepositsBySubaccountId(subaccountId)
	if deposits == nil || len(*deposits) == 0 {
		xlog.Errorf("No deposits found for subaccount: %s", subaccountId)
		return nil, fmt.Errorf("no deposits found")
	}

	var response []DepositHistoryResponse
	for _, deposit := range *deposits {
		resp := DepositHistoryResponse{
			TxnHash:            deposit.TxnHash,
			Amount:             deposit.Amount,
			SourceChainID:      deposit.SourceChainID,
			DestinationChainID: deposit.DestinationChainID,
			CreatedAt:          deposit.CreatedAt.Format("02-01-2006 15:04:05"),
			IsDeposit:          deposit.IsDeposit, // Use the actual value from database

		}
		response = append(response, resp)
	}

	return response, nil
}

// FetchDailyPnlGraph returns daily realized PnL for the given subaccountId
func fetchDailyPnlGraph(subaccountId string) ([]map[string]interface{}, error) {

	// Extract broker ID from the subaccount ID
	brokerId := cutils.ExtractBrokerIdFromSubaccountHex(subaccountId)
	if brokerId == 0 {
		xlog.Errorf("Failed to extract broker ID from subaccount ID: %s", subaccountId)
		return nil, fmt.Errorf("failed to extract broker ID from subaccount ID")
	}

	// Get daily PnL graph data
	dailyPnlData, err := (&db.FillDB{}).GetDailyPnlGraph(subaccountId, brokerId)
	if err != nil {
		xlog.Errorf("Failed to get daily PnL graph for subaccount: %v, subaccountId: %s, brokerId: %d", err, subaccountId, brokerId)
		return nil, err
	}

	return dailyPnlData, nil
}

// FetchOpenPositions returns all open positions for the given subaccountId
func fetchOpenPositions(subaccountId string, redisClient *redis.Client, ctx context.Context) ([]*ctypes.PositionSummary, error) {
	positionMapKey := xredis.GetPositionsMapKey()
	val, err := redisClient.Get(ctx, positionMapKey).Result()
	if err != nil {
		xlog.Errorf("Failed to get positions map from Redis: %v", err)
		return nil, err
	}

	var positionMap map[string]*ctypes.PositionSummary
	if err := json.Unmarshal([]byte(val), &positionMap); err != nil {
		xlog.Errorf("Failed to unmarshal positions map from Redis: %v", err)
		return nil, err
	}

	var openPositions []*ctypes.PositionSummary
	for _, pos := range positionMap {
		if pos == nil {
			continue
		}
		if pos.SubaccountID == subaccountId && pos.TotalAmount != nil && pos.TotalAmount.Cmp(big.NewInt(0)) != 0 {
			openPositions = append(openPositions, pos)
		}
	}
	return openPositions, nil
}

// FetchTradeHistory returns trade history for the given subaccountId with pagination
func fetchTradeHistory(subaccountId string, page, limit int) (*PaginatedTradeHistoryResponse, error) {

	// Get total count first
	totalCount, err := (&db.FillDB{}).TradeCount(subaccountId)
	if err != nil {
		xlog.Errorf("Failed to get total count for subaccount: %v, subaccountId: %s", err, subaccountId)
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	if totalCount == 0 {
		xlog.Errorf("No trade history found for subaccount: %s", subaccountId)
		return &PaginatedTradeHistoryResponse{
			Data:       []TradeHistoryResponse{},
			TotalCount: 0,
			Page:       1,
			Limit:      totalCount,
			TotalPages: 0,
		}, nil
	}

	// Get all trade history data from database using FillDB (fill_tables)
	allTradeHistory := (&db.FillDB{}).GetAllBySubaccountId(subaccountId)
	if allTradeHistory == nil {
		allTradeHistory = &[]db.FillTable{}
	}

	// Convert the FillTable data to TradeHistoryResponse structs
	var response []TradeHistoryResponse
	for _, trade := range *allTradeHistory {
		resp := TradeHistoryResponse{
			Amount:        trade.Amountx18.Val.String(),
			ProductID:     uint64(trade.MarketId),
			RealisedPnl:   trade.RealizedPnlx18.Val.String(),
			CreatedAt:     trade.CreatedAt.Format("02-01-2006"),
			Type:          string(trade.Side),
			Price:         trade.Pricex18.Val.String(),
			Fee:           trade.Feex18.Val.String(),
			IsLiquidation: trade.Type == ctypes.FILL_LIQUIDATION,
		}
		response = append(response, resp)
	}

	return &PaginatedTradeHistoryResponse{
		Data:       response,
		TotalCount: int64(totalCount),
		Page:       1,
		Limit:      totalCount,
		TotalPages: 1,
	}, nil
}

// FetchLiquidationTrades returns the latest liquidation trades for broker ID 2
func (api *PublicApi) FetchLiquidationTrades() (*[]db.FillTable, error) {
	liquidationTrades, err := (&db.FillDB{}).GetLatestLiquidationTrades()
	if err != nil {
		xlog.Errorf("Failed to fetch liquidation trades: %v", err)
		return nil, fmt.Errorf("failed to fetch liquidation trades: %w", err)
	}
	return liquidationTrades, nil
}
