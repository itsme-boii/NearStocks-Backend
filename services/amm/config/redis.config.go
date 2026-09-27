package config

import (
	"context"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var useRedisConfig bool

func SetSlippageRedisConfigEnabled(enabled bool) {
	useRedisConfig = enabled
}

func getSizeFromConfig(marketId uint) float64 {
	if size, exists := MarketIdToSize[marketId]; exists {
		return size
	}
	return 1000.0
}

func getSlippageFromConfig(marketId uint) float64 {
	if slippage, exists := MarketIdToSlippage[marketId]; exists {
		return slippage
	}
	return 0.001
}

func getSizesFromConfig(marketIds []uint) map[uint]float64 {
	result := make(map[uint]float64)
	for _, marketId := range marketIds {
		result[marketId] = getSizeFromConfig(marketId)
	}
	return result
}

func getSlippagesFromConfig(marketIds []uint) map[uint]float64 {
	result := make(map[uint]float64)
	for _, marketId := range marketIds {
		result[marketId] = getSlippageFromConfig(marketId)
	}
	return result
}

func GetSize(redisClient *redis.Client, marketId uint) float64 {
	if !useRedisConfig {
		return getSizeFromConfig(marketId)
	}

	key := xredis.GetAMMSizesKey()
	field := xredis.GetAMMSizesField(marketId)

	result, err := redisClient.HGet(context.Background(), key, field).Result()
	if err != nil {
		return getSizeFromConfig(marketId)
	}

	size, err := strconv.ParseFloat(result, 64)
	if err != nil {
		return getSizeFromConfig(marketId)
	}

	return size
}

func GetSlippage(redisClient *redis.Client, marketId uint) float64 {
	if !useRedisConfig {
		return getSlippageFromConfig(marketId)
	}

	key := xredis.GetAMMSlippageKey()
	field := xredis.GetAMMSlippageField(marketId)

	result, err := redisClient.HGet(context.Background(), key, field).Result()
	if err != nil {
		return getSlippageFromConfig(marketId)
	}

	slippage, err := strconv.ParseFloat(result, 64)
	if err != nil {
		return getSlippageFromConfig(marketId)
	}

	return slippage
}

func GetSizesBatch(redisClient *redis.Client, marketIds []uint) map[uint]float64 {
	if len(marketIds) == 0 {
		return make(map[uint]float64)
	}

	if !useRedisConfig {
		return getSizesFromConfig(marketIds)
	}

	fields := make([]string, len(marketIds))
	for i, marketId := range marketIds {
		fields[i] = strconv.FormatUint(uint64(marketId), 10)
	}

	// Batch Redis call
	key := xredis.GetAMMSizesKey()
	values, err := redisClient.HMGet(context.Background(), key, fields...).Result()
	if err != nil {
		return getSizesFromConfig(marketIds)
	}

	result := make(map[uint]float64)
	for i, value := range values {
		marketId := marketIds[i]
		if value == nil {
			result[marketId] = getSizeFromConfig(marketId)
			continue
		}

		sizeStr, ok := value.(string)
		if !ok {
			result[marketId] = getSizeFromConfig(marketId)
			continue
		}

		size, err := strconv.ParseFloat(sizeStr, 64)
		if err != nil {
			result[marketId] = getSizeFromConfig(marketId)
			continue
		}

		result[marketId] = size
	}

	return result
}

func GetSlippagesBatch(redisClient *redis.Client, marketIds []uint) map[uint]float64 {
	if len(marketIds) == 0 {
		return make(map[uint]float64)
	}

	if !useRedisConfig {
		return getSlippagesFromConfig(marketIds)
	}

	fields := make([]string, len(marketIds))
	for i, marketId := range marketIds {
		fields[i] = strconv.FormatUint(uint64(marketId), 10)
	}

	// Batch Redis call
	key := xredis.GetAMMSlippageKey()
	values, err := redisClient.HMGet(context.Background(), key, fields...).Result()
	if err != nil {
		return getSlippagesFromConfig(marketIds)
	}

	result := make(map[uint]float64)
	for i, value := range values {
		marketId := marketIds[i]
		if value == nil {
			result[marketId] = getSlippageFromConfig(marketId)
			continue
		}

		slippageStr, ok := value.(string)
		if !ok {
			result[marketId] = getSlippageFromConfig(marketId)
			continue
		}

		slippage, err := strconv.ParseFloat(slippageStr, 64)
		if err != nil {
			result[marketId] = getSlippageFromConfig(marketId)
			continue
		}

		result[marketId] = slippage
	}

	return result
}
