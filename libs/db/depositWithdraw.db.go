package db

import (
	"errors"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
)

type DepositWithdrawDB struct{}

func (*DepositWithdrawDB) MarkWithdrawReceivedByMessageId(messageId string, newMessageId string, logxTxnHash string, logxTxnIndex uint64, logxBlockNumber uint64) *DepositWithdrawTable {
	withdrawal := &DepositWithdrawTable{}
	updates := map[string]interface{}{
		"received":     true,
		"txn_hash":     logxTxnHash,
		"txn_index":    logxTxnIndex,
		"block_number": logxBlockNumber,
		"message_id":   newMessageId,
	}

	return GetDbObjOrNil(db.Model(withdrawal).Where("message_id = ? AND is_deposit = ?", messageId, false).Updates(updates), withdrawal)
}

func (*DepositWithdrawDB) MarkWithdrawalFinalisedByMessageId(messageId string, publicTxnHash string, publicTxnIndex uint64, publicBlockNumber uint64) *DepositWithdrawTable {
	withdrawal := &DepositWithdrawTable{}
	updates := map[string]interface{}{
		"received":     true, // Redundant but safe to set
		"finalised":    true,
		"txn_hash":     publicTxnHash,
		"txn_index":    publicTxnIndex,
		"block_number": publicBlockNumber,
	}
	return GetDbObjOrNil(db.Model(withdrawal).Where("message_id = ? AND is_deposit = ?", messageId, false).Updates(updates), withdrawal)
}

func (*DepositWithdrawDB) MarkDepositReceivedByMessageId(messageId string) {
	deposit := &DepositWithdrawTable{}
	updates := map[string]interface{}{
		"received": true,
	}
	db.Model(deposit).Where("message_id = ? AND is_deposit = ?", messageId, true).Updates(updates)
}

func (*DepositWithdrawDB) MarkDepositFinalisedByMessageId(messageId string) *DepositWithdrawTable {
	deposit := &DepositWithdrawTable{}
	updates := map[string]interface{}{
		"finalised": true,
	}
	return GetDbObjOrNil(db.Model(deposit).Where("message_id = ? AND is_deposit = ?", messageId, true).Updates(updates), deposit)
}

func (*DepositWithdrawDB) GetAllUnfinalisedDeposits(lastFetchId uint, limit int) *[]DepositWithdrawTable {
	deposits := []DepositWithdrawTable{}
	return GetDBObjOrEmptyList(db.Where("is_deposit = ? and finalised = ? and failed = ? and id > ?", true, false, false, lastFetchId).Limit(limit).Order("id ASC").Find(&deposits), &deposits)
}

// FinaliseDeposit marks a deposit as finalized by ID
func (*DepositWithdrawDB) FinaliseDeposit(id uint) error {
	result := db.Model(&DepositWithdrawTable{}).
		Where("id = ? AND is_deposit = ?", id, true).
		Update("finalised", true)
	return result.Error
}

// MarkDepositFailed marks a deposit as failed
func (*DepositWithdrawDB) MarkDepositFailed(id uint) error {
	updates := map[string]interface{}{
		"finalised": false,
		"failed":    true,
	}
	result := db.Model(&DepositWithdrawTable{}).
		Where("id = ? AND is_deposit = ?", id, true).
		Updates(updates)
	return result.Error
}

func (*DepositWithdrawDB) GetDepositsBySubaccountId(subaccountId string) *[]DepositWithdrawTable {
	deposits := []DepositWithdrawTable{}
	return GetDBObjOrEmptyList(db.Where("subaccount_id = ? and is_deposit = ?", subaccountId, true).Find(&deposits), &deposits)
}

