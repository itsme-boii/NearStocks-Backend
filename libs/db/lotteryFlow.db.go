package db

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/rand"
	"time"
)

type LotteryFlowDB struct{}

// Create inserts a new record into the LotteryFlowTable
func (*LotteryFlowDB) Create(entry *LotteryFlowTable) *LotteryFlowTable {
	if tx := db.Create(entry); tx.Error != nil {
		xlog.Errorf("Error creating lottery entry: %v", tx.Error)
		return nil
	}
	return entry
}

// GetRandomEntryByTimeRange retrieves a random entry for a specified time range
func (*LotteryFlowDB) GetRandomEntryByTimeRange(startTime, endTime time.Time) (*LotteryFlowTable, error) {
	var entries []LotteryFlowTable
	tx := db.Where("created_at >= ? AND created_at <= ?", startTime, endTime).Find(&entries)

	if tx.Error != nil {
		xlog.Errorf("Error retrieving entries in time range: %v", tx.Error)
		return nil, tx.Error
	}

	if len(entries) == 0 {
		xlog.Warnf("No entries found in the specified time range")
		return nil, fmt.Errorf("no entries found in the specified time range")
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := r.Intn(len(entries))
	return &entries[randomIndex], nil
}

// Retrieve all lottery codes for a specific date range
func (*LotteryFlowDB) GetLotteryCodesForDateRange(startTime, endTime time.Time) ([]string, error) {
	var lotteryCodes []string
	if err := db.Model(&LotteryFlowTable{}).
		Where("created_at >= ? AND created_at < ?", startTime, endTime).
		Pluck("lottery_code", &lotteryCodes).Error; err != nil {
		xlog.Errorf("Error fetching lottery codes: %v", err)
		return nil, err
	}
	return lotteryCodes, nil
}

// GetLotteryCodeIfExists retrieves the lottery code for a given SubaccountId within a specified time range if it exists.
// It returns a boolean indicating existence and the lottery code if found.
func (*LotteryFlowDB) GetLotteryCodeIfExists(subaccountId string, startTime, endTime time.Time) (bool, string, error) {
	var lotteryCode string
	tx := db.Model(&LotteryFlowTable{}).
		Select("lottery_code").
		Where("subaccount_id = ? AND created_at >= ? AND created_at < ?", subaccountId, startTime, endTime).
		Limit(1).
		Pluck("lottery_code", &lotteryCode)

	if tx.Error != nil {
		xlog.Errorf("Error fetching lottery code for subaccount ID %s: %v", subaccountId, tx.Error)
		return false, "", tx.Error
	}

	// Check if a lottery code was retrieved
	exists := lotteryCode != ""
	return exists, lotteryCode, nil
}
