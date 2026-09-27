package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

const (
	DateLayout           = "2006-01-02"
	HourlyDateTimeLayout = "2006-01-02 15:00"
)

// DEX2: USING BROKER IDs - supporting multiple brokers
var SUPPORTED_BROKER_IDS = []uint{1, 2}

// getBrokerSpecificStartDate returns the start date for data queries based on broker ID
func getBrokerSpecificStartDate(brokerId uint) time.Time {
	if brokerId == 2 {
		// For broker 2, start from June 1st 2025
		return time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	}
	// For other brokers, use past 30 days
	return time.Now().UTC().AddDate(0, 0, -30)
}

// Note - not setting distributed redis lock for dashboard since we are not expecting race conditions on these APIs
var redisClient = xredis.GetRedisClient()

func StartDashboardCron() {
	fmt.Println("Dashboard Cron job started")
	c := cron.New(cron.WithSeconds())
	// cron job runs every 10 minutes
	_, err := c.AddFunc("0 */10 * * * *", func() {
		fmt.Println("Dashboard Cron job running:", time.Now())
		updateStatsMetricAtNewday()
		for _, brokerId := range SUPPORTED_BROKER_IDS {
			updateDashboardStats(brokerId)
			updateDailyPnL(brokerId)
			updateDailyFees(brokerId)
			updateDailyVolumes(brokerId)
			updateDailyVolumesInternal()
			updateStatsCache(brokerId)
		}
		updateFundingRates()
	})
	if err != nil {
		fmt.Printf("Error adding cron function: %v\n", err)
	}

	// Start the cron scheduler
	c.Start()
	fmt.Println("Cron scheduler started")

	select {}
}
func fetchMaxLimit(key string, defaultLimit int64) int64 {
	// Fetch max limit from Redis or other reference source
	redisKey := fmt.Sprintf("USER_DATA_MAX_LIMIT_%s", key)
	limitStr, err := redisClient.Get(context.Background(), redisKey).Result()
	if err != nil {
		// Log error and use default limit if reference data is unavailable
		xlog.Warnf("Max limit for key %s not found in Redis, using default limit: %d", key, defaultLimit)
		return defaultLimit
	}

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		// Log error and use default limit if parsing fails
		xlog.Errorf("Error parsing max limit for key %s from Redis: %v. Using default limit: %d", key, err, defaultLimit)
		return defaultLimit
	}

	return limit
}

func updateSpecificKeysWithAdditionalLogic() {
	currentTime := time.Now().UTC()

	redisKeysWithExtraLogic := []string{
		"NEW_USERS_DAILY_DUMMY_DATA",
		"DAILY_ACTIVE_USERS_DUMMY_DATA",
	}

	// Default limits for keys
	defaultLimits := map[string]int64{
		"NEW_USERS_DAILY_DUMMY_DATA":    300,
		"DAILY_ACTIVE_USERS_DUMMY_DATA": 1500,
	}

	for _, key := range redisKeysWithExtraLogic {
		// Fetch the dynamic or default maximum limit
		maxLimit := fetchMaxLimit(key, defaultLimits[key])

		// Fetch data from Redis
		redisData, err := redisClient.Get(context.Background(), key).Result()
		if err != nil {
			xlog.Errorf("Error fetching data for key %s: %v", key, err)
			continue
		}

		var data []int64
		// Unmarshal the JSON data
		err = json.Unmarshal([]byte(redisData), &data)
		if err != nil {
			xlog.Errorf("Error unmarshalling data for key %s: %v", key, err)
			continue
		}

		if len(data) > 1 {
			randomValue := int64(0)

			// Generate random value for reference
			switch key {
			case "NEW_USERS_DAILY_DUMMY_DATA":
				randomValue = int64(rand.Intn(61) + 40) // Random value between 40-100
			case "DAILY_ACTIVE_USERS_DUMMY_DATA":
				randomValue = int64(rand.Intn(101) + 100) // Random value between 100-200
			}

			// Calculate minutes remaining in the day
			minutesRemaining := int64((24-currentTime.Hour())*60 - currentTime.Minute())

			// Enforce maximum limit with an allowance
			if data[0] < maxLimit {
				// Update data[0] using formula only if below maxLimit
				if minutesRemaining > 0 {
					data[0] += (data[1] + randomValue) / (minutesRemaining / 10)
				}
			} else if data[0] > maxLimit+100 {
				// Hard cap at maxLimit if it exceeds maxLimit + 100
				data[0] = maxLimit
			} else if data[0] > maxLimit {
				continue
			}
		}

		// Marshal and update the modified data back to Redis
		updatedData, err := json.Marshal(data)
		if err != nil {
			xlog.Errorf("Error marshalling updated data for key %s: %v", key, err)
			continue
		}

		err = redisClient.Set(context.Background(), key, updatedData, 0).Err()
		if err != nil {
			xlog.Errorf("Error updating data for key %s in Redis: %v", key, err)
		} else {
			xlog.Infof("Successfully applied additional logic and updated key %s in Redis.", key)
		}
	}

	xlog.Infof("Successfully updated specific Redis keys with additional logic.")
}

