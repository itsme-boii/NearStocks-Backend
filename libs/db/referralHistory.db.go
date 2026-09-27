package db

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"math/big"
)

// ReferralHistoryDB handles interactions with the referral history table.
type ReferralHistoryDB struct{}

type ReferralStats struct {
	RefereeUserID           string        `json:"referee_user_id"`
	TotalFees               ctypes.BigInt `json:"total_fees"`
	TotalVolume             ctypes.BigInt `json:"total_volume"`
	TotalReferrerRewards    ctypes.BigInt `json:"total_referrer_rewards"`
	NetVolume               ctypes.BigInt `json:"net_volume"`
	TotalDistinctReferees   int           `json:"total_distinct_referees"`
	TotalRefereeFeesOverall ctypes.BigInt `json:"total_referrer_rewards_overall"`
}

// AddReferralHistory either adds a new referral history entry or updates the existing entry by adding fees, volume, and rewards.
func (*ReferralHistoryDB) AddReferralHistory(referralHistoryData ReferralHistoryTable) (*ReferralHistoryTable, error) {
	var existingEntry ReferralHistoryTable

	// Check if an entry already exists for the combination of RefereeUserID, ReferrerUserID, RefereeProductID, and ReferrerProductID
	if err := db.Where("referee_user_id = ? AND referrer_user_id = ? AND referee_product_id = ? AND referrer_product_id = ?",
		referralHistoryData.RefereeUserID, referralHistoryData.ReferrerUserID,
		referralHistoryData.RefereeProductID, referralHistoryData.ReferrerProductID).
		First(&existingEntry).Error; err == nil {
		// Entry exists, update the referral fees, volume, and rewards by adding the new values

		// Add the new referral fees, volume, and rewards to the existing ones
		existingEntry.RefereeFees = existingEntry.RefereeFees.Add(referralHistoryData.RefereeFees)
		existingEntry.RefereeVolume = existingEntry.RefereeVolume.Add(referralHistoryData.RefereeVolume)
		existingEntry.RefereeReward = existingEntry.RefereeReward.Add(referralHistoryData.RefereeReward)
		existingEntry.ReferrerRewards = existingEntry.ReferrerRewards.Add(referralHistoryData.ReferrerRewards)

		// Save the updated entry
		if err := db.Save(&existingEntry).Error; err != nil {
			return nil, err
		}

		return &existingEntry, nil
	}

	// Entry does not exist, create a new one
	if err := db.Create(&referralHistoryData).Error; err != nil {
		return nil, err
	}

	return &referralHistoryData, nil
}

// CalculateReferralStats calculates stats for a given referrerUserID
func (*ReferralHistoryDB) CalculateReferralStats(referrerUserID string) (*ReferralStats, []ReferralStats, error) {
	var results []ReferralStats
	totalVolume := ctypes.NewBigInt(big.NewInt(0)) // Initialize totalVolume to 0
	totalRefereeFees := ctypes.NewBigInt(big.NewInt(0))

	// Query to get distinct referee users and calculate total fees, volume, and referrer rewards for each referee
	if err := db.Table("referral_history_tables").
		Select("referee_user_id, COALESCE(SUM(CAST(referee_fees AS NUMERIC)), 0) as total_fees, COALESCE(SUM(CAST(referee_volume AS NUMERIC)), 0) as total_volume, COALESCE(SUM(CAST(referrer_rewards AS NUMERIC)), 0) as total_referrer_rewards").
		Where("referrer_user_id = ?", referrerUserID).
		Group("referee_user_id").
		Scan(&results).Error; err != nil {
		return nil, nil, err
	}

	// Aggregate totals across all referees
	for _, result := range results {
		// Ensure non-nil values for fees, volume, and rewards
		if result.TotalFees.IsNil() {
			result.TotalFees = ctypes.NewBigInt(big.NewInt(0))
		}
		if result.TotalVolume.IsNil() {
			result.TotalVolume = ctypes.NewBigInt(big.NewInt(0))
		}
		if result.TotalReferrerRewards.IsNil() {
			result.TotalReferrerRewards = ctypes.NewBigInt(big.NewInt(0))
		}

		totalVolume = totalVolume.Add(result.TotalVolume)
		totalRefereeFees = totalRefereeFees.Add(result.TotalFees)
	}

	// Count distinct referees
	var totalDistinctReferees int64
	if err := db.Table("referral_history_tables").
		Where("referrer_user_id = ?", referrerUserID).
		Count(&totalDistinctReferees).Error; err != nil {
		return nil, nil, err
	}

	// Final aggregated data
	stats := ReferralStats{
		NetVolume:               totalVolume,
		TotalDistinctReferees:   int(totalDistinctReferees),
		TotalRefereeFeesOverall: totalRefereeFees, // New total referee fees
	}

	return &stats, results, nil
}
