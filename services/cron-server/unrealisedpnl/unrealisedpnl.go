package unrealisedpnl

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient       = xredis.GetRedisClient()
	mutex             sync.Mutex // Mutex to ensure only one execution at a time
	excludedMap       = make(map[string]struct{})
	excludedSubLoaded bool
)

func InitializeUnrealisedPnlCron() {
	// Load excluded subaccounts from the environment at application startup
	excludedSubaccounts := os.Getenv("EXCLUDED_SUBACCOUNTS")
	if excludedSubaccounts == "" {
		xlog.Errorf("Unrealised PnL Cron - EXCLUDED_SUBACCOUNTS environment variable not found, exiting...")
		excludedSubLoaded = false
		return
	}
	excludedSubLoaded = true
	for _, id := range strings.Split(excludedSubaccounts, ",") {
		excludedMap[id] = struct{}{}
	}
}

func StartUnrealisedPnlCron() {
	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called to clean up resources

	// Check if excluded subaccounts were loaded
	if !excludedSubLoaded {
		xlog.Errorf("Unrealised Pnl Cron - exiting unrealised pnl cron since excluded subaccounts env variable is not set")
		return // Gracefully exit if not loaded
	}

	unrealisedPnlCronIntervalStr := os.Getenv("UNREALISED_PNL_CRON_INTERVAL")
	// Convert string to an integer
	unrealisedPnlCronInterval, err := strconv.Atoi(unrealisedPnlCronIntervalStr)
	if err != nil {
		fmt.Printf("Error converting UNREALISED_PNL_CRON_INTERVAL to integer: %v\n", err)
		return
	}

	for {
		select {
		case <-ctx.Done(): // Listen for cancellation
			xlog.Infof("Unrealised Pnl Cron - stopping cron due to context cancellation")
			return
		default:
			xlog.Infof("Unrealised PnL Cron - fetching positions...")
			// Run FetchOpenPositions function with context
			go FetchOpenPositions(ctx, cancel)

			// Wait before starting the next run
			time.Sleep(time.Duration(unrealisedPnlCronInterval) * time.Second)
		}
	}
}