func updateStatsMetricAtNewday() {
	currentTime := time.Now().UTC()
	if currentTime.Hour() == 0 && currentTime.Minute() >= 0 && currentTime.Minute() <= 1 {

		redisKeys := []string{
			"DAILY_TRADES_DUMMY_DATA",
			"NEW_USERS_DAILY_DUMMY_DATA",
			"DAILY_ACTIVE_USERS_DUMMY_DATA",
		}

		for _, key := range redisKeys {
			// Fetch data from Redis
			redisData, err := redisClient.Get(context.Background(), key).Result()
			if err != nil {
				xlog.Errorf("Error fetching data for key %s: %v", key, err)
				continue
			}

			var data []int64
			// Unmarshal the JSON data
			err = json.Unmarshal([]byte(redisData), &data)
			if err != nil {
				xlog.Errorf("Error unmarshalling data for key %s: %v", key, err)
				continue
			}

			// Modify the data: remove last element and prepend 0
			if len(data) > 0 {
				data = append([]int64{0}, data[:len(data)-1]...)
			}

			// Marshal and update the modified data back to Redis
			updatedData, err := json.Marshal(data)
			if err != nil {
				xlog.Errorf("Error marshalling updated data for key %s: %v", key, err)
				continue
			}

			err = redisClient.Set(context.Background(), key, updatedData, 0).Err()
			if err != nil {
				xlog.Errorf("Error updating data for key %s in Redis: %v", key, err)
			} else {
				xlog.Infof("Successfully updated data for key %s in Redis.", key)
			}
		}

		xlog.Infof("Successfully updated Redis keys at midnight window.")
	}

	// Independent logic for additional processing
	updateSpecificKeysWithAdditionalLogic()
}

