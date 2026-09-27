package services

// The purpose of this service is to add some locked token amount to the HASH set for subaccount lock on redis
// So, hash key is LOCK_BALANCE_<subaccount> and field is <PRODUCT>_<product_id> and value is <TOKEN_SYMBOL>_<amount>
// Methods will receive subaccount id, product, product id, amount and collateral token symbol
// Methods: LockBalance, UnlockBalance, UpdateLockBalance

// NOTE: We are using mutex locks to lock the subaccount so, only one update at a time can happen for a subaccount

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type BalanceLockerService struct {
	RedisClient *redis.Client
}

// ToDo - should we add distributed locking here ?
//
//	All redis transactions seem to be atomic
func NewBalanceLockerService() *BalanceLockerService {
	return &BalanceLockerService{
		RedisClient: xredis.GetRedisClient(),
	}
}

// Value should be of format <PRODUCT_ID>_<AMOUNT>
func parseLockedAmount(value string) (productId uint32, lockedAmount string, err error) {
	// value is in the format <PRODUCT_ID>_<AMOUNT>
	// Split value by underscore and parse the amount
	// Return the token symbol and locked amount
	vals := strings.Split(value, "_")
	if len(vals) != 2 {
		return 0, "0", fmt.Errorf("invalid value format")
	}

	_productId, err := strconv.ParseUint(vals[0], 10, 32)
	if err != nil {
		return 0, "0", fmt.Errorf("invalid product ID format: %v", err)
	}

	productId = uint32(_productId)
	lockedAmount = vals[1]
	return productId, lockedAmount, nil
}

// NOTE: Currently, this logic is only used for locking QUOTE_SPOT balance (i.e ETH-USDC)
// NOTE: Assume this can be wrong for other product ids
// Contains locking logic over subaccount so, only one update at a time can happen for a subaccount
func (s *BalanceLockerService) AtomicLockBalance(subaccountID string, entity ctypes.LockerEntity, entityId string, ProductId uint32, quoteAmountx18 *big.Int) error {
	start := time.Now()
	fmt.Printf("Locking balance for subaccount: %s, entity: %s, entityId: %s, ProductId: %d, quoteAmountx18: %v\n", subaccountID, entity, entityId, ProductId, quoteAmountx18)

	mutex, err := xredis.AcquireLockWithRetry(fmt.Sprintf("balance-lock-%s", subaccountID))
	if err != nil {
		xlog.Errorf("Failed to acquire lock:", err)
		cutils.LogTime(start, "AtomicLockBalance")
		return err
	}
	defer func() {
		// Release lock
		if err := xredis.ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock:", err)
		}
	}()

	ctx := context.Background()

	_, err = s.RedisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		err := s.updateLockedAmountEntityLevel(pipe, subaccountID, entity, entityId, ProductId, quoteAmountx18)

		if err != nil {
			fmt.Printf("Unable to lock balance: %v\n", err)
			cutils.LogTime(start, "AtomicLockBalance")
			return err
		}

		err = s.updateLockedAmountInSpot(pipe, subaccountID, ProductId, quoteAmountx18)
		if err != nil {
			fmt.Printf("Unable to lock balance: %v\n", err)
			cutils.LogTime(start, "AtomicLockBalance")
			return err
		}
		//add under subAccount level as well in redis
		cutils.LogTime(start, "AtomicLockBalance")
		return nil
	})

	cutils.LogTime(start, "AtomicLockBalance")
	return err
}

