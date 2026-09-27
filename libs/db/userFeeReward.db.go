package db

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
)

type UserRewardsDB struct{}

// Create inserts a new record into the UserRewards table
func (*UserRewardsDB) Create(reward *UserFeeRewardsTable) *UserFeeRewardsTable {
	if tx := db.Create(reward); tx.Error != nil {
		xlog.Errorf("Error creating user reward: ", tx.Error)
		return nil
	}
	return reward
}

// GetAllArbRewardsBySubaccount fetches all rewards by subaccount and maps them by FillTableId
func (*UserRewardsDB) GetAllArbRewardsBySubaccount(subaccountId string) map[uint]ctypes.BigInt {
	var rewards []UserFeeRewardsTable
	rewardsMap := make(map[uint]ctypes.BigInt)

	// Fetch all rewards for the subaccount
	if tx := db.Where("subaccount_id = ?", subaccountId).Find(&rewards); tx.Error != nil {
		xlog.Errorf("Error fetching rewards for subaccount: ", tx.Error)
		return nil
	}

	// Map rewards by FillTableId
	for _, reward := range rewards {
		rewardsMap[reward.FillTableId] = reward.FeeBonusx18
	}

	return rewardsMap
}

func (*UserRewardsDB) GetTotalBonusForSubaccount(subaccountId string, symbol string) (*ctypes.BigInt, error) {
	var totalFeeBonusStr string

	// Query to calculate the total fee bonus for the given SubaccountId and provided symbol in the UserFeeRewardsTable
	err := db.Raw(`
		SELECT COALESCE(SUM(CAST(fee_bonusx18 AS NUMERIC)), 0) as total_fee_bonus
		FROM user_fee_rewards_tables
		WHERE subaccount_id = ? AND symbol = ? AND is_claimed = ?
	`, subaccountId, symbol, false).Scan(&totalFeeBonusStr).Error

	if err != nil {
		return nil, err
	}

	// Convert the string result into big.Int
	totalFeeBonus := new(big.Int)
	_, ok := totalFeeBonus.SetString(totalFeeBonusStr, 10) // Base 10 conversion
	if !ok {
		return nil, fmt.Errorf("failed to convert total fee bonus to big.Int")
	}

	return &ctypes.BigInt{Val: totalFeeBonus}, nil
}

func (*UserRewardsDB) MarkRewardsAsClaimed(subaccountId string, symbol string) error {
	if tx := db.Model(&UserFeeRewardsTable{}).
		Where("subaccount_id = ? AND symbol = ? AND is_claimed = ?", subaccountId, symbol, false).
		Update("is_claimed", true); tx.Error != nil {
		xlog.Errorf("Error marking rewards as claimed for subaccount %s and symbol %s: %v", subaccountId, symbol, tx.Error)
		return tx.Error
	}
	return nil
}