func updateDailyDataWithDummyData(redisKey string, dailyData []db.DailyData, brokerId uint) ([]db.DailyData, error) {
	var minValue, maxValue int64

	switch redisKey {
	case "NEW_USERS_DAILY_DUMMY_DATA":
		if brokerId == 2 {
			minValue = 10
			maxValue = 30
		} else {
			minValue = 80
			maxValue = 120
		}
	case "DAILY_ACTIVE_USERS_DUMMY_DATA":
		if brokerId == 2 {
			minValue = 14500
			maxValue = 15000
		} else {
			minValue = 500
			maxValue = 1000
		}
	case "DAILY_TRADES_DUMMY_DATA":
		if brokerId == 2 {
			minValue = 800
			maxValue = 1000
		} else {
			minValue = 20
			maxValue = 50
		}
	default:
		minValue = 10
		maxValue = 50
	}

	now := time.Now().UTC()

	for i := range dailyData {
		dayOffset := now.AddDate(0, 0, -i)
		dateKey := dayOffset.Format("2006-01-02")

		// Create date-specific cache key
		cacheKey := xredis.GetDummyDataCacheKey(redisKey, brokerId, dateKey)

		// Try to get cached dummy value for this specific date
		cachedValue, err := redisClient.Get(context.Background(), cacheKey).Result()
		if err == nil {
			// Use cached value
			if cachedDummyValue, parseErr := strconv.ParseInt(cachedValue, 10, 64); parseErr == nil {
				dailyData[i].Count += cachedDummyValue
				continue
			}
		}

		// Generate new dummy data for this specific date
		daySeed := dayOffset.Year()*10000 + int(dayOffset.Month())*100 + dayOffset.Day()
		dayRand := rand.New(rand.NewSource(int64(daySeed) + int64(brokerId)*1000))

		// Generate decreasing random values as we go further back in time
		factor := float64(len(dailyData)-i) / float64(len(dailyData))
		adjustedMin := int64(float64(minValue) * factor)
		adjustedMax := int64(float64(maxValue) * factor)

		if adjustedMin < 5 {
			adjustedMin = 5
		}
		if adjustedMax <= adjustedMin {
			adjustedMax = adjustedMin + 10
		}

		dummyValue := adjustedMin + dayRand.Int63n(adjustedMax-adjustedMin+1)

		// Weekend low activity
		if dayOffset.Weekday() == time.Saturday || dayOffset.Weekday() == time.Sunday {
			dummyValue = int64(float64(dummyValue) * 0.8)
		}

		// Cache this value permanently (no expiry)
		redisClient.Set(context.Background(), cacheKey, dummyValue, 0)

		// Add dummy data to dailyData
		dailyData[i].Count += dummyValue

		xlog.Debugf("Generated dummy data for %s (broker %d) date %s: %d",
			redisKey, brokerId, dateKey, dummyValue)
	}

	return dailyData, nil
}

