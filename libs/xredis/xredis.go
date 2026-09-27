package xredis

import (
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"os"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

var (
	redisClient           *redis.Client
	redsyncPool           redsyncredis.Pool
	liquiRedisClient      *redis.Client
	once                  sync.Once
	liquiOnce             sync.Once
	defaultLockExpiration time.Duration
	maxLockRetries        int
	lockRetryDelay        time.Duration
)

// initialize initializes the Redis client and pool once
func Initialize() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	xlog.Infof("Redis Addr: %s", redisClient.Options().Addr)
	redsyncPool = goredis.NewPool(redisClient)
	defaultLockExpiration = 10 * time.Minute
	maxLockRetries = 10
	lockRetryDelay = 500 * time.Millisecond
}

func InitializeLiquiRedis() {
	liquiRedisClient = redis.NewClient(&redis.Options{
		Addr:         os.Getenv("LIQUI_REDIS_ADDR"),
		Password:     os.Getenv("LIQUI_REDIS_PASSWORD"),
		DB:           0,
		ReadTimeout:  -1, // NOTE: This is unsafe, but we are using it for now
		WriteTimeout: -1, // NOTE: This is unsafe, but we are using it for now
	})
}

// GetRedisClient returns the singleton Redis client
func GetRedisClient() *redis.Client {
	once.Do(Initialize)
	return redisClient
}

// GetRedsyncPool returns the singleton Redsync pool
func GetRedsyncPool() redsyncredis.Pool {
	once.Do(Initialize)
	return redsyncPool
}

func GetLiquiRedisClient() *redis.Client {
	liquiOnce.Do(InitializeLiquiRedis)
	return liquiRedisClient
}

// AcquireLock acquires a distributed lock
// AcquireLockWithRetry acquires a distributed lock with a retry mechanism
func AcquireLockWithRetry(lockKey string, lockOptions ...LockOptions) (*redsync.Mutex, error) {
	pool := GetRedsyncPool()
	options := LockOptions{
		LockExpiry: &defaultLockExpiration,
		MaxRetries: &maxLockRetries,
		RetryDelay: &lockRetryDelay,
	}
	if len(lockOptions) > 0 {
		options.Set(lockOptions[0])
	}

	rs := redsync.New(pool)
	mutex := rs.NewMutex(lockKey, redsync.WithExpiry(*options.LockExpiry))

	var err error
	for i := 0; i < *options.MaxRetries; i++ {
		if err = mutex.Lock(); err == nil {
			return mutex, nil
		}
		time.Sleep(*options.RetryDelay)
	}
	return nil, err
}

// ReleaseLock releases a distributed lock
func ReleaseLock(mutex *redsync.Mutex) error {
	_, err := mutex.Unlock()
	return err
}

func WithRedisLock[T any](key string, f func() (*T, error), lockOptions ...LockOptions) (*T, error) {
	mutex, err := AcquireLockWithRetry(key, lockOptions...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock:%+v", err)
		}
	}()

	return f()
}
