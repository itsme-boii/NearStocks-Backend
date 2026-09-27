package db

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"

	"gorm.io/gorm"
)

type LogxTokenDB struct{}

// GetLogxTokenUserByUserAddress fetches the LogxTokenUserTable record for a given user address
func (*LogxTokenDB) GetLogxTokenUserByUserAddress(userAddress string) (*LogxTokenUserTable, error) {
	var tokenUserData LogxTokenUserTable

	// Fetch the record from the database
	if err := db.Where("user_address = ?", userAddress).First(&tokenUserData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			xlog.Infof(fmt.Sprintf("No record found for user_address: %s", userAddress))
			return nil, nil
		}
		xlog.Errorf(fmt.Sprintf("Error fetching LogxTokenUserTable for user_address: %s, Error: %s", userAddress, err.Error()))
		return nil, err
	}

	return &tokenUserData, nil
}

// UpdateClaimedAmtByUserAddress updates the ClaimedAmt for a given user address
func (*LogxTokenDB) UpdateClaimedAmtInLogxTokenUserTable(userAddress string, claimedAmt ctypes.BigInt) error {
	// Find the user record by user address
	var tokenUserData LogxTokenUserTable
	if err := db.Where("user_address = ?", userAddress).First(&tokenUserData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			xlog.Infof(fmt.Sprintf("No record found for user_address: %s", userAddress))
			return fmt.Errorf("no record found for user address: %s", userAddress)
		}
		xlog.Errorf(fmt.Sprintf("Error fetching LogxTokenUserTable for user_address: %s, Error: %s", userAddress, err.Error()))
		return err
	}

	// Update the ClaimedAmt
	tokenUserData.ClaimedAmt = claimedAmt

	// Save the updated record to the database
	if err := db.Save(&tokenUserData).Error; err != nil {
		xlog.Errorf(fmt.Sprintf("Error updating ClaimedAmt for user_address: %s, Error: %s", userAddress, err.Error()))
		return err
	}

	xlog.Infof(fmt.Sprintf("Successfully updated ClaimedAmt for user_address: %s", userAddress))
	return nil
}

// UpdateClaimedAmtInVestingTable adds the claimed amount to the existing ClaimedAmt in the TokenVestingTable for a given userAddress and vesting ID
func (*LogxTokenDB) UpdateClaimedAmtInVestingTable(userAddress string, vestingID uint, claimedAmt ctypes.BigInt) error {
	// Find the vesting record by userAddress and ID
	var vestingData TokenVestingTable
	if err := db.Where("user_address = ? AND id = ?", userAddress, vestingID).First(&vestingData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			xlog.Infof(fmt.Sprintf("No record found for user_address: %s with ID: %d", userAddress, vestingID))
			return fmt.Errorf("no record found for user_address: %s with ID: %d", userAddress, vestingID)
		}
		xlog.Errorf(fmt.Sprintf("Error fetching TokenVestingTable for user_address: %s, ID: %d, Error: %s", userAddress, vestingID, err.Error()))
		return err
	}

	// Add the new claimed amount to the current ClaimedAmt
	vestingData.ClaimedAmt = vestingData.ClaimedAmt.Add(claimedAmt)

	// Save the updated record to the database
	if err := db.Save(&vestingData).Error; err != nil {
		xlog.Errorf(fmt.Sprintf("Error updating ClaimedAmt for user_address: %s, ID: %d, Error: %s", userAddress, vestingID, err.Error()))
		return err
	}

	xlog.Infof(fmt.Sprintf("Successfully updated ClaimedAmt for user_address: %s, ID: %d", userAddress, vestingID))
	return nil
}

// GetAllTokenVestingsByUserAddress fetches all TokenVestingTable entries for a given user address
func (*LogxTokenDB) GetAllTokenVestingsByUserAddress(userAddress string) ([]TokenVestingTable, error) {
	var vestingData []TokenVestingTable

	// Fetch all records for the given userAddress
	if err := db.Where("user_address = ?", userAddress).Find(&vestingData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			xlog.Infof(fmt.Sprintf("No vesting records found for user_address: %s", userAddress))
			return nil, nil
		}
		xlog.Errorf(fmt.Sprintf("Error fetching TokenVestingTable for user_address: %s, Error: %s", userAddress, err.Error()))
		return nil, err
	}

	return vestingData, nil
}