func generateUser24hData(brokerId uint, dailyActiveUsers []db.DailyData) ([]db.DailyData, error) {
	if len(dailyActiveUsers) == 0 {
		return []db.DailyData{}, fmt.Errorf("no daily active users data available")
	}

	var reductionRange int64
	switch brokerId {
	case 2:
		reductionRange = 150 // 0-149 reduction for broker 2
	default:
		reductionRange = 100 // 0-99 reduction for other brokers
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	baseCount := dailyActiveUsers[0].Count
	reduction := r.Int63n(reductionRange)

	// Apply time-based variations
	currentTime := time.Now().UTC()
	hour := currentTime.Hour()

	// Peak hours (9 AM - 9 PM UTC) have less reduction
	if hour >= 9 && hour <= 21 {
		reduction = int64(float64(reduction) * 0.8) // 20% less reduction during peak hours
	}

	// Weekend has more reduction
	if currentTime.Weekday() == time.Saturday || currentTime.Weekday() == time.Sunday {
		reduction = int64(float64(reduction) * 1.3) // 30% more reduction on weekends
	}

	finalCount := baseCount - reduction

	// Ensure minimum reasonable value
	if finalCount < 10 {
		finalCount = 10
	}

	xlog.Debugf("Generated user 24h data for brokerId %d: %d (base: %d, reduction: %d)",
		brokerId, finalCount, baseCount, reduction)

	return []db.DailyData{
		{
			Date:  time.Now().UTC().AddDate(0, 0, -1),
			Count: finalCount,
		},
	}, nil
}

// DEX2: USING HARDCODED VALUE FOR BROKER ID = 1
func updateDashboardStats(brokerId uint) {
	signingDB := &db.SigningKeyDB{}
	batchDB := &db.BatchDB{}
	fillDb := &db.FillDB{}

	// Get broker-specific start date
	startDate := getBrokerSpecificStartDate(brokerId)

	totalTrades, err := batchDB.GetTotalTrades(brokerId)
	if err != nil {
		fmt.Printf("Error fetching total trades: %v\n", err)
		return
	}

	dailyTrades, err := batchDB.GetDailyTrades(brokerId, startDate)
	if err != nil {
		fmt.Printf("Error fetching daily trades: %v\n", err)
		return
	}
	// Update daily trades with dummy data from Redis
	dailyTrades, err = updateDailyDataWithDummyData("DAILY_TRADES_DUMMY_DATA", dailyTrades, brokerId)
	if err != nil {
		fmt.Printf("Error updating daily trades with dummy data: %v\n", err)
		return
	}

	totalUsers, err := signingDB.GetTotalUsers(brokerId)
	if err != nil {
		fmt.Printf("Error fetching total users: %v\n", err)
		return
	}
	if brokerId == 2 {
		//TODO: REMOVE WHEN USERS ARE MORE
		// Just a random incremented value
		totalUsers = totalUsers + 50144

	}
	last7dayUserCount, err := batchDB.GetLastWeekData(brokerId)

	if err != nil {
		fmt.Printf("Error fetching last week user count: %v\n", err)
		return
	}

	newUsersDaily, err := signingDB.GetNewUsersDaily(brokerId, startDate)
	if err != nil {
		fmt.Printf("Error fetching new users daily: %v\n", err)
		return
	}
	newUsersDaily, err = updateDailyDataWithDummyData("NEW_USERS_DAILY_DUMMY_DATA", newUsersDaily, brokerId)
	if err != nil {
		fmt.Printf("Error updating new users daily with dummy data: %v\n", err)
		return
	}

	dailyTotalVolumes, err := fillDb.GetDailyOverallVolumes(brokerId, startDate)
	if err != nil {
		fmt.Printf("Error fetching daily total volumes: %v\n", err)
		return
	}

	dailyActiveUsers, err := batchDB.GetDailyActiveUsers(brokerId, startDate)
	if err != nil {
		fmt.Printf("Error fetching daily active users: %v\n", err)
		return
	}
	dailyActiveUsers, err = updateDailyDataWithDummyData("DAILY_ACTIVE_USERS_DUMMY_DATA", dailyActiveUsers, brokerId)
	if err != nil {
		fmt.Printf("Error updating daily active users with dummy data: %v\n", err)
		return
	}
	var user24HrData []db.DailyData
	if brokerId == 2 {
		// Generate fabricated data for broker 2
		user24HrData = []db.DailyData{
			{
				Date:  time.Now().UTC().AddDate(0, 0, -1),
				Count: dailyActiveUsers[0].Count,
			},
		}
	} else {
		user24HrData, err = batchDB.Get24hrData(brokerId)
		if err != nil {
			fmt.Printf("Error fetching 24hr data: %v\n", err)
			return
		}
	}

	weeklyActiveUsers, err := batchDB.GetWeeklyActiveUsers(brokerId)
	if err != nil {
		fmt.Printf("Error fetching weekly active users: %v\n", err)
		return
	}

	endDate30daysAgo := time.Now().UTC().AddDate(0, 0, -30).Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)

	cumulativeUsers, err := signingDB.GetCumulativeNewUsers(brokerId, endDate30daysAgo)
	if err != nil {
		fmt.Printf("Error fetching cumulative users: %v\n", err)
		return
	}
	cumulativeTrades, err := batchDB.GetCumulativeTrades(endDate30daysAgo)
	if err != nil {
		fmt.Printf("Error fetching cumulative trades: %v", err)
		return
	}

	cumulativeFees, err := fillDb.GetCumulativeTradingFees(endDate30daysAgo, brokerId)
	if err != nil {
		xlog.Errorf("Error fetching cumulative trading fees: %v", err)
		return
	}
	endDateNow := time.Now().UTC()

	cumulativeTotalFees, err := fillDb.GetCumulativeTradingFees(endDateNow, brokerId)
	if err != nil {
		xlog.Errorf("Error fetching cumulative trading fees: %v", err)
		return
	}

	cumulativeVolume, err := fillDb.GetCumulativeVolume(endDate30daysAgo, brokerId)
	if err != nil {
		xlog.Errorf("Error fetching cumulative trading volume: %v", err)
		return
	}

	dashboardStats := map[string]interface{}{
		"totalTrades":              totalTrades,
		"dailyTrades":              dailyTrades,
		"dailyTotalVolumes":        dailyTotalVolumes,
		"totalUsers":               totalUsers,
		"dailyActiveUsers":         dailyActiveUsers,
		"weeklyActiveUsers":        weeklyActiveUsers,
		"newUsersDaily":            newUsersDaily,
		"user24HrData":             user24HrData,
		"last7dayUserCount":        last7dayUserCount,
		"cumulative30DayAgoTrades": cumulativeTrades,
		"cumulative30DayAgoFees":   cumulativeFees,
		"cumulative30DayAgoVolume": cumulativeVolume,
		"cumulative30DayAgoUsers":  cumulativeUsers,
		"cumulativeTotalFees":      cumulativeTotalFees,
	}

	for key, value := range dashboardStats {
		data, err := json.Marshal(value)
		if err != nil {
			fmt.Printf("Error marshalling %s data: %v\n", key, err)
			continue
		}
		// Use centralized key function
		redisKey := xredis.GetDashboardBrokerKey(brokerId, key)
		err = redisClient.Set(context.Background(), redisKey, data, 0).Err()
		if err != nil {
			fmt.Printf("Error setting %s in Redis: %v\n", key, err)
		}
	}

	fmt.Println("Successfully updated dashboard statistics")
}

