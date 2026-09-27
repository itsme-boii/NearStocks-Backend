package db

import (
	"errors"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"math/big"
)

type ReferralRewardDB struct{}

// AddReferralReward either adds a new referral reward entry or updates the RewardAmount if the entry already exists.
func (*ReferralRewardDB) AddReferralReward(referralRewardData ReferralRewardTable) (*ReferralRewardTable, error) {
	var existingEntry ReferralRewardTable

	// Check if an entry already exists for the given UserAddress and ProductID (uint)
	if err := db.Where("user_address = ? AND product_id = ?", referralRewardData.UserAddress, referralRewardData.ProductID).
		First(&existingEntry).Error; err == nil {
		// Entry exists, update the reward amount by adding the new amount

		// Add the new reward amount to the existing one
		existingEntry.RewardAmount = existingEntry.RewardAmount.Add(referralRewardData.RewardAmount)

		// Save the updated entry
		if err := db.Save(&existingEntry).Error; err != nil {
			return nil, err
		}

		return &existingEntry, nil
	}

	// Entry does not exist, create a new one with ClaimedAmount set to 0
	referralRewardData.ClaimedAmount = ctypes.NewBigInt(big.NewInt(0)) // Set ClaimedAmount to 0 for new entries

	// Create the new entry
	if err := db.Create(&referralRewardData).Error; err != nil {
		return nil, err
	}

	return &referralRewardData, nil
}

// UpdateReferralClaimAmount updates the claimed amount by adding the new claimed amount without any limit check on the total reward amount.
func (*ReferralRewardDB) UpdateReferralClaimAmount(userAddress string, productID uint, newClaimAmount ctypes.BigInt) (*ReferralRewardTable, error) {
	var existingEntry ReferralRewardTable

	// Check if an entry exists for the given UserAddress and ProductID (uint)
	if err := db.Where("user_address = ? AND product_id = ?",
		userAddress, productID).First(&existingEntry).Error; err != nil {
		return nil, errors.New("reward entry not found")
	}

	// Add the new claimed amount to the existing claimed amount without checking against RewardAmount
	existingEntry.ClaimedAmount = existingEntry.ClaimedAmount.Add(newClaimAmount)

	// Save the updated entry
	if err := db.Save(&existingEntry).Error; err != nil {
		return nil, err
	}

	return &existingEntry, nil
}

// GetReferralReward retrieves a specific row from ReferralRewardTable based on UserAddress and ProductID.
func (*ReferralRewardDB) GetReferralReward(userAddress string, productID uint) (*ReferralRewardTable, error) {
	var rewardEntry ReferralRewardTable

	// Find the specific entry based on the combination of UserAddress and ProductID (uint)
	if err := db.Where("user_address = ? AND product_id = ?",
		userAddress, productID).First(&rewardEntry).Error; err != nil {
		return nil, errors.New("reward entry not found")
	}

	return &rewardEntry, nil
}
