package db

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BatchDB struct {
}

func (b *BatchDB) Insert(subAccountID1, subAccountID2 string, txn, signature1, signature2 []byte, functionName string, transactionCounter uint, brokerId uint, skipBalanceInsert ...bool) *BatchTable {
	ctx := context.Background()
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID

	newBatch := &BatchTable{
		SubAccountID1:      strings.ToLower(subAccountID1),
		SubAccountID2:      strings.ToLower(subAccountID2),
		Transaction:        txn,
		Signature1:         signature1,
		Signature2:         signature2,
		FunctionName:       functionName,
		TransactionCounter: transactionCounter,
		Balance1:           "",
		Balance2:           "",
		Nonce:              "",
		BrokerId:           brokerId,
	}

	if !(len(skipBalanceInsert) > 0 && skipBalanceInsert[0]) {
		// TODO: Claim the balance update here using transactionCounter
		if functionName == transaction.MATCH_ORDERS_FN || functionName == transaction.LIQUIDATE_SUBACCOUNT_FN {
			updateBalances, err := transaction.ReadBalanceUpdateForTxn(transactionCounter)
			var err2 error
			var updateBalanceMap map[string]string
			if err == nil {
				updateBalanceMap, err2 = updateBalances.MapString()
			}
			// If there is err -> set empty
			if err != nil || err2 != nil {
				xlog.Errorf("Failed to read balance update for transaction counter %v: %v | %v", transactionCounter, err, err2)
				newBatch.Balance1 = ""
				newBatch.Balance2 = ""
			} else {
				newBatch.Balance1 = updateBalanceMap[newBatch.SubAccountID1]
				newBatch.Balance2 = updateBalanceMap[newBatch.SubAccountID2]
				// Delete the txn balance update from the redis
				err = transaction.DeleteBalanceUpdateForTxn(transactionCounter)
				if err != nil {
					xlog.Errorf("Failed to delete balance update for transaction counter %v: %v", transactionCounter, err)
				}
			}
		} else {
			// Update Balance1 and Nonce if subAccountID1 is present
			if newBatch.SubAccountID1 != "" {
				hashKey1 := fmt.Sprintf("v0_balance_%s", newBatch.SubAccountID1)
				balances1, err := xredis.GetRedisClient().HGetAll(ctx, hashKey1).Result()
				if err == redis.Nil {
					xlog.Warnf("No key in Redis to fetch balance for subAccountID1: %s", newBatch.SubAccountID1)
				} else if err != nil {
					xlog.Warnf("Failed to get balances for subAccountID1: %v", err)
				} else {
					balance1Bytes, err := json.Marshal(balances1)
					if err != nil {
						xlog.Warnf("Failed to marshal balance1: %v", err)
					} else {
						balance1Str := string(balance1Bytes)
						if balance1Str == "{}" {
							newBatch.Balance1 = ""
						} else {
							newBatch.Balance1 = balance1Str
						}
					}
				}

			}

			// Update Balance2 if subAccountID2 is present
			if newBatch.SubAccountID2 != "" {
				hashKey2 := fmt.Sprintf("v0_balance_%s", newBatch.SubAccountID2)
				balances2, err := xredis.GetRedisClient().HGetAll(ctx, hashKey2).Result()
				if err == redis.Nil {
					xlog.Warnf("No key in Redis to fetch balance for subAccountID2: %s", newBatch.SubAccountID2)
				} else if err != nil {
					xlog.Warnf("Failed to get balances for subAccountID2: %v", err)
				} else {
					balance2Bytes, err := json.Marshal(balances2)
					if err != nil {
						xlog.Warnf("Failed to marshal balance2: %v", err)
					} else {
						newBatch.Balance2 = string(balance2Bytes)
					}
				}
			}
		}

		// Fetch the nonce for SubAccountID1
		if newBatch.SubAccountID1 != "" && newBatch.SubAccountID1 != ammSubaccountID && newBatch.FunctionName != "FinaliseDeposit" {
			nonceKey1 := xredis.GetNonceKey(newBatch.SubAccountID1)
			nonce1, err := xredis.GetRedisClient().Get(ctx, nonceKey1).Result()
			if err == redis.Nil {
				xlog.Errorf("No nonce found in redis for subAccountID1: %s", newBatch.SubAccountID1)
				nonce1 = "0"
			} else if err != nil {
				xlog.Errorf("Failed to get nonce for subAccountID1: %v", err)
				nonce1 = "0"
			}
			newBatch.Nonce = nonce1
		}
		// Fetch the nonce for SubAccountID2
		if newBatch.SubAccountID2 != "" && newBatch.SubAccountID2 != ammSubaccountID && newBatch.FunctionName != "FinaliseDeposit" {
			nonceKey1 := xredis.GetNonceKey(newBatch.SubAccountID2)
			nonce1, err := xredis.GetRedisClient().Get(ctx, nonceKey1).Result()
			if err == redis.Nil {
				xlog.Errorf("No nonce found in redis for subAccountID2: %s", newBatch.SubAccountID2)
				nonce1 = "0"
			} else if err != nil {
				xlog.Errorf("Failed to get nonce for subAccountID2: %v", err)
				nonce1 = "0"
			}
			newBatch.Nonce = nonce1
		}

		// If transcation is perp tick then add funding rate in db from reddis
		if newBatch.SubAccountID1 == "" && newBatch.SubAccountID2 == "" && newBatch.FunctionName == "PerpTick" {
			var fundingRatesStr string
			for _, productID := range contractUtils.ALL_PERPS_ON_CONTRACT {
				symbol, exists := marketutils.GetFundingSymbolForProduct(productID)
				if !exists {
					xlog.Warnf("Symbol not found for productID %v", productID)
					continue
				}

				redisKey := xredis.GetCumulativeFundingRateKey(symbol)
				cumulativeFundingRateData := xredis.CumulativeFundingRateData{
					CumulativeFundingRate:      "0",
					CumulativeFundingTimestamp: "0",
				}

				// Fetch the cumulative funding rate from Redis
				cumulativeFundingRateDict, err := xredis.GetRedisClient().Get(ctx, redisKey).Result()
				if err != nil && err != redis.Nil {
					xlog.Errorf("error fetching cumulative funding rate from redis for product %v: %v", productID, err)
				}
				if err == nil && cumulativeFundingRateDict != "" {
					err = json.Unmarshal([]byte(cumulativeFundingRateDict), &cumulativeFundingRateData)
					if err != nil {
						xlog.Errorf("error unmarshalling cumulative funding rate from redis for product %v: %v", productID, err)
					}
				}

				// Add the funding rate to the string using productID
				fundingRatesStr += fmt.Sprintf("%d:%s;", productID, cumulativeFundingRateData.CumulativeFundingRate)
			}

			// Store the funding rates string in the database
			newBatch.Balance1 = fundingRatesStr
		}
	}

	// Create the new batch entry in the database
	result := db.Create(newBatch)
	if result.Error != nil {
		return nil
	}

	return newBatch
}