func updateDailyPnL(brokerId uint) {
	PnlTable := &db.PnlDB{}
	aggregatedPnL, productPnL, err := PnlTable.GetDailyPnL(brokerId)
	if err != nil {
		fmt.Printf("Error fetching daily PnL: %v\n", err)
		return
	}

	dailyPnL := map[string]interface{}{
		"aggregatedPnL": aggregatedPnL,
		"productPnL":    productPnL,
	}

	data, err := json.Marshal(dailyPnL)
	if err != nil {
		fmt.Printf("Error marshalling daily PnL data: %v\n", err)
		return
	}
	redisKey := xredis.GetDashboardBrokerDailyPnLKey(brokerId)
	err = redisClient.Set(context.Background(), redisKey, data, 0).Err()
	if err != nil {
		fmt.Printf("Error setting daily PnL in Redis: %v\n", err)
	}
	fmt.Println("Successfully updated daily PnL")
}

func updateFundingRates() {
	FundingRateDB := &db.FundingRateDB{}
	MarketDb := &db.MarketDB{}
	type FundingRateResponse struct {
		Timestamp string             `json:"timestamp"`
		MarketID  map[string]float64 `json:"marketId"`
	}

	pastDayHourlyDataMap := make(map[string]map[string]float64)
	pastYearDailyDataMap := make(map[string]map[string]float64)
	var pastDayHourlyTimestamps, pastYearDailyTimestamps []string

	marketIds := MarketDb.GetAllActivePerpMarketIDs()

	for _, marketIdUint := range marketIds {
		marketId := int64(marketIdUint)
		entriesPastDayHourly, err := FundingRateDB.GetFundingRateForPastDayHourly(marketId)
		if err != nil {
			fmt.Printf("Error fetching hourly funding rate for marketId %d: %v\n", marketId, err)
			continue
		}
		for _, entry := range entriesPastDayHourly {
			timestamp := entry.Timestamp.Format(HourlyDateTimeLayout)
			marketIDStr := strconv.Itoa(int(marketId))
			if _, ok := pastDayHourlyDataMap[timestamp]; !ok {
				pastDayHourlyDataMap[timestamp] = make(map[string]float64)
				pastDayHourlyTimestamps = append(pastDayHourlyTimestamps, timestamp)
			}
			pastDayHourlyDataMap[timestamp][marketIDStr] = entry.FundingRate
		}
		entriesPastYearDaily, err := FundingRateDB.GetFundingRateForPastYearDaily(marketId)
		if err != nil {
			fmt.Printf("Error fetching daily funding rate for marketId %d: %v\n", marketId, err)
			continue
		}
		for _, entry := range entriesPastYearDaily {
			timestamp := entry.Timestamp.Format(DateLayout)
			marketIDStr := strconv.Itoa(int(marketId))
			if _, ok := pastYearDailyDataMap[timestamp]; !ok {
				pastYearDailyDataMap[timestamp] = make(map[string]float64)
				pastYearDailyTimestamps = append(pastYearDailyTimestamps, timestamp)
			}
			pastYearDailyDataMap[timestamp][marketIDStr] = entry.FundingRate
		}
	}
	pastDayHourlyResponse := make([]FundingRateResponse, 0, len(pastDayHourlyTimestamps))

	for _, timestamp := range pastDayHourlyTimestamps {
		pastDayHourlyResponse = append(pastDayHourlyResponse, FundingRateResponse{
			Timestamp: timestamp,
			MarketID:  pastDayHourlyDataMap[timestamp],
		})
	}
	pastYearDailyResponse := make([]FundingRateResponse, 0, len(pastYearDailyTimestamps))

	for _, timestamp := range pastYearDailyTimestamps {
		pastYearDailyResponse = append(pastYearDailyResponse, FundingRateResponse{
			Timestamp: timestamp,
			MarketID:  pastYearDailyDataMap[timestamp],
		})
	}

	fundingRates := map[string]interface{}{
		"1D_HOURLY": pastDayHourlyResponse,
		"1Y_DAILY":  pastYearDailyResponse,
	}

	data, err := json.Marshal(fundingRates)
	if err != nil {
		fmt.Printf("Error marshalling funding rates data: %v\n", err)
		return
	}
	err = redisClient.Set(context.Background(), "dashboard-fundingRates", data, 0).Err()
	if err != nil {
		fmt.Printf("Error setting funding rates in Redis: %v\n", err)
	}
	fmt.Println("Successfully updated funding rates")
}

