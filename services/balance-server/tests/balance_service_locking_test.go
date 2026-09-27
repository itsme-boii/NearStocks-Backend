package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/services"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/testutils"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMultiTokenBalanceLocking(t *testing.T) {
	testutils.SetupBalanceEnv()
	testutils.WithSetupMockRedis(t, func() {
		balanceService := services.NewBalanceService()
		client := xredis.GetRedisClient()
		subaccountID := "test-subaccount"
		productIds := []uint32{2, 4}
		initialBalances := []string{"1000", "2000"}
		updateBalances := []string{"1500", "2500"}
		hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
		lockKey := hashKey + "-lock"

		// Clear existing data
		ctx := context.Background()
		client.Del(ctx, hashKey)

		// First update
		err := balanceService.UpdateMultiTokenBalance(subaccountID, productIds, initialBalances)
		assert.NoError(t, err, "Failed to update token balances (first update)")

		// Verify the first update
		for i, productId := range productIds {
			field := fmt.Sprintf("%d", productId)
			result, err := client.HGet(ctx, hashKey, field).Result()
			assert.NoError(t, err, "Failed to get token balance from Redis (first update)")

			var balance types.Balance
			err = json.Unmarshal([]byte(result), &balance)
			assert.NoError(t, err, "Failed to unmarshal token balance (first update)")
			assert.Equal(t, initialBalances[i], balance.Available, fmt.Sprintf("Token balance for productId %d is incorrect after first update", productId))
		}

		// Second update
		err = balanceService.UpdateMultiTokenBalance(subaccountID, productIds, updateBalances)
		assert.NoError(t, err, "Failed to update token balances (second update)")

		// Verify the second update
		for i, productId := range productIds {
			field := fmt.Sprintf("%d", productId)
			result, err := client.HGet(ctx, hashKey, field).Result()
			assert.NoError(t, err, "Failed to get token balance from Redis (second update)")

			var balance types.Balance
			err = json.Unmarshal([]byte(result), &balance)
			assert.NoError(t, err, "Failed to unmarshal token balance (second update)")

			initialBalance, ok := new(big.Int).SetString(initialBalances[i], 10)
			assert.True(t, ok, fmt.Sprintf("Failed to parse initial balance for productId %d", productId))

			updateBalance, ok := new(big.Int).SetString(updateBalances[i], 10)
			assert.True(t, ok, fmt.Sprintf("Failed to parse update balance for productId %d", productId))

			expectedBalance := new(big.Int).Add(initialBalance, updateBalance)
			assert.Equal(t, expectedBalance.String(), balance.Available, fmt.Sprintf("Token balance for productId %d is incorrect after second update", productId))
		}

		// Try to acquire the lock again to ensure it was released
		mutex, err := xredis.AcquireLockWithRetry(lockKey)
		assert.NoError(t, err, "Failed to acquire lock after function execution")

		// Release the lock acquired in the test
		err = xredis.ReleaseLock(mutex)
		assert.NoError(t, err, "Failed to release lock")
	})
}

// Helper function for the next test
func parseLockedAmount(value string) (uint32, string, error) {
	parts := strings.Split(value, "_")
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid locked amount format")
	}
	productID, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, "", fmt.Errorf("invalid product ID in locked amount: %v", err)
	}
	return uint32(productID), parts[1], nil
}

