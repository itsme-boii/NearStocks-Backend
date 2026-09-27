package db

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

type NearAccountDB struct{}

func (*NearAccountDB) GetByAccountId(accountId string, brokerId uint) *NearAccountTable {
	var row NearAccountTable
	if err := db.Where("account_id = ? AND broker_id = ?", accountId, brokerId).First(&row).Error; err != nil {
		return nil
	}
	return &row
}

func (*NearAccountDB) GetBySubaccountId(subaccountId string) *NearAccountTable {
	var row NearAccountTable
	if err := db.Where("subaccount_id = ?", subaccountId).First(&row).Error; err != nil {
		return nil
	}
	return &row
}

// GetOrCreate returns the mapping for (accountId, brokerId), creating it if missing. A concurrent
// insert of the same pair is resolved by re-reading the winner's row.
func (n *NearAccountDB) GetOrCreate(accountId string, brokerId uint, subaccountId, addr20 string) (*NearAccountTable, error) {
	if row := n.GetByAccountId(accountId, brokerId); row != nil {
		return row, nil
	}
	row := &NearAccountTable{AccountId: accountId, BrokerId: brokerId, SubaccountId: subaccountId, Addr20: addr20}
	if err := db.Create(row).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key") {
			if existing := n.GetByAccountId(accountId, brokerId); existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return row, nil
}