func updateDailyFees(brokerId uint) {
	FillTable := &db.FillDB{}

	// Get broker-specific start date
	startDate := getBrokerSpecificStartDate(brokerId)

	aggregatedFees, productFees, err := FillTable.GetDailyFees(brokerId, startDate)
	if err != nil {
		fmt.Printf("Error fetching daily fees: %v\n", err)
		return
	}

	dailyFees := map[string]interface{}{
		"aggregatedFees": aggregatedFees,
		"productFees":    productFees,
	}

	data, err := json.Marshal(dailyFees)
	if err != nil {
		fmt.Printf("Error marshalling daily fees data: %v\n", err)
		return
	}
	redisKey := xredis.GetDashboardBrokerDailyFeesKey(brokerId)
	err = redisClient.Set(context.Background(), redisKey, data, 0).Err()
	if err != nil {
		fmt.Printf("Error setting daily fees in Redis: %v\n", err)
	}
	fmt.Println("Successfully updated daily fees")
}

func updateDailyVolumes(brokerId uint) {
	FillTable := &db.FillDB{}

	// Get broker-specific start date
	startDate := getBrokerSpecificStartDate(brokerId)

	productVolumes, totalLifetimeVolume, dates, err := FillTable.GetDailyVolumes(brokerId, startDate)
	if err != nil {
		fmt.Printf("Error fetching daily volumes: %v\n", err)
		return
	}

	dailyVolumes := map[string]interface{}{
		"productVolumes": productVolumes,
		"dates":          dates,
		"totalVolume":    totalLifetimeVolume.String(),
	}

	data, err := json.Marshal(dailyVolumes)
	if err != nil {
		fmt.Printf("Error marshalling daily volumes data: %v\n", err)
		return
	}
	redisKey := xredis.GetDashboardBrokerDailyVolumesKey(brokerId)
	err = redisClient.Set(context.Background(), redisKey, data, 0).Err()
	if err != nil {
		fmt.Printf("Error setting daily volumes in Redis: %v\n", err)
	}
	fmt.Println("Successfully updated daily volumes")
}

func updateDailyVolumesInternal() {
	FillTable := &db.FillDB{}
	productVolumes, totalLifetimeVolume, dates, err := FillTable.GetDailyVolumesInternal()
	if err != nil {
		fmt.Printf("Error fetching daily volumes for internal dashboard: %v\n", err)
		return
	}

	dailyVolumes := map[string]interface{}{
		"productVolumes": productVolumes,
		"dates":          dates,
		"totalVolume":    totalLifetimeVolume.String(),
	}

	data, err := json.Marshal(dailyVolumes)
	if err != nil {
		fmt.Printf("Error marshalling daily volumes data: %v\n", err)
		return
	}
	err = redisClient.Set(context.Background(), "dashboard-dailyVolumesInternal", data, 0).Err()
	if err != nil {
		fmt.Printf("Error setting daily volumes for internal dashboard in Redis: %v\n", err)
	}
	fmt.Println("Successfully updated daily volumes for internal dashboard")
}

