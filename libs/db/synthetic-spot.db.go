package db

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"os"
	"time"
)

type SyntheticSpotDB struct{}

type CombinedSyntheticSpotStats struct {
	ProductID uint32 `gorm:"column:product_id" json:"product_id"`

	// Total USD volume =
	//    SUM( IF is_buy THEN amount ELSE quote_delta END )
	// (i.e., if buying, 'amount' is the USD spent; if selling, 'quote_delta' is the USD received)
	TotalVolume ctypes.BigInt `gorm:"column:total_volume" json:"total_volume"`

	// Net open positions (tokens) per product =
	//    SUM( IF is_buy THEN quote_delta ELSE -amount END )
	// (i.e., if buying, user adds tokens; if selling, user reduces tokens)
	NetOpenPositions ctypes.BigInt `gorm:"column:net_open_positions" json:"net_open_positions"`

	// Total tokens sold volume in USD =
	//    SUM( IF is_buy = false THEN quote_delta ELSE 0 END )
	// (i.e., only for sells, 'quote_delta' is the USD the user received)
	SoldVolumeUSD ctypes.BigInt `gorm:"column:sold_volume_usd" json:"sold_volume_usd"`

	// Total tokens bought volume in USD =
	//    SUM( IF is_buy = true THEN amount ELSE 0 END )
	// (i.e., only for buys, 'amount' is the USD the user spent)
	BoughtVolumeUSD ctypes.BigInt `gorm:"column:bought_volume_usd" json:"bought_volume_usd"`

	// Sum of all fees for this product:
	TotalFees ctypes.BigInt `gorm:"column:total_fees" json:"total_fees"`

	// Number of distinct users (subaccounts) for this product:
	UniqueUsers int `gorm:"column:unique_users" json:"unique_users"`

	Volume24H      ctypes.BigInt `gorm:"column:volume_24h" json:"volume_24h"`             // 24-hour volume (USD)
	Fees24H        ctypes.BigInt `gorm:"column:fees_24h" json:"fees_24h"`                 // 24-hour fees
	UniqueUsers24H int           `gorm:"column:unique_users_24h" json:"unique_users_24h"` // 24-hour distinct subaccounts

	NetPnl ctypes.BigInt `json:"net_pnl"`
}

type OverallSyntheticSpotStats struct {
	// Sum of all volumes (for buys, use amount; for sells, use quote_delta)
	TotalVolume ctypes.BigInt `gorm:"column:total_volume" json:"total_volume"` // or use ctypes.BigInt

	// Sum of all fees
	TotalFees ctypes.BigInt `gorm:"column:total_fees" json:"total_fees"` // or use ctypes.BigInt

	// Count of distinct subaccount_id values
	UniqueUsers int `gorm:"column:unique_users" json:"unique_users"`

	// 24-hour volume (USD)
	Volume24H ctypes.BigInt `gorm:"column:volume_24h" json:"volume_24h"`

	// 24-hour fees
	Fees24H ctypes.BigInt `gorm:"column:fees_24h" json:"fees_24h"`

	// 24-hour unique subaccounts
	UniqueUsers24H int `gorm:"column:unique_users_24h" json:"unique_users_24h"`
}

// Insert synthetic spot order
func (*SyntheticSpotDB) InsertSyntheticSpotOrder(syntheticSpotUserOrder SyntheticSpotUserTable) (*SyntheticSpotUserTable, error) {
	if err := db.Create(&syntheticSpotUserOrder).Error; err != nil {
		return nil, err
	}
	return &syntheticSpotUserOrder, nil
}

// Get user history
func (*SyntheticSpotDB) GetUserHistory(subaccountID string) ([]SyntheticSpotUserTable, error) {
	var syntheticSpots []SyntheticSpotUserTable

	if err := db.Where("subaccount_id = ?", subaccountID).Find(&syntheticSpots).Error; err != nil {
		return nil, err
	}

	return syntheticSpots, nil
}

// GetUserHistorySorted retrieves all synthetic spot trades for a user sorted by timestamp (descending)
func (*SyntheticSpotDB) GetUserHistorySorted(subaccountID string) ([]SyntheticSpotUserTable, error) {
	var syntheticSpots []SyntheticSpotUserTable

	if err := db.Where("subaccount_id = ?", subaccountID).
		Order("created_at DESC").
		Find(&syntheticSpots).Error; err != nil {
		return nil, err
	}

	return syntheticSpots, nil
}

