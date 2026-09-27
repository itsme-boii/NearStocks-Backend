package lastdeposit

import (
	"context"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

var lastCheckpoint uint

func InitializeLastDepositCheckpoint() error {
	xlog.Infof("Deposit Cron job started")
	redisClient := xredis.GetRedisClient()
	checkpointKey := xredis.GetLastCachedUserDepositKey()

	// Get last processed checkpoint from Redis
	lastCheckpointStr, err := redisClient.Get(context.Background(), checkpointKey).Result()
	if err != nil && err != redis.Nil {
		xlog.Errorf("Error fetching checkpoint from Redis: %v", err)
		return err // Return error if Redis fetching fails
	}

	// Convert the checkpoint string to uint, or default to 0 for first-time runs
	if lastCheckpointStr != "" {
		parsedCheckpoint, err := strconv.ParseUint(lastCheckpointStr, 10, 64)
		if err != nil {
			xlog.Warnf("Invalid checkpoint found, starting from scratch")
			lastCheckpoint = 0
		} else {
			lastCheckpoint = uint(parsedCheckpoint)
			xlog.Infof("Initialized last checkpoint from Redis: %d", lastCheckpoint)
		}
	} else {
		xlog.Infof("No previous checkpoint found, starting from the beginning")
		lastCheckpoint = 0
	}

	return nil // Return nil to indicate success
}

func UpdateSubaccountLastDeposit() {
	currentTime := time.Now().Format(time.RFC3339)
	xlog.Infof("Deposit cache cron job started at %s and last checkpoint was %d", currentTime, lastCheckpoint)
	ctx := context.Background()
	redisClient := xredis.GetRedisClient()
	checkpointKey := xredis.GetLastCachedUserDepositKey()

	// Fetch new deposit entries from DepositWithdrawTable
	newDeposits := (&db.DepositWithdrawDB{}).GetDepositsAfterCertainId(lastCheckpoint)

	var maxID uint = lastCheckpoint // Initialize maxID with the last checkpoint
	for _, deposit := range *newDeposits {
		subaccountId := deposit.SubaccountId
		sourceChainId := deposit.SourceChainID

		// Fetch the existing entry from SubaccountLastDepositTable
		lastDeposit := (&db.SubaccountLastDepositDB{}).GetBySubaccountAndChain(subaccountId, sourceChainId)
		// If there is no existing entry, it means it's the first deposit for this subaccount and sourceChainId
		if lastDeposit == nil {
			// Create a new entry for the first deposit
			xlog.Infof("First deposit found for subaccount %s on chain %d", subaccountId, sourceChainId)
			// newDeposit := (&db.SubaccountLastDepositDB{}).Create(subaccountId, sourceChainId, deposit.CreatedAt)
			// if newDeposit == nil {
			// 	xlog.Errorf("Error creating first deposit entry for subaccount %s and chain %d", subaccountId, sourceChainId)
			// } else {
			// 	xlog.Infof("First deposit recorded for subaccount %s on chain %d", subaccountId, sourceChainId)
			// }
		}

		// Update the max ID encountered during this loop
		if deposit.ID > maxID {
			maxID = deposit.ID
		}
	}

	// Update the last checkpoint to the most recent processed deposit (maxID)
	if maxID > lastCheckpoint {
		redisClient.Set(ctx, checkpointKey, strconv.FormatUint(uint64(maxID), 10), 0)
		xlog.Infof("Updated checkpoint to ID: %d", maxID)

		// Update the in-memory checkpoint for future cron job runs
		lastCheckpoint = maxID
	} else {
		xlog.Infof("No new deposits to process.")
	}
	currentTime = time.Now().Format(time.RFC3339)
	xlog.Infof("Deposit cache cron job iteration finished at %s and with last checkpoint was %d", currentTime, lastCheckpoint)
}

func StartLastDepositCron() {
	// Define the time zone (e.g., UTC)
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err)
	}
	c := cron.New(cron.WithSeconds(), cron.WithLocation(loc))

	// cron job to run every hour in the specified time zone (UTC)
	c.AddFunc("0 0 * * * *", func() {
		UpdateSubaccountLastDeposit()
	})

	c.Start()
	// Run the job immediately once when the service starts
	go UpdateSubaccountLastDeposit()

	// Keep the cron scheduler running in the background
	select {}
}
