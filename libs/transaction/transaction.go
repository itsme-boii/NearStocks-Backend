package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/metric"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

func IncrementCounter(val int64) (uint, error) {
	defer cutils.LogTime(time.Now(), "IncrementTransactionCounter")
	ctx := context.Background()

	// Acquire lock for the counter
	lockKey := "transaction-counter-lock"
	mutex, err := xredis.AcquireLockWithRetry(lockKey)
	if err != nil {
		xlog.Errorf("Failed to acquire lock: %v", err)
		return 0, err
	}
	defer func() {
		// Release lock
		if err := xredis.ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock: %v", err)
		}
	}()

	// Key for the transaction counter
	counterKey := "transaction_counter"
	redisClient := xredis.GetRedisClient()

	// Retrieve the current counter value from Redis
	existingCounter, err := redisClient.Get(ctx, counterKey).Result()
	if err != nil && err != redis.Nil {
		xlog.Errorf("Failed to get transaction counter: %v", err)
		return 0, err
	}

	// Initialize the counter if it doesn't exist
	counterBigInt := big.NewInt(0) // Initialize counterBigInt
	if existingCounter != "" {
		_, ok := counterBigInt.SetString(existingCounter, 10)
		if !ok {
			xlog.Errorf("Failed to parse existing transaction counter: %s", existingCounter)
			return 0, fmt.Errorf("failed to parse existing transaction counter: %s", existingCounter)
		}
	}

	// Increment the counter
	counterBigInt.Add(counterBigInt, big.NewInt(val))

	// Update the counter value in Redis
	err = redisClient.Set(ctx, counterKey, counterBigInt.String(), 0).Err()
	if err != nil {
		xlog.Errorf("Failed to update transaction counter: %v", err)
		return 0, err
	}

	// Convert the counter to uint to return
	transactionCounterUint := uint(counterBigInt.Uint64())

	return transactionCounterUint, nil
}

type TxnBalanceUpdate struct {
	Subaccount1 subaccountTypes.SubaccountBalances
	Subaccount2 subaccountTypes.SubaccountBalances
}

func IncrementOverAllSettledPnl(val int64) {
	ctx := context.Background()
	xlog.Infof(`Accounting settledpnl for value: %v `, val)
	// Acquire lock for the counter
	lockKey := "settle-pnl-lock"
	mutex, err := xredis.AcquireLockWithRetry(lockKey)
	if err != nil {
		xlog.Errorf("Failed to acquire lock: %v", err)
	}
	defer func() {
		// Release lock
		if err := xredis.ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock: %v", err)
		}
	}()

	// Key for the total settled pnl
	settlePnlKey := "total_settled_pnl"
	redisClient := xredis.GetRedisClient()

	// Retrieve the current total settled pnl value from Redis
	existingSettledPnl, err := redisClient.Get(ctx, settlePnlKey).Result()
	if err != nil && err != redis.Nil {
		xlog.Errorf("Failed to get settled pnl: %v", err)
	}

	// Initialize the totalsettledpnl if it doesn't exist
	settledPnlBigInt := big.NewInt(0) 
	if existingSettledPnl != "" {
		_, ok := settledPnlBigInt.SetString(existingSettledPnl, 10)
		if !ok {
			xlog.Errorf("Failed to parse existing settledPnl: %s", existingSettledPnl)
		}
	}

	// Increment the totalsettledpnl
	settledPnlBigInt.Add(settledPnlBigInt, big.NewInt(val))

	// Update the totalsettledpnl value in Redis
	err = redisClient.Set(ctx, settlePnlKey, settledPnlBigInt.String(), 0).Err()
	if err != nil {
		xlog.Errorf("Failed to update total settled pnl: %v", err)
	}
	
}


// Assumes that both subaccounts are present
func (tbu *TxnBalanceUpdate) MarshalForRedis() (string, error) {
	balanceUpdateMap, err := tbu.MapMap()
	if err != nil {
		xlog.Errorf("Failed to get balance update map: %v", err)
		return "", err
	}
	balanceUpdateJson, err := json.Marshal(balanceUpdateMap)
	if err != nil {
		xlog.Errorf("Failed to marshal balance update: %v", err)
		return "", err
	}

	return string(balanceUpdateJson), nil
}

// This is currently written only for two subaccount updates - Order Matching
func (tbu *TxnBalanceUpdate) UnmarshalForRedis(balanceUpdateJson string) error {
	balanceUpdateMap := make(map[string]map[string]string)
	err := json.Unmarshal([]byte(balanceUpdateJson), &balanceUpdateMap)
	if err != nil {
		xlog.Errorf("Failed to unmarshal balance update: %v", err)
		return err
	}

	subaccountIds := cutils.MapKeys(balanceUpdateMap)

	if len(subaccountIds) != 2 {
		return fmt.Errorf("no subaccount balances found in the balance update")
	}

	subaccount1Balance := subaccountTypes.SubaccountBalances{
		SubaccountId: subaccountIds[0],
	}

	err = subaccount1Balance.UnmarshalForRedis(balanceUpdateMap[subaccount1Balance.SubaccountId], subaccount1Balance.SubaccountId)
	if err != nil {
		xlog.Errorf("Failed to unmarshal subaccount1 balances: %v", err)
		return err
	}

	tbu.Subaccount1 = subaccount1Balance
	subaccount2Balance := subaccountTypes.SubaccountBalances{
		SubaccountId: subaccountIds[1],
	}
	err = subaccount2Balance.UnmarshalForRedis(balanceUpdateMap[subaccount2Balance.SubaccountId], subaccount2Balance.SubaccountId)
	if err != nil {
		xlog.Errorf("Failed to unmarshal subaccount2 balances: %v", err)
		return err
	}
	tbu.Subaccount2 = subaccount2Balance
	return nil
}

