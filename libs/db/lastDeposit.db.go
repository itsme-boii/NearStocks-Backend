package db

import (
	"time"
	// "gorm.io/gorm"
)

type SubaccountLastDepositDB struct{}

// Function to get an entry by subaccountId and sourceChainId
func (*SubaccountLastDepositDB) GetBySubaccountAndChain(subaccountId string, sourceChainId uint64) *SubaccountLastDepositTable {
	lastDeposit := &SubaccountLastDepositTable{}
	return GetDbObjOrNil(db.Where("subaccount_id = ? AND source_chain_id = ?", subaccountId, sourceChainId).First(lastDeposit), lastDeposit)
}

// Function to create a new entry
func (*SubaccountLastDepositDB) Create(subaccountId string, sourceChainId uint64, lastDepositDate time.Time) *SubaccountLastDepositTable {
	newEntry := &SubaccountLastDepositTable{
		SubaccountId:    subaccountId,
		SourceChainID:   sourceChainId,
		LastDepositDate: lastDepositDate,
	}
	return GetDbObjOrNil(db.Create(newEntry), newEntry)
}

// Function to update an existing entry
func (*SubaccountLastDepositDB) Update(subaccountId string, sourceChainId uint64, lastDepositDate time.Time) *SubaccountLastDepositTable {
	updates := map[string]interface{}{
		"last_deposit_date": lastDepositDate,
	}
	lastDeposit := &SubaccountLastDepositTable{}
	return GetDbObjOrNil(db.Model(lastDeposit).
		Where("subaccount_id = ? AND source_chain_id = ?", subaccountId, sourceChainId).
		Updates(updates), lastDeposit)
}
