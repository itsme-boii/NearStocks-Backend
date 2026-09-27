package perputils

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// PerpUtils struct which holds the Redis client
type PerpUtils struct {
	redisClient *redis.Client
}

// NewPerpUtilImpl is a constructor function to create a new instance
// of PerpUtils with a Redis client.
func NewPerpUtils() *PerpUtils {
	return &PerpUtils{
		redisClient: xredis.GetRedisClient(),
	}
}

// factorx18 = (initialMargin + takerFeex18)
// lockable quoteAmtx18 = factorx18 * baseAmtx18 * pricex18
func GetBigLockQuotex18(baseAmtx18 *big.Int, priceAmtx18 *big.Int, market *db.MarketTable) *big.Int {
	quotex18 := cutils.Divx18(new(big.Int).Mul(baseAmtx18, priceAmtx18))
	factorx18 := new(big.Int).Add(market.InitialMarginFractionx18.Val, market.TakerFeeFractionx18.Val)
	bigLockQuotex18 := cutils.Divx18(new(big.Int).Mul(quotex18, factorx18))
	return bigLockQuotex18
}

func InitializeOpenInterest() {
	redisClient := xredis.GetRedisClient()
	for _, productID := range contractUtils.ALL_PERPS_ON_CONTRACT {
		// Convert productID to string
		productIDStr := strconv.FormatUint(uint64(productID), 10)

		// Get the Redis keys for long and short positions
		longKey := xredis.GetTotalLongPositionKey(productIDStr)
		shortKey := xredis.GetTotalShortPositionKey(productIDStr)

		// Check if the keys already exist in Redis
		longExists, err := redisClient.Exists(context.Background(), longKey).Result()
		if err != nil {
			xlog.Errorf("Error checking Redis key existence for Long OI for product ID %s: %v", productIDStr, err)
			continue
		}

		shortExists, err := redisClient.Exists(context.Background(), shortKey).Result()
		if err != nil {
			xlog.Errorf("Error checking Redis key existence for Short OI for product ID %s: %v", productIDStr, err)
			continue
		}

		// Skip fetching from contract if both long and short keys are present
		if longExists > 0 && shortExists > 0 {
			continue
		}

		xclient.GetGlobalDiscordClient().SendWebhookMessage(fmt.Sprintf("Perp Utils - Long and Short OI not found for product ID %s. Try recreating the positions.", productIDStr))
	}
}

// Function to get total long position for a given productID
func (p *PerpUtils) GetTotalLongPosition(productID uint32) (*big.Int, error) {
	// Acquire lock for the specific product ID
	productIDStr := strconv.FormatUint(uint64(productID), 10)
	lockKey := xredis.GetTotalOILockKey(productIDStr)
	mutex, err := xredis.AcquireLockWithRetry(lockKey)
	if err != nil {
		xlog.Errorf("Failed to acquire lock for productID %d: %v", productID, err)
		return nil, err
	}
	// Ensure the lock is released at the end
	defer func() {
		if err := xredis.ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock for productID %d: %v", productID, err)
		}
	}()

	// Generate the Redis key for the total long position
	longKey := xredis.GetTotalLongPositionKey(strconv.FormatUint(uint64(productID), 10))

	// Fetch the total long position from Redis
	longValue, err := p.redisClient.Get(context.Background(), longKey).Result()
	if err == redis.Nil {
		msg := fmt.Sprintf("Perp Utils - Total long position not found in Redis for product ID %v. Fetch by recalculating.", productID)
		xlog.Warnf(msg)
		xclient.GetGlobalDiscordClient().SendWebhookMessage(msg)
	} else if err != nil { // Handle errors other than key not existing
		xlog.Errorf("Error fetching long position from Redis for productID %v: %v", productID, err)
		return nil, err
	}

	// If the key doesn't exist, return 0
	if longValue == "" {
		return big.NewInt(0), nil
	}

	// Convert the fetched value to *big.Int
	totalLongPosition := new(big.Int)
	totalLongPosition, success := totalLongPosition.SetString(longValue, 10)
	if !success {
		xlog.Errorf("Failed to convert long position string to big.Int for productID %v", productID)
		return nil, fmt.Errorf("failed to convert long position string to big.Int")
	}

	return totalLongPosition, nil
}

