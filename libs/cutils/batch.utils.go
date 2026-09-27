package cutils

import (
	"context"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"math/big"
	"strings"

	"github.com/redis/go-redis/v9"
)

func IsSequencerRunning() bool {
	paused, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetPauseSequencerKey()).Result()
	if err == redis.Nil {
		return true
	} else if err != nil {
		return false
	}
	return paused != "1"
}

func IsBatchingCronPaused() bool {
	paused, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetPauseBatchingCronKey()).Result()
	if err == redis.Nil {
		return false
	} else if err != nil {
		return false
	}
	return paused == "1"
}
func IsSubAccountPaused(subaccountId string) bool {
	ctx := context.Background()
	// Get the key for paused subaccounts
	exist, err := xredis.GetRedisClient().HExists(ctx, xredis.GetSubacountPauseSequencerKey(), xredis.GetSubaccountLvlSequencerField(subaccountId)).Result()
	if err != nil {
		xlog.Errorf("Error checking if subaccount is paused: %v", err)
		return false
	}

	return exist
}
func IsSubAccountClaimLogXPaused(subaccountId string) bool {
	ctx := context.Background()
	// Get the key for paused subaccounts
	exist, err := xredis.GetRedisClient().HExists(ctx, xredis.GetSubacountPauseClaimLogxKey(), xredis.GetSubaccountClaimFlowPausedField(subaccountId)).Result()
	if err != nil {
		xlog.Errorf("Error checking if subaccount has claimed logx: %v", err)
		return false
	}

	return exist
}

func HasSubaccountGotExtraFeeBonus(subaccountId string) bool {
	ctx := context.Background()
	// Get the key for paused subaccounts
	exist, err := xredis.GetRedisClient().HExists(ctx, xredis.GetSubacountBonusfeeKey(), xredis.GetSubaccountBonusFeeField(subaccountId)).Result()
	if err != nil {
		xlog.Errorf("Error checking if subaccount has received extra fee bonus: %v", err)
		return false
	}

	return exist
}

func AddTotalArbRewardsAccumulated(rewardAmount *big.Int) (bool, error) {
	ctx := context.Background()

	// Fetch the current total ARB rewards from Redis
	totalRewardsStr, err := xredis.GetRedisClient().Get(ctx, xredis.GetTotalArbRewardsAccumulates()).Result()
	if err != nil && err != redis.Nil {
		xlog.Errorf("Error fetching total ARB rewards accumulated: %v", err)
		return false, err
	}

	// Convert the fetched value (if any) to *big.Int
	totalRewards := new(big.Int)
	if totalRewardsStr != "" {
		totalRewards, _ = totalRewards.SetString(totalRewardsStr, 10)
	}

	// Define the limit as 8000 * 10^18
	maxLimit := new(big.Int).Mul(big.NewInt(10000), big.NewInt(1e18))

	// Check if the new total would exceed the limit
	newTotal := new(big.Int).Add(totalRewards, rewardAmount)
	if newTotal.Cmp(maxLimit) > 0 {
		// If the new total exceeds the limit, do not add and return false
		xlog.Infof("Total ARB rewards exceeded the limit of 8000 * 10^18")
		return false, nil
	}

	// Add the new reward amount to the total rewards and update in Redis
	err = xredis.GetRedisClient().Set(ctx, xredis.GetTotalArbRewardsAccumulates(), newTotal.String(), 0).Err()
	if err != nil {
		xlog.Errorf("Error updating total ARB rewards accumulated: %v", err)
		return false, err
	}

	// Return true if the addition was successful and below the limit
	return true, nil
}

func AddTotalLogxRewardsAccumulated(rewardAmount *big.Int) (bool, error) {
	ctx := context.Background()

	// Fetch the current total LOGX rewards from Redis
	totalRewardsStr, err := xredis.GetRedisClient().Get(ctx, xredis.GetTotalLogxRewardsAccumulated()).Result()
	if err != nil && err != redis.Nil {
		xlog.Errorf("Error fetching total LOGX rewards accumulated: %v", err)
		return false, err
	}

	// Convert the fetched value (if any) to *big.Int
	totalRewards := new(big.Int)
	if totalRewardsStr != "" {
		totalRewards, _ = totalRewards.SetString(totalRewardsStr, 10)
	}

	// Define the limit as 8000 * 10^18
	maxLimit := new(big.Int).Mul(big.NewInt(6000000), big.NewInt(1e18))

	// Check if the new total would exceed the limit
	newTotal := new(big.Int).Add(totalRewards, rewardAmount)
	if newTotal.Cmp(maxLimit) > 0 {
		// If the new total exceeds the limit, do not add and return false
		xlog.Infof("Total LOGX rewards exceeded the limit of 6000000 * 10^18")
		return false, nil
	}

	// Add the new reward amount to the total rewards and update in Redis
	err = xredis.GetRedisClient().Set(ctx, xredis.GetTotalLogxRewardsAccumulated(), newTotal.String(), 0).Err()
	if err != nil {
		xlog.Errorf("Error updating total LOGX rewards accumulated: %v", err)
		return false, err
	}

	// Return true if the addition was successful and below the limit
	return true, nil
}

func IsWhitelistedSubaccount(subaccountId string) bool {
	ctx := context.Background()
	// Get the key for paused subaccounts
	exist, err := xredis.GetRedisClient().HExists(ctx, xredis.GetWhitelistHashKey(), xredis.GetWhitelistSubaccountField(subaccountId)).Result()
	if err != nil {
		xlog.Errorf("Error subaccount is not whitelisted: %v", err)
		return false
	}

	return exist
}

func PauseSequencerForSubaccount(subaccountHex string) {
	subaccountID := strings.ToLower(subaccountHex)
	err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetSubacountPauseSequencerKey(), xredis.GetSubaccountLvlSequencerField(subaccountID), 1).Err()
	if err != nil {
		xlog.Errorf("Batching Cron - Error pausing sequencer for subaccount: %s - %v", subaccountHex, err)
	} else {
		xlog.Infof("Batching Cron - Sequencer is paused for subaccount: %s", subaccountHex)
	}
}

func PauseSubaccountClaimLogXFlow(subaccountHex string) {
	subaccountID := strings.ToLower(subaccountHex)
	err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetSubacountPauseClaimLogxKey(), xredis.GetSubaccountClaimFlowPausedField(subaccountID), 1).Err()
	if err != nil {
		xlog.Errorf("Claim Logx API - Error pausing claim flow for subaccount: %s - %v", subaccountHex, err)
	} else {
		xlog.Infof("Claim Logx API - Claim flow is paused for subaccount: %s", subaccountHex)
	}
}

func SubaccountDoneFirstBonusTrade(subaccountHex string) {
	err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetSubacountBonusfeeKey(), xredis.GetSubaccountBonusFeeField(subaccountHex), 1).Err()
	if err != nil {
		xlog.Errorf("Claim Logx API - Error pausing claim flow for subaccount: %s - %v", subaccountHex, err)
	} else {
		xlog.Infof("Claim Logx API - Claim flow is paused for subaccount: %s", subaccountHex)
	}
}

func IsSubaccountRegisteredForTracking(subaccountIdHex string) bool {
	ctx := context.Background()
	// Get the key for paused subaccounts
	exist, err := xredis.GetRedisClient().HExists(ctx, xredis.GetSubaccountTrackingKey(), xredis.GetSubaccountTrackingField(subaccountIdHex)).Result()
	if err != nil {
		xlog.Errorf("Error checking if subaccount is registered for tracking: %v", err)
		return false
	}

	return exist
}
