package funding

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/openinterest"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

var (
	perpMarketIds               []uint
	perpMarketIdToSymbolMapping map[uint]string
	redisClient                 *redis.Client
)

func InitialiseFundingRates() {
	// Initialize Redis config for funding caps and funding rate factor
	initializeFundingRedisConfig()

	// Initialize maps
	perpMarketIdToSymbolMapping = make(map[uint]string)

	activeMarkets := (&db.MarketDB{}).GetAllActivePerpMarkets()
	for _, market := range activeMarkets {
		perpMarketIds = append(perpMarketIds, market.ID)
		perpMarketIdToSymbolMapping[market.ID] = market.Symbol
	}

	redisClient = xredis.GetRedisClient()
	xlog.Infof("Funding Cron - Successfully Initialised")
}

func initializeFundingRedisConfig() {
	useRedis := os.Getenv("AMM_USE_REDIS_CONFIG") == "1"
	SetFundingRedisConfigEnabled(useRedis)

	if useRedis {
		xlog.Infof("Funding Config - Using Redis-based config")
	} else {
		xlog.Infof("Funding Config - Using file-based config (Redis disabled)")
	}
}

// initializes and start the cron job for funding servikmce
func StartFundingCron() {

	xlog.Infof("Perp Markets %v\n", perpMarketIds)
	c := cron.New(cron.WithSeconds())

	// Acquire lock for 4 minutes | This is to ensure that only one instance of the liquidation service is running
	lockTime := time.Minute * 4
	retryDelay := time.Second * 20
	maxRetries := 10
	lockOptions := xredis.LockOptions{
		LockExpiry: &lockTime,
		MaxRetries: &maxRetries,
		RetryDelay: &retryDelay,
	}
	fundingCronInterval := os.Getenv("FUNDING_CRON_INTERVAL")
	if fundingCronInterval == "" {
		fundingCronInterval = "0 */2 * * * *"
	}

	c.AddFunc(fundingCronInterval, func() {

		if !cutils.IsSequencerRunning() {
			xlog.Infof("Funding Cron - Sequencer is paused. Not submitting transactions.")
			return
		}

		xlog.Infof("Funding Cron job running: %v... Acquiring lock", time.Now())

		_, err := xredis.WithRedisLock(xredis.GetFundingRateLockKey(), func() (*xredis.NOOP, error) {
			openInterestMapping, errOI := fetchAndStoreOpenInterest()
			if errOI != nil {
				xlog.Errorf("Error fetching Open interest for perp markets :%v", errOI)
				return nil, errOI
			}

			var wg sync.WaitGroup
			fundingRates := make([]int64, len(perpMarketIds))

			fundingTimestamp := time.Now().Unix()

			for index, productId := range perpMarketIds {
				wg.Add(1)
				go func(index int, productId uint) {
					defer wg.Done()

					// Fetch Long OI Cap and Short OI Cap from Redis
					longMarketOICapKey := xredis.GetMarketLongOICap(strconv.FormatUint(uint64(productId), 10))
					shortMarketOICapKey := xredis.GetMarketShortOICap(strconv.FormatUint(uint64(productId), 10))

					longCap, errLong := redisClient.Get(context.Background(), longMarketOICapKey).Result()
					if errLong != nil {
						if errLong == redis.Nil {
							xlog.Infof("Long Market OI Cap key %s does not exist, using default value: 5000", longMarketOICapKey)
						} else {
							xlog.Errorf("Error fetching Long Market OI Cap for market %d: %v, using default value: 5000", productId, errLong)
						}
						longCap = "5000" // Default value for both missing key and errors
					}

					shortCap, errShort := redisClient.Get(context.Background(), shortMarketOICapKey).Result()
					if errShort != nil {
						if errShort == redis.Nil {
							xlog.Infof("Short Market OI Cap key %s does not exist, using default value: 5000", shortMarketOICapKey)
						} else {
							xlog.Errorf("Error fetching Short Market OI Cap for market %d: %v, using default value: 5000", productId, errShort)
						}
						shortCap = "5000" // Default value for both missing key and errors
					}

					// Convert longCap and shortCap to *big.Int. A parse failure falls through to the same
					// fallback path as a calculation error, so the rate is still stored (behavior-spec H-1).
					var rate int64
					var err1 error
					longCapInt, okLong := new(big.Int).SetString(longCap, 10)
					shortCapInt, okShort := new(big.Int).SetString(shortCap, 10)
					if !okLong || !okShort {
						err1 = fmt.Errorf("invalid OI cap for market %d: long=%q short=%q", productId, longCap, shortCap)
					} else {
						// Calculate total OI cap
						totalOICap := new(big.Int).Add(longCapInt, shortCapInt)

						// Calculate funding rate
						rate, err1 = CalculateFundingRate(productId, openInterestMapping[productId], totalOICap.String())
					}

					xlog.Infof("Funding Rate For Market %d : %d", productId, rate)

					xlog.Infof("Funding rate Product ID: %v ,timestamp : %d", rate, fundingTimestamp, productId)
					var latestEntry *db.FundingRateTable
					var dbReadErr error
					if err1 != nil {
						xlog.Errorf("Error calculating Funding Fee for market %d : %v", productId, err1)
						latestEntry, dbReadErr = (&db.FundingRateDB{}).GetLatestEntryByMarketId(productId)
					}
					fundingRates[index] = SelectFundingRate(rate, err1, latestEntry, dbReadErr)

					// Always record the rate that is actually sent in PerpTick, including the fallback.
					// Otherwise the Redis cumulative (used for off-chain balances) stops advancing while the
					// on-chain tick still applies the rate, and the two diverge (behavior-spec H-1).
					if err2 := storeFundingRate(productId, fundingRates[index], fundingTimestamp); err2 != nil {
						xlog.Errorf("Error storing Funding fee for market %d : %v", productId, err2)
						xclient.GetGlobalDiscordClient().SendWebhookMessage(fmt.Sprintf(
							"FUNDING DRIFT: market %d rate %d was not stored in Redis at %d but is included in PerpTick: %v",
							productId, fundingRates[index], fundingTimestamp, err2))
					}
				}(index, productId)
			}
			wg.Wait()

			// NOTE: Placement for this might look odd but
			// current thought process to that we want state updates to be in same order on chain as we are having on web2
			// So we are placing the transaction here
			transactionCounter, err := transaction.IncrementCounter(1)
			if err != nil {
				xlog.Errorf("Failed to increment transaction counter: %v", err)
				return nil, err
			}

			fundingRateBigInt := int64ToBigIntArray(fundingRates)
			xlog.Infof("Funding rate symbol array: %v ,timestamp : %d", fundingRates, fundingTimestamp)
			//Write the transaction to batch db for persistence on blockchain
			if err := contract.GlobalContracts.EndpointContract.FundingTick(perpMarketIds, fundingRateBigInt, fundingTimestamp, transactionCounter); err != nil {
				xlog.Errorf("Failed to queue funding tick: %v", err)
				xclient.GetGlobalDiscordClient().SendWebhookMessage(fmt.Sprintf("FUNDING TICK NOT QUEUED at %d: %v", fundingTimestamp, err))
			}
			return nil, nil
		}, lockOptions)

		xlog.Infof("Released funding rate lock")
		if err != nil {
			xlog.Errorf("Error in funding rate calculation: %v", err)
			xclient.GetGlobalDiscordClient().SendWebhookMessage(fmt.Sprintf("Error in funding rate calculation: %v", err))
		}
	})

	// Start the cron scheduler
	c.Start()

	// Keep the cron scheduler running in the background
	select {}
}

