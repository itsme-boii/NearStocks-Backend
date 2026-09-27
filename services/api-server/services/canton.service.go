package services

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"github.com/ethereum/go-ethereum/crypto"
)

type CantonPartyDetails struct {
	PartyId      string
	SubaccountId string
	EvmAddress   string // EVM address in checksum format
}

// GetOrCreateCantonPartyDetails checks if a Canton party id exists and returns the mapping,
// or creates a new EVM address, subaccount, and mapping if it doesn't exist
func GetOrCreateCantonPartyDetails(partyId string, brokerId uint) (*CantonPartyDetails, error) {
	// Check if Canton party id exists
	existingRecord := (&db.CantonAddressDB{}).GetSubAccountIdByPartyId(partyId)
	if existingRecord != nil && existingRecord.SubAccountId != "" {
		// Get subaccount to retrieve EVM address
		subaccount := (&db.SubaccountDB{}).GetByIdBrokerId(brokerId, existingRecord.SubAccountId)
		if subaccount != nil {
			xlog.Infof("Canton Service - Existing mapping found for party id: %s, subaccount id: %s", partyId, existingRecord.SubAccountId)
			return &CantonPartyDetails{
				PartyId:      partyId,
				SubaccountId: existingRecord.SubAccountId,
				EvmAddress:   subaccount.EthAddress,
			}, nil
		}
	}

	// Generate new EVM address (checksum format)
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		xlog.Errorf("Canton Service - Error generating Ethereum wallet: %v", err)
		return nil, err
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	evmAddressChecksummed := address.Hex()
	xlog.Infof("Canton Service - Generated new EVM address: %s for party id: %s", evmAddressChecksummed, partyId)

	// Get subaccount id from EVM address
	subaccountId := cutils.CreateSubaccountId(brokerId, evmAddressChecksummed, 1)

	// Get or create subaccount in SubaccountTable
	subaccount := (&db.SubaccountDB{}).GetByIdBrokerId(brokerId, subaccountId)
	if subaccount == nil {
		var err error
		subaccount, err = (&db.SubaccountDB{}).Create(evmAddressChecksummed, brokerId, subaccountId)
		if err != nil {
			xlog.Errorf("Canton Service - Error creating subaccount: %v", err)
			return nil, err
		}
		xlog.Infof("Canton Service - Created subaccount: %s", subaccountId)
	} else {
		xlog.Infof("Canton Service - Subaccount already exists: %s", subaccountId)
	}

	// Store mapping in CantonPartyTable
	if err := (&db.CantonAddressDB{}).CreateCantonAddressMapping(partyId, subaccount.ID); err != nil {
		xlog.Errorf("Canton Service - Error creating Canton address mapping: %v", err)
		return nil, err
	}
	xlog.Infof("Canton Service - Created mapping for party id: %s, subaccount id: %s", partyId, subaccount.ID)

	return &CantonPartyDetails{
		PartyId:      partyId,
		SubaccountId: subaccount.ID,
		EvmAddress:   evmAddressChecksummed,
	}, nil
}