func (s *BalanceLockerService) updateLockedAmountEntityLevel(pipe redis.Pipeliner, subaccountID string, entity ctypes.LockerEntity, entityId string, ProductId uint32, quoteAmountx18 *big.Int) error {
	key := xredis.GetBalanceLockerKey(subaccountID)
	field := xredis.GetBalanceLockerField(entity, entityId)

	existingValue, err := s.RedisClient.HGet(ctx, key, field).Result()
	// If the value is not found, set the existing value to 0
	if err == redis.Nil {
		if quoteAmountx18.Sign() < 0 {
			fmt.Printf("unlock value(%d) greater than locked balance(%d)", quoteAmountx18, 0)
			return fmt.Errorf("unlock value greater than locked balance")
		} else {
			existingValue = xredis.GetBalanceLockerValue(ProductId, "0")
		}
	} else if err != nil {
		fmt.Printf("Unable to get locked token balance: %v\n", err)
		return err
	}

	_, existingBalance, err := parseLockedAmount(existingValue)
	if err != nil {
		fmt.Printf("Failed to extract locked balance from redis string: %v, updateLockedAmountEntityLevel func ", existingValue)
		return err
	}
	existingBalanceInt, success := new(big.Int).SetString(existingBalance, 10)
	if !success {
		fmt.Printf("cannot create big.Int from string existing locked balance %d", existingBalanceInt)
		return fmt.Errorf("cannot create big.Int from string existing locked balance %d", existingBalanceInt)
	}
	newLockedBalance := new(big.Int).Add(existingBalanceInt, quoteAmountx18)
	//todo: throw error if newLocked is negative
	if newLockedBalance.Sign() < 0 {
		fmt.Printf("unlock value(%d) greater than locked balance(%d)", quoteAmountx18, existingBalanceInt)
		return fmt.Errorf("unlock value greater than locked balance")
	}
	value := xredis.GetBalanceLockerValue(ProductId, newLockedBalance.String())

	err = pipe.HSet(context.Background(), key, field, value).Err()
	if err != nil {
		return err
	}

	return nil
}

func (s *BalanceLockerService) updateLockedAmountInSpot(pipe redis.Pipeliner, subaccountID string, ProductId uint32, quoteAmountx18 *big.Int) error {
	hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
	field := fmt.Sprintf("%d", ProductId)

	// Retrieve the existing token balance for the specified productId
	existingBalance, err := s.RedisClient.HGet(ctx, hashKey, field).Result()
	if err == redis.Nil {
		if quoteAmountx18.Sign() < 0 {
			fmt.Printf("unlock value(%d) greater than locked balance(%d)", quoteAmountx18, 0)
			return fmt.Errorf("unlock value greater than locked balance")
		} else {
			existingBalance = ""
		}
	} else if err != nil {
		fmt.Printf("Unable to get locked token balance: %v\n", err)
		return err
	}

	type Balance struct {
		Available string `json:"available"`
		Locked    string `json:"locked"`
	}

	var balance Balance
	if existingBalance != "" {
		if err := json.Unmarshal([]byte(existingBalance), &balance); err != nil {
			return fmt.Errorf("failed to unmarshal existing balance: %v", err)
		}
	} else {
		balance = Balance{
			Available: "0",
			Locked:    "0",
		}
	}

	existingLockedBigInt := new(big.Int)
	if balance.Locked != "" {
		_, ok := existingLockedBigInt.SetString(balance.Locked, 10)
		if !ok {
			return fmt.Errorf("failed to parse existing lock balance: %s", balance.Available)
		}
	}

	// Convert the new token balance to big.Int
	newLockedBalanceBigInt := new(big.Int)
	newLockedBalanceBigInt.Add(quoteAmountx18, existingLockedBigInt)
	balance.Locked = newLockedBalanceBigInt.String()

	updatedBalanceJSON, err := json.Marshal(balance)
	if err != nil {
		fmt.Printf("Unable to marshal updated balance: %v\n", err)
		return err
	}

	// Store the updated balance in Redis
	err = pipe.HSet(ctx, hashKey, field, updatedBalanceJSON).Err()
	if err != nil {
		fmt.Printf("Unable to update locked token balance: %v\n", err)
		return err
	}
	return nil

}

