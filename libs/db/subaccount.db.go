package db

import (
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"strings"

	"gorm.io/gorm"
)

type SubaccountDB struct{}

// NOTE: Only for testing purposes
func (sdb *SubaccountDB) MockCreate(ethAddress string, brokerId, gid uint) (*SubaccountTable, error) {
	subaccount := SubaccountTable{
		ID:         fmt.Sprintf("%d_%s_1", brokerId, ethAddress),
		BrokerId:   brokerId,
		EthAddress: ethAddress,
		GID:        gid,
	}
	if err := db.Create(&subaccount).Error; err != nil {
		return nil, err
	}
	return &subaccount, nil
}

// TODO: Handle creation errors
// NOTE: We will only allow 2 subaccounts at max per broker per eth address
func (sdb *SubaccountDB) Create(ethAddress string, brokerId uint, id string) (*SubaccountTable, error) {
	ids := sdb.GetIDsByBrokerAddress(brokerId, ethAddress)
	if len(*ids) >= 2 {
		return nil, fmt.Errorf("cannot create more than 2 subaccounts per broker")
	}

	subaccount := SubaccountTable{EthAddress: ethAddress, BrokerId: brokerId, ID: id}
	if err := db.Create(&subaccount).Error; err != nil {
		return nil, err
	}
	return &subaccount, nil
}

/*
  - NOTE: Should be carefully used.
    Please make sure you do not give away any information of
    1. other users
    2. same user but different broker
*/
func (*SubaccountDB) GetById(id string) *SubaccountTable {
	subaccount := SubaccountTable{ID: id}
	return GetDbObjOrNil(db.Where(&subaccount).First(&subaccount), &subaccount)
}

// Much safer than GetById
func (*SubaccountDB) GetByIdBrokerId(brokerId uint, id string) *SubaccountTable {
	subaccount := SubaccountTable{BrokerId: brokerId, ID: id}
	return GetDbObjOrNil(db.Where(&subaccount).Find(&subaccount), &subaccount)
}

// TODO: Handle for errors while fetch and return nil in error cases
func (*SubaccountDB) GetAllByAddressBrokerId(ethAddress string, brokerId uint) *[]SubaccountTable {
	subaccounts := []SubaccountTable{}
	return GetDBObjOrEmptyList(db.Where(SubaccountTable{EthAddress: ethAddress, BrokerId: brokerId}).Find(&subaccounts), &subaccounts)
}

func (*SubaccountDB) GetIDsByBrokerAddress(brokerId uint, ethAddress string) *[]string {
	subaccounts := []SubaccountTable{}
	GetDBObjOrEmptyList(db.Select("id").Where(&SubaccountTable{EthAddress: ethAddress, BrokerId: brokerId}).Find(&subaccounts), &subaccounts)
	ids := make([]string, 0, len(subaccounts))
	for _, x := range subaccounts {
		ids = append(ids, x.ID)
	}
	return &ids
}

// Returns subaccountIds from [startGID, startGID + limit -1] and the last g_id in the range
// Here ordering may not be guaranteed
func (*SubaccountDB) GetAllIdsFromStartGID(startGID uint, limit uint) (*[]string, uint) {
	subaccountIds := []string{}
	// Query to get last g_id < startGID + limit
	lastGID := startGID - 1

	firstGidAfterRange := uint(0)
	db.Model(&SubaccountTable{}).Select("g_id").Where("g_id >= ?", startGID+limit).Order("g_id ASC").First(&firstGidAfterRange)

	if firstGidAfterRange != 0 {
		lastGID = firstGidAfterRange - 1
	} else {
		db.Model(&SubaccountTable{}).Select("g_id").Where("g_id >= ? and g_id < ?", startGID, startGID+limit).Order("g_id DESC").First(&lastGID)
	}

	if lastGID < startGID {
		return &subaccountIds, lastGID
	}

	return GetDBObjOrEmptyList(db.Model(&SubaccountTable{}).Select("id").Where("g_id >= ? and g_id <= ?", startGID, lastGID).Find(&subaccountIds), &subaccountIds), lastGID
}