// Function to get total short position for a given productID
func (p *PerpUtils) GetTotalShortPosition(productID uint32) (*big.Int, error) {

	// Acquire lock for the specific product ID
	productIDStr := strconv.FormatUint(uint64(productID), 10)
	lockKey := xredis.GetTotalOILockKey(productIDStr)
	mutex, err := xredis.AcquireLockWithRetry(lockKey)
	if err != nil {
		xlog.Errorf("Failed to acquire lock for productID %d: %v", productID, err)
		return nil, err
	}
	// Ensure the lock is released at the end
	defer func() {
		if err := xredis.ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock for productID %d: %v", productID, err)
		}
	}()

	// Generate the Redis key for the total short position
	shortKey := xredis.GetTotalShortPositionKey(strconv.FormatUint(uint64(productID), 10))

	// Fetch the total short position from Redis
	shortValue, err := p.redisClient.Get(context.Background(), shortKey).Result()
	if err == redis.Nil {
		msg := fmt.Sprintf("Perp Utils - Total short position not found in Redis for product ID %v. Fetch by recalculating.", productID)
		xlog.Warnf(msg)
		xclient.GetGlobalDiscordClient().SendWebhookMessage(msg)
	} else if err != nil { // Handle errors other than key not existing
		xlog.Errorf("Error fetching long position from Redis for productID %v: %v", productID, err)
		return nil, err
	}

	// If the key doesn't exist, return 0
	if shortValue == "" {
		return big.NewInt(0), nil
	}

	// Convert the fetched value to *big.Int
	totalShortPosition := new(big.Int)
	totalShortPosition, success := totalShortPosition.SetString(shortValue, 10)
	if !success {
		xlog.Errorf("Failed to convert short position string to big.Int for productID %v", productID)
		return nil, fmt.Errorf("failed to convert short position string to big.Int")
	}

	return totalShortPosition, nil
}

// AddLongShortOIAtReddis updates the total long and short positions in Redis based on currentAmount and deltaAmount.

// / TO do - Need to write test for locks
func (p *PerpUtils) AddLongShortOIAtReddis(productID uint32, currentAmount, deltaAmount *big.Int) error {
	productIDStr := strconv.FormatUint(uint64(productID), 10)
	// Acquire lock for the specific product ID
	lockKey := xredis.GetTotalOILockKey(productIDStr)
	mutex, err := xredis.AcquireLockWithRetry(lockKey)
	if err != nil {
		xlog.Errorf("Failed to acquire lock for productID %d: %v", productID, err)
		return err
	}
	// Ensure the lock is released at the end
	defer func() {
		if err := xredis.ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock for productID %d: %v", productID, err)
		}
	}()

	longKey := xredis.GetTotalLongPositionKey(productIDStr)
	shortKey := xredis.GetTotalShortPositionKey(productIDStr)

	getRedisValue := func(key string) (*big.Int, error) {
		value, err := p.redisClient.Get(context.Background(), key).Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}

		result := big.NewInt(0)
		if value != "" {
			result, _ = new(big.Int).SetString(value, 10)
		}
		return result, nil
	}

	// Retrieve the current total long and short positions
	totalLong, err := getRedisValue(longKey)
	if err != nil {
		return err
	}
	totalShort, err := getRedisValue(shortKey)
	if err != nil {
		return err
	}
	xlog.Infof("OI Update - current values of OI for product id %v, Long :%v and Short:%v", productIDStr, totalLong, totalShort)

	// Update the long and short positions using the custom logic
	totalLong, totalShort = p.UpdateLongShortPositions(totalLong, totalShort, currentAmount, deltaAmount)

	// Write the updated long and short positions back to Redis
	err = p.redisClient.Set(context.Background(), longKey, totalLong.String(), 0).Err()
	if err != nil {
		xlog.Errorf("OI Update - error setting long OI value to %v for product Id %v", totalLong.String(), productIDStr)
		return err
	}
	xlog.Infof("OI Update - Long OI for product ID %v, Long OI after Update %v, Current amount of user %v, new position amount %v", productIDStr, totalLong.String(), currentAmount.String(), deltaAmount.String())
	err = p.redisClient.Set(context.Background(), shortKey, totalShort.String(), 0).Err()
	if err != nil {
		xlog.Errorf("OI Update - error setting short OI value to %v for product Id %v", totalLong.String(), productIDStr)
		return err
	}
	xlog.Infof("OI Update - Short OI for product ID %v, Short OI after Update %v, Current amount of user %v, new position amount %v", productIDStr, totalShort.String(), currentAmount.String(), deltaAmount.String())

	return nil
}

