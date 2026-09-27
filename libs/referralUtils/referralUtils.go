package referralUtils

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"sort"

	"github.com/redis/go-redis/v9"
)

type ReferralUtils struct {
	redisClient *redis.Client
}

func NewReferralUtils() *ReferralUtils {
	return &ReferralUtils{
		redisClient: xredis.GetRedisClient(),
	}
}

// The key is an HSET where the fields are user addresses, and the values are fee percentages.
func (r *ReferralUtils) GetReferralRebateMapping() (map[string]string, error) {

	key := xredis.GetReferralRebateMapping()

	rebateMapping, err := r.redisClient.HGetAll(context.Background(), key).Result()
	if err != nil {
		xlog.Errorf("Referral Utils - Error fetching referral rebate mapping for key %s: %v", key, err)
		return nil, err
	}
	if len(rebateMapping) == 0 {
		xlog.Infof("Referral Utils - Key %s not found in Redis, returning empty map", key)
		return make(map[string]string), nil
	}
	return rebateMapping, nil
}

// AddOrUpdateReferralRebateMapping adds or updates the referral rebate mapping for a user address
// If the user address exists, it updates the fee percentage; otherwise, it adds a new entry.
func (r *ReferralUtils) AddOrUpdateReferralRebateMapping(userAddress string, feePercentage string) error {
	key := xredis.GetReferralRebateMapping()

	// Use HSET to add or update the mapping with feePercentage as a string
	err := r.redisClient.HSet(context.Background(), key, userAddress, feePercentage).Err()
	if err != nil {
		xlog.Errorf("Referral Utils - Error adding or updating referral rebate mapping for key %s, userAddress %s: %v", key, userAddress, err)
		return err
	}

	xlog.Infof("Referral Utils - Successfully added/updated referral rebate mapping for key %s, userAddress %s", key, userAddress)
	return nil
}

func (r *ReferralUtils) GetReferralFeePercentage(userAddress string) (string, error) {
	key := xredis.GetReferralRebateMapping() // Get the Redis key

	// Fetch the fee percentage for the given user address
	feePercentage, err := r.redisClient.HGet(context.Background(), key, userAddress).Result()
	if err != nil {
		if err == redis.Nil {
			// If the user address is not found, return an empty string
			xlog.Warnf("Referral Utils - No fee percentage found for userAddress %s", userAddress)
			return "", nil
		}
		xlog.Errorf("Referral Utils - Error fetching fee percentage for userAddress %s: %v", userAddress, err)
		return "", err
	}

	return feePercentage, nil
}

func (r *ReferralUtils) CheckReferralCodeExists(referralCode string) (bool, string, error) {
	key := xredis.GetAffiliateReferralCodesKey()

	exists, err := r.redisClient.HExists(context.Background(), key, referralCode).Result()
	if err != nil {
		xlog.Errorf("Referral Utils - Error checking referral code existence for code %s: %v", referralCode, err)
		return false, "", err
	}

	if !exists {
		return false, "", nil
	}

	affiliateSubaccountId, err := r.redisClient.HGet(context.Background(), key, referralCode).Result()
	if err != nil {
		xlog.Errorf("Referral Utils - Error getting affiliate subaccount ID for code %s: %v", referralCode, err)
		return false, "", err
	}

	return true, affiliateSubaccountId, nil
}

// GetAffiliateCodeBySubaccountId gets the affiliate code for a given subaccount ID from Redis
func (r *ReferralUtils) GetAffiliateCodeBySubaccountId(subaccountId string) (string, error) {
	key := xredis.GetAffiliateReferralCodesKey()

	// Get all affiliate referral codes from Redis
	affiliateCodes, err := r.redisClient.HGetAll(context.Background(), key).Result()
	if err != nil {
		xlog.Errorf("Referral Utils - Error fetching affiliate referral codes for key %s: %v", key, err)
		return "", err
	}

	// Filter affiliate codes that match the given subaccount ID
	var matchingCodes []string
	for referralCode, affiliateSubaccountId := range affiliateCodes {
		if affiliateSubaccountId == subaccountId {
			matchingCodes = append(matchingCodes, referralCode)
		}
	}

	// Return the first matching code (alphabetically sorted) if any exist
	if len(matchingCodes) > 0 {
		// Sort matching codes alphabetically
		sort.Strings(matchingCodes)
		return matchingCodes[0], nil
	}

	return "", fmt.Errorf("affiliate code not found for subaccount %s", subaccountId)
}
