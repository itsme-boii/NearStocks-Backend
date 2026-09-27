package db

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"os"
	"time"
)

type OptionsDB struct{}

// A function a add a new option entry
func (*OptionsDB) InsertOptionBet(optionData OptionsTable) (*OptionsTable, error) {
	// Create the new entry
	if err := db.Create(&optionData).Error; err != nil {
		return nil, err
	}

	return &optionData, nil
}

func (*OptionsDB) CloseOptionBetById(id uint, exitPrice, payoutAmount, feesAmount, userPnl *ctypes.BigInt) (*OptionsTable, error) {
	var option OptionsTable

	// Find the existing entry by ID
	if err := db.First(&option, id).Error; err != nil {
		return nil, err // Return error if the entry is not found
	}

	// Update the fields
	option.ExitPrice = exitPrice
	option.PayoutAmount = payoutAmount
	option.FeesAmount = feesAmount
	option.UserPnl = userPnl

	// Save the updated entry
	if err := db.Save(&option).Error; err != nil {
		return nil, err // Return error if the update fails
	}

	return &option, nil
}

func (*OptionsDB) GetPositionsAndHistoryBySubaccount(subaccountID string) ([]OptionsTable, error) {
	var options []OptionsTable

	// Find all entries with the given subaccount ID
	if err := db.Where("subaccount_id = ?", subaccountID).Find(&options).Error; err != nil {
		return nil, err // Return error if the entries are not found
	}

	return options, nil
}

func (*OptionsDB) GetCurrentOI() ([]OptionsTable, error) {
	var options []OptionsTable

	// Collect exclusion subaccount IDs from environment variables
	optionsSubaccountId := os.Getenv("OPTIONS_SUBACCOUNT_ID")

	// Fetch entries where ExitPrice is NULL and the subaccount ID does not match the exclusion ID
	err := db.Where("exit_price IS NULL AND subaccount_id != ?", optionsSubaccountId).Find(&options).Error
	if err != nil {
		return nil, err
	}

	return options, nil
}

func (*OptionsDB) GetHistoricalData() (map[uint32]map[uint32]map[string]string, error) {
	// Define a struct to fetch only the required fields
	type MinimalOptionsTable struct {
		ProductId  uint32         `json:"product_id"`
		Interval   uint32         `json:"interval"`
		UserPnl    *ctypes.BigInt `json:"user_pnl"`
		QuoteDelta *ctypes.BigInt `json:"quote_delta"`
	}

	var options []MinimalOptionsTable

	// Collect exclusion subaccount IDs from environment variables
	optionsSubaccountId := os.Getenv("OPTIONS_SUBACCOUNT_ID")

	// Use raw SQL to fetch the required data, excluding the specified subaccount ID
	query := `
        SELECT 
            product_id, 
            interval, 
            user_pnl, 
            quote_delta
        FROM options_tables
        WHERE exit_price IS NOT NULL AND subaccount_id != ?
    `
	err := db.Raw(query, optionsSubaccountId).Scan(&options).Error
	if err != nil {
		return nil, err
	}

	// Define the grouped structure
	groupedData := make(map[uint32]map[uint32]map[string]string)

	for _, option := range options {

		// Ensure UserPnl is not nil
		if option.UserPnl == nil || option.UserPnl.Val == nil {
			xlog.Warnf("UserPnl is nil for ProductId: %d, Interval: %d", option.ProductId, option.Interval)
			continue
		}

		// Ensure QuoteDelta is not nil
		if option.QuoteDelta == nil || option.QuoteDelta.Val == nil {
			xlog.Warnf("QuoteDelta is nil for ProductId: %d, Interval: %d", option.ProductId, option.Interval)
			continue
		}

		// Ensure the group for ProductId exists
		if _, productExists := groupedData[option.ProductId]; !productExists {
			groupedData[option.ProductId] = make(map[uint32]map[string]string)
		}

		// Ensure the group for Interval exists
		if _, intervalExists := groupedData[option.ProductId][option.Interval]; !intervalExists {
			groupedData[option.ProductId][option.Interval] = map[string]string{
				"up_pnl":     "0",
				"down_pnl":   "0",
				"up_quote":   "0",
				"down_quote": "0",
			}
		}

		// Calculate the total PnL and QuoteDelta for positive and negative amounts
		if option.QuoteDelta.Val.Sign() > 0 {
			currentUpPnl, _ := new(big.Int).SetString(groupedData[option.ProductId][option.Interval]["up_pnl"], 10)
			groupedData[option.ProductId][option.Interval]["up_pnl"] = new(big.Int).Add(currentUpPnl, option.UserPnl.Val).String()
			currentUpQuote, _ := new(big.Int).SetString(groupedData[option.ProductId][option.Interval]["up_quote"], 10)
			groupedData[option.ProductId][option.Interval]["up_quote"] = new(big.Int).Add(currentUpQuote, option.QuoteDelta.Val).String()
		} else {
			currentDownPnl, _ := new(big.Int).SetString(groupedData[option.ProductId][option.Interval]["down_pnl"], 10)
			groupedData[option.ProductId][option.Interval]["down_pnl"] = new(big.Int).Add(currentDownPnl, option.UserPnl.Val).String()
			currentDownQuote, _ := new(big.Int).SetString(groupedData[option.ProductId][option.Interval]["down_quote"], 10)
			groupedData[option.ProductId][option.Interval]["down_quote"] = new(big.Int).Add(currentDownQuote, option.QuoteDelta.Val).String()
		}
	}

	return groupedData, nil
}

