package funding

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github/eugenix-io/logx-inf-backend/libs/xredis"

	"github.com/redis/go-redis/v9"
)

var useRedisConfig bool

func SetFundingRedisConfigEnabled(enabled bool) {
	useRedisConfig = enabled
}

func getDefaultFundingCaps(marketId uint) *xredis.AMMFundingCaps {
	if maxCap, exists := MarketToAnnualizedFundingCaps[marketId]; exists {
		if minCap, minExists := MarketToMinimumAnnualizedFundingCaps[marketId]; minExists {
			return &xredis.AMMFundingCaps{
				MaxFundingCap: maxCap,
				MinFundingCap: minCap,
			}
		}
	}
	return &xredis.AMMFundingCaps{
		MaxFundingCap: 150,
		MinFundingCap: 7,
	}
}

func GetFundingCaps(redisClient *redis.Client, marketId uint) (*xredis.AMMFundingCaps, error) {
	if !useRedisConfig {
		return getDefaultFundingCaps(marketId), nil
	}

	// If redisClient is nil, return default caps (Added for testing purposes)
	if redisClient == nil {
		return getDefaultFundingCaps(marketId), nil
	}

	fundingCaps, err := GetAMMFundingCaps(redisClient, marketId)
	if err != nil {
		return getDefaultFundingCaps(marketId), nil
	}
	return fundingCaps, nil
}

func GetFundingRateFactor(redisClient *redis.Client) (int64, error) {
	if !useRedisConfig {
		return FundingRateFactor, nil
	}

	// If redisClient is nil, return default funding rate factor (Added for testing purposes)
	if redisClient == nil {
		return FundingRateFactor, nil
	}

	key := xredis.GetAMMFundingRateFactorKey()

	result, err := redisClient.Get(context.Background(), key).Result()
	if err != nil {
		return FundingRateFactor, nil
	}

	factor, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return FundingRateFactor, nil
	}

	return factor, nil
}

func GetAMMFundingCaps(redisClient *redis.Client, marketId uint) (*xredis.AMMFundingCaps, error) {
	if !useRedisConfig {
		return getDefaultFundingCaps(marketId), nil
	}

	// If redisClient is nil, return error to trigger fallback in GetFundingCaps (Added for testing purposes)
	if redisClient == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	key := xredis.GetAMMFundingCapsKey()
	field := xredis.GetAMMFundingCapsField(marketId)

	result, err := redisClient.HGet(context.Background(), key, field).Result()
	if err != nil {
		return nil, fmt.Errorf("error fetching funding caps for market %d: %v", marketId, err)
	}

	var fundingCaps xredis.AMMFundingCaps
	if err := json.Unmarshal([]byte(result), &fundingCaps); err != nil {
		return nil, fmt.Errorf("error unmarshalling funding caps for market %d: %v", marketId, err)
	}

	return &fundingCaps, nil
}