func (*BatchDB) MarkCompletedForIds(ids []uint, subIdx uint64) error {
	batchList := make([]BatchTable, len(ids))
	for i, id := range ids {
		batchList[i] = BatchTable{
			BaseTable: BaseTable{
				ID: id,
			},
		}
	}

	// Update the NSubmissionIdx field for the given IDs as well as deleted_at field
	err := db.Model(&batchList).Updates(map[string]interface{}{
		"n_submission_idx": subIdx,
		"deleted_at":       time.Now(),
	}).Error

	return err
}

// Architecture - We fetch all the transactions in order of transaction counter
// Then we cutoff the transactions after the first transaction with a transaction counter greater than the cutoff time
func (b *BatchDB) GetAllTransactionsWithIDs(limit int, subaccountsToExclude []string, cutoffTime time.Time) ([][]byte, [][]byte, [][]byte, []uint, []string, []string, uint, map[string]string, map[string]string, []uint, []string, error) {
	// Fetch batches from the database ordered by transaction_counter in ascending order
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	var batches []BatchTable
	var result *gorm.DB
	if len(subaccountsToExclude) > 0 {
		// NOTE: Not the most efficient way to exclude subaccounts, but it should be fine for now
		result = db.Where("sub_account_id1 NOT IN (?) AND sub_account_id2 NOT IN (?)", subaccountsToExclude, subaccountsToExclude).Order("transaction_counter ASC").Limit(limit).Find(&batches)
	} else {
		result = db.Order("transaction_counter ASC").Limit(limit).Find(&batches)
	}

	if result.Error != nil {
		return nil, nil, nil, nil, nil, nil, 0, nil, nil, nil, nil, result.Error
	}

	transactions := make([][]byte, 0)
	signatures1 := make([][]byte, 0)
	signatures2 := make([][]byte, 0)
	subaccounts1 := make([]string, 0)
	subaccounts2 := make([]string, 0)
	ids := make([]uint, 0)
	txnCounters := make([]uint, 0)
	func_names := make([]string, 0)

	// Map to store the latest balance and nonce JSON string for each subaccountID
	latestBalances := make(map[string]string)
	latestNonces := make(map[string]string)

	var maxTransactionCounter uint

	// Iterate over the fetched batches
	for i, batch := range batches {

		// Check if the transaction counter's created_at is less than the cutoff time
		if batch.CreatedAt.After(cutoffTime) {
			xlog.Infof("Skipping %v no. of txns out of %v txns due to cutoff time", len(batches)-i, len(batches))
			break
		}

		// Fill the data as usual
		transactions = append(transactions, batch.Transaction)
		signatures1 = append(signatures1, batch.Signature1)
		signatures2 = append(signatures2, batch.Signature2)
		ids = append(ids, batch.ID)
		subaccounts1 = append(subaccounts1, batch.SubAccountID1)
		subaccounts2 = append(subaccounts2, batch.SubAccountID2)
		txnCounters = append(txnCounters, batch.TransactionCounter)
		func_names = append(func_names, batch.FunctionName)

		// Store the JSON string balance and nonce in the latestBalances and latestNonces maps
		if batch.SubAccountID1 != "" {
			latestBalances[batch.SubAccountID1] = batch.Balance1
			if batch.SubAccountID1 != ammSubaccountID {
				latestNonces[batch.SubAccountID1] = batch.Nonce
			}
		}

		if batch.SubAccountID2 != "" {
			latestBalances[batch.SubAccountID2] = batch.Balance2
			if batch.SubAccountID2 != ammSubaccountID {
				latestNonces[batch.SubAccountID2] = batch.Nonce
			}
		}

		// for perp tick transcation
		if batch.SubAccountID1 == "" && batch.SubAccountID2 == "" && batch.FunctionName == "PerpTick" {
			latestBalances["cumulative_funding_rates"] = batch.Balance1
		}

		// Update the maxTransactionCounter
		maxTransactionCounter = batch.TransactionCounter
	}

	return transactions, signatures1, signatures2, ids, subaccounts1, subaccounts2, maxTransactionCounter, latestBalances, latestNonces, txnCounters, func_names, nil
}