// UpdateIsBlockedStatus updates the IsBlocked column for a given userAddress
func (*LogxTokenDB) UpdateIsBlockedStatus(userAddress string, isBlocked bool) error {
	var tokenUserData LogxTokenUserTable

	// Fetch the record for the given userAddress
	if err := db.Where("user_address = ?", userAddress).First(&tokenUserData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			xlog.Infof(fmt.Sprintf("No record found for user_address: %s", userAddress))
			return nil
		}
		xlog.Errorf(fmt.Sprintf("Error fetching LogxTokenUserTable for user_address: %s, Error: %s", userAddress, err.Error()))
		return err
	}

	// Update the IsBlocked column
	tokenUserData.IsBlocked = isBlocked

	// Save the updated record back to the database
	if err := db.Save(&tokenUserData).Error; err != nil {
		xlog.Errorf(fmt.Sprintf("Error updating IsBlocked status for user_address: %s, Error: %s", userAddress, err.Error()))
		return err
	}

	// Log the status update
	xlog.Infof(fmt.Sprintf("Updated IsBlocked status for user_address: %s to %v", userAddress, isBlocked))

	return nil
}

// ShouldBlockUserByVesting checks if a user should be blocked based on their vesting types
func (db *LogxTokenDB) CanBeBlockedCheckvesting(userAddress string) (bool, error) {
	// Fetch all vesting records for the given user address
	vestingData, err := db.GetAllTokenVestingsByUserAddress(userAddress)
	if err != nil {
		return true, err // In case of an error, return true and the error
	}

	// If no vesting data is found, the user can be blocked
	if len(vestingData) == 0 {
		xlog.Infof("User address %s is not in the vesting list. The user can be blocked.", userAddress)
		return true, nil
	}

	// Default to blocking the user unless a valid vesting type is found
	shouldBlock := true

	// Loop through all vesting data for the user
	for _, vesting := range vestingData {
		if vesting.VestingType == "KOL COMISSION" ||
			vesting.VestingType == "Referral" ||
			vesting.VestingType == "KOL" ||
			vesting.VestingType == "OTC DEAL" {
			// If any vesting type matches, the user should not be blocked
			shouldBlock = false
			break
		}
	}

	return shouldBlock, nil
}

func (*LogxTokenDB) CalculateTotalClaimedAmounts() (*big.Int, *big.Int, int64, error) {
	var logxUserSum string
	var vestingSum string
	var distinctUserCount int64

	// Fetch the sum of claimed_amt and count of distinct user_address in logx_token_user_tables
	result := db.Raw(`
		SELECT 
			COALESCE(SUM(CAST(claimed_amt AS NUMERIC)), '0') AS total_claimed_amt, 
			COUNT(DISTINCT user_address) AS distinct_user_count
		FROM logx_token_user_tables
		WHERE CAST(claimed_amt AS NUMERIC) > 0
	`).Row()

	// Scan the result into logxUserSum and distinctUserCount
	err := result.Scan(&logxUserSum, &distinctUserCount)
	if err != nil {
		xlog.Errorf("Error calculating total claimed amount and distinct user count: %v", err)
		return nil, nil, 0, err
	}

	// Fetch the sum of claimed_amt from token_vesting_tables
	result = db.Raw(`
		SELECT COALESCE(SUM(CAST(claimed_amt AS NUMERIC)), '0') 
		FROM token_vesting_tables
	`).Row()

	err = result.Scan(&vestingSum)
	if err != nil {
		xlog.Errorf("Error calculating total claimed amount from TokenVestingTable: %v", err)
		return nil, nil, 0, err
	}

	// Convert the sums to big.Int
	logxUserSumBigInt := new(big.Int)
	vestingSumBigInt := new(big.Int)

	// Set the string values for the BigInt variables
	_, ok1 := logxUserSumBigInt.SetString(logxUserSum, 10)
	_, ok2 := vestingSumBigInt.SetString(vestingSum, 10)
	if !ok1 || !ok2 {
		xlog.Errorf("Error converting string to big.Int")
		return nil, nil, 0, fmt.Errorf("error converting string to big.Int")
	}

	// Return the two claimed amounts and the distinct user count
	return logxUserSumBigInt, vestingSumBigInt, distinctUserCount, nil
}