func (s *BalanceLockerService) UnlockFullBalance(subaccountID string, entity ctypes.LockerEntity, entityId string) error {
	start := time.Now()
	fmt.Printf("Unlocking full balance for subaccount: %s, entity: %s, entityId: %s\n", subaccountID, entity, entityId)

	mutex, err := xredis.AcquireLockWithRetry(fmt.Sprintf("balance-lock-%s", subaccountID))
	if err != nil {
		xlog.Errorf("Failed to acquire lock:", err)
		cutils.LogTime(start, "UnlockFullBalance")
		return err
	}
	defer func() {
		// Release lock
		if err := xredis.ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock:", err)
		}
	}()

	ctx := context.Background()

	key := xredis.GetBalanceLockerKey(subaccountID)
	field := xredis.GetBalanceLockerField(entity, entityId)

	// Get the existing locked balance
	existingValue, err := s.RedisClient.HGet(ctx, key, field).Result()
	// If the value is not found we should throw error. So, no need for redis nil check

	// FIXME: Change this
	if err == redis.Nil {
		fmt.Printf("No locked token balances found for subaccount: %v\n", subaccountID)
		cutils.LogTime(start, "UnlockFullBalance")
		return nil
	}

	if err != nil {
		fmt.Printf("Unable to get locked token balance: %v\n", err)
		cutils.LogTime(start, "UnlockFullBalance")
		return err
	}

	_, err = s.RedisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// Delete the locked balance from the redis
		err = s.RedisClient.HDel(ctx, key, field).Err()
		if err != nil {
			cutils.LogTime(start, "UnlockFullBalance")
			return err
		}

		ProductId, existingBalance, err := parseLockedAmount(existingValue)
		if err != nil {
			fmt.Printf("Failed to extract locked balance from redis string: %v, UnlockFullBalance", existingValue)
			cutils.LogTime(start, "UnlockFullBalance")
			return err
		}
		existingBalanceInt, success := new(big.Int).SetString(existingBalance, 10)
		if !success {
			fmt.Printf("cannot create big.Int from string existing locked balance %d", existingBalanceInt)
			cutils.LogTime(start, "UnlockFullBalance")
			return fmt.Errorf("cannot create big.Int from string existing locked balance %d", existingBalanceInt)
		}

		quoteAmountx18Neg := new(big.Int).Neg(existingBalanceInt)

		err = s.updateLockedAmountInSpot(pipe, subaccountID, ProductId, quoteAmountx18Neg)
		if err != nil {
			fmt.Printf("Unable to unlock balance: %v\n", err)
			cutils.LogTime(start, "UnlockFullBalance")
			return err
		}

		cutils.LogTime(start, "UnlockFullBalance")
		return nil
	})

	cutils.LogTime(start, "UnlockFullBalance")
	return err
}

// Assumption: amount will be from the same quote token
func (s *BalanceLockerService) UnlockPartialBalanceWithoutLock(pipe redis.Pipeliner, subaccountID string, entity ctypes.LockerEntity, entityId string, quoteAmountx18 *big.Int) error {
	fmt.Printf("Unlocking partial balance for subaccount: %s, entity: %s, entityId: %s, quoteAmountx18: %v\n", subaccountID, entity, entityId, quoteAmountx18)
	if quoteAmountx18.Sign() == -1 {
		return fmt.Errorf("amount to unlock should be greater than or equal to 0")
	}

	quoteAmountx18Neg := new(big.Int).Neg(quoteAmountx18)

	ctx := context.Background()

	key := xredis.GetBalanceLockerKey(subaccountID)
	field := xredis.GetBalanceLockerField(entity, entityId)

	// NOTE: We won't use pipe here to immediately get the value
	existingValue, err := s.RedisClient.HGet(ctx, key, field).Result()
	// If the value is not found we should throw error. So, no need to handle redis nil check (key not found check)
	if err != nil {
		fmt.Printf("Unable to get locked token balance: %v\n", err)
		return err
	}

	ProductId, _, err := parseLockedAmount(existingValue)
	if err != nil {
		fmt.Printf("Failed to extract locked balance from redis string %v, UnlockPartialBalance", existingValue)
		return err
	}

	err = s.updateLockedAmountEntityLevel(pipe, subaccountID, entity, entityId, ProductId, quoteAmountx18Neg)

	if err != nil {
		fmt.Printf("Unable to lock balance: %v\n", err)
		return err
	}

	err = s.updateLockedAmountInSpot(pipe, subaccountID, ProductId, quoteAmountx18Neg)
	if err != nil {
		fmt.Printf("Unable to lock balance: %v\n", err)
		return err
	}
	//add under subAccount level as well in redis

	return nil
}

