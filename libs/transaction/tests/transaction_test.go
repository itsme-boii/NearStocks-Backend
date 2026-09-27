package tests

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/testutils"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIncrementTransactionCounterLocking(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		counterKey := "transaction_counter"
		lockKey := "transaction-counter-lock"
		ctx := context.Background()

		redisClient := xredis.GetRedisClient()

		// Increment the counter for the first time
		transactionCounter, err := transaction.IncrementCounter(1)
		assert.NoError(t, err, "Failed to increment transaction counter (first increment)")
		assert.Equal(t, uint(1), transactionCounter, "Transaction counter after first increment should be 1")

		// Verify that the counter value in Redis is 1
		result, err := redisClient.Get(ctx, counterKey).Result()
		assert.NoError(t, err, "Failed to get transaction counter from Redis (after first increment)")
		assert.Equal(t, "1", result, "Transaction counter in Redis after first increment should be 1")

		// Increment the counter for the second time
		transactionCounter, err = transaction.IncrementCounter(1)
		assert.NoError(t, err, "Failed to increment transaction counter (second increment)")
		assert.Equal(t, uint(2), transactionCounter, "Transaction counter after second increment should be 2")

		// Verify that the counter value in Redis is 2
		result, err = redisClient.Get(ctx, counterKey).Result()
		assert.NoError(t, err, "Failed to get transaction counter from Redis (after second increment)")
		assert.Equal(t, "2", result, "Transaction counter in Redis after second increment should be 2")

		// Try to acquire the lock again to ensure it was released
		mutex, err := xredis.AcquireLockWithRetry(lockKey)
		assert.NoError(t, err, "Failed to acquire lock after function execution")

		// Release the lock acquired in the test
		err = xredis.ReleaseLock(mutex)
		assert.NoError(t, err, "Failed to release lock")
	})
}

func TestConcurrentIncrementTransactionCounterLocking(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {

		counterKey := "transaction_counter"
		ctx := context.Background()
		redisClient := xredis.GetRedisClient()

		rand.NewSource(time.Now().UnixNano())
		// Number of concurrent goroutines
		numGoroutines := 10
		wg := sync.WaitGroup{}
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				var timeDelay = time.Duration(rand.Intn(3)) * time.Millisecond
				time.Sleep(timeDelay)
				// Increment the counter
				_, err := transaction.IncrementCounter(1)
				assert.NoError(t, err, "Failed to increment transaction counter in goroutine")
			}()
		}

		wg.Wait()

		// Verify that the counter value in Redis matches the number of increments
		result, err := redisClient.Get(ctx, counterKey).Result()
		assert.NoError(t, err, "Failed to get transaction counter from Redis after concurrent increments")
		assert.Equal(t, fmt.Sprintf("%d", numGoroutines), result, "Transaction counter in Redis should match the number of increments")
	})
}

func TestExclusiveAccessDuringConcurrentIncrement(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		counterKey := "transaction_counter"

		ctx := context.Background()
		redisClient := xredis.GetRedisClient()

		// Number of concurrent goroutines
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			_, err := transaction.IncrementCounter(1)
			assert.NoError(t, err, "Failed to increment transaction counter in goroutine")
		}

		// Verify that the counter value in Redis matches the number of increments
		result, err := redisClient.Get(ctx, counterKey).Result()
		assert.NoError(t, err, "Failed to get transaction counter from Redis after concurrent increments")
		assert.Equal(t, fmt.Sprintf("%d", numGoroutines), result, "Transaction counter in Redis should match the number of increments")
	})
}

func TestTxnBalanceMarshalUnmarshal(t *testing.T) {
	subaccountHex1 := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000002"
	subaccountHex2 := "0x0000000000015631e1e23aeefae435106017bbcee2b91676e43b000000000002"

	txnBalance := transaction.TxnBalanceUpdate{
		Subaccount1: subaccountTypes.SubaccountBalances{
			SubaccountId: subaccountHex1,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
					ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
					Balancex18: cutils.Mulx18(big.NewInt(100)), // 100 USDC
					Lockedx18:  big.NewInt(0),
				},
			},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{
				contractUtils.ETH_MARKET: {
					ProductId:             contractUtils.ETH_MARKET,
					Amountx18:             cutils.Mulx18(big.NewInt(1)),
					VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-99)), // Bought at $99
					LastCumFundingRatex18: cutils.GetBig0(),
				},
				contractUtils.TRUMP_MARKET: {
					ProductId:             contractUtils.TRUMP_MARKET,
					Amountx18:             cutils.Mulx18(big.NewInt(1)),
					VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-300)), // Bought at $300
					LastCumFundingRatex18: cutils.GetBig0(),
				},
			},
		},
		Subaccount2: subaccountTypes.SubaccountBalances{
			SubaccountId: subaccountHex2,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
					ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
					Balancex18: cutils.Mulx18(big.NewInt(10)), // 10 USDC
					Lockedx18:  big.NewInt(0),
				},
			},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{
				contractUtils.BTC_MARKET: {
					ProductId:             contractUtils.BTC_MARKET,
					Amountx18:             cutils.Mulx18(big.NewInt(5)),
					VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-1000)), // Avg price = $200
					LastCumFundingRatex18: cutils.GetBig0(),
				},
			},
		},
	}

	redisData, err := txnBalance.MarshalForRedis()
	assert.NoError(t, err, "Failed to marshal TxnBalance for Redis")

	fmt.Printf("Redis data: %v\n", redisData)

	txnBalance2 := transaction.TxnBalanceUpdate{}
	err = txnBalance2.UnmarshalForRedis(redisData)
	assert.NoError(t, err, "Failed to unmarshal TxnBalance from Redis")

	assert.Truef(t, txnBalance.LooseEquals(&txnBalance2), "Subaccounts should match")
}
