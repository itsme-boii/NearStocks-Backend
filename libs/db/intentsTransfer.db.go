package db

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type IntentsTransferDB struct{}

var ErrIntentsTransferExists = errors.New("intents transfer already exists")

func (*IntentsTransferDB) Create(row *IntentsTransferTable) error {
	if err := db.Create(row).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key") {
			return ErrIntentsTransferExists
		}
		return err
	}
	return nil
}

func (*IntentsTransferDB) GetByKey(key string) *IntentsTransferTable {
	var row IntentsTransferTable
	if err := db.Where("key = ?", key).First(&row).Error; err != nil {
		return nil
	}
	return &row
}

// Pending returns non-terminal 1Click deposits, oldest first, for the status poller.
func (*IntentsTransferDB) Pending(kind, direction string, olderThan time.Time, limit int) []IntentsTransferTable {
	var rows []IntentsTransferTable
	db.Where("kind = ? AND direction = ? AND credited_at IS NULL AND status NOT IN ? AND updated_at < ?",
		kind, direction, []string{"SUCCESS_CREDITED", "REFUNDED", "FAILED", "EXPIRED", "CREDIT_FAILED"}, olderThan).
		Order("id ASC").Limit(limit).Find(&rows)
	return rows
}

func (*IntentsTransferDB) ListBySubaccount(subaccountId string, limit int) []IntentsTransferTable {
	var rows []IntentsTransferTable
	db.Where("subaccount_id = ?", subaccountId).Order("id DESC").Limit(limit).Find(&rows)
	return rows
}

func (*IntentsTransferDB) UpdateStatus(key, status, lastError string) error {
	return db.Model(&IntentsTransferTable{}).Where("key = ? AND credited_at IS NULL", key).
		Updates(map[string]any{"status": status, "last_error": lastError}).Error
}

// ClaimCredit atomically marks the row as credited. Exactly one caller gets true, so the balance
// is credited at most once even with concurrent pollers or retries.
func (*IntentsTransferDB) ClaimCredit(key, amountUSDC, nearTxHash string) (bool, error) {
	now := time.Now()
	res := db.Model(&IntentsTransferTable{}).Where("key = ? AND credited_at IS NULL", key).
		Updates(map[string]any{"credited_at": &now, "status": "SUCCESS_CREDITED", "amount_usdc": amountUSDC, "near_tx_hash": nearTxHash})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// MarkCreditFailed records a balance-server failure after ClaimCredit. It never clears
// credited_at, so the credit can't be retried automatically into a double credit; ops resolve it.
func (*IntentsTransferDB) MarkCreditFailed(key, lastError string) error {
	return db.Model(&IntentsTransferTable{}).Where("key = ?", key).
		Updates(map[string]any{"status": "CREDIT_FAILED", "last_error": lastError}).Error
}

func (*IntentsTransferDB) SetWithdrawResult(key, status, nearTxHash, lastError string) error {
	return db.Model(&IntentsTransferTable{}).Where("key = ?", key).
		Updates(map[string]any{"status": status, "near_tx_hash": nearTxHash, "last_error": lastError}).Error
}

// InsertNearTransfer writes the user-visible deposit/withdraw history row for a NEAR flow.
// MessageId is the intents key, so the row is unique per flow.
func (*DepositWithdrawDB) InsertNearTransfer(key, nearTxHash, subaccountHex, amountX18 string, productId uint32, isDeposit bool, nearChainId uint64) error {
	row := &DepositWithdrawTable{
		MessageId: key, SubaccountId: subaccountHex, Amount: amountX18, ProductId: productId,
		SourceChainID: nearChainId, DestinationChainID: nearChainId,
		Received: BoolPtr(true), Finalised: true, IsDeposit: isDeposit,
	}
	if nearTxHash != "" {
		row.TxnHash = &nearTxHash
	}
	err := db.Create(row).Error
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return nil
	}
	return err
}