// function to get an array of unique SubAccountID after a given ID range to the latest
func (*BatchDB) GetUniqueSubAccountIDsAfterID(startIndex uint, limit uint) ([]string, uint64, error) {
	var uniqueSubAccountIDs []string
	lastIndex := uint64(startIndex + limit)

	// Update the query to include soft-deleted records
	result := db.Raw(`
		SELECT sub_account_id1 AS sub_account_id FROM batch_tables 
		WHERE id > ? AND id <= ? AND sub_account_id1 != ''
		UNION
		SELECT sub_account_id2 AS sub_account_id FROM batch_tables 
		WHERE id > ? AND id <= ? AND sub_account_id2 != ''
	`, startIndex, lastIndex, startIndex, lastIndex).Scan(&uniqueSubAccountIDs)

	if result.Error != nil {
		return nil, uint64(startIndex), result.Error
	}

	// Get the last index
	var lastIndexResult *uint64
	result = db.Raw(`
		SELECT MAX(id) FROM batch_tables WHERE id > ?
	`, startIndex).Scan(&lastIndexResult)

	if result.Error != nil {
		return nil, uint64(startIndex), result.Error
	}

	// If lastIndexResult is nil, set lastIndex to startIndex, other wise set it to the minimum of the two values
	if lastIndexResult != nil {
		lastIndex = min(lastIndex, *lastIndexResult)
	} else {
		lastIndex = uint64(startIndex)
	}

	return uniqueSubAccountIDs, lastIndex, nil
}

