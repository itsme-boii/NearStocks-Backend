package db

import (
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type StakingDB struct{}

func (*StakingDB) Insert(subaccountId, amount, cumulativeEarningsRate, action, earnings, offset, transientEarnings, totalClaimedEarningsx18 string) (*StakingTable, error) {
	newStake := &StakingTable{
		SubaccountId:            subaccountId,
		Amount:                  amount,
		CumulativeEarningsRate:  cumulativeEarningsRate,
		Action:                  action,
		Earnings:                earnings,
		Offsetx18:               offset,
		TransientEarningsx18:    transientEarnings,
		TotalClaimedEarningsx18: totalClaimedEarningsx18,
	}
	result := db.Create(newStake)
	if result.Error != nil {
		return nil, result.Error
	}
	return newStake, nil
}

func (*StakingDB) GetAllBySubaccountId(subaccountId string) ([]StakingTable, error) {
	var stakes []StakingTable
	result := db.Where("subaccount_id = ?", subaccountId).Order("created_at DESC").Find(&stakes)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Return an empty slice without an error
			return []StakingTable{}, nil
		}
		return nil, result.Error
	}

	// No error, return the stakes slice (could be empty if no records were found)
	return stakes, nil
}

func (*StakingDB) GetLastEntryBySubaccountHex(subaccountHex string) *StakingTable {
	var stake StakingTable
	return GetDbObjOrNil(db.Where("subaccount_id = ?", subaccountHex).Order("created_at DESC").First(&stake), &stake)
}

func (*StakingDB) GetTotalEarnings(subaccountId string) (*big.Int, error) {
	var stakes []StakingTable
	result := db.Where("subaccount_id = ?", subaccountId).Find(&stakes)
	if result.Error != nil {
		return nil, result.Error
	}

	totalEarnings := big.NewInt(0)

	// Iterate over the stakes and sum up the earnings
	for _, stake := range stakes {
		earning := new(big.Int)
		earning, ok := earning.SetString(stake.Earnings, 10) // Assuming "Earnings" is stored as a string
		if !ok {
			return nil, fmt.Errorf("failed to parse earnings for subaccount ID: %s", subaccountId)
		}
		totalEarnings.Add(totalEarnings, earning)
	}

	return totalEarnings, nil
}

func (s *StakingDB) GetUnstakeSumOfPendingUnstake(subaccountId string) (*big.Int, error) {
	// Load the minutes value from the .env file
	minuteStr := os.Getenv("UNSTAKE_MINUTES")
	minutes, err := strconv.Atoi(minuteStr)
	if err != nil || minutes <= 0 {
		xlog.Errorf("Invalid UNSTAKE_MINUTES value: %s, defaulting to 60 minutes", minuteStr)
		minutes = 60 // Default to 60 minutes if not set or invalid
	}

	// Calculate the time range for the unstake check
	minutesAgo := time.Now().Add(-time.Duration(minutes) * time.Minute)

	xlog.Debugf("Getting unstake sum for the last %d minutes (since %s) for subaccount: %s", minutes, minutesAgo, subaccountId)

	var unstakes []StakingTable

	result := db.Where("subaccount_id = ? AND action = ? AND created_at >= ?", subaccountId, "unstake", minutesAgo).Find(&unstakes)
	if result.Error != nil {
		return nil, result.Error
	}

	unstakeSum := big.NewInt(0)
	for _, unstake := range unstakes {
		amountInt := big.NewInt(0)
		if _, ok := amountInt.SetString(unstake.Amount, 10); !ok {
			return nil, fmt.Errorf("invalid amount in unstake for subaccount ID: %s", subaccountId)
		}
		unstakeSum.Add(unstakeSum, amountInt)
	}

	return unstakeSum, nil
}

func (*StakingDB) CountDistinctStakers() (int64, error) {
	var count int64

	// Execute the query to count distinct subaccount_ids
	result := db.Table("staking_tables").
		Where("deleted_at IS NULL AND action = ?", "stake").
		Select("COUNT(DISTINCT subaccount_id)").
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

// FIXME: Very expensive operations here - cache or optimize this
func (s *StakingDB) CalculateTotalStakeAmount() (*big.Int, *big.Int, *big.Int, error) {
	// Initialize variables
	last24HourUnstake := big.NewInt(0)
	last24HourStake := big.NewInt(0)
	totalStakeAmount := big.NewInt(0)
	totalUnstakeAmount := big.NewInt(0)

	// Fetch all staking and unstaking records from the database
	var stakes []StakingTable
	result := db.Raw(`SELECT * FROM staking_tables where deleted_at is NULL`).Scan(&stakes)
	if result.Error != nil {
		return nil, nil, nil, result.Error
	}

	// Iterate over all records to calculate totals
	for _, stake := range stakes {
		amountInt := big.NewInt(0)
		if _, ok := amountInt.SetString(stake.Amount, 10); !ok {
			return nil, nil, nil, fmt.Errorf("invalid amount in record ID: %d", stake.ID)
		}

		// Check action and calculate totals
		if stake.Action == "stake" {
			// Sum total stake amount
			totalStakeAmount.Add(totalStakeAmount, amountInt)

			// Check if it was staked in the last 24 hours
			if stake.CreatedAt.After(time.Now().Add(-24 * time.Hour)) {
				last24HourStake.Add(last24HourStake, amountInt)
			}
		} else if stake.Action == "unstake" {
			// Sum total unstake amount
			totalUnstakeAmount.Add(totalUnstakeAmount, amountInt)

			// Check if it was unstaked in the last 24 hours
			if stake.CreatedAt.After(time.Now().Add(-24 * time.Hour)) {
				last24HourUnstake.Add(last24HourUnstake, amountInt)
			}
		}
	}

	// Calculate the net staked amount
	netStakeAmount := new(big.Int).Sub(totalStakeAmount, totalUnstakeAmount)

	// Return the results
	return netStakeAmount, last24HourUnstake, last24HourStake, nil
}

// SetDeletedAtById sets the deleted_at field for the transaction with the given id
func (*StakingDB) SetDeletedAtById(id uint, deletedAt time.Time) error {
	result := db.Model(&StakingTable{}).Where("id = ?", id).Update("deleted_at", deletedAt)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (*StakingDB) GetTotalStakingRewards() (*big.Int, error) {
	var totalClaimedEarnings []string
	result := db.Raw(`SELECT total_claimed_earningsx18 FROM staking_tables WHERE action = 'claim' and total_claimed_earningsx18 is not NULL and deleted_at is NULL`).Scan(&totalClaimedEarnings)
	if result.Error != nil {
		return nil, result.Error
	}

	totalRewards := big.NewInt(0)

	// Iterate over the stakes and sum up the earnings
	for _, totalClaimedEarning := range totalClaimedEarnings {
		earning := new(big.Int)
		_, ok := earning.SetString(totalClaimedEarning, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse totalClaimedEarning")
		}
		totalRewards.Add(totalRewards, earning)
	}

	return totalRewards, nil
}
