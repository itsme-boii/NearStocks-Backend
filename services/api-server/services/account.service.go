package services

import (
	"context"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type AccountService struct {
	redisClient *redis.Client
}

func NewAccountService() *AccountService {
	return &AccountService{
		redisClient: xredis.GetRedisClient(),
	}
}

func GetNonce(subAccountId string) (string, error) {
	redisClient := xredis.GetRedisClient()
	// Create a new context
	ctx := context.Background()

	subAccountId = strings.ToLower(subAccountId)
	key := xredis.GetNonceKey(subAccountId)

	nonce, err := redisClient.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return "0", err
	}
	// FIXME: Have a better way to handle non-existing nonce and keep a first time initialization
	if err == redis.Nil {
		redisClient.Set(ctx, key, "0", 0)
		return "0", nil
	}
	return nonce, nil
}

func WithNonceRedisLock[T any](subaccountHex string, f func(currentNonce string) (*T, error)) (*T, error) {
	return xredis.WithRedisLock(xredis.GetNonceLockKey(subaccountHex), func() (*T, error) {
		currentNonce, err := GetNonce(subaccountHex)
		if err != nil {
			return nil, err
		}
		return f(currentNonce)
	})
}

// mbordia check usage for this function
func IncrementNonce(subaccountIdHex string, currentNonce string, decrement ...bool) error {
	defer cutils.LogTime(time.Now(), "IncrementNonce")
	// Create a new context
	ctx := context.Background()
	nonceUint, err := strconv.ParseUint(currentNonce, 10, 64)
	if err != nil {
		xlog.Errorf("AS - Error while parsing nonce to uint64. Err: %v", err)
		return err
	}

	// Check if the decrement flag is provided and true
	if !(len(decrement) > 0 && decrement[0]) {
		nonceUint++
	}

	if err := xredis.GetRedisClient().Set(ctx, xredis.GetNonceKey(subaccountIdHex), nonceUint, 0).Err(); err != nil {
		xlog.Errorf("AS - Failed to set nonce in Redis: %v", err)
		return err
	}

	return nil
}