// GetTotalTrades includes soft-deleted data and returns the total number of trades
func (*BatchDB) GetTotalTrades(brokerId uint) (int64, error) {
	var count int64
	result := db.Raw(`
		SELECT COUNT(*) FROM batch_tables 
		WHERE (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?
	`, brokerId).Scan(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// GetDailyTrades includes soft-deleted data and returns the number of trades per day
func (*BatchDB) GetDailyTrades(brokerId uint, startDate ...time.Time) ([]DailyData, error) {
	var dailyData []DailyData
	now := time.Now().UTC().Truncate(24 * time.Hour)
	var startOfPeriod time.Time

	if len(startDate) > 0 {
		startOfPeriod = startDate[0].Truncate(24 * time.Hour)
	} else {
		startOfPeriod = now.AddDate(0, 0, -29)
	}

	err := db.Raw(`
		SELECT 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date, 
			COUNT(*) AS count
		FROM 
			batch_tables 
		WHERE 
			created_at BETWEEN ? AND ? 	
			AND (deleted_at IS NULL OR deleted_at IS NOT NULL) 
			AND broker_id = ?
		GROUP BY 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
		ORDER BY 
			date DESC
	`, startOfPeriod, now.Add(24*time.Hour-time.Nanosecond), brokerId).Scan(&dailyData).Error

	if err != nil {
		return nil, err
	}

	// Create a map to store daily counts by date
	dayMap := make(map[string]int64)
	for _, data := range dailyData {
		dayMap[data.Date.Format("2006-01-02")] = data.Count
	}

	dayCount := int(now.Sub(startOfPeriod).Hours()/24) + 1

	finalData := make([]DailyData, dayCount)
	for i := 0; i < dayCount; i++ {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		count := dayMap[dateStr]
		finalData[i] = DailyData{
			Date:  date,
			Count: count,
		}
	}

	return finalData, nil
}
func (*BatchDB) GetCumulativeTrades(endDate time.Time) (int64, error) {
	var cumulativeCount int64
	startDate := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

	err := db.Raw(`
		SELECT 
			COUNT(*) AS cumulative_count
		FROM 
			batch_tables
		WHERE 
			created_at AT TIME ZONE 'UTC' BETWEEN ? AND ? 
		AND (deleted_at IS NULL OR deleted_at IS NOT NULL)
	`, startDate, endDate).Scan(&cumulativeCount).Error

	if err != nil {
		xlog.Errorf("Error while fetching cumulative trades: %v", err)
		return 0, err
	}

	return cumulativeCount, nil
}

// GetDailyActiveUsers includes soft-deleted data and returns the number of active users per day
func (*BatchDB) GetDailyActiveUsers(brokerId uint, startDate ...time.Time) ([]DailyData, error) {
	var dailyData []DailyData
	now := time.Now().UTC()
	var startOfPeriod time.Time

	if len(startDate) > 0 {
		startOfPeriod = startDate[0]
	} else {
		startOfPeriod = now.Truncate(24*time.Hour).AddDate(0, 0, -29)
	}

	// Query to get daily unique subaccount counts, grouped by day, using UNION to combine sub_account_id1 and sub_account_id2
	err := db.Raw(`
		SELECT 
			date, 
			COUNT(DISTINCT sub_account_id) AS count
		FROM (
			SELECT 
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date, 
				sub_account_id1 AS sub_account_id
			FROM 
				batch_tables
			WHERE 
				created_at BETWEEN ? AND ?
				AND sub_account_id1 != '' AND sub_account_id1 IS NOT NULL
				AND (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?
			UNION
			SELECT 
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date, 
				sub_account_id2 AS sub_account_id
			FROM 
				batch_tables
			WHERE 
				created_at BETWEEN ? AND ?
				AND sub_account_id2 != '' AND sub_account_id2 IS NOT NULL
				AND (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?
		) AS combined
		GROUP BY 
			date
		ORDER BY 
			date DESC
	`, startOfPeriod, now, brokerId, startOfPeriod, now, brokerId).Scan(&dailyData).Error

	if err != nil {
		return nil, err
	}

	// A map to store daily counts by date
	dayMap := make(map[string]int64)
	for _, data := range dailyData {
		dayMap[data.Date.Format("2006-01-02")] = data.Count
	}

	// Calculate dynamic day count based on the start date
	dayCount := int(now.Sub(startOfPeriod).Hours()/24) + 1

	// Full range from today and fill in missing days with 0
	finalData := make([]DailyData, dayCount)
	for i := 0; i < dayCount; i++ {
		date := now.Truncate(24*time.Hour).AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		count := dayMap[dateStr] // If date is missing, this defaults to 0
		finalData[i] = DailyData{
			Date:  date,
			Count: count,
		}
	}

	return finalData, nil
}

func (*BatchDB) Get24hrData(brokerId uint) ([]DailyData, error) {
	var dailyData []DailyData
	now := time.Now().UTC()
	startOfDay := now.Add(-24 * time.Hour)
	var uniqueSubAccountIDs []string
	result := db.Raw(`
			SELECT DISTINCT sub_account_id1 AS sub_account_id FROM batch_tables 
			WHERE created_at BETWEEN ? AND ? AND sub_account_id1 != '' AND (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?
			UNION
			SELECT DISTINCT sub_account_id2 AS sub_account_id FROM batch_tables 
			WHERE created_at BETWEEN ? AND ? AND sub_account_id2 != '' AND (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?
		`, startOfDay, now, brokerId, startOfDay, now, brokerId).Scan(&uniqueSubAccountIDs)

	if result.Error != nil {
		return nil, result.Error
	}

	dailyData = append(dailyData, DailyData{
		Date:  startOfDay,
		Count: int64(len(uniqueSubAccountIDs)),
	})
	return dailyData, nil
}

func (*BatchDB) GetLastWeekData(brokerId uint) (int, error) {
	var count int
	now := time.Now().UTC()
	startOfWeek := now.Add(-24 * 7 * time.Hour)
	var uniqueSubAccountIDs []string
	result := db.Raw(`
			SELECT DISTINCT sub_account_id1 AS sub_account_id FROM batch_tables 
			WHERE created_at BETWEEN ? AND ? AND sub_account_id1 != '' AND (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?
			UNION
			SELECT DISTINCT sub_account_id2 AS sub_account_id FROM batch_tables 
			WHERE created_at BETWEEN ? AND ? AND sub_account_id2 != '' AND (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?
		`, startOfWeek, now, brokerId, startOfWeek, now, brokerId).Scan(&uniqueSubAccountIDs)

	if result.Error != nil {
		return 0, result.Error
	}

	count = len(uniqueSubAccountIDs)
	return count, nil
}

// GetWeeklyActiveUsers includes soft-deleted data and returns the number of active users per week
func (*BatchDB) GetWeeklyActiveUsers(brokerId uint) ([]WeeklyData, error) {
	var weeklyData []WeeklyData
	now := time.Now().UTC().Truncate(24 * time.Hour)

	for i := 0; i < 4; i++ {
		startOfWeek := now.AddDate(0, 0, -int(now.Weekday())-7*i)
		endOfWeek := startOfWeek.Add(7 * 24 * time.Hour).Add(-time.Nanosecond)

		var uniqueSubAccountIDs []string
		result := db.Raw(`
			SELECT DISTINCT sub_account_id1 AS sub_account_id FROM batch_tables 
			WHERE created_at BETWEEN ? AND ? AND sub_account_id1 != '' AND (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?
			UNION
			SELECT DISTINCT sub_account_id2 AS sub_account_id FROM batch_tables 
			WHERE created_at BETWEEN ? AND ? AND sub_account_id2 != '' AND (deleted_at IS NULL OR deleted_at IS NOT NULL) AND broker_id = ?	
		`, startOfWeek, endOfWeek, brokerId, startOfWeek, endOfWeek, brokerId).Scan(&uniqueSubAccountIDs)

		if result.Error != nil {
			return nil, result.Error
		}

		weeklyData = append(weeklyData, WeeklyData{
			StartDate: startOfWeek,
			Count:     int64(len(uniqueSubAccountIDs)),
		})
	}

	return weeklyData, nil
}

// SetDeletedAtById sets the deleted_at field for the transaction with the given id
func (*BatchDB) SetDeletedAtById(id uint, deletedAt time.Time) error {
	result := db.Model(&BatchTable{}).Where("id = ?", id).Update("deleted_at", deletedAt)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// This function is not used anywhere
func (b *BatchDB) SetDeletedAtByTransactionCounter(transactionCounter uint, deletedAt time.Time) error {
	// Check if a row with the same TransactionCounter exists
	var batch BatchTable
	result := db.Where("transaction_counter = ?", transactionCounter).First(&batch)

	// If there's an error and it's not because of a missing row, return the error
	if result.Error != nil {
		return result.Error
	}

	// If a row is found, soft delete it by setting the deleted_at timestamp
	if result.RowsAffected > 0 {
		result = db.Model(&BatchTable{}).Where("transaction_counter = ?", transactionCounter).Update("deleted_at", deletedAt)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

// CheckPerpTickExists checks if any entry with function_name 'PerpTick' and non-null deleted_at exists.
func (*BatchDB) CheckPerpTickExists() (bool, error) {
	var count int64

	// Execute the count query
	result := db.Raw(`
		SELECT COUNT(*) FROM batch_tables 
		WHERE function_name = 'PerpTick' AND deleted_at IS NULL
	`).Scan(&count)

	if result.Error != nil {
		return false, result.Error
	}

	// Return false if count is greater than zero, else true
	return count == 0, nil
}

func (*BatchDB) UpdatePausedEntries(subaccountID string) error {
	now := time.Now()
	fixedDate := fmt.Sprintf("2001-%02d-%02d", now.Month(), now.Day())

	result := db.Exec(`
        UPDATE batch_tables
        SET deleted_at = ?
        WHERE sub_account_id1 = ? 
          AND function_name = 'MatchOrders'
          AND deleted_at IS NULL
    `, fixedDate, subaccountID)

	if result.Error != nil {
		return fmt.Errorf("failed to unpause subaccount in DB: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no matching row found to unpause (already unpaused or subaccountID does not exist)")
	}

	return nil
}