// GetSyntheticSpotStatsByProduct aggregates and returns, per product ID:
// 1) total volume in USD
// 2) net open positions (tokens)
// 3) total sold volume in USD
// 4) total bought volume in USD
// 5) total fees
// 6) number of unique users
func (*SyntheticSpotDB) GetSyntheticSpotStatsByProduct() ([]CombinedSyntheticSpotStats, error) {
	var results []CombinedSyntheticSpotStats

	// Fetch exclusion pre_market_id from environment variables
	premarketSubaccountId := os.Getenv("PRE_MARKET_SUBACCOUNT_ID")

	query := db.Table("synthetic_spot_user_tables").Where("subaccount_id != ?", premarketSubaccountId).
		Select(`
            product_id,
            SUM(
                CASE
                    WHEN is_buy = TRUE THEN CAST(amount as numeric)
                    ELSE CAST(quote_delta as numeric)
                END
            ) AS total_volume,
            SUM(
                CASE
                    WHEN is_buy = TRUE THEN CAST(quote_delta as numeric)
                    ELSE -CAST(amount as numeric)
                END
            ) AS net_open_positions,
            SUM(
                CASE
                    WHEN is_buy = FALSE THEN CAST(quote_delta as numeric)
                    ELSE 0
                END
            ) AS sold_volume_usd,
            SUM(
                CASE
                    WHEN is_buy = TRUE THEN CAST(amount as numeric)
                    ELSE 0
                END
            ) AS bought_volume_usd,
            SUM(CAST(fees as numeric)) AS total_fees,
            COUNT(DISTINCT subaccount_id) AS unique_users,

            SUM(
                CASE
                    WHEN created_at AT TIME ZONE 'UTC' >= NOW() - INTERVAL '24 hours' THEN
                        CASE 
                            WHEN is_buy = TRUE THEN CAST(amount AS numeric)
                            ELSE CAST(quote_delta AS numeric)
                        END
                    ELSE 0
                END
            ) AS volume_24h,

            SUM(
                CASE
                    WHEN created_at AT TIME ZONE 'UTC' >= NOW() - INTERVAL '24 hours' THEN CAST(fees AS numeric)
                    ELSE 0
                END
            ) AS fees_24h,

            COUNT(
                DISTINCT CASE
                    WHEN created_at AT TIME ZONE 'UTC' >= NOW() - INTERVAL '24 hours'
                    THEN subaccount_id
                END
            ) AS unique_users_24h
        `).
		Group("product_id")

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (*SyntheticSpotDB) GetOverallSyntheticSpotStats() (*OverallSyntheticSpotStats, error) {
	var result OverallSyntheticSpotStats

	// Replace "synthetic_spot_user_tables" with your actual table name, if different.
	query := db.Table("synthetic_spot_user_tables").
		Select(`
            SUM(
                CASE WHEN is_buy = TRUE THEN CAST(amount as numeric) 
                    ELSE CAST(quote_delta as numeric) 
                END
            ) AS total_volume,

            SUM(CAST(fees as numeric)) AS total_fees,

            COUNT(DISTINCT subaccount_id) AS unique_users,

            SUM(
                CASE 
                    WHEN created_at AT TIME ZONE 'UTC' >= NOW() - INTERVAL '24 hours' THEN
                        CASE WHEN is_buy = TRUE 
                            THEN CAST(amount AS numeric) 
                            ELSE CAST(quote_delta AS numeric) 
                        END
                    ELSE 0
                END
            ) AS volume_24h,

            SUM(
                CASE 
                    WHEN created_at AT TIME ZONE 'UTC' >= NOW() - INTERVAL '24 hours' 
                    THEN CAST(fees AS numeric) 
                    ELSE 0 
                END
            ) AS fees_24h,

            COUNT(
                DISTINCT CASE 
                    WHEN created_at AT TIME ZONE 'UTC' >= NOW() - INTERVAL '24 hours' 
                    THEN subaccount_id 
                END
            ) AS unique_users_24h
        `)

	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}

	return &result, nil
}

func (*SyntheticSpotDB) GetLast30DaysSyntheticSpotStats() (map[string]interface{}, error) {
	// Define a struct to hold the daily query results
	type DailyStats struct {
		Date        time.Time     `json:"date"`
		NumUsers    int           `json:"num_users"`
		TotalVolume ctypes.BigInt `json:"total_volume"`
	}

	var dailyStats []DailyStats

	// SQL query to fetch daily statistics for the last 30 days
	query := `
		SELECT
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') as date,
			COUNT(DISTINCT subaccount_id) AS num_users,
			SUM(
                CASE 
                    WHEN is_buy = TRUE THEN CAST(amount AS numeric) 
                    ELSE CAST(quote_delta AS numeric) 
                END
            ) AS total_volume
		FROM synthetic_spot_user_tables
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