// FIXME: Fetch Open Interest from redis instead of AMM_SUBACCOUNT position
func fetchAndStoreOpenInterest() (map[uint]string, error) {
	totalOI, err := openinterest.FetchTotalOIFromRedis()
	if err != nil {
		xlog.Errorf("Funding Cron - Error fetching total open interest from Redis: %v", err)
		return nil, err
	}

	// Initialize a map to hold the open interest data
	productOpenInterest := make(map[uint]string)

	// Check for missing product IDs from perpMarketIds and set their open interest to '0'
	for _, productID := range perpMarketIds {
		productOpenInterest[uint(productID)] = totalOI.MustGetByProductId(uint32(productID)).String()
	}

	return productOpenInterest, nil
}

// ------- Funding Rate Formula ---------
// We want a linear correlation between the OI and funding rate. When OI = 0, funding rate can be 0.
// Therefore, funding rate is a simple y = mx curve
// Slope of the curve : Funding Rate Cap / Funding Rate Factor
// Reason : when open interest = funding rate factor, the Annualised funding fee should be equal to Annual Funding Rate Cap
func CalculateFundingRate(productId uint, openInterest string, totalOICap string) (int64, error) {
	// Convert openInterest (OI Delta) to a big.Int
	openInterestInt := new(big.Int)
	_, ok := openInterestInt.SetString(openInterest, 10)
	if !ok {
		return 0, fmt.Errorf("error converting open interest to *big.Int")
	}

	// Scale down openInterestInt by dividing by 1e18
	openInterestInt = cutils.Divx18(openInterestInt)

	// Convert totalOICap to a big.Int
	totalOICapInt := new(big.Int)
	_, ok = totalOICapInt.SetString(totalOICap, 10)
	if !ok {
		return 0, fmt.Errorf("error converting total OI cap to *big.Int")
	}

	fundingCaps, err := GetFundingCaps(redisClient, productId)
	if err != nil {
		xlog.Errorf("Error fetching funding caps for product %d: %v", productId, err)
		return 0, fmt.Errorf("failed to get funding caps for product %d: %v", productId, err)
	}

	fundingRateFactor, err := GetFundingRateFactor(redisClient)
	if err != nil {
		xlog.Errorf("Error fetching funding rate factor for product %d: %v", productId, err)
		return 0, fmt.Errorf("failed to get funding rate factor for product %d: %v", productId, err)
	}

	annualizedMaxCap := big.NewInt(fundingCaps.MaxFundingCap)
	annualizedMinCap := big.NewInt(fundingCaps.MinFundingCap)

	// Convert annualized caps to per-second values
	multiple := new(big.Int).Div(big.NewInt(1e18), big.NewInt(365*24*60*60*100))
	maxFundingRate := new(big.Int).Mul(annualizedMaxCap, multiple)
	minFundingRate := new(big.Int).Mul(annualizedMinCap, multiple)

	// Apply randomness to bounds
	randomInt := rand.Intn(6) + 1 // Random integer between 1 and 6
	randomAdjustment := new(big.Int).Mul(big.NewInt(int64(randomInt)), big.NewInt(1e8))
	randomizedMin := new(big.Int).Sub(minFundingRate, randomAdjustment)
	randomizedMax := new(big.Int).Add(maxFundingRate, randomAdjustment)

	// Handle sign inversion for negative openInterest
	if openInterestInt.Sign() < 0 {
		randomizedMin.Neg(randomizedMin)
		randomizedMax.Neg(randomizedMax)
		if randomizedMin.Cmp(randomizedMax) > 0 {
			randomizedMin, randomizedMax = randomizedMax, randomizedMin
		}
	}

	// Calculate funding rate
	numerator := new(big.Int).Mul(new(big.Int).Abs(openInterestInt), new(big.Int).Sub(maxFundingRate, minFundingRate))
	denominator := new(big.Int).Add(totalOICapInt, big.NewInt(fundingRateFactor))
	fundingRate := new(big.Int).Div(numerator, denominator)
	fundingRate.Add(fundingRate, minFundingRate)

	// Apply the sign of the OI Delta
	if openInterestInt.Sign() < 0 {
		fundingRate.Neg(fundingRate)
	}

	// Enforce caps (clamp fundingRate within randomized bounds)
	fundingRate = cutils.Maxi(fundingRate, randomizedMin)
	fundingRate = cutils.Min(fundingRate, randomizedMax)

	// Check if the result fits within the int64 range
	if fundingRate.IsInt64() {
		return fundingRate.Int64(), nil
	} else {
		return 0, fmt.Errorf("result out of int64 range")
	}
}