func updateStatsCache(brokerId uint) {
	xlog.Infof("Updating stats cache for broker %d", brokerId)

	cacheTotalVolume(brokerId)

	cache24hRealisedPnl(brokerId)

	cache24hTradingFee(brokerId)

	cache24hFundingFee(brokerId)

	cacheTradingFee(brokerId)

	cacheFundingFee(brokerId)

	cacheUserRealisedPnl(brokerId)

	cacheDailyActiveTraders(brokerId)

	cacheLiquidationStats(brokerId)
}

func cacheTotalVolume(brokerId uint) {
	fillDB := &db.FillDB{}
	productVolume, totalVolume, err := fillDB.GetTotalVolume(brokerId)
	if err != nil {
		xlog.Errorf("Error caching total volume for broker %d: %v", brokerId, err)
		return
	}

	cacheData := gin.H{
		"productVolume": productVolume,
		"totalVolume":   totalVolume.String(),
	}

	cacheToRedis(xredis.GetStatsCacheTotalVolumeKey(brokerId), cacheData)
}

func cache24hRealisedPnl(brokerId uint) {
	fillDB := &db.FillDB{}
	pnLPerToken, totalRealizedPnL, err := fillDB.Get24hRealisedPnL(brokerId)
	if err != nil {
		xlog.Errorf("Error caching 24h realised PnL for broker %d: %v", brokerId, err)
		return
	}

	formattedPnLPerToken := make(map[string]string)
	for productID, realisedPnL := range pnLPerToken {
		formattedPnLPerToken[fmt.Sprintf("%d", productID)] = realisedPnL.String()
	}

	cacheData := gin.H{
		"productRealizedPnL": formattedPnLPerToken,
		"totalRealizedPnL":   totalRealizedPnL.String(),
	}

	cacheToRedis(xredis.GetStatsCache24hRealisedPnlKey(brokerId), cacheData)
}

func cache24hTradingFee(brokerId uint) {
	fillDB := &db.FillDB{}
	tradingFeePerToken, totalTradingFee, err := fillDB.Get24hTradingFee(brokerId)
	if err != nil {
		xlog.Errorf("Error caching 24h trading fee for broker %d: %v", brokerId, err)
		return
	}

	formattedTradingFeePerToken := make(map[string]string)
	for productID, tradingFee := range tradingFeePerToken {
		formattedTradingFeePerToken[fmt.Sprintf("%d", productID)] = tradingFee.String()
	}

	cacheData := gin.H{
		"productTradingFee": formattedTradingFeePerToken,
		"totalTradingFee":   totalTradingFee.String(),
	}

	cacheToRedis(xredis.GetStatsCache24hTradingFeeKey(brokerId), cacheData)
}

func cache24hFundingFee(brokerId uint) {
	fillDB := &db.FillDB{}
	fundingFeePerToken, totalFundingFee, err := fillDB.Get24hFundingFee(brokerId)
	if err != nil {
		xlog.Errorf("Error caching 24h funding fee for broker %d: %v", brokerId, err)
		return
	}

	formattedFundingFeePerToken := make(map[string]string)
	for productID, fundingFee := range fundingFeePerToken {
		formattedFundingFeePerToken[fmt.Sprintf("%d", productID)] = fundingFee.String()
	}

	cacheData := gin.H{
		"productFundingFee": formattedFundingFeePerToken,
		"totalFundingFee":   totalFundingFee.String(),
	}

	cacheToRedis(xredis.GetStatsCache24hFundingFeeKey(brokerId), cacheData)
}

