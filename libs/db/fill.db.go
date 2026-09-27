package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xcache"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"math/big"
	"os"
	"services/external-campaigns/dtos"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
)

type FillDB struct{}

// converts hex subaccount ID to string format
func convertHexSubaccountID(hexID string) string {
	if hexID == "" {
		return ""
	}
	converted, err := cutils.HackySubaccountHexToId(hexID)
	if err != nil {
		xlog.Errorf("Error converting hex subaccount ID %s: %v", hexID, err)
		return hexID // Return original on error
	}
	return converted
}

// converts hex subaccount IDs to string format
func convertHexSubaccountIDs(hexIDs string) []string {
	if hexIDs == "" {
		return []string{}
	}
	ids := strings.Split(hexIDs, ",")
	converted := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			converted = append(converted, convertHexSubaccountID(trimmed))
		}
	}
	return converted
}

// Cache for leaderboard data with broker ID, start date, end date, and limit as key
var (
	leaderboardCache    = make(map[string]xcache.Cache[[]map[string]interface{}])
	totalVolumeCache    = make(map[string]xcache.Cache[map[string]interface{}])
	cacheMutex          = sync.RWMutex{}
	cacheExpiryDuration = 5 * time.Minute
)

// Cache constants for user dashboard data
const (
	USER_DASHBOARD_DATA_KEY       = "user_dashboard:data:"
	USER_DASHBOARD_BY_ADDRESS_KEY = "user_dashboard:address:"
	CACHE_EXPIRY_DURATION         = 15 * time.Minute
)

var FundingFeesMigrationCutoffDate = time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC)

type DailyData struct {
	Date  time.Time `json:"date"`
	Count int64     `json:"count"`
}

type WeeklyData struct {
	StartDate time.Time `json:"startDate"`
	Count     int64     `json:"count"`
}

type PaginatedResponse struct {
	Data       []FillTable `json:"data"`
	TotalRows  int64       `json:"total_rows"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

func (*FillDB) GetAllByOrderId(orderId string) *[]FillTable {
	fills := []FillTable{}
	return GetDBObjOrEmptyList(db.Where("order_id = ?", orderId).Find(&fills), &fills)
}

func (*FillDB) Create(fill *FillTable) *FillTable {
	if tx := db.Create(fill); tx.Error != nil {
		fmt.Println("Error creating market: ", tx.Error)
		return nil
	}
	return fill
}

// NOTE: Returns first 200 entries only
func (*FillDB) GetAllBySubaccountId(subaccountId string) *[]FillTable {
	fills := []FillTable{}
	return GetDBObjOrEmptyList(db.Where("subaccount_id = ?", subaccountId).Order("id DESC").Limit(200).Find(&fills), &fills)
}

func (*FillDB) EntryExistBySubaccountId(subaccountId string) bool {
	var count int64
	if err := db.Model(&FillTable{}).
		Where("subaccount_id = ?", subaccountId).
		Count(&count).Error; err != nil {
		return false
	}

	// Return true if any entries are found, otherwise return false
	return count > 0
}

// DEX2: Not including broker id for now because this is only used to show trade history.
func (*FillDB) GetLatestTradeHistory() (*[]FillTable, error) {
	var tradeHistory []FillTable

	// query to get latest trade for each productID
	err := db.Raw(`
		SELECT DISTINCT ON (market_id) *
		FROM fill_tables
		ORDER BY market_id, created_at DESC;
	`).Scan(&tradeHistory).Error

	if err != nil {
		return nil, err
	}

	return &tradeHistory, nil
}

func (*FillDB) TradeCount(subaccountId string) (int, error) {
	var count int
	query := `
        SELECT COUNT(*)
        FROM fill_tables
        WHERE subaccount_id = ?
    `
	row := db.Raw(query, subaccountId).Row()
	if err := row.Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No matching user address found
		}
		return 0, err
	}
	return count, nil
}

func (*FillDB) TradedToday(subaccountId string) (bool, error) {
	var count int
	// Get the start of today
	todayStart := time.Now().Truncate(24 * time.Hour)

	query := `
        SELECT COUNT(*)
        FROM fill_tables
        WHERE subaccount_id = ?
          AND updated_at >= ?
    `
	row := db.Raw(query, subaccountId, todayStart).Row()
	if err := row.Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return false, nil // No matching user address found
		}
		return false, err
	}

	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (*FillDB) GetTopUsersByPnlInRange(brokerId uint, startDateStr, endDateStr string) ([]dtos.LeaderboardAllResult, error) {
	var results []dtos.LeaderboardAllResult
	internalSubAccountIdList := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")
	ammSubAccountIdList := []string{contractUtils.AMM_SUBACCOUNT_ID_1}
	mergedSubAccountIdList := append(internalSubAccountIdList, ammSubAccountIdList...)

	err := db.Raw(`
	WITH casted_pnl AS (
		SELECT 
			subaccount_id,
			CAST(realized_pnlx18 AS NUMERIC) AS realized_pnl,
			is_reduce
		FROM fill_tables
		WHERE broker_id = ?
		AND created_at BETWEEN ? AND ?
		AND subaccount_id NOT IN (?)
	),
	top_users AS (
		SELECT 
			subaccount_id AS user_address, 
			COUNT(*) AS total_trades, 
			COALESCE(SUM(realized_pnl), 0) AS total_realized_pnl,  
			SUM(CASE WHEN realized_pnl > 0 THEN 1 ELSE 0 END) AS winning_trades,
			SUM(CASE WHEN realized_pnl < 0 THEN 1 ELSE 0 END) AS losing_trades,
			SUM(CASE WHEN is_reduce = TRUE AND realized_pnl = 0 THEN 1 ELSE 0 END) AS reduce_zero_pnl_trades,
			ABS(COALESCE(SUM(realized_pnl), 0)) AS abs_total_realized_pnl
		FROM casted_pnl
		GROUP BY subaccount_id 
		ORDER BY abs_total_realized_pnl DESC 
		LIMIT 10
	)
	SELECT 
		subaccount_tables.eth_address, 
		top_users.total_trades, 
		top_users.total_realized_pnl, 
		top_users.winning_trades, 
		top_users.losing_trades,
		top_users.reduce_zero_pnl_trades,
		subaccount_tables.user_name 
	FROM top_users 
	INNER JOIN subaccount_tables ON top_users.user_address = subaccount_tables.id
	ORDER BY abs_total_realized_pnl DESC;
	`, brokerId, startDateStr, endDateStr, mergedSubAccountIdList).Scan(&results).Error

	if err != nil {
		return nil, err
	}
	return results, nil
}

func (*FillDB) GetAllUsersByPnlInRange(brokerId uint, startDate, endDate string) ([]dtos.LeaderboardResult, error) {
	var results []dtos.LeaderboardResult

	err := db.Model(&FillTable{}).
		Select(`
			subaccount_id AS user_address, 
			COUNT(*) as total_trades, 
			COALESCE(SUM(CAST(realized_pnlx18 AS NUMERIC)), 0) as total_realized_pnl,
			SUM(CASE WHEN CAST(realized_pnlx18 AS NUMERIC) > 0 THEN 1 ELSE 0 END) as winning_trades,
			SUM(CASE WHEN CAST(realized_pnlx18 AS NUMERIC) < 0 THEN 1 ELSE 0 END) as losing_trades,
			SUM(CASE WHEN is_reduce = TRUE AND CAST(realized_pnlx18 AS NUMERIC) = 0 THEN 1 ELSE 0 END) as reduce_zero_pnl_trades
		`).
		Where("broker_id = ? AND created_at BETWEEN ? AND ?", brokerId, startDate, endDate).
		Group("subaccount_id").
		Order("total_realized_pnl DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (*FillDB) GetAllUsersStats(address, startDate, endDate string) (dtos.LeaderboardResult, error) {
	var result dtos.LeaderboardResult

	err := db.Model(&FillTable{}).
		Select(`
			subaccount_id AS user_address, 
			COUNT(*) as total_trades, 
			COALESCE(SUM(CAST(realized_pnlx18 AS NUMERIC)), 0) as total_realized_pnl,
			SUM(CASE WHEN CAST(realized_pnlx18 AS NUMERIC) > 0 THEN 1 ELSE 0 END) as winning_trades,
			SUM(CASE WHEN CAST(realized_pnlx18 AS NUMERIC) < 0 THEN 1 ELSE 0 END) as losing_trades,
			SUM(CASE WHEN is_reduce = TRUE AND CAST(realized_pnlx18 AS NUMERIC) = 0 THEN 1 ELSE 0 END) as reduce_zero_pnl_trades
		`).
		Where("created_at BETWEEN ? AND ? AND subaccount_id = ?", startDate, endDate, address).
		Group("subaccount_id").
		Order("total_realized_pnl DESC").
		Scan(&result).Error

	if err != nil {
		return dtos.LeaderboardResult{}, err
	}

	return result, nil
}

func (*FillDB) GetUserVolumeAndFees(subaccountId string, days int) (string, string, error) {
	var result struct {
		TotalVolume string
		TotalFee    string
	}

	// Set the specific start time and the time limit based on the number of days
	startTime := time.Date(2024, 1, 1, 15, 52, 43, 0, time.UTC)
	timeLimit := time.Now().AddDate(0, 0, -days)

	// Query to calculate total volume and total fees
	err := db.Raw(`
		SELECT
			COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e18) FILTER (WHERE created_at >= ?), '0') AS total_volume,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at >= ? THEN CAST(feex18 AS NUMERIC) ELSE 0 END), '0') AS total_fee
		FROM fill_tables
		WHERE subaccount_id = ?`, startTime, startTime, timeLimit, subaccountId).Scan(&result).Error

	if err != nil {
		return "0", "0", err
	}

	return result.TotalVolume, result.TotalFee, nil
}

// DEX2: Need handling on FE
func (*FillDB) Get24hMarketVolumes() (map[uint]ctypes.BigInt, error) {
	// Initialize a map to hold market volumes
	marketVolumes := make(map[uint]ctypes.BigInt)

	// Calculate the time range: last 24 hours until now
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour) // Start time 24 hours ago
	endOfPeriod := now                        // End time is now

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1

	// Struct to store query result
	var marketData []struct {
		MarketID uint
		Volume   ctypes.BigInt
	}

	// Query to calculate per market_id volume
	err := db.Raw(`
		SELECT 
			market_id, 
			SUM(ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC) / 1e18) AS volume
		FROM 
			fill_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND subaccount_id != ?
		GROUP BY 
			market_id
		ORDER BY 
			market_id ASC
	`, startOfPeriod, endOfPeriod, ammSubaccountID).Scan(&marketData).Error

	if err != nil {
		xlog.Errorf("Error querying market volumes for the last 24 hours: %v", err)
		return nil, err
	}

	// Populate the marketVolumes map with the queried data
	for _, m := range marketData {
		marketVolumes[m.MarketID] = m.Volume
	}

	return marketVolumes, nil
}

func (*FillDB) GetUserVolumeAfterCertainDate(subaccountId string, startTime time.Time) (*big.Float, error) {
	var result struct {
		TotalVolume string
	}

	err := db.Raw(`
		SELECT
			COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e36) FILTER (WHERE created_at >= ?), '0') AS total_volume
		FROM fill_tables
		WHERE subaccount_id = ?`, startTime, subaccountId).Scan(&result).Error

	if err != nil {
		return new(big.Float).SetFloat64(0), err
	}

	totalVolume := new(big.Float)
	_, ok := totalVolume.SetString(result.TotalVolume)
	if !ok {
		return new(big.Float).SetFloat64(0), fmt.Errorf("failed to convert total volume to big.Float")
	}
	return totalVolume, nil
}

func (*FillDB) DefillamaStats(startOfPeriod time.Time, endTime time.Time, brokerId uint) (ctypes.BigInt, ctypes.BigInt, error) {
	// Set the fixed start time (startOfPeriod) and calculate last 24 hours from the provided endTime
	last24Hours := endTime.Add(-24 * time.Hour)
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1

	// Variables to store the total volume and last 24-hour volume
	var totalVolumeStr, last24HourVolumeStr string

	// Query to get both total volume since the specified start time (startOfPeriod) and the volume in the last 24 hours
	err := db.Raw(`
		SELECT 
			COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e18) FILTER (WHERE created_at >= ? AND created_at <= ?), '0') AS last_24_hour_volume,
			COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e18) FILTER (WHERE created_at >= ? AND created_at <= ?), '0') AS total_volume
		FROM 
			fill_tables
		WHERE 
			subaccount_id != ? and broker_id = ?
	`, last24Hours, endTime, startOfPeriod, endTime, ammSubaccountID, brokerId).Row().Scan(&last24HourVolumeStr, &totalVolumeStr)

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

func (*FillDB) GetDailyTotalVolumePast30Days() ([]DailyData, error) {
	startOfPeriod := time.Now().AddDate(0, 0, -30)
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1

	var results []DailyData
	var volumeData []struct {
		Date        time.Time
		DailyVolume string
	}
	err := db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e18), '0') AS daily_volume
		FROM 
			fill_tables
		WHERE 
			created_at >= ? 
			AND subaccount_id != ?
		GROUP BY 
			DATE(created_at)
		ORDER BY 
			DATE(created_at) ASC
	`, startOfPeriod, ammSubaccountID).Scan(&volumeData).Error

	if err != nil {
		xlog.Errorf("Error querying daily total volume: %v", err)
		return nil, err
	}

	for _, vol := range volumeData {
		volume := new(big.Float)
		_, success := volume.SetString(vol.DailyVolume)
		if !success {
			xlog.Errorf("Error converting daily volume to big.Float: %v", vol.DailyVolume)
			continue
		}
		divisor := new(big.Float).SetFloat64(1e18)
		volume.Quo(volume, divisor)
		scaledVolume, _ := volume.Int64()

		results = append(results, DailyData{
			Date:  vol.Date,
			Count: scaledVolume,
		})
	}

	return results, nil
}

// ReferralSubaccountData struct to hold the aggregated data for each subaccount
type ReferralSubaccountData struct {
	SubaccountID string        `json:"subaccount_id"`
	TotalValue   ctypes.BigInt `json:"total_value"` // amountx18 * pricex18
	TotalFees    ctypes.BigInt `json:"total_fees"`  // feex18
}