// Note - not adding redis distributed locking to this function since we are not expecting race conditions
func (s *BalanceLockerService) GetTotalLockBalanceMap(subaccountID string) (tokenToTotalAmount map[uint32]*big.Int, err error) {
	key := xredis.GetBalanceLockerKey(subaccountID)

	tokenToTotalAmount = make(map[uint32]*big.Int)

	lockedBalances, err := s.RedisClient.HGetAll(context.Background(), key).Result()
	if err == redis.Nil {
		fmt.Printf("No lock balance map found for subaccount: %v\n", subaccountID)
		return tokenToTotalAmount, nil
	} else if err != nil {
		return nil, err
	}

	for _, value := range lockedBalances {
		ProductId, lockedAmountStr, err := parseLockedAmount(value)
		if err != nil {
			fmt.Printf("Invalid amount value stored in redis: %s", value)
			panic(err)
		}
		lockedAmountx18, success := new(big.Int).SetString(lockedAmountStr, 10)
		if !success {
			return nil, fmt.Errorf("invalid locked amount value")
		}
		if tokenToTotalAmount[ProductId] == nil {
			tokenToTotalAmount[ProductId] = new(big.Int).Set(lockedAmountx18)
		} else {
			tokenToTotalAmount[ProductId] = new(big.Int).Add(tokenToTotalAmount[ProductId], lockedAmountx18)
		}
	}

	return tokenToTotalAmount, nil
}

func (t *BalanceLockerService) UnlockAllLockedBalance(subaccountID string) error {
	start := time.Now()
	fmt.Printf("Unlocking all locked balance for subaccount: %s\n", subaccountID)

	mutex, err := xredis.AcquireLockWithRetry(fmt.Sprintf("balance-lock-%s", subaccountID))
	if err != nil {
		xlog.Errorf("Failed to acquire lock:", err)
		cutils.LogTime(start, "UnlockAllLockedBalance")
		return err
	}
	defer func() {
		// Release lock
		if err := xredis.ReleaseLock(mutex); err != nil {
			xlog.Errorf("Failed to release lock:", err)
		}
	}()

	ctx := context.Background()

	key := xredis.GetBalanceLockerKey(subaccountID)

	// Get the existing locked balance
	err = t.RedisClient.Del(ctx, key).Err()
	if err != nil {
		fmt.Printf("Unable to unlock all locked token balance: %v\n", err)
		cutils.LogTime(start, "UnlockAllLockedBalance")
		return err
	}
	xlog.Infof("Key deleted: %v", key)

	key2 := xredis.GetBalanceKey(subaccountID)

	err = t.RedisClient.Del(ctx, key2).Err()
	if err != nil {
		fmt.Printf("Unable to unlock all locked token balance: %v\n", err)
		cutils.LogTime(start, "UnlockAllLockedBalance")
		return err
	}
	xlog.Infof("Key deleted: ", key2)
	cutils.LogTime(start, "UnlockAllLockedBalance")
	return nil
}

type ErrorUnlockingBalance struct {
	TakerError bool
	MakerError bool
}

func (e *ErrorUnlockingBalance) Error() string {
	if e.TakerError && e.MakerError {
		return "Error unlocking both taker and maker balance"
	}
	if e.TakerError {
		return "Error unlocking taker balance"
	}
	if e.MakerError {
		return "Error unlocking maker balance"
	}
	return ""
}

func (e *ErrorUnlockingBalance) IsError() bool {
	return e.TakerError || e.MakerError
}

