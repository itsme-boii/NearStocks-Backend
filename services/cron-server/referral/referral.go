package referralCron

import (
	"context"
	"math/big"
	"strconv"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/referralUtils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

var (
	FillTable         = &db.FillDB{} // Replace BatchTable with FillDB
	ReferralUserTable = &db.ReferralDB{}
)

// NOTE: Function uses hardcoded BROKER ID = 1
// StartReferralSyncCron starts the referral sync cron job
func StartReferralSyncCron() {
	brokerId := uint(1)

	xlog.Infof("Referral Sync Cron job started")
	c := cron.New(cron.WithSeconds())
	// Cron job runs every 1 hour
	_, err := c.AddFunc("0 0 * * * *", func() {

		xlog.Infof("Referral Sync Cron job running: %v", time.Now())

		ctx := context.Background()
		key := xredis.GetReferralCronStartIndex()
		startIndex := uint64(0)
		startIndexStr, err := xredis.GetRedisClient().Get(ctx, key).Result()
		if err == redis.Nil {
			// If key is not found in Redis, set the startIndex to the max ID from fill_tables
			xlog.Infof("Key not found in Redis: %v, fetching max ID from fill_tables", key)

			// Fetch the max ID (last processed ID till the current time)
			maxID, err := FillTable.GetMaxID()
			if err != nil {
				xlog.Errorf("Error fetching max ID from fill_tables: %v", err)
				return
			}

			// Set the startIndex to maxID
			startIndex = maxID

			// Store this maxID as the startIndex in Redis for future use
			err = xredis.GetRedisClient().Set(ctx, key, strconv.FormatUint(startIndex, 10), 0).Err()
			if err != nil {
				xlog.Errorf("Error storing max ID in Redis: %v", err)
				c.Stop()
				return
			}

			xlog.Infof("Successfully set the max ID %d as startIndex in Redis", startIndex)
		} else if err != nil {
			xlog.Errorf("Error fetching start index from Redis: %v", err)
			return
		} else {
			// Convert the retrieved start index string to uint64
			startIndexVal, err := strconv.ParseUint(startIndexStr, 10, 64)
			if err != nil {
				xlog.Errorf("Error parsing start index from Redis: %v", err)
				return
			}
			startIndex = startIndexVal
		}

		xlog.Infof("Referral Sync startIndex: %d", startIndex)

		// Fetch the max ID (last processed ID till the current time)
		maxID, err := FillTable.GetMaxID()
		if err != nil {
			xlog.Errorf("Error fetching max ID from fill_tables: %v", err)
			return
		}
		xlog.Infof("Max ID for this run: %d", maxID)

		limit := uint(1000) // Batch size of 1000
		for startIndex < maxID {
			// Fetch unique subaccount data in the current batch
			referralSubaccountData, err := FillTable.GetUniqueSubAccountsWithTotal(brokerId, startIndex, limit)
			if err != nil {
				xlog.Errorf("Error fetching subaccount data for batch starting at ID %d: %v", startIndex, err)
				// Store the last processed max ID in Redis and stop further processing
				err = xredis.GetRedisClient().Set(ctx, key, strconv.FormatUint(startIndex, 10), 0).Err()
				if err != nil {
					xlog.Errorf("Error storing last start index in Redis after failure: %v", err)
				}
				return
			}

			if len(referralSubaccountData) == 0 {
				xlog.Infof("No subaccount data found for batch starting at ID %d, continuing to next batch.", startIndex)
				startIndex += uint64(limit)

				// Store the updated startIndex in Redis
				err = xredis.GetRedisClient().Set(ctx, key, strconv.FormatUint(startIndex, 10), 0).Err()
				if err != nil {
					xlog.Errorf("Error storing new start index in Redis: %v", err)
					return
				}
				// Continue to the next iteration
				continue
			}

			// Process the referral subaccounts with the retrieved referralSubaccountData
			if err := processReferralSubAccounts(referralSubaccountData); err != nil {
				xlog.Errorf("Error processing referral subaccounts for batch starting at ID %d: %v", startIndex, err)
				// Store the last processed max ID in Redis and stop further processing
				err = xredis.GetRedisClient().Set(ctx, key, strconv.FormatUint(startIndex, 10), 0).Err()
				if err != nil {
					xlog.Errorf("Error storing last start index in Redis after failure: %v", err)
					xclient.GlobalDiscordClient.SendWebhookMessage("******************** ALERT >>>>>>>>>>> Stopped Referral Cron Server due issue in setting ID <<<<<<<<<<<<< ALERT **************")
					c.Stop()
				}
				return
			}

			// Move to the next batch by adding limit to the startIndex
			startIndex += uint64(limit)

			// After each successful batch, update Redis with the current startIndex
			err = xredis.GetRedisClient().Set(ctx, key, strconv.FormatUint(startIndex, 10), 0).Err()
			if err != nil {
				xlog.Errorf("Error storing new start index in Redis: %v", err)
				xclient.GlobalDiscordClient.SendWebhookMessage("******************** ALERT >>>>>>>>>>> Stopped Referral Cron Server due issue in setting ID <<<<<<<<<<<<< ALERT **************")
				c.Stop()
				return
			}
		}

		// After processing all batches, update Redis with the final max ID
		err = xredis.GetRedisClient().Set(ctx, key, strconv.FormatUint(maxID, 10), 0).Err()
		if err != nil {
			xlog.Errorf("Error storing final max ID in Redis: %v", err)
			xclient.GlobalDiscordClient.SendWebhookMessage("******************** ALERT >>>>>>>>>>> Stopped Referral Cron Server due issue in setting ID <<<<<<<<<<<<< ALERT **************")
			c.Stop()
			return
		}

		xlog.Infof("Referral Sync completed, last processed ID: %d", maxID)
	})
	if err != nil {
		xlog.Errorf("Error adding cron function: %v", err)
	}

	// Start the cron scheduler
	delay := 3 * time.Minute
	time.AfterFunc(delay, func() {
		c.Start()
		xlog.Infof("Referral Sync Cron - Cron started after a delay of %v", delay)
	})
	xlog.Infof("Referral Sync Cron scheduler started")

	// Keep the cron scheduler running in the background
	select {}
}

// processReferralSubAccounts processes the complete referralSubaccountData for each subaccount
func processReferralSubAccounts(referralSubaccountData []db.ReferralSubaccountData) error {
	// Log the number of subaccounts being processed
	xlog.Infof("Processing %d subaccounts for referral sync", len(referralSubaccountData))

	// Initialize ReferralUtils
	referralUtils := referralUtils.NewReferralUtils()

	// Create a map to hold Ethereum addresses mapped to SubaccountIDs
	ethereumAddresses := make(map[string]string)

	// Extract Ethereum addresses for all subaccounts in one go
	var userAddresses []string
	for _, data := range referralSubaccountData {
		ethAddress, err := cutils.ExtractEthereumAddress(data.SubaccountID)
		if err != nil {
			xlog.Errorf("Error extracting Ethereum address from SubaccountID %s: %v", data.SubaccountID, err)
			continue // Skip if there's an error extracting the Ethereum address
		}
		userAddresses = append(userAddresses, ethAddress)
		ethereumAddresses[data.SubaccountID] = ethAddress // Map SubaccountID to Ethereum address
	}

	// Call GetReferrerMappingForActiveUsers to filter active users with referrer
	referrerMapping, err := ReferralUserTable.GetReferrerMappingForActiveUsers(userAddresses)
	if err != nil {
		xlog.Errorf("Error fetching referrer mapping for active users: %v", err)
		return err
	}

	// Fetch the rebate mapping from Redis
	rebateMapping, err := referralUtils.GetReferralRebateMapping()
	if err != nil {
		xlog.Errorf("Error fetching referral rebate mapping: %v", err)
		return err
	}

	// Default rebate percentage as a BigInt (10%)
	defaultRebatePercentage := ctypes.NewBigInt(big.NewInt(contractUtils.REFERRER_REBATE_PERCENTAGE)) // Default 10% using cutils.MulxCust

	// Final array to hold referral history data
	var referralHistories []db.ReferralHistoryTable

	// Process each ReferralSubaccountData entry
	for _, data := range referralSubaccountData {
		// Get the Ethereum address for the current SubaccountID from the map
		userAddress, exists := ethereumAddresses[data.SubaccountID]
		if !exists {
			xlog.Errorf("Ethereum address not found for SubaccountID %s", data.SubaccountID)
			continue // Skip if the Ethereum address was not extracted correctly
		}

		// Only proceed if the userAddress is in the referrer mapping
		referrerUserAddress, referrerExists := referrerMapping[userAddress]
		if !referrerExists {
			continue // Skip if no referrer is found
		}

		// Determine the rebate percentage as a BigInt
		rebatePercentage := defaultRebatePercentage // Default to 10%
		if rebatePercentageStr, rebateExists := rebateMapping[referrerUserAddress]; rebateExists {
			// Convert rebatePercentageStr to BigInt
			rebatePercentage = ctypes.NewBigIntFromString(rebatePercentageStr)
			if rebatePercentage.IsNil() {
				xlog.Warnf("Error parsing rebate percentage for user %s. Falling back to default rebate percentage.", userAddress)
				rebatePercentage = defaultRebatePercentage // Fall back to default if parsing fails
			}
		}

		// Calculate referrer rewards: referrerRewards = (fees * rebatePercentage) / 100
		feeBigInt := data.TotalFees // Assume TotalFees is in ctypes.BigInt format
		// First, multiply the fees by the rebate percentage

		multipliedRewards := feeBigInt.Mul(rebatePercentage)

		// Then, divide by 100 to get the final rewards
		referrerRewards := multipliedRewards.Div(ctypes.NewBigInt(cutils.MulxCust(big.NewInt(10), 1)))
		// Prepare the referral history entry
		referralHistory := db.ReferralHistoryTable{
			RefereeUserID:     userAddress,
			ReferrerUserID:    referrerUserAddress,
			RefereeFees:       data.TotalFees,                   // Fees in ctypes.BigInt (10^18 format)
			RefereeVolume:     data.TotalValue,                  // Total Value, assuming in ctypes.BigInt format
			RefereeReward:     ctypes.NewBigIntFromString("0"),  // Referee reward set to 0
			ReferrerRewards:   referrerRewards,                  // Calculated referrer rewards
			RefereeProductID:  contractUtils.REFERRE_REWARD_ID,  // current product ID for referee
			ReferrerProductID: contractUtils.REFERRER_REWARD_ID, // current product ID for referrer
		}

		// Add the history to the list of histories to be processed
		referralHistories = append(referralHistories, referralHistory)

		// Call AddReferralReward for referrerUserAddress with ReferrerRewards
		referralRewardData := db.ReferralRewardTable{
			UserAddress:  referrerUserAddress,
			ProductID:    contractUtils.REFERRER_REWARD_ID,
			RewardAmount: referrerRewards, // Referrer rewards as reward amount
		}

		// Initialize ReferralRewardDB and call AddReferralReward
		referralRewardDB := db.ReferralRewardDB{}
		if _, err := referralRewardDB.AddReferralReward(referralRewardData); err != nil {
			xlog.Errorf("Error adding/updating referral reward for user %s: %v. Continuing with next.", referrerUserAddress, err)
			continue // Continue to next entry in case of error
		}
	}

	// Add/update referral history in the database
	referralHistoryDB := db.ReferralHistoryDB{}
	for _, history := range referralHistories {
		if _, err := referralHistoryDB.AddReferralHistory(history); err != nil {
			xlog.Errorf("Error adding/updating referral history for user %s: %v. Continuing with next.", history.RefereeUserID, err)
			continue // Continue to next entry in case of error
		}
	}

	xlog.Infof("Successfully processed referral sync for %d users", len(referralHistories))
	return nil
}