func storeFundingRate(productId uint, fundingRateSecond int64, fundingTimestamp int64) error {
	// Insert funding rate into the database
	_, dbWriteErr := (&db.FundingRateDB{}).Insert(productId, fundingRateSecond, fundingTimestamp)
	if dbWriteErr != nil {
		xlog.Errorf("Error writing funding rate of market %d to database : %v", productId, dbWriteErr)
		return dbWriteErr
	}

	// Add funding rate and timestamp to Redis
	fundingRateDict := xredis.FundingRateData{
		FundingRate:      strconv.FormatInt(fundingRateSecond, 10),
		FundingTimestamp: strconv.FormatInt(fundingTimestamp, 10),
	}

	// Marshal the struct to JSON
	fundingRateJson, err := json.Marshal(fundingRateDict)
	if err != nil {
		xlog.Errorf("Error marshalling JSON: %v", err)
		return err
	}

	fundingRateKey := xredis.GetFundingRateKey(perpMarketIdToSymbolMapping[productId])
	err = redisClient.Set(context.Background(), fundingRateKey, fundingRateJson, 0).Err()
	if err != nil {
		xlog.Errorf("funding rate calculation : Error storing funding rate %v\n", err)
		return err
	}

	// Handle cumulative funding rate
	cumulativeFundingRateKey := xredis.GetCumulativeFundingRateKey(perpMarketIdToSymbolMapping[productId])

	var cumulativeFundingRate *big.Int
	var cumulativeFundingTimestamp int64
	cumulativeFundingRateDict, err := redisClient.Get(context.Background(), cumulativeFundingRateKey).Result()
	if err != nil {
		if err == redis.Nil {
			xlog.Infof("cumulative funding rate key %v does not exist\n", cumulativeFundingRateKey)
			cumulativeFundingRate = big.NewInt(0)
			cumulativeFundingTimestamp = fundingTimestamp
		} else {
			xlog.Errorf("Error getting value for key %v: %v\n", fundingRateKey, err)
			return err
		}
	} else {
		if cumulativeFundingRateDict == "" {
			xlog.Infof("cumulative funding rate key %v is empty\n", cumulativeFundingRateKey)
			cumulativeFundingRate = big.NewInt(0)
			cumulativeFundingTimestamp = fundingTimestamp
		} else {
			// Unmarshal the JSON string into the struct
			var cumulativeFundingRateData xredis.CumulativeFundingRateData
			err = json.Unmarshal([]byte(cumulativeFundingRateDict), &cumulativeFundingRateData)
			if err != nil {
				xlog.Errorf("Error unmarshalling cumulative funding rate JSON: %v", err)
				return err
			}
			cumulativeFundingRateInt, ok := new(big.Int).SetString(cumulativeFundingRateData.CumulativeFundingRate, 10)
			if !ok {
				xlog.Errorf("Error converting cumulative funding rate from string to big.Int")
				return fmt.Errorf("error converting cumulative funding rate from string to big.Int")
			}
			cumulativeFundingRate = cumulativeFundingRateInt
			cumulativeFundingTimestamp, err = strconv.ParseInt(cumulativeFundingRateData.CumulativeFundingTimestamp, 10, 64)
			if err != nil {
				xlog.Errorf("Error converting cumulative funding timestamp from string to int64 %v", err)
				return err
			}
		}
	}

	timeDelta := fundingTimestamp - cumulativeFundingTimestamp
	cumulativeFundingRate.Add(cumulativeFundingRate, new(big.Int).Mul(big.NewInt(fundingRateSecond), big.NewInt(timeDelta)))
	newCumulativeFundingRateDict := xredis.CumulativeFundingRateData{
		CumulativeFundingRate:      cumulativeFundingRate.String(),
		CumulativeFundingTimestamp: strconv.FormatInt(fundingTimestamp, 10),
	}

	newCumulativeFundingRateJson, err := json.Marshal(newCumulativeFundingRateDict)
	if err != nil {
		xlog.Errorf("Error marshalling JSON: %v", err)
		return err
	}
	err = redisClient.Set(context.Background(), cumulativeFundingRateKey, newCumulativeFundingRateJson, 0).Err()
	if err != nil {
		xlog.Errorf("cumulative funding rate calculation : Error storing funding rate %v\n", err)
		return err
	} else {
		xlog.Infof("Cumulative Funding Rate value set for key %v : %v", cumulativeFundingRateKey, newCumulativeFundingRateDict)
	}

	return nil
}

// SelectFundingRate picks the per-second rate for a market: the freshly calculated one, or on
// failure the latest stored rate, or 0 when there is none.
func SelectFundingRate(calculated int64, calcErr error, latest *db.FundingRateTable, latestErr error) int64 {
	if calcErr == nil {
		return calculated
	}
	if latestErr != nil || latest == nil {
		return 0
	}
	return latest.FundingRate
}

// Convert an array of int64 to an array of *big.Int
func int64ToBigIntArray(values []int64) []*big.Int {
	result := make([]*big.Int, len(values))

	for i, value := range values {
		result[i] = big.NewInt(value)
	}

	return result
}