// UpdateLongShortPositions contains the logic to calculate the new long and short positions based on currentAmount and deltaAmount.
func (p *PerpUtils) UpdateLongShortPositions(totalLong, totalShort, currentAmount, deltaAmount *big.Int) (*big.Int, *big.Int) {
	// Case 1: last Position was zero
	if currentAmount.Sign() == 0 {
		if deltaAmount.Sign() > 0 { // new position is long
			totalLong = new(big.Int).Add(totalLong, new(big.Int).Abs(deltaAmount))
		} else if deltaAmount.Sign() < 0 { // new position is short
			totalShort = new(big.Int).Add(totalShort, new(big.Int).Abs(deltaAmount))
		}
	} else if currentAmount.Sign() > 0 { // Case 2: last Position is long
		if deltaAmount.Sign() > 0 { // new position is long
			totalLong = new(big.Int).Add(totalLong, new(big.Int).Abs(deltaAmount))
		} else { // new position is short
			x := new(big.Int).Abs(currentAmount)
			y := new(big.Int).Abs(deltaAmount)

			if y.Cmp(x) > 0 { // new position is short but less than open long position
				totalLong = new(big.Int).Sub(totalLong, x)
				totalShort = new(big.Int).Add(totalShort, new(big.Int).Sub(y, x))
			} else if y.Cmp(x) == 0 { // new position is short but equal to open long position
				totalLong = new(big.Int).Sub(totalLong, x)
			} else { // new position is short but more than open long position
				totalLong = new(big.Int).Sub(totalLong, y)
			}
		}
	} else {
		if deltaAmount.Sign() < 0 { // new position is short
			totalShort = new(big.Int).Add(totalShort, new(big.Int).Abs(deltaAmount))
		} else { // new position is long
			x := new(big.Int).Abs(currentAmount)
			y := new(big.Int).Abs(deltaAmount)

			if y.Cmp(x) > 0 { // new position is long but less than open short position
				// totalShort = new(big.Int).Sub(totalShort, x)
				totalShort = new(big.Int).Sub(totalShort, x)
				totalLong = new(big.Int).Add(totalLong, new(big.Int).Sub(y, x))
			} else if y.Cmp(x) == 0 { // new position is long but equal to open short position
				totalShort = new(big.Int).Sub(totalShort, x)
			} else { // new position is long but more than open short position
				totalShort = new(big.Int).Sub(totalShort, y)
			}
		}
	}

	zero := big.NewInt(0)
	// If totalLong is less than zero, set it to zero
	if totalLong.Cmp(zero) < 0 {
		totalLong.Set(zero)
		xlog.Errorf("Error in total Long, cannot be less than zero for given productID")
	}

	// If totalShort is less than zero, set it to zero
	if totalShort.Cmp(zero) < 0 {
		totalShort.Set(zero)
		xlog.Errorf("Error in total Long, cannot be less than zero for given productID ")
	}

	return totalLong, totalShort
}

// Fetch the market-level OI cap from Redis if available, otherwise return the default
func (p *PerpUtils) GetMarketOICap(marketId uint32) (*big.Int, *big.Int, error) {
	productIDStr := strconv.FormatUint(uint64(marketId), 10)
	longRedisKey := xredis.GetMarketLongOICap(productIDStr)
	shortRedisKey := xredis.GetMarketShortOICap(productIDStr)

	longCapStr, errLong := p.redisClient.Get(context.Background(), longRedisKey).Result()
	shortCapStr, errShort := p.redisClient.Get(context.Background(), shortRedisKey).Result()

	var longOiCap, shortOiCap *big.Int

	// Handle long OI cap
	if errLong != nil {
		// Key not found, use default
		xlog.Errorf("Perp Utils - error while fetching market OI Cap for Long for market id %v, error is %v", marketId, errLong)
		longOiCap = p.GetDefaultMarketLimit(marketId)
	} else {
		longOiCap = new(big.Int)
		if _, ok := longOiCap.SetString(longCapStr, 10); !ok {
			return nil, nil, fmt.Errorf("invalid long market OI cap format")
		}
	}

	// Handle short OI cap
	if errShort != nil {
		// Key not found, use default
		xlog.Errorf("Perp Utils - error while fetching market OI Cap for Short for market id %v, error is %v", marketId, errShort)
		shortOiCap = p.GetDefaultMarketLimit(marketId)
	} else {
		shortOiCap = new(big.Int)
		if _, ok := shortOiCap.SetString(shortCapStr, 10); !ok {
			return nil, nil, fmt.Errorf("invalid short market OI cap format")
		}
	}

	return longOiCap, shortOiCap, nil
}