func (*DepositWithdrawDB) CountDepositsBySubaccountAndChain(subaccountId string, sourceChainId uint64) (int64, error) {
	var count int64
	err := db.Model(&DepositWithdrawTable{}).
		Where("subaccount_id = ? AND is_deposit = ? AND source_chain_id = ?", subaccountId, true, sourceChainId).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

func (*DepositWithdrawDB) GetDepositsAfterCertainId(lastId uint) *[]DepositWithdrawTable {
	deposits := []DepositWithdrawTable{}
	return GetDBObjOrEmptyList(db.Where("id > ? AND is_deposit = ?", lastId, true).Order("id asc").Find(&deposits), &deposits)
}

func (*DepositWithdrawDB) GetWithdrawalsBySubaccountId(subaccountId string) *[]DepositWithdrawTable {
	withdrawals := []DepositWithdrawTable{}
	return GetDBObjOrEmptyList(db.Where("subaccount_id = ? and is_deposit = ?", subaccountId, false).Find(&withdrawals), &withdrawals)
}

// for tradeHistory of LOGX
func (*DepositWithdrawDB) GetAllBySubaccountId(subaccountId string) ([]DepositWithdrawTable, error) {
	var depositsWithdrawals []DepositWithdrawTable

	result := db.Where("subaccount_id = ? AND product_id = ?", subaccountId, 0).
		Order("created_at DESC").
		Find(&depositsWithdrawals)

	if result.Error != nil {
		return nil, result.Error
	}

	return depositsWithdrawals, nil
}

func (*DepositWithdrawDB) GetAllUnfinalisedWithdrawalsMessageIdsForProductId(productId uint32) []string {
	// Initialize a map to store ProductId and corresponding MessageIds
	results := []string{}

	// Query the database
	var withdrawals []DepositWithdrawTable
	err := db.Model(&DepositWithdrawTable{}).
		Select("message_id").
		Where("product_id = ? AND is_deposit = ? AND received = ? AND finalised = ?", productId, false, true, false).Order("id ASC").
		Find(&withdrawals).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Return an empty slice without an error
		return results
	}

	if err != nil {
		xlog.Errorf("Failed to retrieve withdrawals: %s", err)
		return results // Return an empty map in case of an error
	}

	// Group MessageIds by ProductId
	for _, withdrawal := range withdrawals {
		results = append(results, withdrawal.MessageId)
	}

	return results
}

func (*DepositWithdrawDB) GetPendingUnfinalizedWithdrawalsCount(productId uint32) int64 {
	var count int64
	err := db.Model(&DepositWithdrawTable{}).
		Where("product_id = ? AND is_deposit = ? AND finalised = ?", productId, false, false).
		Count(&count).Error

	if err != nil {
		xlog.Errorf("Failed to count unfinalized withdrawals: %s", err)
		return 0
	}

	return count
}

func (*DepositWithdrawDB) GetFirstXUnreceivedWithdrawalsMessageIdsForProductId(productId uint32, limit int) []string {
	// Initialize a map to store ProductId and corresponding MessageIds
	results := []string{}

	// Query the database
	var withdrawals []DepositWithdrawTable
	err := db.Model(&DepositWithdrawTable{}).
		Select("message_id").
		Where("product_id = ? AND is_deposit = ? AND received = ? AND finalised = ?", productId, false, false, false).Order("id DESC").Limit(limit).
		Find(&withdrawals).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Return an empty slice without an error
		return results
	}

	if err != nil {
		xlog.Errorf("Failed to retrieve withdrawals: %s", err)
		return results // Return an empty map in case of an error
	}

	// Group MessageIds by ProductId
	for _, withdrawal := range withdrawals {
		results = append(results, withdrawal.MessageId)
	}

	return results
}

// Gives the first X deposits which are only received on source chain and not yet received on LogX Chain
func (*DepositWithdrawDB) GetFirstXSourceOnlyReceivedDepositsMessageIdsForProductId(productId uint32, limit int) []string {
	// Initialize a map to store ProductId and corresponding MessageIds
	results := []string{}

	// Query the database
	var deposits []DepositWithdrawTable
	err := db.Model(&DepositWithdrawTable{}).
		Select("message_id").
		Where("product_id = ? AND is_deposit = ? AND received = ? AND finalised = ?", productId, true, false, false).Order("id DESC").Limit(limit).
		Find(&deposits).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Return an empty slice without an error
		return results
	}

	if err != nil {
		xlog.Errorf("Failed to retrieve deposits: %s", err)
		return results // Return an empty map in case of an error
	}

	// Group MessageIds by ProductId
	for _, deposit := range deposits {
		results = append(results, deposit.MessageId)
	}

	return results
}

