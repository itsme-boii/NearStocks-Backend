package db

import (
	"errors"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type PreMarketsDB struct{}

type PreMarketHolding struct {
	SubaccountID string        `json:"subaccount_id"`
	Holdings     ctypes.BigInt `json:"holdings"`
}

type OverallPreMarketStats struct {
	// Sum of all volumes (for buys, use amount; for sells, use quote_delta)
	TotalVolume ctypes.BigInt `gorm:"column:total_volume" json:"total_volume"` // or use ctypes.BigInt

	USDDeposited ctypes.BigInt `gorm:"column:usd_deposited" json:"usd_deposited"` // or use ctypes.BigInt

	// Sum of all fees
	TotalFees ctypes.BigInt `gorm:"column:total_fees" json:"total_fees"` // or use ctypes.BigInt

	// Count of distinct subaccount_id values
	UniqueUsers int `gorm:"column:unique_users" json:"unique_users"`

	// 24-hour volume (USD)
	Volume24H ctypes.BigInt `gorm:"column:volume_24h" json:"volume_24h"`

	// 24-hour usdc deposited
	USDDeposited24H ctypes.BigInt `gorm:"column:usd_deposited_24h" json:"usd_deposited_24h"`

	// 24-hour fees
	Fees24H ctypes.BigInt `gorm:"column:fees_24h" json:"fees_24h"`

	// 24-hour unique subaccounts
	UniqueUsers24H int `gorm:"column:unique_users_24h" json:"unique_users_24h"`
}

// AddPreMarket adds a new pre-market record to the database
func (*PreMarketsDB) AddPreMarket(preMarketData PreMarketTable) (*PreMarketTable, error) {
	if err := db.Create(&preMarketData).Error; err != nil {
		return nil, err
	}
	return &preMarketData, nil
}

func (*PreMarketsDB) InsertPreMarketOrder(preMarketUser PreMarketUserTable) (*PreMarketUserTable, error) {
	if err := db.Create(&preMarketUser).Error; err != nil {
		return nil, err
	}
	return &preMarketUser, nil
}

// IsProductEnabled checks if a product is enabled and returns a boolean
func (*PreMarketsDB) IsProductEnabled(productId uint32) (bool, int64, error) {
	var preMarket PreMarketTable

	if err := db.Select("is_enabled, closing_timestamp").Where("product_id = ?", productId).First(&preMarket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0, nil // Product not found
		}
		return false, 0, err // Other errors
	}

	return preMarket.IsEnabled, preMarket.ClosingTimestamp, nil
}

func (*PreMarketsDB) GetUserHistory(subaccountID string) ([]PreMarketUserTable, error) {
	var preMarkets []PreMarketUserTable

	if err := db.Where("subaccount_id = ?", subaccountID).Find(&preMarkets).Error; err != nil {
		return nil, err
	}

	return preMarkets, nil
}

func (*PreMarketsDB) GetMarketData() ([]PreMarketTable, error) {
	var preMarkets []PreMarketTable

	// Select only the required fields
	if err := db.Select("product_id, closing_timestamp, delivery_timestamp, is_enabled, details").
		Find(&preMarkets).Error; err != nil {
		return nil, err
	}

	return preMarkets, nil
}