func (tbu *TxnBalanceUpdate) MapMap() (map[string]map[string]string, error) {
	if tbu.Subaccount1.SubaccountId == "" || tbu.Subaccount2.SubaccountId == "" {
		return nil, fmt.Errorf("subaccount ids are empty")
	}

	var err error
	balanceUpdateMap := make(map[string]map[string]string)

	subaccount1BalanceMap, err := tbu.Subaccount1.MarshalForRedis()
	if err != nil {
		xlog.Errorf("Failed to marshal subaccount1 balances: %v", err)
		return nil, err
	}

	balanceUpdateMap[tbu.Subaccount1.SubaccountId] = subaccount1BalanceMap

	subaccount2BalanceMap, err := tbu.Subaccount2.MarshalForRedis()
	if err != nil {
		xlog.Errorf("Failed to marshal subaccount2 balances: %v", err)
		return nil, err
	}

	balanceUpdateMap[tbu.Subaccount2.SubaccountId] = subaccount2BalanceMap
	
	return balanceUpdateMap, nil
}

func (tbu *TxnBalanceUpdate) MapString() (map[string]string, error) {
	balanceUpdateMap, err := tbu.MapMap()
	if err != nil {
		xlog.Errorf("Failed to get balance update map: %v", err)
		return nil, err
	}

	balanceUpdateStringMap := make(map[string]string)
	for key, value := range balanceUpdateMap {
		jsonValue, err := json.Marshal(value)
		if err != nil {
			xlog.Errorf("Failed to marshal balance update: %v", err)
			return nil, err
		}
		balanceUpdateStringMap[key] = string(jsonValue)
	}

	return balanceUpdateStringMap, nil
}

func (tbu *TxnBalanceUpdate) Equals(other *TxnBalanceUpdate) bool {
	return tbu.Subaccount1.Equals(&other.Subaccount1) && tbu.Subaccount2.Equals(&other.Subaccount2)
}

// LooseEquals is used to compare two TxnBalanceUpdates where the subaccounts can be in any order
func (tbu *TxnBalanceUpdate) LooseEquals(other *TxnBalanceUpdate) bool {
	if tbu.Subaccount1.SubaccountId == other.Subaccount1.SubaccountId {
		return tbu.Subaccount1.Equals(&other.Subaccount1) && tbu.Subaccount2.Equals(&other.Subaccount2)
	} else {
		return tbu.Subaccount1.Equals(&other.Subaccount2) && tbu.Subaccount2.Equals(&other.Subaccount1)
	}
}

/* This will be the map that we'll store in Redis
hashKey: TXN_BALANCE_UPDATE
field: txnCounter
{
	"<subaccountId1>": <balance we store in redis>,
	"<subaccountId2>": <balance we store in redis>,
}
*/

// Expiry time = 10 minutes
// No need to acquire any lock as there will be only one writer for each txnCounter
func WriteBalanceUpdateForTxn(pipe redis.Pipeliner, txnCounter uint, txnBalanceUpdateRequest TxnBalanceUpdate) error {
	defer cutils.LogTime(time.Now(), metric.WRITE_BALANCES_FOR_TXN)
	ctx := context.Background()
	txnUpdateBalanceData, err := txnBalanceUpdateRequest.MarshalForRedis()
	if err != nil {
		xlog.Errorf("Failed to marshal balance update: %v", err)
		return err
	}

	hashKey := xredis.GetTxnBalanceUpdateKey(txnCounter)

	// Expiry time = 10 minutes
	pipe.Set(ctx, hashKey, txnUpdateBalanceData, 10*time.Minute)
	return nil
}

func ReadBalanceUpdateForTxn(txnCounter uint) (*TxnBalanceUpdate, error) {
	defer cutils.LogTime(time.Now(), metric.READ_BALANCES_FOR_TXN)
	ctx := context.Background()
	redisClient := xredis.GetRedisClient()

	hashKey := xredis.GetTxnBalanceUpdateKey(txnCounter)
	txnUpdateBalanceData, err := redisClient.Get(ctx, hashKey).Result()
	// If the balance update is not found then it's an issue in code
	if err != nil {
		xlog.Errorf("Failed to get balance update for txnCounter %d: %v", txnCounter, err)
		return nil, err
	}

	txnBalanceUpdate := TxnBalanceUpdate{}
	err = txnBalanceUpdate.UnmarshalForRedis(txnUpdateBalanceData)
	if err != nil {
		xlog.Errorf("Failed to unmarshal balance update for txnCounter %d: %v", txnCounter, err)
		return nil, err
	}

	return &txnBalanceUpdate, nil
}

func DeleteBalanceUpdateForTxn(txnCounter uint) error {
	defer cutils.LogTime(time.Now(), metric.DELETE_BALANCES_FOR_TXN)
	ctx := context.Background()
	redisClient := xredis.GetRedisClient()

	hashKey := xredis.GetTxnBalanceUpdateKey(txnCounter)
	_, err := redisClient.Del(ctx, hashKey).Result()
	if err != nil {
		xlog.Errorf("Failed to delete balance update for txnCounter %d: %v", txnCounter, err)
		return err
	}

	return nil
}