// GetUniqueSubAccountsWithTotal fetches unique subaccount IDs and calculates total value, total fees,
// and retrieves the maximum `ID` for up to `limit` entries at a time, starting from a given ID.
func (*FillDB) GetUniqueSubAccountsWithTotal(brokerId uint, startID uint64, limit uint) ([]ReferralSubaccountData, error) {
	var subaccountData []ReferralSubaccountData
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1

	// Calculate the endIndex
	endIndex := startID + uint64(limit)

	// Query to fetch unique subaccount IDs, total value, and total fees between startID and endIndex
	rows, err := db.Raw(`
		SELECT 
			subaccount_id AS subaccount_id, 
			COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC))), '0') AS total_value,
			COALESCE(SUM(CAST(feex18 AS NUMERIC)), '0') AS total_fees
		FROM 
			fill_tables
		WHERE 
			id > ? AND id <= ? AND
			broker_id = ?
			AND subaccount_id != ?
		GROUP BY 
			subaccount_id
		ORDER BY 
			subaccount_id ASC
	`, startID, endIndex, brokerId, ammSubaccountID).Rows()

	if err != nil {
		xlog.Errorf("Error querying total value and fees: %v", err)
		return nil, err
	}
	defer rows.Close()

	divisor := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)
	for rows.Next() {
		var data ReferralSubaccountData
		var totalValueStr, totalFeesStr string

		if err := rows.Scan(&data.SubaccountID, &totalValueStr, &totalFeesStr); err != nil {
			xlog.Errorf("Error scanning row: %v", err)
			return nil, err
		}

		// Convert string results to *big.Int
		totalValue := new(big.Int)
		totalValue.SetString(totalValueStr, 10)
		totalFees := new(big.Int)
		totalFees.SetString(totalFeesStr, 10)

		// Divide totalValue by 10^18
		totalValue.Div(totalValue, divisor)

		// Set values to ReferralSubaccountData
		data.TotalValue = ctypes.NewBigInt(totalValue)
		data.TotalFees = ctypes.NewBigInt(totalFees)

		subaccountData = append(subaccountData, data)
	}
	return subaccountData, nil
}

// Function to fetch the max ID
func (*FillDB) GetMaxID() (uint64, error) {
	var maxID uint64
	result := db.Raw(`
		SELECT COALESCE(MAX(id), 0) AS max_id 
		FROM fill_tables
	`).Scan(&maxID)

	if result.Error != nil {
		return 0, result.Error
	}
	return maxID, nil
}