func (*PreMarketsDB) CloseProduct(productId uint32, amount ctypes.BigInt) error {
	var product PreMarketTable

	// Retrieve the product from the database
	if err := db.Where("product_id = ?", productId).First(&product).Error; err != nil {
		return err
	}

	// Calculate to_be_delivered as max_supply - amount
	toBeDelivered := product.MaxSupply.Sub(amount) // Perform subtraction

	// Update the product's is_enabled and to_be_delivered fields
	result := db.Model(&PreMarketTable{}).Where("product_id = ?", productId).Updates(map[string]interface{}{
		"is_enabled":      false,
		"to_be_delivered": &toBeDelivered,
	})

	// Check for errors in the update operation
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Get all unique product IDs
func (*PreMarketsDB) GetUniqueProductIDs() ([]uint32, error) {
	var productIDs []uint32

	// Select only the product IDs where is_enabled is true
	if err := db.Model(&PreMarketTable{}).Where("is_enabled = ?", true).Select("product_id").Find(&productIDs).Error; err != nil {
		return nil, err
	}

	return productIDs, nil
}

func (*PreMarketsDB) GetProductWiseData() (map[uint32]map[string]string, error) {
	type MinimalProductTable struct {
		ProductId  uint32        `json:"product_id"`
		BuyUSDC    ctypes.BigInt `json:"buy_usdc"`
		SellUSDC   ctypes.BigInt `json:"sell_usdc"`
		BuyToken   ctypes.BigInt `json:"buy_token"`
		SellToken  ctypes.BigInt `json:"sell_token"`
		TotalFees  ctypes.BigInt `json:"total_fees"`  // sum of fees
		TotalUsers int           `json:"total_users"` // distinct user count
	}

	// Fetch exclusion pre_market_id from environment variables
	premarketSubaccountId := os.Getenv("PRE_MARKET_SUBACCOUNT_ID")

	// Query to fetch aggregated data
	query := `
	SELECT 
    product_id,
    COALESCE(SUM(CASE WHEN is_buy = true THEN CAST(amount AS numeric) END), 0) AS buy_usdc,
    COALESCE(SUM(CASE WHEN is_buy = false THEN CAST(quote_delta AS numeric) END), 0) AS sell_usdc,
    COALESCE(SUM(CASE WHEN is_buy = true THEN CAST(quote_delta AS numeric) END), 0) AS buy_token,
    COALESCE(SUM(CASE WHEN is_buy = false THEN CAST(amount AS numeric) END), 0) AS sell_token,
	COALESCE(SUM(CAST(fees AS numeric)), 0) AS total_fees,
    COUNT(DISTINCT subaccount_id) AS total_users
	FROM pre_market_user_tables
	WHERE subaccount_id != ?
	GROUP BY product_id
    `

	var productData []MinimalProductTable

	// Execute query
	err := db.Raw(query, premarketSubaccountId).Scan(&productData).Error
	if err != nil {
		return nil, err
	}

	// Define structured data map
	groupedData := make(map[uint32]map[string]string)

	xlog.Debugf("Product Data: %v", productData)

	for _, product := range productData {
		// Ensure total_buy_amount is not nil
		if product.BuyUSDC.Val == nil {
			xlog.Warnf("USDC Buy is nil for ProductId: %d", product.ProductId)
			continue
		}

		// Ensure total_sell_quote is not nil
		if product.SellUSDC.Val == nil {
			xlog.Warnf("USDC Sell is nil for ProductId: %d", product.ProductId)
			continue
		}

		// Ensure total_buy_quote is not nil
		if product.BuyToken.Val == nil {
			xlog.Warnf("Token Buy is nil for ProductId: %d", product.ProductId)
			continue
		}

		// Ensure total_sell_amount is not nil
		if product.SellToken.Val == nil {
			xlog.Warnf("Token Sell is nil for ProductId: %d", product.ProductId)
			continue
		}

		if product.TotalFees.Val == nil {
			// If there's a possibility that fees is never nil, you can skip.
			xlog.Warnf("Total Fees is nil for ProductId: %d", product.ProductId)
			continue
		}

		// Ensure the product_id group exists
		if _, productExists := groupedData[product.ProductId]; !productExists {
			groupedData[product.ProductId] = map[string]string{
				"usdc_buy":   "0",
				"usdc_sell":  "0",
				"token_buy":  "0",
				"token_sell": "0",
			}
		}

		// Compute required values
		totalVolume := new(big.Int).Add(product.BuyUSDC.Val, product.SellUSDC.Val)
		currentUSDCDeposit := new(big.Int).Sub(product.BuyUSDC.Val, product.SellUSDC.Val)
		currentTokensSold := new(big.Int).Sub(product.BuyToken.Val, product.SellToken.Val)

		// Compute average price safely (avoid division by zero)
		averagePrice := big.NewInt(0)
		if currentTokensSold.Cmp(big.NewInt(0)) != 0 {
			currentUSDCDepositX36 := new(big.Int).Mul(currentUSDCDeposit, big.NewInt(1e18))
			averagePrice = new(big.Int).Div(currentUSDCDepositX36, currentTokensSold)
		}

		// Store results in a structured map
		groupedData[product.ProductId] = map[string]string{
			"total_volume":         totalVolume.String(),
			"current_usdc_deposit": currentUSDCDeposit.String(),
			"current_tokens_sold":  currentTokensSold.String(),
			"average_price":        averagePrice.String(),
			"total_fees":           product.TotalFees.Val.String(),
			"total_users":          strconv.Itoa(product.TotalUsers),
		}
	}

	return groupedData, nil
}

func (*PreMarketsDB) GetProductWiseVolumeOnly() (map[uint32]map[string]string, error) {
	// Define a minimal struct to receive the query result
	type MinimalVolume struct {
		ProductId uint32        `json:"product_id"`
		Volume    ctypes.BigInt `json:"volume"`
	}

	// Query only computing total volume (Buy + Sell) in a single column
	query := `
        SELECT 
            product_id,
            COALESCE(
                SUM(
                    CASE WHEN is_buy = true 
                        THEN CAST(amount AS numeric)
                        ELSE CAST(quote_delta AS numeric)
                    END
                ), 0
            ) AS volume
        FROM pre_market_user_tables
        GROUP BY product_id
    `

	var rows []MinimalVolume
	// Perform the raw query
	if err := db.Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Build the final map: same shape as your existing GetProductWiseData()
	volumeMap := make(map[uint32]map[string]string, len(rows))
	for _, row := range rows {
		// If row.Volume.Val is nil, skip or log a warning
		if row.Volume.Val == nil {
			xlog.Warnf("Volume is nil for ProductId %d", row.ProductId)
			continue
		}

		volumeMap[row.ProductId] = map[string]string{
			"total_volume": row.Volume.Val.String(),
		}
	}

	return volumeMap, nil
}

func (*PreMarketsDB) GetHoldingsForProduct(productId uint32) ([]PreMarketHolding, error) {
	var holdings []PreMarketHolding

	// Fetch exclusion pre_market_id from environment variables
	premarketSubaccountId := os.Getenv("PRE_MARKET_SUBACCOUNT_ID")

	query := `
		SELECT subaccount_id, 
		       SUM(
		          CASE WHEN is_buy = 'f' THEN -CAST(amount AS numeric) 
		          ELSE CAST(quote_delta AS numeric) 
		          END
		       ) AS holdings
		FROM pre_market_user_tables 
		WHERE product_id = ?
		AND subaccount_id != ?
		GROUP BY subaccount_id 
		ORDER BY holdings DESC
	`

	if err := db.Raw(query, productId, premarketSubaccountId).Scan(&holdings).Error; err != nil {
		return nil, err
	}

	return holdings, nil
}

func (*PreMarketsDB) GetTokensSoldForProductForSpot(productId uint32) (big.Int, error) {
	var tokenSoldStr string

	// Fetch exclusion pre_market_id from environment variables
	premarketSubaccountId := os.Getenv("PRE_MARKET_SUBACCOUNT_ID")

	query := db.Table("pre_market_user_tables").
		Select(`
		SUM(
            CASE
            WHEN is_buy = TRUE THEN CAST(quote_delta as numeric)
            ELSE -CAST(amount as numeric)
            END
            ) AS net_tokens_sold
		`).Where("product_id = ? AND subaccount_id != ?", productId, premarketSubaccountId)

	if err := query.Scan(&tokenSoldStr).Error; err != nil {
		return *big.NewInt(0), err
	}

	tokenSold := new(big.Int)
	tokenSold.SetString(tokenSoldStr, 10)

	// Multiply with factor and then return
	factor := contractUtils.PRE_MARKET_TO_SPOT_FACTOR_MAP[productId]
	factorBigInt := new(big.Int).SetUint64(uint64(factor))
	tokenSold.Mul(tokenSold, factorBigInt)
	// divide by 100
	tokenSold.Div(tokenSold, big.NewInt(100))

	return *tokenSold, nil
}

func (*PreMarketsDB) GetOverAllPreMarketsStats() (*OverallPreMarketStats, error) {
	var result OverallPreMarketStats

	// Fetch exclusion pre_market_id from environment variables
	premarketSubaccountId := os.Getenv("PRE_MARKET_SUBACCOUNT_ID")

	query := db.Table("pre_market_user_tables").Where("subaccount_id != ?", premarketSubaccountId).
		Select(`
            SUM(
                CASE WHEN is_buy = TRUE THEN CAST(amount as numeric) 
                    ELSE CAST(quote_delta as numeric) 
                END
            ) AS total_volume,

			SUM(
                CASE
                    WHEN is_buy = TRUE THEN CAST(amount as numeric)
                    ELSE -CAST(quote_delta as numeric)
                END
            ) AS usd_deposited,

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
                    WHEN created_at AT TIME ZONE 'UTC' >= NOW() - INTERVAL '24 hours' THEN
                        CASE 
                            WHEN is_buy = TRUE THEN CAST(amount AS numeric)
                            ELSE -CAST(quote_delta AS numeric)
                        END
                    ELSE 0
                END
            ) AS usd_deposited_24h,

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

func (*PreMarketsDB) GetLast30DaysPreMarketStats() (map[string]interface{}, error) {
	type DailyStats struct {
		Date        time.Time     `json:"date"`
		NumUsers    int           `json:"num_users"`
		TotalVolume ctypes.BigInt `json:"total_volume"`
	}

	var dailyStats []DailyStats

	// Fetch exclusion pre_market_id from environment variables
	premarketSubaccountId := os.Getenv("PRE_MARKET_SUBACCOUNT_ID")

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
		FROM pre_market_user_tables
		WHERE created_at >= NOW() - INTERVAL '30 DAYS'
		AND subaccount_id != ?
		GROUP BY date
		ORDER BY date DESC
	`

	// Execute the query
	err := db.Raw(query, premarketSubaccountId).Scan(&dailyStats).Error
	if err != nil {
		return nil, err
	}

	// Prepare the response
	response := map[string]interface{}{
		"dailyStats": dailyStats,
	}

	return response, nil
}

func (*PreMarketsDB) GetFinalUsdDepositedForProduct(productId uint32) (big.Int, error) {
	var usdDepositedStr string

	// Fetch exclusion pre_market_id from environment variables
	premarketSubaccountId := os.Getenv("PRE_MARKET_SUBACCOUNT_ID")

	query := db.Table("pre_market_user_tables").
		Select(`
		SUM(
            CASE
            WHEN is_buy = TRUE THEN CAST(amount as numeric)
            ELSE -CAST(quote_delta as numeric)
            END
            ) AS net_usd_deposited
		`).Where("product_id = ? AND subaccount_id != ?", productId, premarketSubaccountId)

	if err := query.Scan(&usdDepositedStr).Error; err != nil {
		return *big.NewInt(0), err
	}

	usdDeposited := new(big.Int)
	usdDeposited.SetString(usdDepositedStr, 10)

	return *usdDeposited, nil
}

// Get24HourHighLowPrices retrieves the highest and lowest prices for each product in the last 24 hours
func (*PreMarketsDB) Get24HourHighLowPrices() (map[uint32]map[string]string, error) {
	// Calculate the timestamp for 24 hours ago in milliseconds
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour).UnixMilli()

	// Query to get all products
	var productIDs []uint32
	if err := db.Model(&PreMarketTable{}).Where("is_enabled = ?", true).Select("product_id").Find(&productIDs).Error; err != nil {
		return nil, err
	}

	// Initialize the result map
	result := make(map[uint32]map[string]string)

	// For each product, find the highest and lowest prices in the last 24 hours
	for _, productID := range productIDs {
		var highPrice, lowPrice, price24HoursAgo string
		resultMap := map[string]string{
			"high_price":    "0", // Default value
			"low_price":     "0", // Default value
			"price_24h_ago": "0", // Default value
		}

		// Query for the highest price
		highQuery := db.Model(&PreMarketCandleTable{}).
			Where("product_id = ? AND end_time >= ? AND interval = ?", productID, twentyFourHoursAgo, "1m").
			Select("MAX(CAST(high_price AS numeric)) as max_price")

		if err := highQuery.Row().Scan(&highPrice); err != nil {
			xlog.Errorf("Error querying high price for product %d: %v", productID, err)
		} else if highPrice != "" {
			resultMap["high_price"] = highPrice
		}

		// Query for the lowest price
		lowQuery := db.Model(&PreMarketCandleTable{}).
			Where("product_id = ? AND end_time >= ? AND interval = ?", productID, twentyFourHoursAgo, "1m").
			Select("MIN(CAST(low_price AS numeric)) as min_price")

		if err := lowQuery.Row().Scan(&lowPrice); err != nil {
			xlog.Errorf("Error querying low price for product %d: %v", productID, err)
		} else if lowPrice != "" {
			resultMap["low_price"] = lowPrice
		}

		// Query for the price closest to 24 hours ago
		priceQuery := db.Model(&PreMarketCandleTable{}).
			Where("product_id = ? AND interval = ?", productID, "1m").
			Order("ABS(end_time - " + strconv.FormatInt(twentyFourHoursAgo, 10) + ") ASC").
			Limit(1).
			Select("close_price")

		if err := priceQuery.Row().Scan(&price24HoursAgo); err != nil {
			xlog.Errorf("Error querying price 24 hours ago for product %d: %v", productID, err)
		} else if price24HoursAgo != "" {
			resultMap["price_24h_ago"] = price24HoursAgo
		}

		// Add the result map for this product
		result[productID] = resultMap
	}

	return result, nil
}

// GetUserHistorySorted retrieves all pre-market trades for a user sorted by timestamp (descending)
func (*PreMarketsDB) GetUserHistorySorted(subaccountID string) ([]PreMarketUserTable, error) {
	var preMarkets []PreMarketUserTable

	if err := db.Where("subaccount_id = ?", subaccountID).
		Order("created_at DESC").
		Find(&preMarkets).Error; err != nil {
		return nil, err
	}

	return preMarkets, nil
}