func FetchOpenPositions(ctx context.Context, cancel context.CancelFunc) {
	mutex.Lock() // Acquire the lock to ensure no parallel execution
	defer mutex.Unlock()

	// Get the last recorded row ID from Redis
	id, err := getLastIdFromRedis(ctx)
	if err != nil {
		errString := fmt.Sprintf("Unrealised PnL Cron %v", err)
		xlog.Errorf(errString)
		xclient.GlobalDiscordClient.SendWebhookMessage(errString)
		cancel() // Cancel the context to stop the cron
		return
	}

	// Get the position map from Redis
	positionMap, err := getPositionMapFromRedis(ctx)
	if err != nil {
		errString := fmt.Sprintf("Unrealised PnL Cron - error fetching position map from Redis: %v", err)
		xlog.Errorf(errString)
		xclient.GlobalDiscordClient.SendWebhookMessage(errString)
		cancel() // Cancel the context to stop the cron
		return
	}

	sb := subaccount.NewSubaccountBalanceImpl()

	batchSize := 250
	for {
		xlog.Infof("Unrealised PnL Cron - fetching %v rows from %v", batchSize, id)
		// Fetch the next batch of rows from the database
		fills, err := (&db.FillDB{}).GetRowsAfterID(id, batchSize)
		if err != nil {
			xlog.Errorf("Unrealised Pnl Cron - error while fetching rows from fills db, aborting...")
			return
		}

		// Break the loop if no more rows are found
		if len(fills) == 0 {
			xlog.Infof("Unrealised Pnl Cron - no fills rows returned, exiting...")
			break
		}

		subaccountsCovered := make(map[string]bool)

		for _, fill := range fills {
			if _, exists := subaccountsCovered[fill.SubaccountId]; exists {
				// Skip processing if this subaccount has already been covered in this batch
				id = uint64(fill.ID)
				continue
			}

			subaccountsCovered[fill.SubaccountId] = true

			// Check if the current subaccount ID is in the exclusion list
			// Remove all existing positions of this subaccount from the position map
			if _, exists := excludedMap[fill.SubaccountId]; exists {
				for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
					key := fmt.Sprintf("%s-%d", fill.SubaccountId, productId)
					delete(positionMap, key)
				}
				id = uint64(fill.ID)
				continue // Skip this fill if it is in the exclusion list
			}

			subaccountHex, err := cutils.SubaccountIdToHex(fill.SubaccountId)
			if err != nil {
				msg := fmt.Sprintf("Unrealised Pnl Cron - error converting subaccount ID to hex: %v", err)
				xclient.GetGlobalDiscordClient().SendWebhookMessage(msg)
				xlog.Errorf(msg)
				id = uint64(fill.ID)
				continue
			}

			// Iterate through all positions of user and update the values
			_, err1 := xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(subaccountHex), func() (*xredis.NOOP, error) {
				balance := sb.MustGetSubaccountBalancesFromIds([]string{subaccountHex})[0]

				// Iterate through all perp positions of the user and update the position map
				for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
					positionKey := fmt.Sprintf("%s-%d", fill.SubaccountId, productId)
					pos, exists := balance.PerpBalances[productId]
					if !exists || pos.Amountx18.Sign() == 0 {
						delete(positionMap, positionKey)
					} else {
						positionMap[positionKey] = &ctypes.PositionSummary{
							SubaccountID: fill.SubaccountId,
							MarketID:     uint(productId),
							TotalAmount:  pos.Amountx18,
							OpenPrice:    pos.GetAvgOpenPricex18().String(),
						}
					}
				}

				return nil, nil
			})

			if err1 != nil {
				xlog.Errorf("Unrealised Pnl Cron - error fetching balance from redis for subaccount %s: %v", fill.SubaccountId, err1)
			}

			// Update the last processed ID
			id = uint64(fill.ID)
		}

		// Save the updated position map and last ID to Redis
		if err := storePositionMapInRedis(ctx, positionMap); err != nil {
			errString := fmt.Sprintf("Unrealised PnL Cron - error storing position map in Redis: %v", err)
			xlog.Errorf(errString)
			xclient.GlobalDiscordClient.SendWebhookMessage(errString)
			cancel() // Cancel the context to stop the cron
			return
		}
		if err := setLastIdToRedis(ctx, id); err != nil {
			errString := fmt.Sprintf("Unrealised PnL Cron - error storing last ID in Redis: %v", err)
			xlog.Errorf(errString)
			xclient.GlobalDiscordClient.SendWebhookMessage(errString)
			cancel() // Cancel the context to stop the cron
			return
		}

		// Sleep for 2 seconds before the next iteration to avoid too much load on the database.
		time.Sleep(2 * time.Second)
	}
}

func getLastIdFromRedis(ctx context.Context) (uint64, error) {
	key := xredis.GetUnrealisedPnlFillDbIdKey()

	id, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		} else {
			return 0, fmt.Errorf("error while fetching last recorded row ID in redis :%v", err)
		}
	}

	idInt, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error while converting last recorded row ID into uint64 :%v", err)
	}
	return idInt, nil
}

func setLastIdToRedis(ctx context.Context, id uint64) error {
	key := xredis.GetUnrealisedPnlFillDbIdKey()

	// Convert the id to a string before storing it in Redis
	idStr := strconv.FormatUint(id, 10)
	err := redisClient.Set(ctx, key, idStr, 0).Err()
	if err != nil {
		return fmt.Errorf("error while setting last recorded row ID in redis: %v", err)
	}
	return nil
}

func getPositionMapFromRedis(ctx context.Context) (map[string]*ctypes.PositionSummary, error) {
	positionMapKey := xredis.GetPositionsMapKey()
	val, err := redisClient.Get(ctx, positionMapKey).Result()
	if err != nil {
		// Return an empty map if key is not found
		if err == redis.Nil {
			return make(map[string]*ctypes.PositionSummary), nil
		}
		return nil, err
	}

	var positionMap map[string]*ctypes.PositionSummary
	if err := json.Unmarshal([]byte(val), &positionMap); err != nil {
		return nil, err
	}
	return positionMap, nil
}

func storePositionMapInRedis(ctx context.Context, positionMap map[string]*ctypes.PositionSummary) error {
	positionMapKey := xredis.GetPositionsMapKey()
	data, err := json.Marshal(positionMap)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, positionMapKey, data, 0).Err()
}