func (*OptionsDB) GetTradesBreakdown() (map[uint32]map[uint32]map[string]int, error) {
	var results []struct {
		ProductId uint32         `json:"product_id"`
		Interval  uint32         `json:"interval"`
		UserPnl   *ctypes.BigInt `json:"user_pnl"`
	}

	// Collect exclusion subaccount IDs from environment variables
	optionsSubaccountId := os.Getenv("OPTIONS_SUBACCOUNT_ID")

	// Query to fetch only the necessary data, excluding the specified subaccount ID
	err := db.Raw(`
        SELECT 
            product_id AS product_id, 
            interval AS interval,
            user_pnl AS user_pnl
        FROM options_tables
        WHERE exit_price IS NOT NULL AND subaccount_id != ?
    `, optionsSubaccountId).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Group the results in Go
	groupedData := make(map[uint32]map[uint32]map[string]int)

	for _, entry := range results {
		// If userPnl is nil, skip the entry
		if entry.UserPnl == nil || entry.UserPnl.Val == nil {
			xlog.Warnf("UserPnl is nil for ProductId: %d, Interval: %d", entry.ProductId, entry.Interval)
			continue
		}

		// Ensure productId exists in groupedData
		if _, productExists := groupedData[entry.ProductId]; !productExists {
			groupedData[entry.ProductId] = make(map[uint32]map[string]int)
		}

		// Ensure interval exists for the given productId
		if _, intervalExists := groupedData[entry.ProductId][entry.Interval]; !intervalExists {
			groupedData[entry.ProductId][entry.Interval] = map[string]int{
				"wins":   0,
				"losses": 0,
				"total":  0,
			}
		}

		breakdown := groupedData[entry.ProductId][entry.Interval]

		// Increment the total
		breakdown["total"]++

		// Check if it's a win or a loss
		if entry.UserPnl.Val.Sign() <= 0 {
			breakdown["wins"]++
		} else {
			breakdown["losses"]++
		}
	}

	return groupedData, nil
}

func (*OptionsDB) GetLast24HourStats() (numTrades int, numUsers int, totalQuoteDelta ctypes.BigInt, totalUniqueSubaccounts int, err error) {
	// Define a struct to hold the query results
	type Result struct {
		NumTrades           int
		NumUsers            int
		TotalQuoteDelta     ctypes.BigInt
		TotalUniqueAccounts int
	}

	var result Result

	// Single query to fetch all required stats
	// Collect exclusion subaccount IDs from environment variables
	optionsSubaccountId := os.Getenv("OPTIONS_SUBACCOUNT_ID")

	query := `
			SELECT
				COUNT(*) AS num_trades,
				COUNT(DISTINCT subaccount_id) AS num_users,
				COALESCE(SUM(ABS(CAST(quote_delta AS numeric))), 0) AS total_quote_delta,
				(SELECT COUNT(DISTINCT subaccount_id) FROM options_tables WHERE subaccount_id != ?) AS total_unique_accounts
			FROM options_tables
			WHERE created_at >= NOW() - INTERVAL '24 HOURS' AND subaccount_id != ?
		`

	// Execute the query
	err = db.Raw(query, optionsSubaccountId, optionsSubaccountId).Scan(&result).Error
	if err != nil {
		return 0, 0, ctypes.BigInt{}, 0, err
	}

	// Return the results
	return result.NumTrades, result.NumUsers, result.TotalQuoteDelta, result.TotalUniqueAccounts, nil
}

