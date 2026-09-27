// NOTE: These operations are just for testing of liquidation. Please use this carefully as this might be deprecated

package db

type LiquidationDB struct{}

// TODO: Evaluate if this is optimal
func (*LiquidationDB) GetAllWithGreaterGID(startGID uint, limit int) *[]LiquidationTable {
	liquidationSubaccounts := []LiquidationTable{}
	return GetDBObjOrEmptyList(db.Select("*").Where("id > ?", startGID).Order("id ASC").Limit(limit).Find(&liquidationSubaccounts), &liquidationSubaccounts)
}
