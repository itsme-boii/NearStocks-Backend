package tests

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/testutils"
	"log"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestAtomicTxn(t *testing.T) {
	// TestNewRedisClient tests NewRedisClient function
	// It should return a new redis client
	t.Skip("Skip atomic txn test")

	client := xredis.GetRedisClient()
	if client == nil {
		t.Errorf("Expected a new redis client, got nil")
		return
	}

	ctx := context.Background()

	err := client.Watch(ctx, func(tx *redis.Tx) error {

		startValue, err := tx.Get(ctx, "Test").Int64()
		if err != nil && err != redis.Nil {
			t.Errorf("Start val: Expected nil, got %v", err)
			return err
		}

		startValue2, err := tx.Get(ctx, "Test2").Int64()

		if err != nil && err != redis.Nil {
			t.Errorf("Start val: Expected nil, got %v", err)
			return err
		}

		fmt.Printf("Start Values: Test: %v | Test2: %v\n", startValue, startValue2)

		_, err2 := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			err = pipe.Set(ctx, "Test", startValue+1, 0).Err()

			// return fmt.Errorf("Mock error")

			if err != nil {
				return err
			}

			startValue, err := tx.Get(ctx, "Test").Int64()
			if err != nil && err != redis.Nil {
				t.Errorf("Start val: Expected nil, got %v", err)
				return err
			}

			startValue2, err := tx.Get(ctx, "Test2").Int64()
			if err != nil && err != redis.Nil {
				t.Errorf("Start val: Expected nil, got %v", err)
				return err
			}

			fmt.Printf("Mid Values: Test: %v | Test2: %v\n", startValue, startValue2)

			err = pipe.Set(ctx, "Test2", startValue2+1, 0).Err()
			// return fmt.Errorf("Mock error")

			if err != nil {
				return err
			}

			return nil
		})

		endValue, err := tx.Get(ctx, "Test").Int64()
		if err != nil && err != redis.Nil {
			return err
		}

		endValue2, err := tx.Get(ctx, "Test2").Int64()

		if err != nil && err != redis.Nil {
			return err
		}

		fmt.Printf("End Values: Test: %v | Test2: %v\n", endValue, endValue2)

		if err2 != nil {
			return err2
		}

		return nil
	})

	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}

// Testing AcquireLockWithRetry() and ReleaseLock() functions in xredis library
func TestAcquireAndReleaseLock(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		client := xredis.GetRedisClient()
		if client == nil {
			t.Errorf("Expected a new redis client, got nil")
			return
		}

		ctx := context.Background()
		lockKey := "test-lock"
		testKey := "Test"

		// Acquire the lock
		mutex, err := xredis.AcquireLockWithRetry(lockKey)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		// Ensure the lock is held
		if mutex.Until().Before(time.Now()) {
			t.Fatalf("Lock is not held after acquisition")
		}

		// Perform some operations while the lock is held
		startValue, err := client.Get(ctx, testKey).Int64()
		if err != nil && err != redis.Nil {
			t.Errorf("Start val: Expected nil or an int64, got %v", err)
			return
		}

		fmt.Printf("Start Value: %v\n", startValue)

		err = client.Set(ctx, testKey, startValue+1, 0).Err()
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		endValue, err := client.Get(ctx, testKey).Int64()
		if err != nil && err != redis.Nil {
			t.Errorf("End val: Expected nil or an int64, got %v", err)
			return
		}

		fmt.Printf("End Value: %v\n", endValue)

		// Release the lock
		err = xredis.ReleaseLock(mutex)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}

		// Attempt to acquire the lock again to ensure it can be re-acquired
		mutex, err = xredis.AcquireLockWithRetry(lockKey)
		if err != nil {
			t.Fatalf("Failed to acquire lock after release: %v", err)
		}

		// Clean up by releasing the lock again
		err = xredis.ReleaseLock(mutex)
		if err != nil {
			t.Fatalf("Failed to release lock during cleanup: %v", err)
		}
	})
}

func holdLockTemporarily(lockKey string, duration time.Duration) {
	mutex, err := xredis.AcquireLockWithRetry(lockKey)
	if err != nil {
		log.Printf("Failed to temporarily hold lock: %v\n", err)
		return
	}
	defer xredis.ReleaseLock(mutex)

	log.Printf("Holding lock for %v\n", duration)
	time.Sleep(duration)
	log.Printf("Released lock after %v\n", duration)
}

func TestLockRetry(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		client := xredis.GetRedisClient()
		if client == nil {
			t.Errorf("Expected a new redis client, got nil")
			return
		}

		lockKey := "test-retry-lock"
		testKey := "TestRetry"

		// Private configuration which has been hardcoded for the sake of this test
		lockRetryDelay := 500 * time.Millisecond
		maxLockRetries := 10

		// Hold the lock for a duration longer than one retry delay but shorter than the total retry time
		go holdLockTemporarily(lockKey, 3*lockRetryDelay)

		// Add a small delay to ensure holdLockTemporarily goroutine acquires the lock first
		time.Sleep(100 * time.Millisecond)

		startTime := time.Now()
		mutex, err := xredis.AcquireLockWithRetry(lockKey)
		elapsedTime := time.Since(startTime)

		assert.NoError(t, err, "Failed to acquire lock with retry")
		assert.NotNil(t, mutex, "Mutex should not be nil")

		// Ensure that the total retry duration is within an acceptable range
		expectedMinDuration := 3 * lockRetryDelay
		expectedMaxDuration := time.Duration(maxLockRetries) * lockRetryDelay
		marginOfError := 150 * time.Millisecond // Adding a margin of error to account for timing discrepancies
		assert.GreaterOrEqual(t, elapsedTime, expectedMinDuration-marginOfError, "Elapsed time should be at least the expected minimum duration")
		assert.LessOrEqual(t, elapsedTime, expectedMaxDuration, "Elapsed time should be at most the expected maximum duration")

		// Perform some operations while the lock is held
		ctx := context.Background()
		startValue, err := client.Get(ctx, testKey).Int64()
		if err != nil && err != redis.Nil {
			t.Errorf("Start val: Expected nil or an int64, got %v", err)
			return
		}

		log.Printf("Start Value: %v\n", startValue)

		err = client.Set(ctx, testKey, startValue+1, 0).Err()
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		endValue, err := client.Get(ctx, testKey).Int64()
		if err != nil && err != redis.Nil {
			t.Errorf("End val: Expected nil or an int64, got %v", err)
			return
		}

		log.Printf("End Value: %v\n", endValue)

		// Release the lock
		err = xredis.ReleaseLock(mutex)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}

		// Clean up by deleting the test key
		client.Del(ctx, testKey)
	})
}
