package xredis

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/redis/go-redis/v9"
)

func GetCumulativeFundingRateForSymbol(redisClient *redis.Client, symbol string) (*big.Int, error) {
	cumulativeFundingRateKey := GetCumulativeFundingRateKey(symbol)
	cumulativeFundingRateStr, err := redisClient.Get(context.Background(), cumulativeFundingRateKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("funding rate not found for symbol %s", symbol)
		}
		return nil, fmt.Errorf("error fetching funding for symbol %s: %v", symbol, err)
	}

	if cumulativeFundingRateStr == "" {
		return nil, fmt.Errorf("funding rate not found for symbol %s", symbol)
	}

	var cumulativeFundingRateData CumulativeFundingRateData
	err = json.Unmarshal([]byte(cumulativeFundingRateStr), &cumulativeFundingRateData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling funding rate JSON for symbol %s: %v", symbol, err)
	}

	cumulativeFundingRate, ok := new(big.Int).SetString(cumulativeFundingRateData.CumulativeFundingRate, 10)
	if !ok {
		return nil, fmt.Errorf("error converting funding rate string to big.Int for symbol %s", symbol)
	}

	return cumulativeFundingRate, nil
}