func (*SubaccountDB) DeleteByGID(gid string) {
	db.Delete(&SubaccountTable{}, gid)
}

func (*SubaccountDB) GetUserName(id string) (*string, error) {
	var subaccount SubaccountTable
	result := db.Where("id = ?", id).First(&subaccount)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return subaccount.UserName, nil
}

func (subaccountDB *SubaccountDB) UpdateUserName(id string, newUserName string) error {
	var subaccount SubaccountTable
	result := db.Where("id = ?", id).First(&subaccount)
	if result.Error != nil {
		return result.Error
	}

	// If the subaccount already has a username, no need to update again, just return nil
	if subaccount.UserName != nil && *subaccount.UserName != "" {
		return nil
	}

	// Check if the new username already exists in the system
	var existingSubaccount SubaccountTable
	result = db.Where("user_name = ?", newUserName).First(&existingSubaccount)
	if result.Error == nil {
		return errors.New("username already taken")
	}

	// Update the username
	subaccount.UserName = &newUserName

	// Save the updated subaccount
	if err := db.Save(&subaccount).Error; err != nil {
		return err
	}

	// Initialize the referralDB instance
	referralDB := ReferralDB{}

	// Call function to update the referral code in ReferralDB
	err := referralDB.UpdateReferralCodeWithUserNameForUser(subaccount.EthAddress, newUserName)
	if err != nil {
		return err
	}

	return nil
}

func (*SubaccountDB) CreateUserName(subAccountId string, userName string, ethAddress string) {
	var subaccount SubaccountTable
	result := db.Where("id = ?", subAccountId).First(&subaccount)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		xlog.Errorf("Failed to connect to db")
		return
	}
	var existingSubaccount SubaccountTable
	result = db.Where("user_name = ?", userName).First(&existingSubaccount)
	if result.RowsAffected >= 1 {
		xlog.Errorf("CreateUsername - Failed to create username: %v for subaccount: %v", subAccountId, userName)
		return
	}
	if subaccount.UserName != nil {
		xlog.Errorf("CreateUsername - Differnet username exists on mainnet for account: %v", subAccountId)
		return
	}
	subaccountCreated := SubaccountTable{EthAddress: ethAddress, BrokerId: 1, ID: subAccountId, UserName: &userName}
	if err := db.Create(&subaccountCreated).Error; err != nil {
		xlog.Errorf("CreateUsername - Failed to create entry in db for account: %v, username: %v", subAccountId, userName)
		return
	}
	xlog.Infof("Created username successfully for account: %v", subAccountId)
}

func (*SubaccountDB) CheckUserNameAvailability(username string) (bool, error) {
	var subaccount SubaccountTable
	lowerUsername := strings.ToLower(username)
	result := db.Where("user_name = lower(?)", lowerUsername).First(&subaccount)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, result.Error
	}
	return false, nil
}

func (*SubaccountDB) GetUserNamesByEthAddresses(ethAddresses []string) (map[string]string, error) {
	// Initialize the result map to store Ethereum address -> UserName mapping
	userNameMapping := make(map[string]string)
	// Define a slice to hold the result
	var subaccounts []SubaccountTable
	// Query the database for matching Ethereum addresses with non-null usernames
	result := db.Where("eth_address IN ? AND user_name IS NOT NULL", ethAddresses).Find(&subaccounts)
	if result.Error != nil {
		return nil, result.Error
	}
	// Iterate over the fetched subaccounts and populate the map
	for _, subaccount := range subaccounts {
		if subaccount.UserName != nil {
			userNameMapping[subaccount.EthAddress] = *subaccount.UserName + ".logX"
		}
	}
	return userNameMapping, nil
}
func (*SubaccountDB) CheckUserExistsInSubaccount(userAddress string, brokerId uint) (bool, error) {
	var subaccount SubaccountTable
	err := db.Where("eth_address = ? AND broker_id = ?", userAddress, brokerId).First(&subaccount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // User doesn't exist
		}
		return false, err // Database error
	}
	return true, nil // User exists
}