func BoolPtr(b bool) *bool {
	return &b
}

func (*DepositWithdrawDB) InsertDeposit(txnHash string, txnIndex uint64, blockNumber uint64, messageId string, subaccountId string, amount string, productId uint32, sourceChainID uint64) (*DepositWithdrawTable, error) {
	// Create a new record for the DepositWithdrawTable
	newEntry := &DepositWithdrawTable{
		TxnHash:            &txnHash,
		TxnIndex:           &txnIndex,
		BlockNumber:        &blockNumber,
		MessageId:          messageId,
		SubaccountId:       subaccountId,
		Amount:             amount,
		ProductId:          productId,
		SourceChainID:      sourceChainID,
		DestinationChainID: uint64(contractUtils.LOGX_CHAIN_ID),
		Received:           BoolPtr(true),
		Finalised:          false,
		IsDeposit:          true,
	}

	// Insert into the database
	result := db.Create(newEntry)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key value violates unique constraint") {
			return nil, nil
		}
		return nil, result.Error
	}

	// Return the inserted entry
	return newEntry, nil
}

// Here messageId is the encodedStruct and not the mailbox message id
func (*DepositWithdrawDB) InsertWithdraw(messageId string, subaccountId string, amount string, productId uint32, sourceChainID uint64, destinationChainID uint64) (*DepositWithdrawTable, error) {
	// Create a new record for the DepositWithdrawTable
	newEntry := &DepositWithdrawTable{
		TxnHash:            nil,
		TxnIndex:           nil,
		BlockNumber:        nil,
		MessageId:          messageId,
		SubaccountId:       subaccountId,
		Amount:             amount,
		ProductId:          productId,
		SourceChainID:      sourceChainID,
		DestinationChainID: destinationChainID,
		Received:           BoolPtr(false),
		Finalised:          false,
		IsDeposit:          false,
	}

	// Insert into the database
	result := db.Create(newEntry)
	if result.Error != nil {
		return nil, result.Error
	}

	// Return the inserted entry
	return newEntry, nil
}