func TestAtomicLockBalance(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		client := xredis.GetRedisClient()
		subAccountId := "test-subaccount"
		entity := ctypes.LockerEntity("test-entity")
		entityId := "test-entity-id"
		productId := uint32(1)
		quoteAmountx18 := big.NewInt(1)

		ctx := context.Background()
		key := xredis.GetBalanceLockerKey(subAccountId)
		field := xredis.GetBalanceLockerField(entity, entityId)
		balanceKey := fmt.Sprintf("v0_balance_%s", subAccountId)
		balanceField := fmt.Sprintf("%d", productId)

		// Clear existing data
		client.Del(ctx, key)
		client.Del(ctx, balanceKey)

		// Number of concurrent increments
		const numGoroutines = 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Mutex to protect output
		var mu sync.Mutex

		// Test: Increment balance concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				defer wg.Done()
				balanceLockerService := services.NewBalanceLockerService()

				err := balanceLockerService.AtomicLockBalance(subAccountId, entity, entityId, productId, quoteAmountx18)
				if err != nil {
					t.Errorf("Failed to lock balance: %v", err)
					return
				}

				// Fetch and print locked balance after each increment
				mu.Lock()
				defer mu.Unlock()

				lockedBalance, err := client.HGet(ctx, key, field).Result()
				if err != nil && err != redis.Nil {
					t.Errorf("Failed to get locked balance: %v", err)
					return
				}
				_, balanceStr, err := parseLockedAmount(lockedBalance)
				if err != nil {
					t.Errorf("Failed to parse locked balance: %v", err)
					return
				}
				balanceBigInt := new(big.Int)
				if _, success := balanceBigInt.SetString(balanceStr, 10); !success {
					t.Errorf("Failed to parse locked balance: %v", balanceStr)
					return
				}

				balanceData, err := client.HGet(ctx, balanceKey, balanceField).Result()
				if err != nil && err != redis.Nil {
					t.Errorf("Failed to get balance data from v0_balance key: %v", err)
					return
				}

				var balance struct {
					Available string `json:"available"`
					Locked    string `json:"locked"`
				}
				if balanceData != "" {
					err = json.Unmarshal([]byte(balanceData), &balance)
					if err != nil {
						t.Errorf("Failed to unmarshal balance data: %v", err)
						return
					}
				}

				lockedBalanceInt, success := new(big.Int).SetString(balance.Locked, 10)
				if !success {
					t.Errorf("Failed to parse locked balance in v0_balance key: %v", balance.Locked)
					return
				}

				t.Logf("Increment %d: Locked Balance = %v, v0_balance Locked Balance = %v\n", i+1, balanceBigInt, lockedBalanceInt)
			}(i)
		}

		// Wait for all goroutines to finish
		wg.Wait()

		// Verify that the locked balance was incremented correctly
		lockedBalance, err := client.HGet(ctx, key, field).Result()
		assert.NoError(t, err, "Failed to get locked balance")
		_, balanceStr, err := parseLockedAmount(lockedBalance)
		assert.NoError(t, err, "Failed to parse locked balance")
		balanceBigInt := new(big.Int)
		_, success := balanceBigInt.SetString(balanceStr, 10)
		assert.True(t, success, "Failed to parse locked balance")
		expectedBalance := big.NewInt(int64(numGoroutines))
		assert.Equal(t, expectedBalance, balanceBigInt, "Locked balance was not incremented correctly")

		// Verify that the balance in the v0_balance_<subaccountID> key is updated correctly
		balanceData, err := client.HGet(ctx, balanceKey, balanceField).Result()
		assert.NoError(t, err, "Failed to get balance data from v0_balance key")

		var balance struct {
			Available string `json:"available"`
			Locked    string `json:"locked"`
		}
		err = json.Unmarshal([]byte(balanceData), &balance)
		assert.NoError(t, err, "Failed to unmarshal balance data")

		lockedBalanceInt, success := new(big.Int).SetString(balance.Locked, 10)
		assert.True(t, success, "Failed to parse locked balance in v0_balance key")
		assert.Equal(t, expectedBalance, lockedBalanceInt, "Locked balance in v0_balance key was not incremented correctly")
	})
}

func TestUnlockFullBalance(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {

		client := xredis.GetRedisClient()
		subAccountId := "test-subaccount"
		entity := ctypes.LOCKER_ENTITY_ORDER
		entityId := "test-entity-id"
		productId := uint32(1)
		initialLockAmount := big.NewInt(1000)

		ctx := context.Background()
		key := xredis.GetBalanceLockerKey(subAccountId)
		field := xredis.GetBalanceLockerField(entity, entityId)
		balanceKey := fmt.Sprintf("v0_balance_%s", subAccountId)
		balanceField := fmt.Sprintf("%d", productId)

		// Clear existing data
		client.Del(ctx, key)
		client.Del(ctx, balanceKey)

		balanceLockerService := services.NewBalanceLockerService()

		// Lock the initial balance
		err := balanceLockerService.AtomicLockBalance(subAccountId, entity, entityId, productId, initialLockAmount)
		assert.NoError(t, err, "Failed to lock initial balance")

		// Unlock the full balance
		err = balanceLockerService.UnlockFullBalance(subAccountId, entity, entityId)
		assert.NoError(t, err, "Failed to unlock full balance")

		// Verify that the locked balance was unlocked correctly
		lockedBalance, err := client.HGet(ctx, key, field).Result()
		if err == redis.Nil {
			lockedBalance = "0_0"
		} else {
			assert.NoError(t, err, "Failed to get locked balance")
		}
		_, balanceStr, err := parseLockedAmount(lockedBalance)
		assert.NoError(t, err, "Failed to parse locked balance")
		balanceBigInt := new(big.Int)
		_, success := balanceBigInt.SetString(balanceStr, 10)
		assert.True(t, success, "Failed to parse locked balance")
		assert.Equal(t, big.NewInt(0), balanceBigInt, "Locked balance was not fully unlocked correctly")

		// Verify that the balance in the v0_balance_<subaccountID> key is updated correctly
		balanceData, err := client.HGet(ctx, balanceKey, balanceField).Result()
		assert.NoError(t, err, "Failed to get balance data from v0_balance key")

		var balance struct {
			Available string `json:"available"`
			Locked    string `json:"locked"`
		}
		err = json.Unmarshal([]byte(balanceData), &balance)
		assert.NoError(t, err, "Failed to unmarshal balance data")

		lockedBalanceInt, success := new(big.Int).SetString(balance.Locked, 10)
		assert.True(t, success, "Failed to parse locked balance in v0_balance key")
		assert.Equal(t, big.NewInt(0), lockedBalanceInt, "Locked balance in v0_balance key was not fully unlocked correctly")
	})
}

