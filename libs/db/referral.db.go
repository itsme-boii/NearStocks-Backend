package db

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"strings"
)

type ReferralDB struct{}

// GenerateReferralCode generates a referral code for a given user address
func GenerateReferralCode(userAddress string) string {
	hash := sha256.New()
	hash.Write([]byte(userAddress))
	referralCodeValue := hash.Sum(nil)
	return strings.ToUpper(hex.EncodeToString(referralCodeValue)[:8])
}

// It only updates the ReferrerUserID if it's not already set; otherwise, it returns an error.
func (rdb *ReferralDB) AddReferrerMapping(referralData ReferralUserTable) (*ReferralUserTable, error) {
	var existingEntry ReferralUserTable

	// Check if the user address already exists
	if err := db.Where("user_address = ?", referralData.UserAddress).First(&existingEntry).Error; err == nil {
		// Entry exists, check if ReferrerUserID is already set
		if existingEntry.ReferrerUserID != "" {
			// ReferrerUserID already exists, return an error
			return nil, errors.New("referrer user ID already present for this user")
		}

		// ReferrerUserID is not set, so we can update it
		if referralData.ReferrerUserID != "" {
			existingEntry.ReferrerUserID = referralData.ReferrerUserID

			// Save the updated referrerUserID
			if err := db.Save(&existingEntry).Error; err != nil {
				return nil, err
			}
		}
		return &existingEntry, nil
	}

	// Entry does not exist, create a new one
	// Entry does not exist, create a new one using the CreateReferralUser function
	createdUser, err := rdb.CreateReferralUser(referralData)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

// CreateReferralUser creates a new entry in the referral_users table, but skips if the same user address already exists
func (*ReferralDB) CreateReferralUser(referralUserData ReferralUserTable) (*ReferralUserTable, error) {
	// Check if the referral user already exists
	var existingEntry ReferralUserTable
	if err := db.Where("user_address = ?", referralUserData.UserAddress).First(&existingEntry).Error; err == nil {
		// Entry already exists, skip creation
		return &existingEntry, nil
	}

	// Inline call to check if the subaccount has a username to use as the referral code
	var subaccount SubaccountTable
	if err := db.Where("eth_address = ?", referralUserData.UserAddress).First(&subaccount).Error; err == nil && subaccount.UserName != nil && *subaccount.UserName != "" {
		referralUserData.ReferralCode = *subaccount.UserName + ".logX"
	} else if referralUserData.ReferralCode == "" {
		referralUserData.ReferralCode = GenerateReferralCode(referralUserData.UserAddress) + ".logX"
	}

	// Create the new referral user entry
	if err := db.Create(&referralUserData).Error; err != nil {
		return nil, err
	}
	return &referralUserData, nil
}

// GetReferralUserByCode fetches a referral user by the referral code only if the user is active
func (rdb *ReferralDB) GetReferralUserByCode(referralCode string) (*ReferralUserTable, error) {
	var referralUser ReferralUserTable

	// Fetch the referral user where the referral code matches and the user is active
	if err := db.Where("referral_code = ?", referralCode).First(&referralUser).Error; err != nil {
		return nil, err
	}

	// Return the referral user if found and active
	return &referralUser, nil
}

// GetReferralUserByAddress fetches a referral user by the user address
func (rdb *ReferralDB) GetReferralUserByAddress(userAddress string) (*ReferralUserTable, bool, error) {
	var referralUser ReferralUserTable

	// Fetch the referral user by user address
	err := db.Where("user_address = ?", userAddress).First(&referralUser).Error
	if err != nil {
		// Handle the creation of the new user
		newReferralUser := ReferralUserTable{
			UserAddress: userAddress,
			IsActive:    false, // Initially inactive
		}
		createdUser, createErr := rdb.CreateReferralUser(newReferralUser)
		if createErr != nil {
			return nil, false, createErr
		}
		referralUser = *createdUser
	}

	// Attempt to activate the referral user
	status, err := rdb.ActivateReferralUser(referralUser.UserAddress)
	if err != nil {
		return nil, false, fmt.Errorf("failed to activate referral user: %w", err)
	}
	if !status {
		return &referralUser, false, nil // Return with false status if activation was unsuccessful
	}

	// Refetch the referral user after activation to get updated details
	if err := db.Where("user_address = ?", userAddress).First(&referralUser).Error; err != nil {
		return nil, false, err
	}

	// Return the referral user, true status, and nil error if successful
	return &referralUser, true, nil
}

// GetReferralByReferredUserID fetches a referral entry by the referred user ID
func (*ReferralDB) GetReferralByReferredUserID(referredUserID string) (*ReferralUserTable, error) {
	var referral ReferralUserTable
	if err := db.Where("user_address = ?", referredUserID).First(&referral).Error; err != nil {
		return nil, err
	}
	return &referral, nil
}

// NOTE: This is hardcoded for BROKER ID 1
// updates the IsActive state to true for a given user address if there are FillTable entries for the subaccountId
func (*ReferralDB) ActivateReferralUser(userAddress string) (bool, error) {
	// Get the referral user entry by user address
	var referralUser ReferralUserTable
	if err := db.Where("user_address = ?", userAddress).First(&referralUser).Error; err != nil {
		return false, err
	}

	// If IsActive is already true, no need to update
	if referralUser.IsActive {
		return true, nil
	}

	// Get Subaccount ID from the referral user
	subaccountId := cutils.CreateSubaccountId(1, userAddress, 1)

	// Check if there are any entries in FillTable for the given subaccountId
	fillDB := &FillDB{}
	if !fillDB.EntryExistBySubaccountId(subaccountId) {
		return false, nil
	}

	// Update the IsActive state to true
	referralUser.IsActive = true
	if err := db.Save(&referralUser).Error; err != nil {
		return false, err
	}

	return true, nil
}

// Count of referrer for given referrerUserId
func (*ReferralDB) CountReferredUserID(referrerUserID string) (int64, error) {
	var count int64

	err := db.Table("referral_tables").
		Joins("JOIN referral_user_tables ON referral_tables.referred_user_id = referral_user_tables.user_address").
		Where("referral_tables.referrer_user_id = ? AND referral_user_tables.is_active = ?", referrerUserID, true).
		Count(&count).Error

	if err != nil {
		return 0, err
	}
	return count, nil
}

// / for inital time while migrating usernames as referral codes
func (*ReferralDB) UpdateReferralCodesWithUsernamesByIDRange(startID, endID uint) (uint, error) {
	var referralUsers []ReferralUserTable

	// Fetch all entries in the given ID range
	if err := db.Where("id >= ? AND id <= ?", startID, endID).Find(&referralUsers).Error; err != nil {
		return startID, err
	}

	// Track the last successfully processed ID
	var lastProcessedID uint = startID

	// Iterate over each referral user in the specified ID range
	for _, referralUser := range referralUsers {
		var subaccount SubaccountTable

		// Try to find the corresponding username in SubaccountTable based on the user address
		err := db.Where("eth_address = ?", referralUser.UserAddress).First(&subaccount).Error

		// If subaccount is found and UserName is valid, append the UserName with '.logX'
		if err == nil && subaccount.UserName != nil && *subaccount.UserName != "" {
			referralUser.ReferralCode = *subaccount.UserName + ".logX"
		} else {
			// If subaccount is not found or UserName is empty, append '.logX' to the existing ReferralCode
			referralUser.ReferralCode = referralUser.ReferralCode + ".logX"
		}

		// Save the updated referral code in the database
		if err := db.Save(&referralUser).Error; err != nil {
			return lastProcessedID, err // Return the last processed ID if saving fails
		}

		// Update the last successfully processed ID
		lastProcessedID = referralUser.ID
	}

	// Return the last processed ID and no error if all went well
	return lastProcessedID, nil
}

func (referralDB *ReferralDB) UpdateReferralCodeWithUserNameForUser(ethAddress string, newUserName string) error {
	// Fetch the referral entry based on the EthAddress
	var referralUser ReferralUserTable
	result := db.Where("user_address = ?", ethAddress).First(&referralUser)
	if result.Error != nil {
		return result.Error
	}

	// Update the referral code with the new username + ".logX"
	referralUser.ReferralCode = newUserName + ".logX"

	// Save the updated referral user
	return db.Save(&referralUser).Error
}

// GetReferrerMappingForActiveUsers takes an array of user addresses and returns a map of
// referrer user IDs where IsActive is true
func (*ReferralDB) GetReferrerMappingForActiveUsers(userAddresses []string) (map[string]string, error) {
	// Map to store the referrer user IDs for active users
	referrerMapping := make(map[string]string)

	if len(userAddresses) == 0 {
		return referrerMapping, errors.New("user address list is empty")
	}

	var referralUsers []ReferralUserTable
	// Add condition to only include entries where ReferrerUserID is not an empty string
	err := db.Where("user_address IN (?) AND is_active = ? AND referrer_user_id != ?", userAddresses, true, "").Find(&referralUsers).Error
	if err != nil {
		return nil, err
	}

	// Populate the referrer mapping for active users with non-empty ReferrerUserID
	for _, user := range referralUsers {
		referrerMapping[user.UserAddress] = user.ReferrerUserID
	}

	return referrerMapping, nil
}
func (*ReferralDB) CountActiveReferralsByReferrerID(referrerUserID string) (int64, error) {
	var count int64

	err := db.Table("referral_user_tables").
		Where("referrer_user_id = ? AND is_active = ?", referrerUserID, true).
		Count(&count).Error

	if err != nil {
		xlog.Errorf(err.Error())
		return 0, err
	}

	return count, nil
}