func (*DepositWithdrawDB) CalculateDepositWithdrawAmounts() (totalWithdrawals, totalDeposits, total24hrWithdrawals, total24hrDeposits *big.Int, err error) {
	// Initialize big.Int variables
	totalWithdrawals = big.NewInt(0)
	totalDeposits = big.NewInt(0)
	total24hrWithdrawals = big.NewInt(0)
	total24hrDeposits = big.NewInt(0)

	// Get the current time and time 24 hours ago
	now := time.Now()
	twentyFourHoursAgo := now.Add(-24 * time.Hour)

	// Calculate total deposits
	var totalDepositsResult struct {
		Amount string
	}

	// Define the subaccount IDs to exclude
	excludeSubaccountIDs := []string{
		"0x0000000000010000000000000000000000000000000000000003000000000001",
		"0x0000000000010000000000000000000000000000000000000006000000000001",
	}

	err = db.Model(&DepositWithdrawTable{}).
		Select("COALESCE(SUM(CAST(amount AS NUMERIC)), '0') as amount").
		Where("is_deposit = 't' AND product_id = '0' AND subaccount_id NOT IN (?)", excludeSubaccountIDs).
		Scan(&totalDepositsResult).Error
	if err != nil {
		return
	}
	totalDeposits.SetString(totalDepositsResult.Amount, 10)

	// Calculate total withdrawals
	var totalWithdrawalsResult struct {
		Amount string
	}
	err = db.Model(&DepositWithdrawTable{}).
		Select("COALESCE(SUM(CAST(amount AS NUMERIC)), '0') as amount").
		Where("is_deposit = 'f' AND product_id = '0' AND subaccount_id NOT IN (?)", excludeSubaccountIDs).
		Scan(&totalWithdrawalsResult).Error
	if err != nil {
		return
	}
	totalWithdrawals.SetString(totalWithdrawalsResult.Amount, 10)

	// Calculate 24-hour deposits
	var total24hrDepositsResult struct {
		Amount string
	}
	err = db.Model(&DepositWithdrawTable{}).
		Select("COALESCE(SUM(CAST(amount AS NUMERIC)), '0') as amount").
		Where("is_deposit = 't' AND product_id = '0' AND created_at >= ? AND subaccount_id NOT IN (?)", twentyFourHoursAgo, excludeSubaccountIDs).
		Scan(&total24hrDepositsResult).Error
	if err != nil {
		return
	}
	total24hrDeposits.SetString(total24hrDepositsResult.Amount, 10)

	// Calculate 24-hour withdrawals
	var total24hrWithdrawalsResult struct {
		Amount string
	}
	err = db.Model(&DepositWithdrawTable{}).
		Select("COALESCE(SUM(CAST(amount AS NUMERIC)), '0') as amount").
		Where("is_deposit = 'f' AND product_id = '0' AND created_at >= ? AND subaccount_id NOT IN (?)", twentyFourHoursAgo, excludeSubaccountIDs).
		Scan(&total24hrWithdrawalsResult).Error
	if err != nil {
		return
	}
	total24hrWithdrawals.SetString(total24hrWithdrawalsResult.Amount, 10)

	return
}

// Function to get the current net sum of deposits and withdrawals for a specific SourceChainID
func (*DepositWithdrawDB) GetNetSumForSourceChain(sourceChainId uint64) ([]ctypes.NetDepositResult, error) {
	var netDepositResults []ctypes.NetDepositResult

	// Step 1: Fetch subaccount_ids from SubaccountLastDepositTable where source_chain_id = ?
	var subaccounts []string
	err := db.Table("subaccount_last_deposit_tables").
		Where("source_chain_id = ?", sourceChainId).
		Pluck("subaccount_id", &subaccounts).
		Error
	if err != nil {
		xlog.Errorf("Failed to retrieve subaccounts: %s", err)
		return nil, err
	}

	if len(subaccounts) == 0 {
		return netDepositResults, nil
	}

	// Step 2: Calculate the net sum of deposits and withdrawals from DepositWithdrawTable for these subaccounts
	rows, err := db.Table("deposit_withdraw_tables").
		Select("subaccount_id, SUM(CASE WHEN is_deposit = true THEN amount::numeric ELSE -amount::numeric END) AS net_sum").
		Where("subaccount_id IN (?) AND (source_chain_id = ? OR destination_chain_id = ?)", subaccounts, sourceChainId, sourceChainId).
		Group("subaccount_id").
		Rows()

	if err != nil {
		xlog.Errorf("Failed to retrieve deposits/withdrawals: %s", err)
		return nil, err
	}
	defer rows.Close()

	// Step 3: Iterate over the results and populate the NetDepositResult slice
	for rows.Next() {
		var result ctypes.NetDepositResult
		var netSumStr string

		err = rows.Scan(&result.SubaccountId, &netSumStr)
		if err != nil {
			xlog.Errorf("Failed to scan result row: %s", err)
			return nil, err
		}

		netSum := new(big.Int)
		_, ok := netSum.SetString(netSumStr, 10)
		if !ok {
			xlog.Errorf("Failed to convert net sum to big.Int")
			continue
		}

		if netSum.Cmp(big.NewInt(0)) <= 0 {
			continue
		}

		result.NetSum = netSum
		netDepositResults = append(netDepositResults, result)
	}

	return netDepositResults, nil
}