func (*OptionsDB) GetOptions24HData() (map[uint32]ctypes.BigInt, error) {
	var results []struct {
		ProductID     uint32        `gorm:"column:product_id"`
		Last24HourVol ctypes.BigInt `gorm:"column:last_24_hour_volume"`
	}

	err := db.Raw(`
        SELECT
            product_id,
            COALESCE(SUM(ABS(CAST(quote_delta AS numeric))), 0) AS last_24_hour_volume
        FROM options_tables
        WHERE created_at AT TIME ZONE 'UTC' >= NOW() - INTERVAL '24 hours'
        GROUP BY product_id
    `).Scan(&results).Error

	if err != nil {
		xlog.Errorf("Error querying last 24-hour volume for options: %v", err)
		return nil, err
	}

	productVolumes := make(map[uint32]ctypes.BigInt)
	for _, result := range results {
		productVolumes[result.ProductID] = result.Last24HourVol
	}
	return productVolumes, nil
}

func (*OptionsDB) GetLast30DaysStats() (map[string]interface{}, error) {
	// Define a struct to hold the daily query results
	type DailyStats struct {
		Date       time.Time     `json:"date"`
		NumTrades  int           `json:"num_trades"`
		NumUsers   int           `json:"num_users"`
		QuoteDelta ctypes.BigInt `json:"quote_delta"`
	}

	var dailyStats []DailyStats

	// SQL query to fetch daily statistics for the last 30 days
	optionsSubaccountId := os.Getenv("OPTIONS_SUBACCOUNT_ID")

	query := `
		SELECT
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') as date,
			COUNT(*) AS num_trades,
			COUNT(DISTINCT subaccount_id) AS num_users,
			COALESCE(SUM(ABS(CAST(quote_delta AS numeric))), 0) AS quote_delta
		FROM options_tables
		WHERE created_at >= NOW() - INTERVAL '30 DAYS' AND subaccount_id != ?
		GROUP BY date
		ORDER BY date DESC
	`

	// Execute the query
	err := db.Raw(query, optionsSubaccountId).Scan(&dailyStats).Error
	if err != nil {
		return nil, err
	}

	// Prepare the response
	response := map[string]interface{}{
		"dailyStats": dailyStats,
	}

	return response, nil
}

func (*OptionsDB) DefillamaStats(endTime time.Time) (ctypes.BigInt, ctypes.BigInt, error) {
	// Set the fixed start time (startOfPeriod) and calculate last 24 hours from the provided endTime
	startOfPeriod := time.Date(2024, 12, 04, 06, 40, 25, 0, time.UTC)
	last24Hours := endTime.Add(-24 * time.Hour) // Last 24 hours from endTime

	// Variables to store the total volume and last 24-hour volume
	var totalVolumeStr, last24HourVolumeStr string

	// Query to get both total volume since the specified start time (startOfPeriod) and the volume in the last 24 hours
	err := db.Raw(`
		SELECT 
			COALESCE(SUM(ABS(CAST(quote_delta AS numeric))) FILTER (WHERE created_at >= ? AND created_at <= ?), 0) AS last_24_hour_volume,
			COALESCE(SUM(ABS(CAST(quote_delta AS numeric))) FILTER (WHERE created_at >= ? AND created_at <= ?), 0) AS total_volume
		FROM 
			options_tables
	`, last24Hours, endTime, startOfPeriod, endTime).Row().Scan(&last24HourVolumeStr, &totalVolumeStr)

	if err != nil {
		xlog.Errorf("Error querying total and last 24-hour volume: %v", err)
		return ctypes.NewBigInt(big.NewInt(0)), ctypes.NewBigInt(big.NewInt(0)), err
	}

	// Convert the string results to *big.Int
	totalVolume := new(big.Int)
	totalVolume.SetString(totalVolumeStr, 10)

	last24HourVolume := new(big.Int)
	last24HourVolume.SetString(last24HourVolumeStr, 10)

	divisor := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)

	// Divide both totalVolume and last24HourVolume by 10^18
	totalVolume.Div(totalVolume, divisor)
	last24HourVolume.Div(last24HourVolume, divisor)

	return ctypes.NewBigInt(totalVolume), ctypes.NewBigInt(last24HourVolume), nil
}