// This function will return nil or ErrorUnlockingBalance error
func (t *BalanceLockerService) AtomicUpdateSubaccountForMatch(req types.UpdateSubaccountForMatchRequest) *ErrorUnlockingBalance {
	// Add debug to monitor the state
	start := time.Now()

	ctx := context.Background()

	unlockMaker := !req.Maker.SkipUnlock && !contractUtils.IsAMMAccount(req.Maker.SubaccountId)
	unlockTaker := !req.Taker.SkipUnlock && !contractUtils.IsAMMAccount(req.Taker.SubaccountId)

	// Locks are taken before the pipeline and released after it EXECs. (Before, they were taken
	// inside the TxPipelined callback and released by its defers, i.e. before EXEC, so two matches
	// on one order could both read the same locked amount and one unlock was lost.)
	if unlockMaker {
		mutex1, err := xredis.AcquireLockWithRetry(fmt.Sprintf("balance-lock-%s", req.Maker.SubaccountId))
		if err != nil {
			xlog.Errorf("Failed to acquire lock:", err)
			cutils.LogTime(start, "AtomicUpdateSubaccountForMatch")
			return nil
		}
		defer func() {
			if err := xredis.ReleaseLock(mutex1); err != nil {
				xlog.Errorf("Failed to release lock:", err)
			}
		}()
	}
	if unlockTaker {
		mutex2, err := xredis.AcquireLockWithRetry(fmt.Sprintf("balance-lock-%s", req.Taker.SubaccountId))
		if err != nil {
			xlog.Errorf("Failed to acquire lock:", err)
			cutils.LogTime(start, "AtomicUpdateSubaccountForMatch")
			return nil
		}
		defer func() {
			if err := xredis.ReleaseLock(mutex2); err != nil {
				xlog.Errorf("Failed to release lock:", err)
			}
		}()
	}

	// Maybe FIXME: Not watching any key for now
	_, err := t.RedisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		var errUnlockTaker, errUnlockMaker error
		var errUnlock ErrorUnlockingBalance

		if unlockMaker {
			errUnlockMaker = t.UnlockPartialBalanceWithoutLock(pipe, req.Maker.SubaccountId, ctypes.LOCKER_ENTITY_ORDER, fmt.Sprintf("%d", req.Maker.OrderId), req.Maker.UnlockQuotex18)
			if errUnlockMaker != nil {
				xlog.Errorf("System fault: Unable to unlock maker locked balance: %v\n. This means either a) there is some issue in the api server related to locking of balance or b) the unlocking of balance is already done or c) Liquidation engine removed the locked balance for clearing some locked margin", errUnlockMaker)
				errUnlock.MakerError = true
			}
		}

		if unlockTaker {
			errUnlockTaker = t.UnlockPartialBalanceWithoutLock(pipe, req.Taker.SubaccountId, ctypes.LOCKER_ENTITY_ORDER, fmt.Sprintf("%d", req.Taker.OrderId), req.Taker.UnlockQuotex18)
			if errUnlockTaker != nil {
				xlog.Errorf("System fault: Unable to unlock taker locked balance: %v\n. This means either a) there is some issue in the api server related to locking of balance or b) the unlocking of balance is already done or c) Liquidation engine removed the locked balance for clearing some locked margin", errUnlockTaker)
				errUnlock.TakerError = true
			}
		}

		if errUnlock.IsError() {
			cutils.LogTime(start, "AtomicUpdateSubaccountForMatch")
			return &errUnlock
		}
		cutils.LogTime(start, "AtomicUpdateSubaccountForMatch")
		return nil
	})

	if err != nil {
		var errUnlock *ErrorUnlockingBalance
		if errors.As(err, &errUnlock) {
			cutils.LogTime(start, "AtomicUpdateSubaccountForMatch")
			return errUnlock
		} else {
			cutils.LogTime(start, "AtomicUpdateSubaccountForMatch")
			panic("This should never happen")
		}
	}
	cutils.LogTime(start, "AtomicUpdateSubaccountForMatch")
	return nil
}