// Fetch the subaccount-level OI cap from Redis if available, otherwise return the default
func (p *PerpUtils) GetSubaccountOICap() (*big.Int, error) {
	redisKey := xredis.GetSubaccountOICap()

	capStr, err := p.redisClient.Get(context.Background(), redisKey).Result()
	if err == redis.Nil {
		// Key not found, fallback to default of 500,000
		return big.NewInt(500000), nil
	} else if err != nil {
		return nil, fmt.Errorf("error fetching subaccount OI cap from Redis: %v", err)
	}

	oiCap := new(big.Int)
	if _, ok := oiCap.SetString(capStr, 10); !ok {
		return nil, fmt.Errorf("invalid subaccount OI cap format")
	}

	return oiCap, nil
}

// Fetch the subaccount-level OI cap from Redis if available, otherwise return the default
func (p *PerpUtils) GetSubaccountMarketCap(productID string) (*big.Int, error) {
	redisKey := xredis.GetSubaccountMarketCap(productID)

	capStr, err := p.redisClient.Get(context.Background(), redisKey).Result()
	if err == redis.Nil {
		// Key not found, fallback to default of 500,000
		return big.NewInt(500000), nil
	} else if err != nil {
		return nil, fmt.Errorf("error fetching subaccount OI cap from Redis: %v", err)
	}

	subaccountMarketCap := new(big.Int)
	if _, ok := subaccountMarketCap.SetString(capStr, 10); !ok {
		return nil, fmt.Errorf("invalid subaccount OI cap format")
	}

	return subaccountMarketCap, nil
}

// Fetch the global OI cap from Redis if available, otherwise return the default
func (p *PerpUtils) GetGlobalNegativeOICap() (*big.Int, error) {
	redisKey := xredis.GetGlobalNegativeOICap()

	capStr, err := p.redisClient.Get(context.Background(), redisKey).Result()
	if err == redis.Nil {
		// Key not found, fallback to default of 1,000,000
		return big.NewInt(1000000), nil
	} else if err != nil {
		return nil, fmt.Errorf("error fetching global OI cap from Redis: %v", err)
	}

	oiCap := new(big.Int)
	if _, ok := oiCap.SetString(capStr, 10); !ok {
		return nil, fmt.Errorf("invalid global OI cap format")
	}

	return oiCap, nil
}

func (p *PerpUtils) GetDefaultMarketLimit(marketId uint32) *big.Int {
	// switch marketId {
	// case 1, 3:
	// 	return big.NewInt(2500000) // 2.5 million for productId 1 and 3
	// default:
	// 	return big.NewInt(1000000) // 1 million for other product IDs
	// }
	return big.NewInt(0)
}

// Returns dollar value of the spot product that you are providing
func GetQuoteValuex36(productId uint32, tokenAmountx18 *big.Int, oraclePricesMap map[string]ctypes.OraclePrice) (*big.Int, error) {
	oracleSymbol, exists := marketutils.GetBaseSymbolForProduct(productId)
	if !exists {
		xlog.Errorf("Oracle symbol not found for product id: %d. This means there is a bug in contractUtils.PRODUCT_ID_SYMBOL_TO_MAP", productId)
		return nil, fmt.Errorf("oracle symbol not found for product id: %d. This means there is a bug in contractUtils.PRODUCT_ID_SYMBOL_TO_MAP", productId)
	}

	if oraclePrice, ok := oraclePricesMap[oracleSymbol]; ok {
		return new(big.Int).Mul(tokenAmountx18, oraclePrice.Pricex18), nil
	}
	return nil, fmt.Errorf("oracle price not found for symbol: %s. This means there is a bug in oracle", oracleSymbol)
}