func (*OptionsDB) GetPublicCharts() (map[string]interface{}, error) {
	// Define a struct to hold the daily query results
	type DailyStats struct {
		Date       time.Time     `json:"date"`
		NumTrades  int           `json:"num_trades"`
		QuoteDelta ctypes.BigInt `json:"quote_delta"`
	}

	var dailyStats []DailyStats

	query := `
		SELECT
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') as date,
			COUNT(*) AS num_trades,
			COALESCE(SUM(ABS(CAST(quote_delta AS numeric))), 0) AS quote_delta
		FROM options_tables
		WHERE created_at >= NOW() - INTERVAL '30 DAYS'
		GROUP BY date
		ORDER BY date DESC
	`

	// Execute the query
	err := db.Raw(query).Scan(&dailyStats).Error
	if err != nil {
		return nil, err
	}

	// Prepare the response
	response := map[string]interface{}{
		"dailyStats": dailyStats,
	}

	return response, nil
}

func (*OptionsDB) GetOptionsPublicData() (ctypes.BigInt, int, ctypes.BigInt, int, error) {
	// Get total volume and total trades and 24 hours data
	var totalVolumeStr string
	var totalTrades int
	var last24HourVolumeStr string
	var last24HourTrades int

	// Query to get total volume, total trades, 24-hour volume, and 24-hour trades
	err := db.Raw(`
		SELECT
			COALESCE(SUM(ABS(CAST(quote_delta AS numeric))), 0) AS total_volume,
			COUNT(*) AS total_trades,
			COALESCE(SUM(ABS(CAST(quote_delta AS numeric))) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours'), 0) AS last_24_hour_volume,
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours') AS last_24_hour_trades
		FROM options_tables
	`).Row().Scan(&totalVolumeStr, &totalTrades, &last24HourVolumeStr, &last24HourTrades)

	if err != nil {
		xlog.Errorf("Error querying data for public stats for options: %v", err)
		return ctypes.NewBigInt(big.NewInt(0)), 0, ctypes.NewBigInt(big.NewInt(0)), 0, err
	}

	// Convert the string results to *big.Int
	totalVolume := new(big.Int)
	totalVolume.SetString(totalVolumeStr, 10)

	last24HourVolume := new(big.Int)
	last24HourVolume.SetString(last24HourVolumeStr, 10)

	return ctypes.NewBigInt(totalVolume), totalTrades, ctypes.NewBigInt(last24HourVolume), last24HourTrades, nil
}

// GetOptionBetById retrieves an option bet by its ID without modifying it
func (*OptionsDB) GetOptionBetById(id uint) (*OptionsTable, error) {
	var option OptionsTable

	// Find the existing entry by ID
	if err := db.First(&option, id).Error; err != nil {
		return nil, err // Return error if the entry is not found
	}

	return &option, nil
}

func (*OptionsDB) HasTradedToday(subaccountId string) (int, error) {
	var count int

	err := db.Raw(`
		SELECT
			COUNT(*)
		FROM options_tables
		WHERE subaccount_id = ? AND DATE(created_at) = CURRENT_DATE`, subaccountId).Scan(&count).Error

	if err != nil {
		xlog.Errorf("error querying trade entries for today: %w", err)
		return 0, err
	}

	return count, nil
}

func (*OptionsDB) GetOptionsVolumeAfterCertainDate(subaccountId string, startTime time.Time) (*big.Float, error) {
	var result struct {
		TotalVolume string
	}

	err := db.Raw(`
		SELECT
			COALESCE(ROUND(SUM(ABS(CAST(quote_delta AS NUMERIC))) / 1e18, 4), 0) AS total_volume
		FROM options_tables
		WHERE subaccount_id = ? AND created_at >= ?`, subaccountId, startTime).Scan(&result).Error

	if err != nil {
		xlog.Errorf("error querying options volume: %w", err)
		return new(big.Float).SetFloat64(0), err
	}

	totalVolume := new(big.Float)
	totalVolume.SetString(result.TotalVolume)

	return totalVolume, nil
}

// GetHistoryBySubaccount retrieves only completed trades (with ExitPrice) for a given subaccount
func (*OptionsDB) GetHistoryBySubaccount(subaccountID string) ([]OptionsTable, error) {
	var options []OptionsTable

	// Find all completed trades (where ExitPrice is not NULL) for the given subaccount ID
	if err := db.Where("subaccount_id = ? AND exit_price IS NOT NULL", subaccountID).
		Order("entry_time DESC").
		Find(&options).Error; err != nil {
		return nil, err // Return error if the entries are not found
	}

	return options, nil
}