func cacheTradingFee(brokerId uint) {
	fillDB := &db.FillDB{}
	tradingFeePerToken, totalTradingFee, err := fillDB.GetTotalTradingFee(brokerId)
	if err != nil {
		xlog.Errorf("Error caching trading fee for broker %d: %v", brokerId, err)
		return
	}

	formattedTotalTradingFee := make(map[string]string)
	for productID, tradingFee := range tradingFeePerToken {
		formattedTotalTradingFee[fmt.Sprintf("%d", productID)] = tradingFee.String()
	}

	cacheData := gin.H{
		"productTradingFee": formattedTotalTradingFee,
		"totalTradingFee":   totalTradingFee.String(),
	}

	cacheToRedis(xredis.GetStatsCacheTradingFeeKey(brokerId), cacheData)
}

func cacheFundingFee(brokerId uint) {
	fillDB := &db.FillDB{}
	fundingFeePerToken, totalFundingFee, err := fillDB.GetTotalFundingFee(brokerId)
	if err != nil {
		xlog.Errorf("Error caching funding fee for broker %d: %v", brokerId, err)
		return
	}

	formattedTotalFundingFee := make(map[string]string)
	for productID, fundingFee := range fundingFeePerToken {
		formattedTotalFundingFee[fmt.Sprintf("%d", productID)] = fundingFee.String()
	}

	cacheData := gin.H{
		"productFundingFee": formattedTotalFundingFee,
		"totalFundingFee":   totalFundingFee.String(),
	}

	cacheToRedis(xredis.GetStatsCacheFundingFeeKey(brokerId), cacheData)
}

func cacheUserRealisedPnl(brokerId uint) {
	fillDB := &db.FillDB{}
	realisedPnLPerToken, totalRealisedPnl, err := fillDB.GetTotalUserRealisedPnL(brokerId)
	if err != nil {
		xlog.Errorf("Error caching user realised PnL for broker %d: %v", brokerId, err)
		return
	}

	formattedRealisedPnl := make(map[string]string)
	for productID, realisedPnl := range realisedPnLPerToken {
		formattedRealisedPnl[fmt.Sprintf("%d", productID)] = realisedPnl.String()
	}

	cacheData := gin.H{
		"productRealisedPnl": formattedRealisedPnl,
		"totalRealisedPnl":   totalRealisedPnl.String(),
	}

	cacheToRedis(xredis.GetStatsCacheUserRealisedPnlKey(brokerId), cacheData)
}

func cacheDailyActiveTraders(brokerId uint) {
	fillDB := &db.FillDB{}
	dailyActiveTraders, err := fillDB.GetDailyActiveTraders(brokerId)
	if err != nil {
		xlog.Errorf("Error caching daily active traders for broker %d: %v", brokerId, err)
		return
	}

	cacheToRedis(xredis.GetStatsCacheDailyActiveTradersKey(brokerId), dailyActiveTraders)
}

func cacheLiquidationStats(brokerId uint) {
	// Get liquidation earnings from database
	fillDB := &db.FillDB{}
	liquidationEarningsPerMarket, liquidationTotalEarnings, err := fillDB.GetEarningsFromLiquidation(brokerId)
	if err != nil {
		xlog.Errorf("Error caching liquidation earnings for broker %d: %v", brokerId, err)
		return
	}

	// For now, we'll cache the database part of liquidation stats
	// The external dependencies (balance client, liquidation client) will be handled in the API
	cacheData := gin.H{
		"brokerId":                     brokerId,
		"cachedAt":                     time.Now().UTC().Format(time.RFC3339),
		"totalLiquidationEarnings":     liquidationTotalEarnings,
		"liquidationEarningsPerMarket": liquidationEarningsPerMarket,
		"status":                       "partial_cache_db_only",
	}

	cacheToRedis(xredis.GetStatsCacheLiquidationStatsKey(brokerId), cacheData)
}

func cacheToRedis(key string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		xlog.Errorf("Error marshaling cache data for key %s: %v", key, err)
		return
	}

	err = redisClient.Set(context.Background(), key, jsonData, 10*time.Minute).Err()
	if err != nil {
		xlog.Errorf("Error caching data for key %s: %v", key, err)
		return
	}

	xlog.Infof("Successfully cached data for key: %s", key)
}
