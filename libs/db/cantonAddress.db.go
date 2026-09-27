package db

type CantonAddressDB struct{}

// CreateCantonAddressMapping creates a Canton address mapping with PartyId -> SubAccountId if it doesn't exist
func (*CantonAddressDB) CreateCantonAddressMapping(partyId string, subAccountId string) error {
	var existingRecord CantonPartyTable
	if err := db.Where("party_id = ?", partyId).First(&existingRecord).Error; err != nil {
		// Create new mapping: party id -> subaccount id
		cantonMapping := CantonPartyTable{
			SubAccountId: subAccountId,
			PartyId:      partyId,
		}
		if err := db.Create(&cantonMapping).Error; err != nil {
			return err
		}
		return nil
	}

	// Mapping already exists, no update needed
	return nil
}

// GetSubAccountIdByPartyId fetches subaccount id for a given party id
func (*CantonAddressDB) GetSubAccountIdByPartyId(partyId string) *CantonPartyTable {
	var cantonMapping CantonPartyTable
	result := db.Where("party_id = ?", partyId).First(&cantonMapping)
	if result.Error != nil {
		return nil
	}
	return &cantonMapping
}
