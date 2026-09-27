package db

import "strings"

type CountryCodeDB struct{}

// CreateCountryCode creates a new country code entry in the database
func (*CountryCodeDB) CreateCountryCode(subaccountID string, countryCode string) (*IpTable, error) {
	var existingRecord IpTable
	// Check if the record already exists
	if err := db.Where("subaccount_id = ?", subaccountID).First(&existingRecord).Error; err == nil {
		// Record already exists, return the existing record
		return &existingRecord, nil
	}

	// Record does not exist, create a new one
	countryCodeData := IpTable{
		SubaccountID: subaccountID,
		CountryCode:  countryCode,
	}
	if err := db.Create(&countryCodeData).Error; err != nil {
		return nil, err
	}
	return &countryCodeData, nil
}

// GetCountryCodesBySubaccount fetches country code entries for a specific subaccount ID
func (*CountryCodeDB) GetCountryCodeBySubaccount(subaccountID string) *IpTable {
	var countryCode IpTable
	subaccountIDLower := strings.ToLower(subaccountID)
	result := db.Where("subaccount_id = ?", subaccountIDLower).First(&countryCode)
	if result.Error != nil {
		return nil
	}
	return &countryCode
}