// DEX2: Not adding BROKER ID as this is used in internal dashboard
// NOTE: This isn't very expensive call for now but will
func (*FillDB) GetEarningsFromLiquidation(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	// Get the total earnings from liquidation for each market
	var earningsByMarket []struct {
		MarketID uint          `gorm:"column:market_id"`
		Earnings ctypes.BigInt `gorm:"column:total_earnings"`
	}

	totalEarningsAllMarkets := ctypes.NewBigInt(big.NewInt(0))
	excludedSubaccounts := os.Getenv("BOT_ACCOUNT_UNREALIZED")

	// Query to fetch total earnings from liquidation for each market
	query := `
		SELECT 
			market_id,
			COALESCE(-SUM(CAST(realized_pnlx18 AS NUMERIC)), 0) AS total_earnings
		FROM fill_tables
		WHERE type = 'LIQUIDATION' AND broker_id = ?`

	if excludedSubaccounts != "" {
		query += fmt.Sprintf(" AND subaccount_id NOT IN ('%s')",
			strings.ReplaceAll(excludedSubaccounts, ",", "','"))
	}

	query += " GROUP BY market_id"

	err := db.Raw(query, brokerId).Scan(&earningsByMarket).Error

	if err != nil {
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	var earningsMap = make(map[uint]ctypes.BigInt)
	for _, marketEarnings := range earningsByMarket {
		earningsMap[marketEarnings.MarketID] = marketEarnings.Earnings
		totalEarningsAllMarkets = totalEarningsAllMarkets.Add(marketEarnings.Earnings)
	}

	return earningsMap, totalEarningsAllMarkets, nil
}

// -------- functions to create triggers on table ----------
func (*FillDB) MatchOrderTriggerFunction() {
	triggerFunctionExists := false

	//Check if the trigger function exists
	rows, err := db.Raw(`
		SELECT EXISTS (
            SELECT 1
            FROM pg_proc
            WHERE proname = 'notify_match_order'
        )
	`).Rows()
	if err != nil {
		xlog.Errorf("Fill Table Trigger - Failed to check trigger function existence: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var exists bool
		if err := rows.Scan(&exists); err != nil {
			xlog.Errorf("Fill Table Trigger - Failed to scan trigger function existence %v\n", err)
		}
	}

	// Create trigger function if it does not exist
	if !triggerFunctionExists {
		triggerFunctionSQL := `
        CREATE OR REPLACE FUNCTION notify_match_order_channel() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_notify('match_order_channel', row_to_json(NEW)::text);
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
        `
		if err := db.Exec(triggerFunctionSQL).Error; err != nil {
			xlog.Errorf("Fill Table Trigger - Failed to create trigger function: %v\n", err)
		} else {
			xlog.Infof("Fill Table Trigger - Trigger function 'notify_match_order' created.")
		}
	} else {
		xlog.Infof("Fill Table Trigger - Trigger function 'notify_match_order' already exists.")
	}
}

// NO BROKER ID filtering required
func (*FillDB) MatchOrderTrigger() {
	triggerExists := false

	// Check if the trigger exists on fill_tables
	rows, err := db.Raw(`
		SELECT EXISTS (
            SELECT 1
            FROM pg_trigger
            WHERE tgname = 'match_order'
            AND tgrelid = 'fill_tables'::regclass
        )
	`).Rows()
	if err != nil {
		xlog.Errorf("Fill Table Trigger - Failed to check trigger existence: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var exists bool
		if err := rows.Scan(&exists); err != nil {
			xlog.Errorf("Fill Table Trigger - Failed to scan trigger existence: %v\n", err)
			return
		}
		triggerExists = exists
	}

	if !triggerExists {
		// Create the trigger
		triggerSQL := `
        CREATE TRIGGER match_order
        AFTER INSERT ON fill_tables
        FOR EACH ROW
        EXECUTE FUNCTION notify_match_order();
        `
		if err := db.Exec(triggerSQL).Error; err != nil {
			xlog.Errorf("Fill Table Trigger - Failed to create trigger: %v\n", err)
		} else {
			xlog.Infof("Fill Table Trigger - Trigger match_order created on 'fill_tables'.")
		}
	} else {
		xlog.Infof("Fill Table Trigger - Trigger 'match_order' already exists on 'fill_tables'.")
	}
}

func (*FillDB) GetVolumeAndTraderCount(brokerId uint, startDate, endDate time.Time, limit int, specificSubaccountId string) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {
	var leaderboard []map[string]interface{}
	var userRankInfo map[string]interface{}

	// Retrieve internal and AMM subaccount IDs from environment variables
	internalSubAccountIdList := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")
	ammSubAccountIdList := []string{contractUtils.AMM_SUBACCOUNT_ID_1}
	mergedSubAccountIdList := append(internalSubAccountIdList, ammSubAccountIdList...)

	// Generate cache key for leaderboard
	cacheKey := fmt.Sprintf("%d_%d_%d_%d", brokerId, startDate.Unix(), endDate.Unix(), limit)

	// Try to get leaderboard from cache
	cacheMutex.RLock()
	leaderboardCacheEntry, exists := leaderboardCache[cacheKey]
	cacheMutex.RUnlock()

	if exists && !leaderboardCacheEntry.IsExpired() {
		// Use cached leaderboard data
		if cachedLeaderboard := leaderboardCacheEntry.Get(); cachedLeaderboard != nil {
			leaderboard = *cachedLeaderboard
		}
	} else {
		// Base query with conditional market ID filtering
		params := []interface{}{brokerId, startDate, endDate, pq.Array(mergedSubAccountIdList), limit}

		// Query to get top N users by volume within the date range and specified market IDs
		rows, err := db.Raw(`
			SELECT 
				subaccount_id,
				COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e36), '0') AS total_volume,
				COUNT(*) AS total_trades
			FROM fill_tables
			WHERE broker_id = ? AND
			created_at BETWEEN ? AND ?
			AND subaccount_id != ALL(?)
			GROUP BY subaccount_id
			ORDER BY total_volume DESC
			LIMIT ?
		`, params...).Rows()
		if err != nil {
			return nil, nil, nil, err
		}
		defer rows.Close()

		rank := 1 // Initialize rank for leaderboard entries
		// Populate leaderboard with query results
		for rows.Next() {
			var subaccountID string
			var totalVolume string
			var totalTrades int

			err := rows.Scan(&subaccountID, &totalVolume, &totalTrades)
			if err != nil {
				return nil, nil, nil, err
			}

			entry := map[string]interface{}{
				"subaccount_id": subaccountID,
				"total_volume":  totalVolume,
				"total_trades":  totalTrades,
				"rank":          rank,
			}
			leaderboard = append(leaderboard, entry)
			rank++
		}

		// Cache the leaderboard data
		cacheMutex.Lock()
		leaderboardCache[cacheKey] = xcache.NewCache(leaderboard, time.Time{})
		leaderboardCache[cacheKey].Set(leaderboard, cacheExpiryDuration)
		cacheMutex.Unlock()
	}

	// Find the address of specific subaccount in leaderboard and take user rank info
	// If not found in leaderboard return rank as > limit  (like '> 100')
	foundSpecificUser := false
	for _, entry := range leaderboard {
		if entry["subaccount_id"] == specificSubaccountId {
			userRankInfo = map[string]interface{}{
				"total_volume": entry["total_volume"],
				"total_trades": entry["total_trades"],
				"rank":         entry["rank"],
			}
			foundSpecificUser = true
			break
		}
	}

	// If specificSubaccountId is provided, get that user's total volume and trader count
	if !foundSpecificUser && specificSubaccountId != "" {
		var result struct {
			TotalVolume string
			TraderCount int
		}

		userParams := []interface{}{brokerId, specificSubaccountId, startDate, endDate}

		err := db.Raw(`
			SELECT 
				COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e36), '0') AS total_volume,
				COUNT(DISTINCT subaccount_id) AS trader_count
			FROM fill_tables
			WHERE broker_id = ? AND subaccount_id = ? AND created_at BETWEEN ? AND ?
		`, userParams...).Scan(&result).Error

		if err != nil {
			return nil, nil, nil, err
		}

		userRankInfo = map[string]interface{}{
			"total_volume": result.TotalVolume,
			"total_trades": result.TraderCount,
			"rank":         fmt.Sprintf("> %d", limit),
		}
	}

	// Try to get total volume data from cache
	totalVolumeCacheKey := fmt.Sprintf("total_%s", cacheKey)

	cacheMutex.RLock()
	totalVolumeCacheEntry, exists := totalVolumeCache[totalVolumeCacheKey]
	cacheMutex.RUnlock()

	var totalTradeMap map[string]interface{}

	if exists && !totalVolumeCacheEntry.IsExpired() {
		// Use cached total volume data
		if cachedTotalVolume := totalVolumeCacheEntry.Get(); cachedTotalVolume != nil {
			totalTradeMap = *cachedTotalVolume
		}
	} else {
		// Get total volume and trader count across all users
		var totalVolumeResult struct {
			TotalVolume string
			TotalTrades int
		}

		err := db.Raw(`
			SELECT 
				COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e36), '0') AS total_volume,
				COUNT(*) AS total_trades
			FROM fill_tables
			WHERE broker_id = ? AND subaccount_id != ALL(?) AND created_at BETWEEN ? AND ?
		`, brokerId, pq.Array(mergedSubAccountIdList), startDate, endDate).Scan(&totalVolumeResult).Error

		if err != nil {
			return leaderboard, userRankInfo, nil, err
		}

		totalTradeMap = map[string]interface{}{
			"total_volume": totalVolumeResult.TotalVolume,
			"total_trades": totalVolumeResult.TotalTrades,
		}

		// Cache the total volume data
		cacheMutex.Lock()
		totalVolumeCache[totalVolumeCacheKey] = xcache.NewCache(totalTradeMap, time.Time{})
		totalVolumeCache[totalVolumeCacheKey].Set(totalTradeMap, cacheExpiryDuration)
		cacheMutex.Unlock()
	}

	return leaderboard, userRankInfo, totalTradeMap, nil
}

// DEX2: Broker ID handling is done where function is being used.
func (*FillDB) GetRowsAfterID(id uint64, count int) ([]FillTable, error) {
	var fills []FillTable

	// Query to select rows after the specified ID with a limit of 'count'
	err := db.Where("id > ?", id).
		Order("id ASC").
		Limit(count).
		Find(&fills).Error

	if err != nil {
		return nil, err
	}

	return fills, nil
}

// DEX2: BROKER ID handling is skipped as this is used in internal dashboard
// Function to get paginated and filtered data from the database
func (*FillDB) GetPaginatedAndFilteredData(page, pageSize int, marketId *uint, subaccountId, fillType, side *string) (PaginatedResponse, error) {
	var fills []FillTable
	var totalRows int64

	offset := (page - 1) * pageSize
	query := db.Model(&FillTable{})

	// Apply filters if provided
	if marketId != nil {
		query = query.Where("market_id = ?", *marketId)
	}
	if subaccountId != nil && *subaccountId != "" {
		query = query.Where("subaccount_id = ?", *subaccountId)
	}
	if fillType != nil && *fillType != "" {
		query = query.Where("type = ?", *fillType)
	}
	if side != nil && *side != "" {
		query = query.Where("side = ?", *side)
	}

	// Count total rows after applying filters
	err := query.Count(&totalRows).Error
	if err != nil {
		return PaginatedResponse{}, err
	}

	// Fetch paginated rows with filters applied
	err = query.Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&fills).Error
	if err != nil {
		return PaginatedResponse{}, err
	}

	totalPages := int((totalRows + int64(pageSize) - 1) / int64(pageSize))

	return PaginatedResponse{
		Data:       fills,
		TotalRows:  totalRows,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (*FillDB) GetTradeHistoryWithMarketSymbols(subaccountID string) (*[]map[string]interface{}, error) {

	if subaccountID == "" {
		return nil, fmt.Errorf("subaccountID is required")
	}

	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	query := `
		SELECT 
			f.id,
			f.created_at,
			f.subaccount_id,
			f.market_id,
			f.side,
			f.amountx18,
			f.pricex18,
			f.feex18,
			f.realized_pnlx18,
			m.symbol as market_symbol,
			COALESCE(r.fee_bonusx18, '0') as fee_bonus_x18,
            CAST((CAST(f.pricex18 AS NUMERIC) * CAST(f.amountx18 AS NUMERIC)/1e18) AS TEXT) as total_value_x18
		FROM 
			fill_tables f
		LEFT JOIN 
			market_tables m ON f.market_id = m.id
		LEFT JOIN 
			user_fee_rewards_tables r ON f.id = r.fill_table_id AND r.subaccount_id = ?
		WHERE 
			f.subaccount_id = ? 
			AND f.created_at >= ?
		ORDER BY f.created_at DESC
	`

	params := []interface{}{subaccountID, subaccountID, sixMonthsAgo}

	var results []map[string]interface{}
	err := db.Raw(query, params...).Scan(&results).Error
	if err != nil {
		xlog.Errorf("Database error when fetching trade history with market symbols: %v", err)
		return nil, fmt.Errorf("failed to fetch trade history: %w", err)
	}

	return &results, nil
}

// GetLatestLiquidationTrades returns the latest 50 liquidation trades for broker ID 2
func (*FillDB) GetLatestLiquidationTrades() (*[]FillTable, error) {
	fills := []FillTable{}

	err := db.Raw(`
		SELECT * FROM fill_tables 
		WHERE type = 'LIQUIDATION' 
		  AND broker_id = 2
		ORDER BY created_at DESC
		LIMIT 50
	`).Scan(&fills).Error

	if err != nil {
		return nil, err
	}

	return &fills, nil
}

// GetTotalVolumeForSubaccounts calculates the total volume generated by a list of subaccount IDs
func (*FillDB) GetTotalVolumeForSubaccounts(subaccountIds []string) (string, error) {
	if len(subaccountIds) == 0 {
		return "0", nil
	}

	var result struct {
		TotalVolume string `gorm:"column:total_volume"`
	}

	err := db.Raw(`
		SELECT 
			COALESCE(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e18), '0') AS total_volume
		FROM fill_tables
		WHERE subaccount_id IN (?)
	`, subaccountIds).Scan(&result).Error

	if err != nil {
		return "0", err
	}

	return result.TotalVolume, nil
}

//-------------------------- Functions that are not used on mainnet anymore -------------------------//
// ------------------------- Functions that are not used on mainnet anymore -------------------------//
//-------------------------- Functions that are not used on mainnet anymore -------------------------//

func (*FillDB) CountBySubaccountIdAndSide(subaccountId string, side string) (int64, error) {
	var count int64
	err := db.Raw(`
        SELECT COUNT(*)
        FROM fill_tables
        WHERE subaccount_id = ? AND side = ?
    `, subaccountId, side).Scan(&count).Error

	if err != nil {
		fmt.Println("Error querying database:", err)
		return 0, err
	}
	return count, nil
}

func (*FillDB) CountBySubaccountIdAndMarketId(subaccountId string, marketId int64, side string) (int64, error) {
	var count int64
	err := db.Raw(`
        SELECT COUNT(*)
        FROM fill_tables
        WHERE subaccount_id = ? AND market_id = ? AND side = ? 
    `, subaccountId, marketId, side).Scan(&count).Error

	if err != nil {
		fmt.Println("Error querying database:", err)
		return 0, err
	}
	return count, nil
}
func (*FillDB) CountBySubaccountIdAndMarketIdAndType(subaccountId string, marketId int64, typeOfOrder string) (int64, error) {
	var count int64
	err := db.Raw(`
        SELECT COUNT(*)
        FROM fill_tables
        WHERE subaccount_id = ? AND market_id = ? AND type = ?
    `, subaccountId, marketId, typeOfOrder).Scan(&count).Error

	if err != nil {
		fmt.Println("Error querying database:", err)
		return 0, err
	}
	return count, nil
}

func (*FillDB) CountBySubaccountIdAndMarketIdAndSide(subaccountId string, marketId int64, side string, typeOfOrder string) (int64, error) {
	var count int64
	err := db.Raw(`
        SELECT COUNT(*)
        FROM fill_tables
        WHERE subaccount_id = ? AND market_id = ? AND side = ? AND type = ?
    `, subaccountId, marketId, side, typeOfOrder).Scan(&count).Error

	if err != nil {
		fmt.Println("Error querying database:", err)
		return 0, err
	}
	return count, nil
}

func (*FillDB) UniqueDaysTradesCount(subaccountId string) (int64, error) {
	var count int64
	err := db.Raw(`
        SELECT COUNT(DISTINCT DATE(created_at)) AS active_days
		FROM fill_tables
		WHERE created_at >= '2000-01-01' AND subaccount_id = ?
    `, subaccountId).Scan(&count).Error

	if err != nil {
		fmt.Println("Error querying database:", err)
		return 0, err
	}
	return count, nil
}

func (*FillDB) GetRecentFillsByMarketIds(marketIds []uint, limit int) map[uint]*[]FillTable {
	if len(marketIds) == 0 {
		return make(map[uint]*[]FillTable)
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	result := make(map[uint]*[]FillTable)

	for _, marketId := range marketIds {
		result[marketId] = &[]FillTable{}
	}

	// Process each market individually for optimal index usage: this method might seem inefficient at glance
	// Reason: Db is not sharded -> too large to use window functions
	for _, marketId := range marketIds {
		fills := []FillTable{}
		tx := db.Where("market_id = ? AND subaccount_id != ?", marketId, ammSubaccountID).
			Order("id DESC").
			Limit(limit).
			Find(&fills)

		if tx.Error != nil {
			xlog.Errorf("Error fetching fills for marketId=%d: %v", marketId, tx.Error)
			continue
		}

		if len(fills) > 0 {
			result[marketId] = &fills
		}
	}

	return result
}

// GetUserAddressFromSubaccount extracts the Ethereum address from subaccount ID
func (f *FillTable) GetUserAddressFromSubaccount() (string, error) {
	return cutils.ExtractETHAddress(f.SubaccountId)
}

// GetProductIDFromMarketID returns the product ID (same as market ID for perp)
func (f *FillTable) GetProductIDFromMarketID() uint32 {
	return uint32(f.MarketId)
}

// GetIsTakerFromLiquidity returns whether the fill is from a taker order
func (f *FillTable) GetIsTakerFromLiquidity() bool {
	return f.Liquidity == ctypes.LIQUIDITY_TAKER
}

// GetBaseDelta returns the base delta (same as amount for perp)
func (f *FillTable) GetBaseDelta() ctypes.BigInt {
	return f.Amountx18
}

// GetQuoteDelta calculates the quote delta from amount and price
func (f *FillTable) GetQuoteDelta() ctypes.BigInt {
	// QuoteDelta = Amountx18 * Pricex18 / 1e18
	quoteValue := new(big.Int).Mul(f.Amountx18.Val, f.Pricex18.Val)
	quoteValue = new(big.Int).Div(quoteValue, big.NewInt(1e18))
	return ctypes.NewBigInt(quoteValue)
}

// GetRealisedPnl returns the realized PnL (same as RealizedPnlx18)
func (f *FillTable) GetRealisedPnl() ctypes.BigInt {
	return f.RealizedPnlx18
}

// GetFeeAmount returns the fee amount (same as Feex18)
func (f *FillTable) GetFeeAmount() ctypes.BigInt {
	return f.Feex18
}

// GetFundingFees returns the funding fees
func (f *FillTable) GetFundingFees() ctypes.BigInt {
	return f.FundingFeesx18
}

// GetDailyFees returns daily fees aggregated by date for a broker
// NOTE: Uses fill_order_tables for dates before cutoff date, fill_tables for dates after cutoff date
func (*FillDB) GetDailyFees(brokerId uint, startDate ...time.Time) (map[string][]interface{}, map[uint]map[string][]interface{}, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)

	var startOfPeriod time.Time
	var dayCount int
	if len(startDate) > 0 {
		startOfPeriod = startDate[0].UTC().Truncate(24 * time.Hour)
		dayCount = int(now.Sub(startOfPeriod).Hours()/24) + 1
	} else {
		startOfPeriod = now.AddDate(0, 0, -29)
		dayCount = 30
	}
	if dayCount <= 0 {
		dayCount = 1
	}

	aggregatedFees := map[string][]interface{}{
		"date":        make([]interface{}, dayCount),
		"funding_fee": make([]interface{}, dayCount),
		"trading_fee": make([]interface{}, dayCount),
	}
	productFees := make(map[uint]map[string][]interface{})
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	ammSubaccountIDForFillOrder := contractUtils.AMM_SUBACCOUNT_ID

	endOfPeriod := now.Add(24 * time.Hour).Add(-time.Nanosecond)
	cutoffDate := FundingFeesMigrationCutoffDate

	var aggregatedData []struct {
		Date        time.Time
		FundingFees ctypes.BigInt
		TradingFees ctypes.BigInt
	}

	// For dates before cutoff, use fill_order_tables
	// For dates after cutoff, use fill_tables
	// For dates spanning cutoff, use UNION to query both tables in one query
	if endOfPeriod.Before(cutoffDate) {
		// Entire period is before cutoff - use fill_order_tables only
		err := db.Raw(`
			SELECT 
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
				COALESCE(SUM(CAST(funding_fees AS NUMERIC)), 0) AS funding_fees,
				COALESCE(SUM(CAST(fee_amount AS NUMERIC)), 0) AS trading_fees
			FROM fill_order_tables
			WHERE created_at BETWEEN ? AND ? 
			  AND broker_id = ?
			  AND sub_account_id != ?
			GROUP BY date
			ORDER BY date DESC
		`, startOfPeriod, endOfPeriod, brokerId, ammSubaccountIDForFillOrder).Scan(&aggregatedData).Error
		if err != nil {
			return nil, nil, err
		}
	} else if startOfPeriod.After(cutoffDate) || startOfPeriod.Equal(cutoffDate) {
		// Entire period is after cutoff - use fill_tables only
		err := db.Raw(`
			SELECT 
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
				COALESCE(SUM(CAST(funding_feesx18 AS NUMERIC)), 0) AS funding_fees,
				COALESCE(SUM(CAST(feex18 AS NUMERIC)), 0) AS trading_fees
			FROM fill_tables
			WHERE created_at BETWEEN ? AND ? 
			  AND broker_id = ?
			  AND subaccount_id != ?
			GROUP BY date
			ORDER BY date DESC
		`, startOfPeriod, endOfPeriod, brokerId, ammSubaccountID).Scan(&aggregatedData).Error
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Period spans cutoff date - use UNION to query both tables in one query
		err := db.Raw(`
			SELECT 
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
				COALESCE(SUM(funding_fees), 0) AS funding_fees,
				COALESCE(SUM(trading_fees), 0) AS trading_fees
			FROM (
				SELECT 
					created_at,
					CAST(funding_fees AS NUMERIC) AS funding_fees,
					CAST(fee_amount AS NUMERIC) AS trading_fees
				FROM fill_order_tables
				WHERE created_at BETWEEN ? AND ? 
				  AND broker_id = ?
				  AND sub_account_id != ?
				UNION ALL
				SELECT 
					created_at,
					CAST(funding_feesx18 AS NUMERIC) AS funding_fees,
					CAST(feex18 AS NUMERIC) AS trading_fees
				FROM fill_tables
				WHERE created_at BETWEEN ? AND ? 
				  AND broker_id = ?
				  AND subaccount_id != ?
			) combined
			GROUP BY date
			ORDER BY date DESC
		`, startOfPeriod, cutoffDate.Add(-time.Nanosecond), brokerId, ammSubaccountIDForFillOrder,
			cutoffDate, endOfPeriod, brokerId, ammSubaccountID).Scan(&aggregatedData).Error
		if err != nil {
			return nil, nil, err
		}
	}

	for i := 0; i < dayCount; i++ {
		date := now.AddDate(0, 0, -i)
		aggregatedFees["date"][i] = date
		aggregatedFees["funding_fee"][i] = "0"
		aggregatedFees["trading_fee"][i] = "0"
	}
	for _, day := range aggregatedData {
		dayIndex := int(now.Sub(day.Date).Hours() / 24)
		if dayIndex >= 0 && dayIndex < dayCount {
			aggregatedFees["funding_fee"][dayIndex] = day.FundingFees.String()
			aggregatedFees["trading_fee"][dayIndex] = day.TradingFees.String()
		}
	}

	var productData []struct {
		Date        time.Time
		MarketID    uint
		FundingFees ctypes.BigInt
		TradingFees ctypes.BigInt
	}

	// Query product-level data based on cutoff date
	if endOfPeriod.Before(cutoffDate) {
		// Entire period is before cutoff - use fill_order_tables only
		err := db.Raw(`
			SELECT 
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
				product_id AS market_id,
				COALESCE(SUM(CAST(funding_fees AS NUMERIC)), 0) AS funding_fees,
				COALESCE(SUM(CAST(fee_amount AS NUMERIC)), 0) AS trading_fees
			FROM fill_order_tables
			WHERE created_at BETWEEN ? AND ? 
			  AND broker_id = ?
			  AND sub_account_id != ?
			GROUP BY date, market_id
			ORDER BY date DESC
		`, startOfPeriod, endOfPeriod, brokerId, ammSubaccountIDForFillOrder).Scan(&productData).Error
		if err != nil {
			return nil, nil, err
		}
	} else if startOfPeriod.After(cutoffDate) || startOfPeriod.Equal(cutoffDate) {
		// Entire period is after cutoff - use fill_tables only
		err := db.Raw(`
			SELECT 
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
				market_id,
				COALESCE(SUM(CAST(funding_feesx18 AS NUMERIC)), 0) AS funding_fees,
				COALESCE(SUM(CAST(feex18 AS NUMERIC)), 0) AS trading_fees
			FROM fill_tables
			WHERE created_at BETWEEN ? AND ? 
			  AND broker_id = ?
			  AND subaccount_id != ?
			GROUP BY date, market_id
			ORDER BY date DESC
		`, startOfPeriod, endOfPeriod, brokerId, ammSubaccountID).Scan(&productData).Error
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Period spans cutoff date - use UNION to query both tables in one query
		err := db.Raw(`
			SELECT 
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
				market_id,
				COALESCE(SUM(funding_fees), 0) AS funding_fees,
				COALESCE(SUM(trading_fees), 0) AS trading_fees
			FROM (
				SELECT 
					created_at,
					product_id AS market_id,
					CAST(funding_fees AS NUMERIC) AS funding_fees,
					CAST(fee_amount AS NUMERIC) AS trading_fees
				FROM fill_order_tables
				WHERE created_at BETWEEN ? AND ? 
				  AND broker_id = ?
				  AND sub_account_id != ?
				UNION ALL
				SELECT 
					created_at,
					market_id,
					CAST(funding_feesx18 AS NUMERIC) AS funding_fees,
					CAST(feex18 AS NUMERIC) AS trading_fees
				FROM fill_tables
				WHERE created_at BETWEEN ? AND ? 
				  AND broker_id = ?
				  AND subaccount_id != ?
			) combined
			GROUP BY date, market_id
			ORDER BY date DESC
		`, startOfPeriod, cutoffDate.Add(-time.Nanosecond), brokerId, ammSubaccountIDForFillOrder,
			cutoffDate, endOfPeriod, brokerId, ammSubaccountID).Scan(&productData).Error
		if err != nil {
			return nil, nil, err
		}
	}

	for _, row := range productData {
		if _, ok := productFees[row.MarketID]; !ok {
			productFees[row.MarketID] = map[string][]interface{}{
				"funding_fee": make([]interface{}, dayCount),
				"trading_fee": make([]interface{}, dayCount),
			}
			for i := 0; i < dayCount; i++ {
				productFees[row.MarketID]["funding_fee"][i] = "0"
				productFees[row.MarketID]["trading_fee"][i] = "0"
			}
		}
		dayIndex := int(now.Sub(row.Date).Hours() / 24)
		if dayIndex >= 0 && dayIndex < dayCount {
			productFees[row.MarketID]["funding_fee"][dayIndex] = row.FundingFees.String()
			productFees[row.MarketID]["trading_fee"][dayIndex] = row.TradingFees.String()
		}
	}

	// Manual fee data for specific dates (only for broker ID 1)
	if brokerId == 1 {
		manualFees := map[time.Time]string{
			time.Date(2025, 06, 17, 0, 0, 0, 0, time.UTC): "1811231018382918867194",
			time.Date(2025, 06, 18, 0, 0, 0, 0, time.UTC): "1627834818382918867194",
			time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC): "1583561456278654321291",
			time.Date(2025, 06, 20, 0, 0, 0, 0, time.UTC): "1498936286403948304061",
			time.Date(2025, 06, 21, 0, 0, 0, 0, time.UTC): "1733561456278654321291",
			time.Date(2025, 06, 22, 0, 0, 0, 0, time.UTC): "1227834818382918867194",
			time.Date(2025, 06, 23, 0, 0, 0, 0, time.UTC): "1301287026894111241241",
			time.Date(2025, 06, 24, 0, 0, 0, 0, time.UTC): "1291782326894111241241",
			time.Date(2025, 06, 26, 0, 0, 0, 0, time.UTC): "1583561456278654321291",
			time.Date(2025, 06, 27, 0, 0, 0, 0, time.UTC): "1227834818382918867194",
			time.Date(2025, 06, 28, 0, 0, 0, 0, time.UTC): "1301287026894111241241",
			time.Date(2025, 06, 29, 0, 0, 0, 0, time.UTC): "1291782326894111241241",
			time.Date(2025, 06, 30, 0, 0, 0, 0, time.UTC): "1733561456278654321291",
			time.Date(2025, 07, 01, 0, 0, 0, 0, time.UTC): "1862936286403948304061",
			time.Date(2025, 07, 03, 0, 0, 0, 0, time.UTC): "1583561456278654321291",
			time.Date(2025, 07, 04, 0, 0, 0, 0, time.UTC): "1483561456278654321291",
			time.Date(2025, 07, 05, 0, 0, 0, 0, time.UTC): "1733561456278654321291",
			time.Date(2025, 07, 06, 0, 0, 0, 0, time.UTC): "1602231456278654321291",
			time.Date(2025, 07, 07, 0, 0, 0, 0, time.UTC): "1227834818382918867194",
			time.Date(2025, 07, 10, 0, 0, 0, 0, time.UTC): "2056147921190945642275",
			time.Date(2025, 07, 11, 0, 0, 0, 0, time.UTC): "1733561456278654321291",
			time.Date(2025, 07, 12, 0, 0, 0, 0, time.UTC): "1665561456278654321291",
			time.Date(2025, 07, 27, 0, 0, 0, 0, time.UTC): "10000",
		}
		for i, date := range aggregatedFees["date"] {
			if fee, ok := manualFees[date.(time.Time)]; ok {
				aggregatedFees["trading_fee"][i] = fee
			}
		}
	}

	return aggregatedFees, productFees, nil
}

// GetDailyVolumes returns daily volumes aggregated by date for a broker
func (*FillDB) GetDailyVolumes(brokerId uint, startDate ...time.Time) (map[uint]map[string][]interface{}, ctypes.BigInt, []time.Time, error) {
	productVolumes := make(map[uint]map[string][]interface{})
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1

	now := time.Now().UTC().Truncate(24 * time.Hour)

	var dayCount int
	if len(startDate) > 0 {
		dayCount = int(now.Sub(startDate[0]).Hours()/24) + 1
	} else {
		dayCount = 30
	}
	if dayCount <= 0 {
		dayCount = 1
	}

	dates := make([]time.Time, dayCount)
	for i := 0; i < dayCount; i++ {
		dates[i] = now.AddDate(0, 0, -i)
	}
	latestDate := dates[0]
	startOfWindow := dates[dayCount-1]
	endExclusive := latestDate.Add(24 * time.Hour)

	var lifetimeVolumeData struct {
		Volume ctypes.BigInt
	}
	if err := db.Raw(`
        SELECT 
            SUM(ABS((amountx18::numeric * pricex18::numeric) / 1e18)) as volume
        FROM 
            fill_tables
        WHERE 
            subaccount_id NOT IN (?) AND broker_id = ?
    `, ammSubaccountID, brokerId).Scan(&lifetimeVolumeData).Error; err != nil {
		xlog.Errorf("Error querying lifetime volume: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), nil, err
	}
	totalLifetimeVolume := lifetimeVolumeData.Volume

	// Add manual volume adjustment only for broker ID 1
	if brokerId == 1 {
		additionalVolume := new(big.Int)
		additionalVolume.SetString("15376775000000000000000000", 10)
		if totalLifetimeVolume.Val == nil {
			totalLifetimeVolume.Val = new(big.Int)
		}
		totalLifetimeVolume.Val.Add(totalLifetimeVolume.Val, additionalVolume)
	}

	// Query per-market daily volumes
	var volumeData []struct {
		MarketID uint
		Volume   ctypes.BigInt
		Date     time.Time
	}
	if err := db.Raw(`
        SELECT 
            market_id, 
            SUM(ABS((amountx18::numeric * pricex18::numeric) / 1e18)) as volume,
            DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') as date
        FROM 
            fill_tables
        WHERE 
            created_at >= ? AND created_at < ?
            AND broker_id = ?
            AND subaccount_id NOT IN (?)
        GROUP BY
            market_id, DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
        ORDER BY
            date DESC
    `, startOfWindow, endExclusive, brokerId, ammSubaccountID).Scan(&volumeData).Error; err != nil {
		xlog.Errorf("Error querying product volumes: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), nil, err
	}

	// --- Manual adjustments scaffolding (only for broker 1) ---
	var targetDates map[time.Time]*big.Float
	var targetDateVolumes map[time.Time]map[uint]*big.Int
	if brokerId == 1 {
		targetDates = map[time.Time]*big.Float{
			time.Date(2025, 06, 17, 0, 0, 0, 0, time.UTC): big.NewFloat(18112310000000000000000000),
			time.Date(2025, 06, 18, 0, 0, 0, 0, time.UTC): big.NewFloat(16278348000000000000000000),
			time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC): big.NewFloat(15835614000000000000000000),
			time.Date(2025, 06, 20, 0, 0, 0, 0, time.UTC): big.NewFloat(14989362000000000000000000),
			time.Date(2025, 06, 21, 0, 0, 0, 0, time.UTC): big.NewFloat(17335614000000000000000000),
			time.Date(2025, 06, 22, 0, 0, 0, 0, time.UTC): big.NewFloat(12278348000000000000000000),
			time.Date(2025, 06, 23, 0, 0, 0, 0, time.UTC): big.NewFloat(13012870000000000000000000),
			time.Date(2025, 06, 24, 0, 0, 0, 0, time.UTC): big.NewFloat(12917823000000000000000000),
			time.Date(2025, 06, 26, 0, 0, 0, 0, time.UTC): big.NewFloat(15835614000000000000000000),
			time.Date(2025, 06, 27, 0, 0, 0, 0, time.UTC): big.NewFloat(12278348000000000000000000),
			time.Date(2025, 06, 28, 0, 0, 0, 0, time.UTC): big.NewFloat(13012870000000000000000000),
			time.Date(2025, 06, 29, 0, 0, 0, 0, time.UTC): big.NewFloat(12917823000000000000000000),
			time.Date(2025, 06, 30, 0, 0, 0, 0, time.UTC): big.NewFloat(17335614000000000000000000),
			time.Date(2025, 07, 01, 0, 0, 0, 0, time.UTC): big.NewFloat(18629363000000000000000000),
			time.Date(2025, 07, 03, 0, 0, 0, 0, time.UTC): big.NewFloat(15835614000000000000000000),
			time.Date(2025, 07, 04, 0, 0, 0, 0, time.UTC): big.NewFloat(14835614000000000000000000),
			time.Date(2025, 07, 05, 0, 0, 0, 0, time.UTC): big.NewFloat(17335614000000000000000000),
			time.Date(2025, 07, 06, 0, 0, 0, 0, time.UTC): big.NewFloat(16022314000000000000000000),
			time.Date(2025, 07, 07, 0, 0, 0, 0, time.UTC): big.NewFloat(12278348000000000000000000),
			time.Date(2025, 07, 10, 0, 0, 0, 0, time.UTC): big.NewFloat(20561479000000000000000000),
			time.Date(2025, 07, 11, 0, 0, 0, 0, time.UTC): big.NewFloat(17335614000000000000000000),
			time.Date(2025, 07, 12, 0, 0, 0, 0, time.UTC): big.NewFloat(16655614000000000000000000),
			time.Date(2025, 07, 27, 0, 0, 0, 0, time.UTC): big.NewFloat(53245123000000000000000000),
		}
		targetDateVolumes = make(map[time.Time]map[uint]*big.Int)
		for d := range targetDates {
			targetDateVolumes[d] = make(map[uint]*big.Int)
		}
	}

	// Fill per-market arrays with zeros up-front as needed and load data
	for _, row := range volumeData {
		dateIndex := int(latestDate.Sub(row.Date.UTC()).Hours() / 24)
		if dateIndex < 0 || dateIndex >= dayCount {
			continue
		}

		if _, ok := productVolumes[row.MarketID]; !ok {
			productVolumes[row.MarketID] = map[string][]interface{}{
				"volume": make([]interface{}, dayCount),
			}
			zero := ctypes.NewBigInt(big.NewInt(0))
			zeroStr := (&zero).String()
			for j := 0; j < dayCount; j++ {
				productVolumes[row.MarketID]["volume"][j] = zeroStr
			}
		}
		productVolumes[row.MarketID]["volume"][dateIndex] = (&row.Volume).String()

		if bucket, ok := targetDateVolumes[row.Date.UTC()]; ok {
			bucket[row.MarketID] = row.Volume.Val
		}
	}

	// Manual redistribution across BTC/ETH/SOL for broker 1
	if brokerId == 1 && targetDates != nil {
		// Market IDs corresponding to BTC, ETH, SOL
		targetSet := map[uint]bool{3: true, 1: true, 5: true}
		for d, target := range targetDates {
			current := big.NewInt(0)
			for mid, vol := range targetDateVolumes[d] {
				if targetSet[mid] {
					current.Add(current, vol)
				}
			}
			if current.Sign() > 0 {
				scale := new(big.Float).Quo(target, new(big.Float).SetInt(current))
				for mid, vol := range targetDateVolumes[d] {
					if targetSet[mid] {
						adjF := new(big.Float).Mul(new(big.Float).SetInt(vol), scale)
						adjI, _ := adjF.Int(nil)
						adj := ctypes.NewBigInt(adjI)
						idx := int(latestDate.Sub(d).Hours() / 24)
						if arr, ok := productVolumes[mid]; ok && idx >= 0 && idx < dayCount {
							arr["volume"][idx] = (&adj).String()
						}
					}
				}
			}
		}
	}

	return productVolumes, totalLifetimeVolume, dates, nil
}

// GetDailyVolumesInternal returns daily volumes for internal dashboard
func (*FillDB) GetDailyVolumesInternal() (map[uint]map[string][]interface{}, ctypes.BigInt, []time.Time, error) {
	productVolumes := make(map[uint]map[string][]interface{})
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Combine the AMM subaccount and internal subaccounts into one slice
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	// Initialize dates array for the last 30 days from today
	dates := make([]time.Time, 30)
	for i := 0; i < 30; i++ {
		dates[i] = time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -i)
	}

	// Calculate the total lifetime volume excluding specified subaccount IDs
	var lifetimeVolumeData struct {
		Volume ctypes.BigInt
	}
	err := db.Raw(`
		SELECT 
			SUM(ABS((amountx18::numeric * pricex18::numeric) / 1e18)) as volume
		FROM 
			fill_tables
		WHERE 
			subaccount_id NOT IN (?)
	`, excludeSubaccountIDs).Scan(&lifetimeVolumeData).Error

	if err != nil {
		xlog.Errorf("Error querying lifetime volume: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), nil, err
	}
	totalLifetimeVolume := lifetimeVolumeData.Volume

	// Query product volumes grouped by day for the last 30 days
	var volumeData []struct {
		MarketId uint
		Volume   ctypes.BigInt
		Date     time.Time
	}
	endDate := dates[0].Add(24 * time.Hour)
	err = db.Raw(`
			SELECT 
				market_id, 
				SUM(ABS((amountx18::numeric * pricex18::numeric) / 1e18)) as volume,
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') as date
			FROM 
				fill_tables
			WHERE 
				created_at BETWEEN ? AND ? 
				AND subaccount_id NOT IN (?)
			GROUP BY 
				market_id, DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
			ORDER BY 
				date DESC
		`, dates[29], endDate, excludeSubaccountIDs).Scan(&volumeData).Error

	if err != nil {
		xlog.Errorf("Error querying volume data: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), nil, err
	}

	// Set the reference date to the latest date in the data (i.e., dates[0] or today)
	latestDate := dates[0]

	// Process the query results and map data to productVolumes
	for _, data := range volumeData {
		// Calculate the date index based on latestDate
		dateIndex := int(latestDate.Sub(data.Date.UTC()).Hours() / 24)
		if dateIndex < 0 || dateIndex >= 30 {
			continue // Skip out-of-range data
		}

		// Initialize volume array if it doesn't exist
		if _, ok := productVolumes[data.MarketId]; !ok {
			productVolumes[data.MarketId] = map[string][]interface{}{
				"volume": make([]interface{}, 30),
			}
			for j := 0; j < 30; j++ {
				productVolumes[data.MarketId]["volume"][j] = ctypes.NewBigInt(big.NewInt(0))
			}
		}
		productVolumes[data.MarketId]["volume"][dateIndex] = data.Volume.String()
	}

	return productVolumes, totalLifetimeVolume, dates, nil
}

// GetDailyOverallVolumes returns daily overall volumes for dashboard
func (*FillDB) GetDailyOverallVolumes(brokerId uint, startDate ...time.Time) ([]DailyData, error) {
	var startOfPeriod time.Time
	if len(startDate) > 0 {
		startOfPeriod = startDate[0]
	} else {
		startOfPeriod = time.Now().AddDate(0, 0, -30).UTC()
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1

	var results []DailyData
	err := db.Raw(`
		SELECT 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
			FLOOR(SUM(ABS((amountx18::numeric * pricex18::numeric) / 1e36)))::bigint AS count
		FROM fill_tables
		WHERE created_at >= ?
		  AND subaccount_id != ? AND broker_id = ?
		GROUP BY 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
		ORDER BY 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') ASC
	`, startOfPeriod, ammSubaccountID, brokerId).Scan(&results).Error
	if err != nil {
		xlog.Errorf("Error querying daily total volume: %v", err)
		return nil, err
	}

	// Override volumes for specific dates (only for broker ID 1, only from June 16th onwards)
	if brokerId == 1 {
		overrideVolumes := map[string]int64{
			"2025-06-17": 18112310,
			"2025-06-18": 16278348,
			"2025-06-19": 15835614,
			"2025-06-20": 14989362,
			"2025-06-21": 17335614,
			"2025-06-22": 12278348,
			"2025-06-23": 13012870,
			"2025-06-24": 12917823,
			"2025-06-26": 15835614,
			"2025-06-27": 12278348,
			"2025-06-28": 13012870,
			"2025-06-29": 12917823,
			"2025-06-30": 17335614,
			"2025-07-01": 18629363,
			"2025-07-03": 15835614,
			"2025-07-04": 14835614,
			"2025-07-05": 17335614,
			"2025-07-06": 16022314,
			"2025-07-07": 12278348,
			"2025-07-10": 20561479,
			"2025-07-11": 17335614,
			"2025-07-12": 16655614,
			"2025-07-27": 53245123,
		}
		for i, data := range results {
			dateStr := data.Date.Format("2006-01-02")
			if v, ok := overrideVolumes[dateStr]; ok {
				results[i].Count = v
			}
		}
	}

	return results, nil
}

// GetCumulativeTradingFees returns cumulative trading fees up to a specific date
func (*FillDB) GetCumulativeTradingFees(endDate time.Time, brokerId uint) (string, error) {
	var cumulativeFees float64
	startDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	err := db.Raw(`
		SELECT 
			COALESCE(SUM(ABS(feex18::numeric)) / 1e18, 0) AS cumulative_fees
		FROM 
			fill_tables
		WHERE 
			created_at AT TIME ZONE 'UTC' BETWEEN ? AND ? AND subaccount_id != ? AND broker_id = ?
	`, startDate, endDate, ammSubaccountID, brokerId).Scan(&cumulativeFees).Error

	if err != nil {
		xlog.Errorf("Error while fetching cumulative trading fees: %v", err)
		return "0", err
	}

	october31 := time.Date(2024, 10, 31, 0, 0, 0, 0, time.UTC)
	november9 := time.Date(2024, 11, 9, 0, 0, 0, 0, time.UTC)
	march02 := time.Date(2024, 03, 02, 0, 0, 0, 0, time.UTC)
	march03 := time.Date(2024, 03, 03, 0, 0, 0, 0, time.UTC)
	april25 := time.Date(2025, 04, 25, 0, 0, 0, 0, time.UTC)
	april26 := time.Date(2025, 04, 26, 0, 0, 0, 0, time.UTC)
	april27 := time.Date(2025, 04, 27, 0, 0, 0, 0, time.UTC)
	april20 := time.Date(2025, 04, 20, 0, 0, 0, 0, time.UTC)
	may04 := time.Date(2025, 05, 04, 0, 0, 0, 0, time.UTC)
	may09 := time.Date(2025, 05, 9, 0, 0, 0, 0, time.UTC)
	may10 := time.Date(2025, 05, 10, 0, 0, 0, 0, time.UTC)
	may11 := time.Date(2025, 05, 11, 0, 0, 0, 0, time.UTC)
	may17 := time.Date(2025, 05, 17, 0, 0, 0, 0, time.UTC)
	may18 := time.Date(2025, 05, 18, 0, 0, 0, 0, time.UTC)
	may24 := time.Date(2025, 05, 24, 0, 0, 0, 0, time.UTC)
	may25 := time.Date(2025, 05, 25, 0, 0, 0, 0, time.UTC)
	june01 := time.Date(2025, 06, 01, 0, 0, 0, 0, time.UTC)
	june02 := time.Date(2025, 06, 02, 0, 0, 0, 0, time.UTC)
	june04 := time.Date(2025, 06, 04, 0, 0, 0, 0, time.UTC)
	june05 := time.Date(2025, 06, 05, 0, 0, 0, 0, time.UTC)
	june06 := time.Date(2025, 06, 06, 0, 0, 0, 0, time.UTC)
	june07 := time.Date(2025, 06, 07, 0, 0, 0, 0, time.UTC)
	june10 := time.Date(2025, 06, 10, 0, 0, 0, 0, time.UTC)
	june11 := time.Date(2025, 06, 11, 0, 0, 0, 0, time.UTC)
	june12 := time.Date(2025, 06, 12, 0, 0, 0, 0, time.UTC)
	june13 := time.Date(2025, 06, 13, 0, 0, 0, 0, time.UTC)
	june14 := time.Date(2025, 06, 14, 0, 0, 0, 0, time.UTC)
	june15 := time.Date(2025, 06, 15, 0, 0, 0, 0, time.UTC)
	june17 := time.Date(2025, 06, 17, 0, 0, 0, 0, time.UTC)
	june18 := time.Date(2025, 06, 18, 0, 0, 0, 0, time.UTC)
	june19 := time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC)
	june20 := time.Date(2025, 06, 20, 0, 0, 0, 0, time.UTC)
	june21 := time.Date(2025, 06, 21, 0, 0, 0, 0, time.UTC)
	june22 := time.Date(2025, 06, 22, 0, 0, 0, 0, time.UTC)
	june23 := time.Date(2025, 06, 23, 0, 0, 0, 0, time.UTC)
	june24 := time.Date(2025, 06, 24, 0, 0, 0, 0, time.UTC)
	june26 := time.Date(2025, 06, 26, 0, 0, 0, 0, time.UTC)
	june27 := time.Date(2025, 06, 27, 0, 0, 0, 0, time.UTC)
	june28 := time.Date(2025, 06, 28, 0, 0, 0, 0, time.UTC)
	june29 := time.Date(2025, 06, 29, 0, 0, 0, 0, time.UTC)
	june30 := time.Date(2025, 06, 30, 0, 0, 0, 0, time.UTC)
	july01 := time.Date(2025, 07, 01, 0, 0, 0, 0, time.UTC)
	july03 := time.Date(2025, 07, 03, 0, 0, 0, 0, time.UTC)
	july04 := time.Date(2025, 07, 04, 0, 0, 0, 0, time.UTC)
	july05 := time.Date(2025, 07, 05, 0, 0, 0, 0, time.UTC)
	july06 := time.Date(2025, 07, 06, 0, 0, 0, 0, time.UTC)
	july07 := time.Date(2025, 07, 07, 0, 0, 0, 0, time.UTC)
	july10 := time.Date(2025, 07, 10, 0, 0, 0, 0, time.UTC)
	july11 := time.Date(2025, 07, 11, 0, 0, 0, 0, time.UTC)
	july12 := time.Date(2025, 07, 12, 0, 0, 0, 0, time.UTC)
	july27 := time.Date(2025, 07, 27, 0, 0, 0, 0, time.UTC)

	// Manual cumulative fee adjustments only for broker ID 1
	if brokerId == 1 {
		// these are for missed dates for internal subaccount fees
		if endDate.After(october31) {
			cumulativeFees += 4546.88
		}
		if endDate.After(november9) {
			cumulativeFees += 3141.51
		}

		if endDate.After(march02) {
			cumulativeFees += 11062.51
		}
		if endDate.After(march03) {
			cumulativeFees += 12556.38
		}
		if endDate.After(april25) {
			cumulativeFees += 2227.83
		}
		if endDate.After(april26) {
			cumulativeFees += 2862.94
		}
		if endDate.After(april27) {
			cumulativeFees += 2056.15
		}
		if endDate.After(may04) {
			cumulativeFees += 2583.56
		}
		if endDate.After(may09) {
			cumulativeFees += 3602.23
		}
		if endDate.After(may10) {
			cumulativeFees += 2301.29
		}
		if endDate.After(may11) {
			cumulativeFees += 2602.23
		}

		if endDate.After(april20) {
			cumulativeFees += 2492.67
		}

		if endDate.After(may17) {
			cumulativeFees += 2583.56
		}
		if endDate.After(may18) {
			cumulativeFees += 1765.56
		}
		if endDate.After(may24) {
			cumulativeFees += 2056.15
		}
		if endDate.After(may25) {
			cumulativeFees += 2227.83
		}
		if endDate.After(june01) {
			cumulativeFees += 2583.56
		}
		if endDate.After(june02) {
			cumulativeFees += 2862.94
		}
		if endDate.After(june04) {
			cumulativeFees += 2498.32
		}
		if endDate.After(june05) {
			cumulativeFees += 1733.56
		}
		if endDate.After(june06) {
			cumulativeFees += 2583.56
		}
		if endDate.After(june07) {
			cumulativeFees += 2583.56
		}
		if endDate.After(june10) {
			cumulativeFees += 2227.83
		}
		if endDate.After(june11) {
			cumulativeFees += 2056.15
		}
		if endDate.After(june12) {
			cumulativeFees += 2056.15
		}
		if endDate.After(june13) {
			cumulativeFees += 2056.15
		}
		if endDate.After(june14) {
			cumulativeFees += 1765.56
		}
		if endDate.After(june15) {
			cumulativeFees += 1827.83
		}
		if endDate.After(june17) {
			cumulativeFees += 1811.23
		}
		if endDate.After(june18) {
			cumulativeFees += 1627.83
		}
		if endDate.After(june19) {
			cumulativeFees += 1583.56
		}
		if endDate.After(june20) {
			cumulativeFees += 1498.94
		}
		if endDate.After(june21) {
			cumulativeFees += 1733.56
		}
		if endDate.After(june22) {
			cumulativeFees += 1227.83
		}
		if endDate.After(june23) {
			cumulativeFees += 1301.29
		}
		if endDate.After(june24) {
			cumulativeFees += 1291.78
		}
		if endDate.After(june26) {
			cumulativeFees += 1583.56
		}
		if endDate.After(june27) {
			cumulativeFees += 1227.83
		}
		if endDate.After(june28) {
			cumulativeFees += 1301.29
		}
		if endDate.After(june29) {
			cumulativeFees += 1291.78
		}
		if endDate.After(june30) {
			cumulativeFees += 1733.56
		}
		if endDate.After(july01) {
			cumulativeFees += 1862.94
		}
		if endDate.After(july03) {
			cumulativeFees += 1583.56
		}
		if endDate.After(july04) {
			cumulativeFees += 1733.56
		}
		if endDate.After(july05) {
			cumulativeFees += 1862.94
		}
		if endDate.After(july06) {
			cumulativeFees += 1602.23
		}
		if endDate.After(july07) {
			cumulativeFees += 1227.83
		}
		if endDate.After(july10) {
			cumulativeFees += 2056.15
		}
		if endDate.After(july11) {
			cumulativeFees += 1733.56
		}
		if endDate.After(july12) {
			cumulativeFees += 1665.56
		}
		if endDate.After(july27) {
			cumulativeFees += 0
		}
	}
	cumulativeFeesString := strconv.FormatFloat(cumulativeFees, 'f', 2, 64)

	return cumulativeFeesString, nil
}

// GetCumulativeVolume returns cumulative volume up to a specific date
func (*FillDB) GetCumulativeVolume(endDate time.Time, brokerId uint) (string, error) {
	var cumulativeVolume float64
	startDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	err := db.Raw(`
		SELECT 
			COALESCE(SUM(ABS((amountx18::numeric * pricex18::numeric) / 1e36)), 0) AS cumulative_volume
		FROM 
			fill_tables
		WHERE 
			created_at AT TIME ZONE 'UTC' BETWEEN ? AND ? AND subaccount_id != ? AND broker_id = ?
	`, startDate, endDate, ammSubaccountID, brokerId).Scan(&cumulativeVolume).Error

	if err != nil {
		xlog.Errorf("Error while fetching cumulative volume: %v", err)
		return "0", err
	}

	october31 := time.Date(2024, 10, 31, 0, 0, 0, 0, time.UTC)
	november9 := time.Date(2024, 11, 9, 0, 0, 0, 0, time.UTC)
	feb19 := time.Date(2025, 02, 19, 0, 0, 0, 0, time.UTC)
	march02 := time.Date(2024, 03, 2, 0, 0, 0, 0, time.UTC)
	march03 := time.Date(2024, 03, 3, 0, 0, 0, 0, time.UTC)
	april25 := time.Date(2025, 04, 25, 0, 0, 0, 0, time.UTC)
	april26 := time.Date(2025, 04, 26, 0, 0, 0, 0, time.UTC)
	april27 := time.Date(2025, 04, 27, 0, 0, 0, 0, time.UTC)
	april20 := time.Date(2025, 04, 20, 0, 0, 0, 0, time.UTC)
	may04 := time.Date(2025, 05, 04, 0, 0, 0, 0, time.UTC)
	may09 := time.Date(2025, 05, 9, 0, 0, 0, 0, time.UTC)
	may10 := time.Date(2025, 05, 10, 0, 0, 0, 0, time.UTC)
	may11 := time.Date(2025, 05, 11, 0, 0, 0, 0, time.UTC)
	may17 := time.Date(2025, 05, 17, 0, 0, 0, 0, time.UTC)
	may18 := time.Date(2025, 05, 18, 0, 0, 0, 0, time.UTC)
	may24 := time.Date(2025, 05, 24, 0, 0, 0, 0, time.UTC)
	may25 := time.Date(2025, 05, 25, 0, 0, 0, 0, time.UTC)
	june01 := time.Date(2025, 06, 01, 0, 0, 0, 0, time.UTC)
	june02 := time.Date(2025, 06, 02, 0, 0, 0, 0, time.UTC)
	june04 := time.Date(2025, 06, 04, 0, 0, 0, 0, time.UTC)
	june05 := time.Date(2025, 06, 05, 0, 0, 0, 0, time.UTC)
	june06 := time.Date(2025, 06, 06, 0, 0, 0, 0, time.UTC)
	june07 := time.Date(2025, 06, 07, 0, 0, 0, 0, time.UTC)
	june10 := time.Date(2025, 06, 10, 0, 0, 0, 0, time.UTC)
	june11 := time.Date(2025, 06, 11, 0, 0, 0, 0, time.UTC)
	june12 := time.Date(2025, 06, 12, 0, 0, 0, 0, time.UTC)
	june13 := time.Date(2025, 06, 13, 0, 0, 0, 0, time.UTC)
	june14 := time.Date(2025, 06, 14, 0, 0, 0, 0, time.UTC)
	june15 := time.Date(2025, 06, 15, 0, 0, 0, 0, time.UTC)
	june17 := time.Date(2025, 06, 17, 0, 0, 0, 0, time.UTC)
	june18 := time.Date(2025, 06, 18, 0, 0, 0, 0, time.UTC)
	june19 := time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC)
	june20 := time.Date(2025, 06, 20, 0, 0, 0, 0, time.UTC)
	june21 := time.Date(2025, 06, 21, 0, 0, 0, 0, time.UTC)
	june22 := time.Date(2025, 06, 22, 0, 0, 0, 0, time.UTC)
	june23 := time.Date(2025, 06, 23, 0, 0, 0, 0, time.UTC)
	june24 := time.Date(2025, 06, 24, 0, 0, 0, 0, time.UTC)
	june26 := time.Date(2025, 06, 26, 0, 0, 0, 0, time.UTC)
	june27 := time.Date(2025, 06, 27, 0, 0, 0, 0, time.UTC)
	june28 := time.Date(2025, 06, 28, 0, 0, 0, 0, time.UTC)
	june29 := time.Date(2025, 06, 29, 0, 0, 0, 0, time.UTC)
	june30 := time.Date(2025, 06, 30, 0, 0, 0, 0, time.UTC)
	july01 := time.Date(2025, 07, 01, 0, 0, 0, 0, time.UTC)
	july03 := time.Date(2025, 07, 03, 0, 0, 0, 0, time.UTC)
	july04 := time.Date(2025, 07, 04, 0, 0, 0, 0, time.UTC)
	july05 := time.Date(2025, 07, 05, 0, 0, 0, 0, time.UTC)
	july06 := time.Date(2025, 07, 06, 0, 0, 0, 0, time.UTC)
	july07 := time.Date(2025, 07, 07, 0, 0, 0, 0, time.UTC)
	july10 := time.Date(2025, 07, 10, 0, 0, 0, 0, time.UTC)
	july11 := time.Date(2025, 07, 11, 0, 0, 0, 0, time.UTC)
	july12 := time.Date(2025, 07, 12, 0, 0, 0, 0, time.UTC)
	july27 := time.Date(2025, 07, 27, 0, 0, 0, 0, time.UTC)

	// Manual cumulative volume adjustments only for broker ID 1
	if brokerId == 1 {
		// these are for missed dates for internal subaccount volumes
		if endDate.After(october31) {
			cumulativeVolume += 9093763
		}
		if endDate.After(november9) {
			cumulativeVolume += 6283012
		}
		if endDate.After(feb19) {
			cumulativeVolume += 12167123
		}
		if endDate.After(march02) {
			cumulativeVolume += 20434700
		}
		if endDate.After(march03) {
			cumulativeVolume += 24105552
		}
		if endDate.After(april25) {
			cumulativeVolume += 10113154
		}
		if endDate.After(april26) {
			cumulativeVolume += 9112532
		}
		if endDate.After(april27) {
			cumulativeVolume += 11112310
		}
		if endDate.After(may04) {
			cumulativeVolume += 25835614
		}
		if endDate.After(may09) {
			cumulativeVolume += 18112310
		}
		if endDate.After(may10) {
			cumulativeVolume += 23012870
		}
		if endDate.After(may11) {
			cumulativeVolume += 20611472
		}
		if endDate.After(april20) {
			cumulativeVolume += 24926733
		}
		if endDate.After(may17) {
			cumulativeVolume += 15835614
		}
		if endDate.After(may18) {
			cumulativeVolume += 12926733
		}
		if endDate.After(june01) {
			cumulativeVolume += 15856145
		}
		if endDate.After(june02) {
			cumulativeVolume += 18629363
		}
		if endDate.After(june04) {
			cumulativeVolume += 24983232
		}
		if endDate.After(june05) {
			cumulativeVolume += 17335614
		}
		if endDate.After(june06) {
			cumulativeVolume += 15835614
		}
		if endDate.After(june07) {
			cumulativeVolume += 18629363
		}
		if endDate.After(june10) {
			cumulativeVolume += 18278348
		}
		if endDate.After(june11) {
			cumulativeVolume += 20561479
		}
		if endDate.After(june12) {
			cumulativeVolume += 17917823
		}
		if endDate.After(june13) {
			cumulativeVolume += 205615
		}
		if endDate.After(june14) {
			cumulativeVolume += 17655614
		}
		if endDate.After(june26) {
			cumulativeVolume += 15835614
		}
		if endDate.After(may24) {
			cumulativeVolume += 20561500
		}
		if endDate.After(may25) {
			cumulativeVolume += 22278300
		}
		if endDate.After(june15) {
			cumulativeVolume += 18278300
		}
		if endDate.After(june17) {
			cumulativeVolume += 18112300
		}
		if endDate.After(june18) {
			cumulativeVolume += 16278300
		}
		if endDate.After(june19) {
			cumulativeVolume += 15835614
		}
		if endDate.After(june20) {
			cumulativeVolume += 14989400
		}
		if endDate.After(june21) {
			cumulativeVolume += 17335614
		}
		if endDate.After(june22) {
			cumulativeVolume += 12278300
		}
		if endDate.After(june23) {
			cumulativeVolume += 13012900
		}
		if endDate.After(june24) {
			cumulativeVolume += 12917800
		}
		if endDate.After(june27) {
			cumulativeVolume += 12278300
		}
		if endDate.After(june28) {
			cumulativeVolume += 13012900
		}
		if endDate.After(june29) {
			cumulativeVolume += 12917800
		}
		if endDate.After(june30) {
			cumulativeVolume += 17335614
		}
		if endDate.After(july01) {
			cumulativeVolume += 18629400
		}
		if endDate.After(july03) {
			cumulativeVolume += 15835614
		}
		if endDate.After(july04) {
			cumulativeVolume += 14835614
		}
		if endDate.After(july05) {
			cumulativeVolume += 17335614
		}
		if endDate.After(july06) {
			cumulativeVolume += 16022314
		}
		if endDate.After(july07) {
			cumulativeVolume += 12278300
		}
		if endDate.After(july10) {
			cumulativeVolume += 20561500
		}
		if endDate.After(july11) {
			cumulativeVolume += 17335614
		}
		if endDate.After(july12) {
			cumulativeVolume += 16655614
		}
		if endDate.After(july27) {
			cumulativeVolume += 53245123
		}
	}
	cumulativeVolumeString := strconv.FormatFloat(cumulativeVolume, 'f', 2, 64)

	return cumulativeVolumeString, nil
}

// GetTotalVolume returns total volume by product and overall
func (*FillDB) GetTotalVolume(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	startOfPeriod := time.Date(2024, 8, 14, 15, 52, 43, 0, time.UTC)
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	var results []struct {
		MarketId uint
		Volume   ctypes.BigInt
	}

	err := db.Raw(`
		SELECT 
			market_id,
			COALESCE(CAST(SUM(ABS((CAST(amountx18 AS NUMERIC) * CAST(pricex18 AS NUMERIC)) / 1e18)) AS TEXT), '0') AS volume
		FROM 
			fill_tables
		WHERE 
			created_at > ? 
			AND subaccount_id != ?
			AND subaccount_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			market_id
	`, startOfPeriod, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&results).Error

	if err != nil {
		xlog.Errorf("Error querying total volume since the specified timestamp: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	productVolumes := make(map[uint]ctypes.BigInt)
	totalVolume := ctypes.NewBigInt(big.NewInt(0))

	for _, result := range results {
		productVolumes[result.MarketId] = result.Volume
		totalVolume = totalVolume.Add(result.Volume)
	}

	return productVolumes, totalVolume, nil
}

// Get24hRealisedPnL returns 24h realized PnL by product and total
func (*FillDB) Get24hRealisedPnL(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	endOfPeriod := now.Format("2006-01-02 15:04:05")

	type PnlData struct {
		MarketId    uint
		RealisedPnl ctypes.BigInt
	}

	var ammPnlData, internalAccountPnlData []PnlData
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err1 := db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(realized_pnlx18 AS NUMERIC)) AS realised_pnl
		FROM fill_tables
		WHERE created_at BETWEEN ? AND ?
			AND subaccount_id = ? AND broker_id = ?
		GROUP BY market_id
	`, startOfPeriod, endOfPeriod, ammSubaccountID, brokerId).Scan(&ammPnlData).Error
	if err1 != nil {
		xlog.Errorf("Error while fetching 24h realised PnL %v", err1)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err1
	}

	err2 := db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(realized_pnlx18 AS NUMERIC)) AS realised_pnl
		FROM fill_tables
		WHERE created_at BETWEEN ? AND ?
			AND subaccount_id IN (?) AND broker_id = ?
		GROUP BY market_id
	`, startOfPeriod, endOfPeriod, internalSubaccountIDs, brokerId).Scan(&internalAccountPnlData).Error
	if err2 != nil {
		xlog.Errorf("Error while fetching 24h realised PnL %v", err2)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err2
	}

	// Create a map for internalAccountPnlData for fast lookup
	internalPnLMap := make(map[uint]ctypes.BigInt)
	for _, pnl := range internalAccountPnlData {
		internalPnLMap[pnl.MarketId] = pnl.RealisedPnl
	}

	totalRealizedPnL := ctypes.NewBigInt(big.NewInt(0))
	pnLPerMarket := make(map[uint]ctypes.BigInt)

	for _, pnl := range ammPnlData {
		marketId := pnl.MarketId
		ammPnl := pnl.RealisedPnl
		if internalPnl, ok := internalPnLMap[marketId]; ok {
			sum := ammPnl.Add(internalPnl)
			pnLPerMarket[marketId] = sum
			totalRealizedPnL = totalRealizedPnL.Add(sum)
		} else {
			pnLPerMarket[marketId] = ammPnl
			totalRealizedPnL = totalRealizedPnL.Add(ammPnl)
		}
	}

	return pnLPerMarket, totalRealizedPnL, nil
}

// Get24hTradingFee returns 24h trading fees by product and total
func (*FillDB) Get24hTradingFee(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	endOfPeriod := now.Format("2006-01-02 15:04:05")

	var feeData []struct {
		MarketId  uint
		FeeAmount ctypes.BigInt
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err := db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(feex18 AS NUMERIC)) as fee_amount
		FROM 
			fill_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND subaccount_id != ?
			AND subaccount_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			market_id
	`, startOfPeriod, endOfPeriod, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&feeData).Error

	if err != nil {
		xlog.Errorf("Error executing query:", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalFeeAmount := ctypes.NewBigInt(big.NewInt(0))
	feePerToken := make(map[uint]ctypes.BigInt)
	for _, fee := range feeData {
		if currentFee, exists := feePerToken[fee.MarketId]; exists {
			feePerToken[fee.MarketId] = currentFee.Add(fee.FeeAmount)
		} else {
			feePerToken[fee.MarketId] = fee.FeeAmount
		}
		totalFeeAmount = totalFeeAmount.Add(fee.FeeAmount)
	}

	return feePerToken, totalFeeAmount, nil
}

// Get24hFundingFee returns 24h funding fees by product and total
func (*FillDB) Get24hFundingFee(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	endOfPeriod := now.Format("2006-01-02 15:04:05")

	var feeData []struct {
		MarketId  uint
		FeeAmount ctypes.BigInt
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err := db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(funding_feesx18 AS NUMERIC)) as fee_amount
		FROM 
			fill_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND subaccount_id != ?
			AND subaccount_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			market_id
	`, startOfPeriod, endOfPeriod, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&feeData).Error

	if err != nil {
		xlog.Errorf("Error executing query:", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalFeeAmount := ctypes.NewBigInt(big.NewInt(0))
	feePerToken := make(map[uint]ctypes.BigInt)
	for _, fee := range feeData {
		if currentFee, exists := feePerToken[fee.MarketId]; exists {
			feePerToken[fee.MarketId] = currentFee.Add(fee.FeeAmount)
		} else {
			feePerToken[fee.MarketId] = fee.FeeAmount
		}
		totalFeeAmount = totalFeeAmount.Add(fee.FeeAmount)
	}

	return feePerToken, totalFeeAmount, nil
}

// GetTotalTradingFee returns total trading fees by product and overall
func (*FillDB) GetTotalTradingFee(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	var tradingFeeData []struct {
		MarketId  uint
		FeeAmount ctypes.BigInt
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err := db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(feex18 AS NUMERIC)) as fee_amount
		FROM 
			fill_tables
		WHERE 
			subaccount_id != ?
			AND subaccount_id NOT IN (?) AND broker_id = ?
			AND created_at > '2024-08-14 15:52:43'
		GROUP BY 
			market_id
	`, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&tradingFeeData).Error

	if err != nil {
		xlog.Errorf("Error while fetching total realised PnL for AMM subaccount: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalTradingFee := ctypes.NewBigInt(big.NewInt(0))
	tradingFeePerToken := make(map[uint]ctypes.BigInt)
	for _, tradingFee := range tradingFeeData {
		if currentFee, exists := tradingFeePerToken[tradingFee.MarketId]; exists {
			tradingFeePerToken[tradingFee.MarketId] = currentFee.Add(tradingFee.FeeAmount)
		} else {
			tradingFeePerToken[tradingFee.MarketId] = tradingFee.FeeAmount
		}
		totalTradingFee = totalTradingFee.Add(tradingFee.FeeAmount)
	}

	return tradingFeePerToken, totalTradingFee, nil
}

// GetTotalFundingFee returns total funding fees by product and overall
// NOTE: Uses fill_order_tables for dates before cutoff date, fill_tables for dates after cutoff date
func (*FillDB) GetTotalFundingFee(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	var fundingFeeData []struct {
		MarketId    uint
		FundingFees ctypes.BigInt
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	ammSubaccountIDForFillOrder := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")
	internalSubaccountIDsHex := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")
	cutoffDate := FundingFeesMigrationCutoffDate

	// Query both tables: fill_order_tables before cutoff, fill_tables after cutoff
	var oldData, newData []struct {
		MarketId    uint
		FundingFees ctypes.BigInt
	}

	// Query fill_order_tables for period before cutoff
	err := db.Raw(`
		SELECT 
			product_id AS market_id, 
			SUM(CAST(funding_fees AS NUMERIC)) as funding_fees
		FROM fill_order_tables
		WHERE sub_account_id != ?
		  AND sub_account_id NOT IN (?) 
		  AND broker_id = ?
		  AND created_at > '2024-08-14 15:52:43'
		  AND created_at < ?
		GROUP BY market_id
	`, ammSubaccountIDForFillOrder, internalSubaccountIDsHex, brokerId, cutoffDate).Scan(&oldData).Error
	if err != nil {
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	// Query fill_tables for period after cutoff
	err = db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(funding_feesx18 AS NUMERIC)) as funding_fees
		FROM fill_tables
		WHERE subaccount_id != ?
		  AND subaccount_id NOT IN (?) 
		  AND broker_id = ?
		  AND created_at >= ?
		GROUP BY market_id
	`, ammSubaccountID, internalSubaccountIDs, brokerId, cutoffDate).Scan(&newData).Error
	if err != nil {
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	// Combine and aggregate by market_id
	feeMap := make(map[uint]*big.Int)
	for _, d := range oldData {
		if feeMap[d.MarketId] == nil {
			feeMap[d.MarketId] = big.NewInt(0)
		}
		feeMap[d.MarketId].Add(feeMap[d.MarketId], d.FundingFees.Val)
	}
	for _, d := range newData {
		if feeMap[d.MarketId] == nil {
			feeMap[d.MarketId] = big.NewInt(0)
		}
		feeMap[d.MarketId].Add(feeMap[d.MarketId], d.FundingFees.Val)
	}
	for marketId, amount := range feeMap {
		fundingFeeData = append(fundingFeeData, struct {
			MarketId    uint
			FundingFees ctypes.BigInt
		}{MarketId: marketId, FundingFees: ctypes.NewBigInt(amount)})
	}

	if err != nil {
		xlog.Errorf("Error while fetching total funding fees for AMM subaccount: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalFundingFee := ctypes.NewBigInt(big.NewInt(0))
	fundingFeePerToken := make(map[uint]ctypes.BigInt)
	for _, fundingFee := range fundingFeeData {
		if currentFee, exists := fundingFeePerToken[fundingFee.MarketId]; exists {
			fundingFeePerToken[fundingFee.MarketId] = currentFee.Add(fundingFee.FundingFees)
		} else {
			fundingFeePerToken[fundingFee.MarketId] = fundingFee.FundingFees
		}
		totalFundingFee = totalFundingFee.Add(fundingFee.FundingFees)
	}

	return fundingFeePerToken, totalFundingFee, nil
}

// GetTotalUserRealisedPnL returns total user realized PnL by product and overall
func (*FillDB) GetTotalUserRealisedPnL(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	type PnlData struct {
		MarketId    uint
		RealisedPnl ctypes.BigInt
	}

	var pnLData []PnlData

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Combine the AMM subaccount and internal subaccounts into one slice
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	// Perform the query
	err := db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(realized_pnlx18 AS NUMERIC)) as realised_pnl
		FROM 
			fill_tables
		WHERE 
			subaccount_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			market_id
	`, excludeSubaccountIDs, brokerId).Scan(&pnLData).Error

	if err != nil {
		xlog.Errorf("Error while fetching total realised PnL: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	// Process the result and calculate total realized PnL
	totalRealizedPnL := ctypes.NewBigInt(big.NewInt(0))
	pnLPerToken := make(map[uint]ctypes.BigInt)

	for _, pnl := range pnLData {
		pnLPerToken[pnl.MarketId] = pnl.RealisedPnl
		totalRealizedPnL = totalRealizedPnL.Add(pnl.RealisedPnl)
	}

	return pnLPerToken, totalRealizedPnL, nil
}

// GetDailyActiveTraders returns daily active traders data
func (*FillDB) GetDailyActiveTraders(brokerId uint) ([]map[string]interface{}, error) {
	// Define the struct within the function scope
	type DailyActiveTrader struct {
		Count int       `json:"count"`
		Date  time.Time `json:"date"`
	}

	// Collect exclusion subaccount IDs from environment variables
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Combine the AMM and internal subaccounts
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	// Slice to hold query results
	var results []DailyActiveTrader

	// Execute the query to get daily active traders
	err := db.Raw(`
		SELECT 
			COUNT(DISTINCT subaccount_id) AS count,
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date
		FROM 
			fill_tables
		WHERE 
			subaccount_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
		ORDER BY 
			date DESC
		LIMIT 30
	`, excludeSubaccountIDs, brokerId).Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("error querying daily active traders: %v", err)
	}

	// Convert results to []map[string]interface{} for flexibility
	dailyActiveTraders := make([]map[string]interface{}, len(results))
	for i, record := range results {
		dailyActiveTraders[i] = map[string]interface{}{
			"count": record.Count,
			"date":  record.Date,
		}
	}

	return dailyActiveTraders, nil
}

// Get24hVolumes returns 24h volumes by product and total
func (*FillDB) Get24hVolumes(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	productVolumes := make(map[uint]ctypes.BigInt)
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour) // Start time 24 hours ago
	endOfPeriod := now                        // End time is now
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Query to get the 24-hour volume for each product and the total volume in one go
	var productData []struct {
		MarketId uint
		Volume   ctypes.BigInt
	}
	err := db.Raw(`
		SELECT 
			market_id, 
			COALESCE(CAST(SUM((ABS(CAST(amountx18 AS NUMERIC)) * CAST(pricex18 AS NUMERIC)) / 1e18) AS TEXT), '0') as volume
		FROM 
			fill_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND subaccount_id != ?
			AND subaccount_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			market_id
	`, startOfPeriod, endOfPeriod, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&productData).Error

	if err != nil {
		xlog.Errorf("Error querying product volumes for the last 24 hours: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	// Initialize the total volume
	totalVolume := ctypes.NewBigInt(big.NewInt(0))

	// Populate the productVolumes map with the queried data and calculate the total volume
	for _, p := range productData {
		productVolumes[p.MarketId] = p.Volume

		// Add the current product volume to the total volume
		totalVolume = totalVolume.Add(p.Volume)
	}

	return productVolumes, totalVolume, nil
}

// fetchUserDashboardData is a helper function to fetch user dashboard data from database
func (*FillDB) fetchUserDashboardData(brokerId uint, limit, offset int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := db.Raw(`
		WITH user_stats AS (
			SELECT 
				ft.subaccount_id                         AS sub_account_id,
				SPLIT_PART(ft.subaccount_id, '_', 2)     AS user_address,
				COUNT(*)                                  AS total_trades,
				SUM(ABS((ft.amountx18::numeric * ft.pricex18::numeric) / 1e36))                      AS total_volume_usd,
				SUM(ft.realized_pnlx18::numeric) / 1e18   AS total_net_pnl_usd,
				SUM(CASE WHEN ft.realized_pnlx18::numeric > 0 THEN 1 ELSE 0 END) AS winning_trades,
				AVG(ABS((ft.amountx18::numeric * ft.pricex18::numeric) / 1e36))                      AS avg_trade_size,
				MODE() WITHIN GROUP (ORDER BY ft.market_id) AS favorite_product_id,
				COUNT(DISTINCT DATE(ft.created_at))       AS active_days,
				MAX(ft.created_at)                         AS last_trade,
				MIN(ft.created_at)                         AS first_trade
			FROM fill_tables ft
			WHERE ft.broker_id = ?
			GROUP BY ft.subaccount_id
		)
		SELECT 
			us.sub_account_id,
			us.user_address,
			us.total_trades,
			us.total_volume_usd,
			us.total_net_pnl_usd,
			us.winning_trades,
			CASE 
				WHEN us.total_trades > 0 THEN (us.winning_trades::DECIMAL / us.total_trades::DECIMAL * 100)
				ELSE 0 
			END AS win_rate,
			us.avg_trade_size,
			us.favorite_product_id,
			us.active_days,
			us.last_trade,
			us.first_trade,
			EXTRACT(EPOCH FROM (NOW() - us.first_trade)) / 86400 AS account_age_days
		FROM user_stats us
		ORDER BY us.total_volume_usd DESC
		LIMIT ? OFFSET ?
	`, brokerId, limit, offset).Scan(&results).Error

	if err != nil {
		xlog.Errorf("Error fetching user dashboard data: %v", err)
		return nil, err
	}
	return results, nil
}

// GetUserDashboardData returns user dashboard data with pagination
func (*FillDB) GetUserDashboardData(limit, offset int) ([]map[string]interface{}, int64, error) {
	redisClient := xredis.GetRedisClient()
	ctx := context.Background()

	cacheKey := fmt.Sprintf("%s%d_%d", USER_DASHBOARD_DATA_KEY, limit, offset)

	// Try cache first
	cachedData, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var results []map[string]interface{}
		if err := json.Unmarshal([]byte(cachedData), &results); err == nil {
			return results, int64(len(results)), nil
		}
	}
	brokerId := uint(2)

	results, err := (*FillDB)(nil).fetchUserDashboardData(brokerId, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	if len(results) > 0 {
		if dataJSON, err := json.Marshal(results); err == nil {
			redisClient.Set(ctx, cacheKey, dataJSON, CACHE_EXPIRY_DURATION)
		} else {
			xlog.Errorf("Error marshaling data for cache: %v", err)
		}
	}

	return results, int64(len(results)), nil
}

// fetchUserDashboardByAddress is a helper function to fetch user dashboard data by address
func (*FillDB) fetchUserDashboardByAddress(userAddress string) (map[string]interface{}, error) {
	brokerId := uint(2)
	subaccountId := fmt.Sprintf("2_%s_1", userAddress)

	var result map[string]interface{}
	err := db.Raw(`
		WITH user_stats AS (
			SELECT 
				f.subaccount_id                                                AS sub_account_id,
				COUNT(*)                                                       AS total_trades,
				SUM(ABS((f.amountx18::numeric * f.pricex18::numeric) / 1e36))  AS total_volume_usd,
				SUM(f.realized_pnlx18::numeric) / 1e18                         AS total_net_pnl_usd,
				SUM(CASE WHEN f.realized_pnlx18::numeric > 0 THEN 1 ELSE 0 END) AS winning_trades,
				AVG(ABS((f.amountx18::numeric * f.pricex18::numeric) / 1e36))  AS avg_trade_size,
				MODE() WITHIN GROUP (ORDER BY f.market_id)                     AS favorite_product_id,
				COUNT(DISTINCT DATE(f.created_at))                             AS active_days,
				MAX(f.created_at)                                              AS last_trade,
				MIN(f.created_at)                                              AS first_trade
			FROM fill_tables f
			WHERE f.subaccount_id = ?
			  AND f.broker_id   = ?
			GROUP BY f.subaccount_id
		)
		SELECT 
			SPLIT_PART(us.sub_account_id, '_', 2)                             AS user_address,
			us.total_trades,
			us.total_volume_usd,
			us.total_net_pnl_usd,
			us.winning_trades,
			CASE 
				WHEN us.total_trades > 0 THEN (us.winning_trades::DECIMAL / us.total_trades::DECIMAL * 100)
				ELSE 0 
			END AS win_rate,
			us.avg_trade_size,
			us.favorite_product_id,
			us.active_days,
			us.last_trade,
			us.first_trade,
			EXTRACT(EPOCH FROM (NOW() - us.first_trade)) / 86400 AS account_age_days
		FROM user_stats us
	`, subaccountId, brokerId).Scan(&result).Error

	return result, err
}

// GetUserDashboardByAddress returns user dashboard data for a specific address
func (*FillDB) GetUserDashboardByAddress(userAddress string) (map[string]interface{}, error) {
	redisClient := xredis.GetRedisClient()
	ctx := context.Background()

	// Create cache key for this specific address
	cacheKey := fmt.Sprintf("%s%s", USER_DASHBOARD_BY_ADDRESS_KEY, userAddress)

	// Try to get from cache first
	cachedData, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return result, nil
		}
	}

	// Fetch user dashboard data
	result, err := (*FillDB)(nil).fetchUserDashboardByAddress(userAddress)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if len(result) > 0 {
		if dataJSON, err := json.Marshal(result); err == nil {
			redisClient.Set(ctx, cacheKey, dataJSON, CACHE_EXPIRY_DURATION)
		} else {
			xlog.Errorf("Error marshaling data for cache: %v", err)
		}
	}

	return result, nil
}

type FundingResult struct {
	FundingFee string `gorm:"column:funding_fee"`
}

// GetFundingFeeBySubaccountId returns funding fee for a specific subaccount
// NOTE: Uses fill_order_tables for dates before cutoff date, fill_tables for dates after cutoff date
func (*FillDB) GetFundingFeeBySubaccountId(subaccountId string, brokerId uint) (string, error) {
	subaccountIDStr, err := cutils.HackySubaccountHexToId(subaccountId)
	if err != nil {
		return "0", err
	}
	cutoffDate := FundingFeesMigrationCutoffDate

	var totalFee string
	err = db.Raw(`
		SELECT COALESCE(
			(SELECT COALESCE(SUM(CAST(funding_fees AS NUMERIC))/1e18, 0) 
			 FROM fill_order_tables
			 WHERE sub_account_id = ? AND broker_id = ? AND created_at < ?) +
			(SELECT COALESCE(SUM(CAST(funding_feesx18 AS NUMERIC))/1e18, 0)
			 FROM fill_tables
			 WHERE subaccount_id = ? AND broker_id = ? AND created_at >= ?),
			0
		) as funding_fee
	`, subaccountId, brokerId, cutoffDate, subaccountIDStr, brokerId, cutoffDate).Row().Scan(&totalFee)
	if err != nil && err != sql.ErrNoRows {
		return "0", err
	}
	return totalFee, nil
}

// GetDailyPnlGraph returns daily PnL data for a subaccount
func (*FillDB) GetDailyPnlGraph(subaccountId string, brokerId uint) ([]map[string]interface{}, error) {
	subaccountIDStr, err := cutils.HackySubaccountHexToId(subaccountId)
	if err != nil {
		return nil, err
	}

	var results []struct {
		Date        time.Time `gorm:"column:date"`
		RealizedPnL string    `gorm:"column:realized_pnl"`
	}

	err = db.Raw(`
		SELECT 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
			COALESCE(SUM(CAST(realized_pnlx18 AS NUMERIC))/1e18, 0) AS realized_pnl
		FROM fill_tables
		WHERE subaccount_id = ? AND broker_id = ?
		GROUP BY date
		ORDER BY date ASC
	`, subaccountIDStr, brokerId).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	var graphData []map[string]interface{}

	for _, row := range results {
		pnlFloat, err := strconv.ParseFloat(row.RealizedPnL, 64)
		if err != nil {
			fmt.Printf("Error parsing RealizedPnL: %v\n", err)
			pnlFloat = 0 // fallback
		}
		graphData = append(graphData, map[string]interface{}{
			"date":         row.Date.Format("02-01-2006"),
			"realized_pnl": fmt.Sprintf("%.18f", pnlFloat),
		})
	}
	return graphData, nil
}

// Get24hRealisedPnlPerSubaccount returns 24h realized PnL per subaccount
func (*FillDB) Get24hRealisedPnlPerSubaccount() (map[string]*big.Int, error) {
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	endOfPeriod := now.Format("2006-01-02 15:04:05")

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	type Result struct {
		SubaccountID string
		RealisedPnl  string
	}
	var results []Result

	err := db.Raw(`
		SELECT subaccount_id, SUM(CAST(realized_pnlx18 AS NUMERIC)) AS realised_pnl
		FROM fill_tables
		WHERE created_at BETWEEN ? AND ? AND subaccount_id NOT IN (?)
		GROUP BY subaccount_id
	`, startOfPeriod, endOfPeriod, excludeSubaccountIDs).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	out := make(map[string]*big.Int)
	for _, r := range results {
		val := new(big.Int)
		val.SetString(r.RealisedPnl, 10)
		out[r.SubaccountID] = val
	}
	return out, nil
}

// GetTotalRealisedPnL returns total realized PnL by product and overall
func (*FillDB) GetTotalRealisedPnL(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	type PnlData struct {
		MarketId    uint
		RealisedPnl ctypes.BigInt
	}

	var ammPnlData, internalAccountPnlData []PnlData

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err1 := db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(realized_pnlx18 AS NUMERIC)) as realised_pnl
		FROM 
			fill_tables
		WHERE 
			subaccount_id = ? AND broker_id = ?
			AND created_at > '2024-08-14 15:52:43'
		GROUP BY 
			market_id
	`, ammSubaccountID, brokerId).Scan(&ammPnlData).Error

	if err1 != nil {
		xlog.Errorf("Error while fetching total realised PnL for AMM subaccount: %v", err1)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err1
	}

	err2 := db.Raw(`
		SELECT 
			market_id, 
			SUM(CAST(realized_pnlx18 AS NUMERIC)) as realised_pnl
		FROM 
			fill_tables
		WHERE 
			subaccount_id IN (?) AND broker_id = ?
			AND created_at > '2024-08-14 15:52:43'
		GROUP BY 
			market_id
	`, internalSubaccountIDs, brokerId).Scan(&internalAccountPnlData).Error

	if err2 != nil {
		xlog.Errorf("Error while fetching total realised PnL %v", err2)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err2
	}

	// Create a map for internalAccountPnlData for fast lookup
	internalPnLMap := make(map[uint]ctypes.BigInt)
	for _, pnl := range internalAccountPnlData {
		internalPnLMap[pnl.MarketId] = pnl.RealisedPnl
	}

	totalRealizedPnL := ctypes.NewBigInt(big.NewInt(0))
	pnLPerToken := make(map[uint]ctypes.BigInt)
	for _, pnl := range ammPnlData {
		marketId := pnl.MarketId
		ammProductPnl := pnl.RealisedPnl

		// Check if the marketId exists in internalAccountPnlData
		if internalProductPnl, exists := internalPnLMap[marketId]; exists {
			// If it exists, add the AMM and internal account PnL
			finalPnl := ammProductPnl.Add(internalProductPnl)
			pnLPerToken[marketId] = finalPnl
			totalRealizedPnL = totalRealizedPnL.Add(finalPnl)
		} else {
			// If it doesn't exist, just add the AMM PnL
			pnLPerToken[marketId] = ammProductPnl
			totalRealizedPnL = totalRealizedPnL.Add(ammProductPnl)
		}
	}

	return pnLPerToken, totalRealizedPnL, nil
}

// GetPerpUserStatistics returns perp user statistics
func (*FillDB) GetPerpUserStatistics(brokerId uint) (map[string]interface{}, error) {
	// Collect exclusion subaccount IDs from environment variables
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Combine the AMM and internal subaccounts
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	// Variables to store results
	var last24hUsers int
	var totalUsers int

	// Query to get the number of unique users in the last 24 hours
	err := db.Raw(`
		SELECT COUNT(DISTINCT subaccount_id)
		FROM fill_tables
		WHERE subaccount_id NOT IN (?) AND broker_id = ?
		AND created_at >= NOW() - INTERVAL '24 hours'
	`, excludeSubaccountIDs, brokerId).Scan(&last24hUsers).Error

	if err != nil {
		return nil, fmt.Errorf("error querying 24-hour users: %v", err)
	}

	// Query to get the total number of unique users
	err = db.Raw(`
		SELECT COUNT(DISTINCT subaccount_id)
		FROM fill_tables
		WHERE subaccount_id NOT IN (?) AND broker_id = ?
	`, excludeSubaccountIDs, brokerId).Scan(&totalUsers).Error

	if err != nil {
		return nil, fmt.Errorf("error querying total users: %v", err)
	}

	// Return results as a map
	return map[string]interface{}{
		"24h_users":   last24hUsers,
		"total_users": totalUsers,
	}, nil
}

// GetTotalFeesIncludingInternal returns total fees including internal accounts
func (*FillDB) GetTotalFeesIncludingInternal(brokerId uint) (map[string]string, error) {
	// Get AMM subaccount ID and deposit skip subaccounts
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	depositSkipSubaccountIDs := convertHexSubaccountIDs(os.Getenv("DEPOSIT_SUBACCOUNT_SKIP"))

	// Combine all subaccount IDs to exclude
	excludeSubaccountIDs := append([]string{ammSubaccountID}, depositSkipSubaccountIDs...)

	// Struct to hold our results
	var result struct {
		TotalTradingFee  ctypes.BigInt `json:"total_trading_fee"`
		TotalFundingFee  ctypes.BigInt `json:"total_funding_fee"`
		TotalRealisedPnl ctypes.BigInt `json:"total_realised_pnl"`
	}

	ammSubaccountIDForFillOrder := contractUtils.AMM_SUBACCOUNT_ID
	depositSkipSubaccountIDsHex := cutils.GetTrimmedSplitEnv("DEPOSIT_SUBACCOUNT_SKIP", ",")
	excludeSubaccountIDsHex := append([]string{ammSubaccountIDForFillOrder}, depositSkipSubaccountIDsHex...)
	cutoffDate := FundingFeesMigrationCutoffDate

	// Query both tables: fill_order_tables before cutoff, fill_tables after cutoff
	var oldResult, newResult struct {
		TotalTradingFee  ctypes.BigInt
		TotalFundingFee  ctypes.BigInt
		TotalRealisedPnl ctypes.BigInt
	}

	// Query fill_order_tables for period before cutoff
	err := db.Raw(`
		SELECT 
			COALESCE(SUM(CAST(fee_amount AS NUMERIC)), 0) as total_trading_fee,
			COALESCE(SUM(CAST(funding_fees AS NUMERIC)), 0) as total_funding_fee,
			COALESCE(SUM(CAST(realised_pnl AS NUMERIC)), 0) as total_realised_pnl
		FROM fill_order_tables
		WHERE sub_account_id NOT IN (?) AND broker_id = ?
		  AND created_at > '2024-08-14 15:52:43'
		  AND created_at < ?
	`, excludeSubaccountIDsHex, brokerId, cutoffDate).Scan(&oldResult).Error
	if err != nil {
		xlog.Errorf("Error fetching total fees and PnL from fill_order_tables: %v", err)
		return nil, err
	}

	// Query fill_tables for period after cutoff
	err = db.Raw(`
		SELECT 
			COALESCE(SUM(CAST(feex18 AS NUMERIC)), 0) as total_trading_fee,
			COALESCE(SUM(CAST(funding_feesx18 AS NUMERIC)), 0) as total_funding_fee,
			COALESCE(SUM(CAST(realized_pnlx18 AS NUMERIC)), 0) as total_realised_pnl
		FROM fill_tables
		WHERE subaccount_id NOT IN (?) AND broker_id = ?
		  AND created_at >= ?
	`, excludeSubaccountIDs, brokerId, cutoffDate).Scan(&newResult).Error
	if err != nil {
		xlog.Errorf("Error fetching total fees and PnL from fill_tables: %v", err)
		return nil, err
	}

	// Combine results
	result.TotalTradingFee = oldResult.TotalTradingFee.Add(newResult.TotalTradingFee)
	result.TotalFundingFee = oldResult.TotalFundingFee.Add(newResult.TotalFundingFee)
	result.TotalRealisedPnl = oldResult.TotalRealisedPnl.Add(newResult.TotalRealisedPnl)

	if err != nil {
		xlog.Errorf("Error fetching total fees and PnL: %v", err)
		return nil, err
	}

	// Construct the response
	response := map[string]string{
		"total_trading_fee":  result.TotalTradingFee.String(),
		"total_funding_fee":  result.TotalFundingFee.String(),
		"total_realised_pnl": result.TotalRealisedPnl.String(),
	}

	return response, nil
}

// GetPreviousMonthStats returns previous month statistics
func (*FillDB) GetPreviousMonthStats() (map[string]string, error) {
	now := time.Now().UTC()
	firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfPreviousMonth := firstDayOfCurrentMonth.Add(-time.Second)
	firstDayOfPreviousMonth := time.Date(lastDayOfPreviousMonth.Year(), lastDayOfPreviousMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	excludeSubaccountIDs := cutils.GetTrimmedSplitEnv("EXCLUDED_SUBACCOUNT_IDS", ",")

	type FeeStats struct {
		TradingFees     string `json:"trading_fees"`
		LiquidationFees string `json:"liquidation_fees"`
	}

	// Build trading/liquidation fees query with proper NOT IN clause
	tradingQuery := `
		SELECT
		  COALESCE(SUM(CASE WHEN type != 'LIQUIDATION' THEN CAST(feex18 AS NUMERIC) END)/1e18, 0) AS trading_fees,
		  COALESCE(SUM(CASE WHEN type = 'LIQUIDATION' THEN CAST(feex18 AS NUMERIC) END)/1e18, 0) AS liquidation_fees
		FROM fill_tables
		WHERE created_at >= ? AND created_at <= ?`

	// Add NOT IN clause only if there are excluded IDs
	if len(excludeSubaccountIDs) > 0 {
		tradingQuery += ` AND subaccount_id NOT IN (`
		for i := range excludeSubaccountIDs {
			if i > 0 {
				tradingQuery += ","
			}
			tradingQuery += "?"
		}
		tradingQuery += ")"
	}

	// Build parameters for trading query
	tradingParams := []interface{}{firstDayOfPreviousMonth, lastDayOfPreviousMonth}
	for _, id := range excludeSubaccountIDs {
		tradingParams = append(tradingParams, id)
	}

	var feeStats FeeStats
	err := db.Raw(tradingQuery, tradingParams...).Scan(&feeStats).Error
	if err != nil {
		xlog.Errorf("Error fetching previous month trading/liquidation fees: %v", err)
		return nil, err
	}

	cutoffDate := FundingFeesMigrationCutoffDate
	fundingFeeExcludeIDsHex := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")
	fundingFeeExcludeIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	var oldFundingFees, newFundingFees, fundingFees string

	if lastDayOfPreviousMonth.Before(cutoffDate) {
		fundingQuery := `
			SELECT COALESCE(SUM(CAST(funding_fees AS NUMERIC))/1e18, 0) as funding_fees
			FROM fill_order_tables
			WHERE created_at >= ? AND created_at <= ?`

		if len(fundingFeeExcludeIDsHex) > 0 {
			fundingQuery += ` AND sub_account_id NOT IN (`
			for i := range fundingFeeExcludeIDsHex {
				if i > 0 {
					fundingQuery += ","
				}
				fundingQuery += "?"
			}
			fundingQuery += ")"
		}

		fundingParams := []interface{}{firstDayOfPreviousMonth, lastDayOfPreviousMonth}
		for _, id := range fundingFeeExcludeIDsHex {
			fundingParams = append(fundingParams, id)
		}

		if err = db.Raw(fundingQuery, fundingParams...).Row().Scan(&oldFundingFees); err != nil {
			xlog.Errorf("Error fetching previous month funding fees from fill_order_tables: %v", err)
			return nil, err
		}
		fundingFees = oldFundingFees
	} else if !firstDayOfPreviousMonth.Before(cutoffDate) {
		fundingQuery := `
			SELECT COALESCE(SUM(CAST(funding_feesx18 AS NUMERIC))/1e18, 0) as funding_fees
			FROM fill_tables
			WHERE created_at >= ? AND created_at <= ?`

		// Add NOT IN clause only if there are excluded IDs
		if len(fundingFeeExcludeIDs) > 0 {
			fundingQuery += ` AND subaccount_id NOT IN (`
			for i := range fundingFeeExcludeIDs {
				if i > 0 {
					fundingQuery += ","
				}
				fundingQuery += "?"
			}
			fundingQuery += ")"
		}

		fundingParams := []interface{}{firstDayOfPreviousMonth, lastDayOfPreviousMonth}
		for _, id := range fundingFeeExcludeIDs {
			fundingParams = append(fundingParams, id)
		}

		if err = db.Raw(fundingQuery, fundingParams...).Row().Scan(&newFundingFees); err != nil {
			xlog.Errorf("Error fetching previous month funding fees from fill_tables: %v", err)
			return nil, err
		}
		fundingFees = newFundingFees
	} else {
		fundingQueryOld := `
			SELECT COALESCE(SUM(CAST(funding_fees AS NUMERIC))/1e18, 0) as funding_fees
			FROM fill_order_tables
			WHERE created_at >= ? AND created_at < ?`

		if len(fundingFeeExcludeIDsHex) > 0 {
			fundingQueryOld += ` AND sub_account_id NOT IN (`
			for i := range fundingFeeExcludeIDsHex {
				if i > 0 {
					fundingQueryOld += ","
				}
				fundingQueryOld += "?"
			}
			fundingQueryOld += ")"
		}

		fundingParamsOld := []interface{}{firstDayOfPreviousMonth, cutoffDate}
		for _, id := range fundingFeeExcludeIDsHex {
			fundingParamsOld = append(fundingParamsOld, id)
		}

		if err = db.Raw(fundingQueryOld, fundingParamsOld...).Row().Scan(&oldFundingFees); err != nil {
			xlog.Errorf("Error fetching previous month funding fees from fill_order_tables: %v", err)
			return nil, err
		}

		fundingQueryNew := `
			SELECT COALESCE(SUM(CAST(funding_feesx18 AS NUMERIC))/1e18, 0) as funding_fees
			FROM fill_tables
			WHERE created_at >= ? AND created_at <= ?`

		if len(fundingFeeExcludeIDs) > 0 {
			fundingQueryNew += ` AND subaccount_id NOT IN (`
			for i := range fundingFeeExcludeIDs {
				if i > 0 {
					fundingQueryNew += ","
				}
				fundingQueryNew += "?"
			}
			fundingQueryNew += ")"
		}

		fundingParamsNew := []interface{}{cutoffDate, lastDayOfPreviousMonth}
		for _, id := range fundingFeeExcludeIDs {
			fundingParamsNew = append(fundingParamsNew, id)
		}

		if err = db.Raw(fundingQueryNew, fundingParamsNew...).Row().Scan(&newFundingFees); err != nil {
			xlog.Errorf("Error fetching previous month funding fees from fill_tables: %v", err)
			return nil, err
		}
		fundingFees = cutils.X18ToFloatStr(new(big.Int).Add(cutils.FloatStrToX18(oldFundingFees), cutils.FloatStrToX18(newFundingFees)))
	}

	ammExcludeIDs := cutils.GetTrimmedSplitEnv("AMM_EXCLUDED_SUBACCOUNT_IDS", ",")

	ammQuery := `
		SELECT COALESCE(SUM(CAST(realized_pnlx18 AS NUMERIC))/1e18, 0) as amm_pnl
		FROM fill_tables
		WHERE created_at >= ? AND created_at <= ?`

	// Add NOT IN clause only if there are excluded IDs
	if len(ammExcludeIDs) > 0 {
		ammQuery += ` AND subaccount_id NOT IN (`
		for i := range ammExcludeIDs {
			if i > 0 {
				ammQuery += ","
			}
			ammQuery += "?"
		}
		ammQuery += ")"
	}

	// Build parameters for AMM query
	ammParams := []interface{}{firstDayOfPreviousMonth, lastDayOfPreviousMonth}
	for _, id := range ammExcludeIDs {
		ammParams = append(ammParams, id)
	}

	var ammPnL string
	err = db.Raw(ammQuery, ammParams...).Row().Scan(&ammPnL)
	if err != nil {
		xlog.Errorf("Error fetching previous month AMM PnL: %v", err)
		return nil, err
	}

	return map[string]string{
		"trading_fees":     feeStats.TradingFees,
		"funding_fees":     cutils.X18ToFloatStr(cutils.FloatStrToX18(fundingFees)),
		"liquidation_fees": feeStats.LiquidationFees,
		"amm_pnl":          ammPnL,
		"month":            firstDayOfPreviousMonth.Format("2006-01"),
		"start_date":       firstDayOfPreviousMonth.Format("2006-01-02"),
		"end_date":         lastDayOfPreviousMonth.Format("2006-01-02"),
	}, nil
}

// GetDailyRealizedPnL returns daily realized PnL for a subaccount
func (*FillDB) GetDailyRealizedPnL(subaccountID string, specificDate time.Time) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	subaccountIDStr, err := cutils.HackySubaccountHexToId(subaccountID)
	if err != nil {
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	startOfDay := specificDate.UTC().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	var pnlData []struct {
		ProductID   uint          `gorm:"column:product_id"`
		RealisedPnl ctypes.BigInt `gorm:"column:realised_pnl"`
	}

	if err = db.Model(&FillTable{}).
		Select("market_id AS product_id, SUM(CAST(realized_pnlx18 AS NUMERIC)) AS realised_pnl").
		Where("subaccount_id = ? AND created_at BETWEEN ? AND ?", subaccountIDStr, startOfDay, endOfDay).
		Group("market_id").
		Scan(&pnlData).Error; err != nil {
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalRealizedPnL := ctypes.NewBigInt(big.NewInt(0))
	pnLPerToken := make(map[uint]ctypes.BigInt)

	for _, pnl := range pnlData {
		if current, ok := pnLPerToken[pnl.ProductID]; ok {
			pnLPerToken[pnl.ProductID] = current.Add(pnl.RealisedPnl)
		} else {
			pnLPerToken[pnl.ProductID] = pnl.RealisedPnl
		}
		totalRealizedPnL = totalRealizedPnL.Add(pnl.RealisedPnl)
	}

	return pnLPerToken, totalRealizedPnL, nil
}
