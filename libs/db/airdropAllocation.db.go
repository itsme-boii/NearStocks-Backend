package db

import (
	"errors"
	"gorm.io/gorm"
)

type AirdropAllocationDB struct{}

// GetByAddress retrieves airdrop allocation for a specific address
func (*AirdropAllocationDB) GetByAddress(address string) (*AirdropAllocationTable, error) {
	var allocation AirdropAllocationTable
	result := db.Where("address = ?", address).First(&allocation)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil without error if not found
		}
		return nil, result.Error
	}
	return &allocation, nil
}