func TestUnlockAllLockedBalance(t *testing.T) {
	testutils.SetupBalanceEnv()
	testutils.WithSetupMockRedis(t, func() {
		client := xredis.GetRedisClient()
		subAccountId := "test-subaccount"
		entity := ctypes.LOCKER_ENTITY_ORDER
		entityId := "test-entity-id"
		productId := uint32(1)
		initialLockAmount := big.NewInt(1000)

		ctx := context.Background()
		key := xredis.GetBalanceLockerKey(subAccountId)
		field := xredis.GetBalanceLockerField(entity, entityId)

		// Clear existing data
		client.Del(ctx, key)

		balanceLockerService := services.NewBalanceLockerService()

		// Lock the initial balance
		err := balanceLockerService.AtomicLockBalance(subAccountId, entity, entityId, productId, initialLockAmount)
		assert.NoError(t, err, "Failed to lock initial balance")

		// Unlock all locked balances
		err = balanceLockerService.UnlockAllLockedBalance(subAccountId)
		assert.NoError(t, err, "Failed to unlock all locked balances")

		// Verify that all locked balances were unlocked correctly
		lockedBalance, err := client.HGet(ctx, key, field).Result()
		assert.Error(t, err, "redis: nil", "Expected error redis: nil, got %v", err)
		assert.Equal(t, redis.Nil, err, "Expected redis.Nil error, got %v", err)
		assert.Equal(t, "", lockedBalance, "Expected empty locked balance, got %v", lockedBalance)

		key2 := xredis.GetBalanceKey(subAccountId)
		lockedBalance2, err := client.HGet(ctx, key2, field).Result()
		assert.Error(t, err, "redis: nil", "Expected error redis: nil, got %v", err)
		assert.Equal(t, redis.Nil, err, "Expected redis.Nil error, got %v", err)
		assert.Equal(t, "", lockedBalance2, "Expected empty locked balance, got %v", lockedBalance2)
	})
}