// GetWithdrawalsWithConditions retrieves withdrawals based on the specified conditions
func (*DepositWithdrawDB) GetWithdrawalsWithConditions() (*[]DepositWithdrawTable, error) {
	withdrawals := []DepositWithdrawTable{}

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	result := db.Where("received = ? AND finalised = ? AND is_deposit = ? AND product_id = ? AND created_at >= ?",
		true, true, false, 0, sevenDaysAgo).
		Order("created_at DESC").
		Find(&withdrawals)

	if result.Error != nil {
		xlog.Errorf("Error fetching withdrawals: %v", result.Error)
		return nil, result.Error
	}

	return &withdrawals, nil
}

func (*DepositWithdrawDB) GetSubaccountsWithOtherTokens() ([]string, error) {
	var subaccounts []string

	internalSubaccountIDs := strings.Split(os.Getenv("DEPOSIT_SUBACCOUNT_SKIP"), ",")

	err := db.Model(&DepositWithdrawTable{}).
		Select("DISTINCT subaccount_id").
		Where("product_id NOT IN (0, 2, 72, 74) AND finalised = true").
		Where("subaccount_id NOT IN (?)", internalSubaccountIDs).
		Find(&subaccounts).Error

	if err != nil {
		xlog.Errorf("Error fetching subaccounts with other tokens: %v", err)
		return nil, err
	}

	return subaccounts, nil
}

func (*DepositWithdrawDB) GetTotalDepositsAndWithdrawals() (map[string]string, error) {
	internalSubaccountIDs := strings.Split(os.Getenv("DEPOSIT_SUBACCOUNT_SKIP"), ",")
	result := make(map[string]string)

	// Combined query for both deposits and withdrawals
	var queryResult struct {
		TotalDeposits    string `json:"total_deposits"`
		TotalWithdrawals string `json:"total_withdrawals"`
	}

	err := db.Raw(`
		SELECT 
			COALESCE(SUM(CASE WHEN is_deposit = true THEN amount::numeric ELSE 0 END), '0') as total_deposits,
			COALESCE(SUM(CASE WHEN is_deposit = false THEN amount::numeric ELSE 0 END), '0') as total_withdrawals
		FROM deposit_withdraw_tables
		WHERE finalised = true AND product_id NOT IN (0, 2)
		AND subaccount_id NOT IN (?)
	`, internalSubaccountIDs).Scan(&queryResult).Error

	if err != nil {
		xlog.Errorf("Error fetching total deposits and withdrawals: %v", err)
		return nil, err
	}

	result["totalDeposits"] = queryResult.TotalDeposits
	result["totalWithdrawals"] = queryResult.TotalWithdrawals

	return result, nil
}

func (*DepositWithdrawDB) GetFirstXUnfinalisedWithdrawDetails(limit int) ([]DepositWithdrawTable, error) {
	var withdrawals []DepositWithdrawTable
	result := db.Where("is_deposit = ? AND finalised = ? AND failed = ? AND received = ?", false, false, false, false).
		Order("created_at ASC").
		Limit(limit).
		Find(&withdrawals)

	if result.Error != nil {
		xlog.Errorf("Failed to get pending withdrawals: %v", result.Error)
		return nil, result.Error
	}

	return withdrawals, nil
}

func (*DepositWithdrawDB) MarkWithdrawalFailedByMessageId(messageID string) error {
	withdrawal := &DepositWithdrawTable{}
	updates := map[string]interface{}{
		"failed": true,
	}
	result := db.Model(withdrawal).Where("message_id = ? AND is_deposit = ?", messageID, false).Updates(updates)
	if result.Error != nil {
		xlog.Errorf("Failed to mark withdrawal as failed for message ID %s: %v", messageID, result.Error)
		return result.Error
	}
	return nil
}

func (*DepositWithdrawDB) ExistsMessageID(messageID string) (bool, error) {
	var count int64
	result := db.Model(&DepositWithdrawTable{}).Where("message_id = ?", messageID).Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}