func TestAtomicUpdateSubaccountForMatch(t *testing.T) {
	testutils.SetupBalanceEnv()
	testutils.WithSetupMockRedis(t, func() {
		client := xredis.GetRedisClient()
		makerSubAccountId := "maker-subaccount"
		takerSubAccountId := "taker-subaccount"
		entity := ctypes.LOCKER_ENTITY_ORDER
		makerOrderId := uint(1)
		takerOrderId := uint(2)
		productId := uint32(1)
		makerUnlockQuotex18 := big.NewInt(1000) // Adjusted for a higher unlock amount
		takerUnlockQuotex18 := big.NewInt(500)  // Adjusted for a higher unlock amount

		ctx := context.Background()
		makerKey := xredis.GetBalanceLockerKey(makerSubAccountId)
		makerField := xredis.GetBalanceLockerField(entity, fmt.Sprintf("%d", makerOrderId))
		takerKey := xredis.GetBalanceLockerKey(takerSubAccountId)
		takerField := xredis.GetBalanceLockerField(entity, fmt.Sprintf("%d", takerOrderId))

		// Clear existing data
		client.Del(ctx, makerKey)
		client.Del(ctx, takerKey)

		// Initialize and lock initial balances to match the unlock amounts times the number of goroutines
		const numGoroutines = 10
		initialMakerLock := big.NewInt(0).Mul(makerUnlockQuotex18, big.NewInt(int64(numGoroutines)))
		initialTakerLock := big.NewInt(0).Mul(takerUnlockQuotex18, big.NewInt(int64(numGoroutines)))

		balanceLockerService := services.NewBalanceLockerService()

		// Lock the initial balances
		err := balanceLockerService.AtomicLockBalance(makerSubAccountId, entity, fmt.Sprintf("%d", makerOrderId), productId, initialMakerLock)
		assert.NoError(t, err, "Failed to lock initial maker balance")

		err = balanceLockerService.AtomicLockBalance(takerSubAccountId, entity, fmt.Sprintf("%d", takerOrderId), productId, initialTakerLock)
		assert.NoError(t, err, "Failed to lock initial taker balance")

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Mutex to protect output
		var mu sync.Mutex

		// Test: Concurrently update subaccounts for match
		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				defer wg.Done()

				balanceLockerService := services.NewBalanceLockerService()

				req := types.UpdateSubaccountForMatchRequest{
					TxnCounter: &ONE,
					MarketId:   1,
					Maker: types.BalancePerpPayload{
						SubaccountId:   makerSubAccountId,
						OrderId:        makerOrderId,
						UnlockQuotex18: makerUnlockQuotex18,
					},
					Taker: types.BalancePerpPayload{
						SubaccountId:   takerSubAccountId,
						OrderId:        takerOrderId,
						UnlockQuotex18: takerUnlockQuotex18,
					},
				}

				err := balanceLockerService.AtomicUpdateSubaccountForMatch(req)
				if err != nil {
					t.Errorf("Failed to update subaccount for match: %v", err)
					return
				}

				// Fetch and print locked balance after each update
				mu.Lock()
				makerLockedBalance, makerErr := client.HGet(ctx, makerKey, makerField).Result()
				if makerErr != nil && makerErr != redis.Nil {
					t.Errorf("Failed to get maker locked balance: %v", makerErr)
					mu.Unlock()
					return
				}
				takerLockedBalance, takerErr := client.HGet(ctx, takerKey, takerField).Result()
				if takerErr != nil && takerErr != redis.Nil {
					t.Errorf("Failed to get taker locked balance: %v", takerErr)
					mu.Unlock()
					return
				}
				mu.Unlock()

				_, makerBalanceStr, errParse := parseLockedAmount(makerLockedBalance)
				if errParse != nil {
					t.Errorf("Failed to parse maker locked balance: %v", errParse)
					return
				}
				makerBalanceBigInt := new(big.Int)
				if _, success := makerBalanceBigInt.SetString(makerBalanceStr, 10); !success {
					t.Errorf("Failed to parse maker locked balance: %v", makerBalanceStr)
					return
				}

				_, takerBalanceStr, errParse := parseLockedAmount(takerLockedBalance)
				if errParse != nil {
					t.Errorf("Failed to parse taker locked balance: %v", errParse)
					return
				}
				takerBalanceBigInt := new(big.Int)
				if _, success := takerBalanceBigInt.SetString(takerBalanceStr, 10); !success {
					t.Errorf("Failed to parse taker locked balance: %v", takerBalanceStr)
					return
				}

				t.Logf("Increment %d: Maker Locked Balance = %v, Taker Locked Balance = %v\n", i+1, makerBalanceBigInt, takerBalanceBigInt)
			}(i)
		}

		// Wait for all goroutines to finish
		wg.Wait()

		// Verify that the locked balances were updated correctly
		makerLockedBalance, err := client.HGet(ctx, makerKey, makerField).Result()
		assert.NoError(t, err, "Failed to get maker locked balance")
		_, makerBalanceStr, err := parseLockedAmount(makerLockedBalance)
		assert.NoError(t, err, "Failed to parse maker locked balance")
		makerBalanceBigInt := new(big.Int)
		_, success := makerBalanceBigInt.SetString(makerBalanceStr, 10)
		assert.True(t, success, "Failed to parse maker locked balance")

		takerLockedBalance, err := client.HGet(ctx, takerKey, takerField).Result()
		assert.NoError(t, err, "Failed to get taker locked balance")
		_, takerBalanceStr, err := parseLockedAmount(takerLockedBalance)
		assert.NoError(t, err, "Failed to parse taker locked balance")
		takerBalanceBigInt := new(big.Int)
		_, success = takerBalanceBigInt.SetString(takerBalanceStr, 10)
		assert.True(t, success, "Failed to parse taker locked balance")

		expectedMakerBalance := big.NewInt(0)
		expectedTakerBalance := big.NewInt(0)
		assert.Equal(t, expectedMakerBalance, makerBalanceBigInt, "Maker locked balance was not decremented correctly")
		assert.Equal(t, expectedTakerBalance, takerBalanceBigInt, "Taker locked balance was not decremented correctly")
	})
}
