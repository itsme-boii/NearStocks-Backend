package logxTokenUtils

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"time"
)

// LogxTokenUtils provides utilities for interacting with LogxToken data
type LogxTokenUtils struct {
	TokenDB       *db.LogxTokenDB
	CountryCodeDB *db.CountryCodeDB
}

// NewLogxTokenUtils initializes the LogxTokenUtils with the required database instance
func NewLogxTokenUtils() *LogxTokenUtils {
	return &LogxTokenUtils{
		TokenDB:       &db.LogxTokenDB{}, // Initializing LogxTokenDB
		CountryCodeDB: &db.CountryCodeDB{},
	}
}

// LogxTokenUserResponse represents the response structure for token user data
type LogxTokenUserResponse struct {
	TotalAmount      ctypes.BigInt `json:"total_amount"`
	ClaimableAmount  ctypes.BigInt `json:"claimable_amount"` // This now represents the amount that can still be claimed
	HasVesting       bool          `json:"has_vesting"`
	IsBlocked        bool          `json:"is_blocked"`
	MinTimeRemaining int64         `json:"min_time_remaining"` // Minimum time remaining for next unlock
	NextUnlockAmount ctypes.BigInt `json:"next_unlock_amount"` // Next unlock amount
}

// Define the response structure for the checker scenario
type LogxTokenUserResponseChecker struct {
	AmountDay1         ctypes.BigInt `json:"amount_day1"`
	AmountUnderVesting ctypes.BigInt `json:"amount_under_vesting"`
	HasVesting         bool          `json:"has_vesting"`
	IsBlocked          bool          `json:"is_blocked"`
}

type LogxTokenVestingResponseChecker struct {
	AmountDay1         ctypes.BigInt `json:"amount_day1"`
	AmountUnderVesting ctypes.BigInt `json:"amount_under_vesting"`
	HasVesting         bool          `json:"has_vesting"`
	IsBlocked          bool          `json:"is_blocked"`
	ClaimedAmount      ctypes.BigInt `json:"claimed_amount"`
}

// GetTokenUserDataByUserAddress fetches and returns a structured response for a user by their address
func (utils *LogxTokenUtils) GetTokenUserDataByUserAddress(userAddress string) (*LogxTokenUserResponse, error) {
	// Fetch the user data from the database
	tokenUserData, err := utils.TokenDB.GetLogxTokenUserByUserAddress(userAddress)
	if err != nil {
		xlog.Errorf(fmt.Sprintf("Error fetching token user data for userAddress %s: %v", userAddress, err))
		return nil, err
	}

	// If no record is found, return a response with zero amounts and false flags
	if tokenUserData == nil {
		return &LogxTokenUserResponse{
			TotalAmount:      ctypes.NewBigInt(big.NewInt(0)),
			ClaimableAmount:  ctypes.NewBigInt(big.NewInt(0)),
			HasVesting:       false,
			IsBlocked:        true,
			MinTimeRemaining: 0,
			NextUnlockAmount: ctypes.NewBigInt(big.NewInt(0)),
		}, nil
	}

	// Check if the user is blocked
	if tokenUserData.IsBlocked && !tokenUserData.HasVesting {
		return &LogxTokenUserResponse{
			TotalAmount:      ctypes.NewBigInt(big.NewInt(0)),
			ClaimableAmount:  ctypes.NewBigInt(big.NewInt(0)),
			HasVesting:       false,
			IsBlocked:        true,
			MinTimeRemaining: 0,
			NextUnlockAmount: ctypes.NewBigInt(big.NewInt(0)),
		}, nil
	}

	// Check if the user is blocked by IP
	// ipBlocked, err := utils.IsIPBlocked(userAddress)
	// if err != nil {
	// 	xlog.Errorf(fmt.Sprintf("Error checking IP block status for userAddress %s: %v", userAddress, err))
	// 	return nil, err
	// }

	// If user is blocked by IP, return a blocked response
	// if ipBlocked && !tokenUserData.HasVesting {
	// 	return &LogxTokenUserResponse{
	// 		TotalAmount:      ctypes.NewBigInt(big.NewInt(0)),
	// 		ClaimableAmount:  ctypes.NewBigInt(big.NewInt(0)),
	// 		HasVesting:       false,
	// 		IsBlocked:        true,
	// 		MinTimeRemaining: 0,
	// 		NextUnlockAmount: ctypes.NewBigInt(big.NewInt(0)),
	// 	}, nil
	// }

	// Check if user has vesting enabled
	if tokenUserData.HasVesting {
		// Fetch all vesting records for the given userAddress
		vestingData, err := utils.TokenDB.GetAllTokenVestingsByUserAddress(userAddress)
		if err != nil {
			xlog.Errorf(fmt.Sprintf("Error fetching vesting data for userAddress %s: %v", userAddress, err))
			return nil, err
		}
		// Get the current time
		now := time.Now().UTC()
		// Calculate vesting amount by passing tokenUserData and vestingData
		claimableAmount, totalAmount, minTimeRemaining, nextUnlockAmount, _, err := utils.ClaimableAmountifVesting(tokenUserData, vestingData, now)
		if err != nil {
			xlog.Errorf(fmt.Sprintf("Error calculating vesting amount for userAddress %s: %v", userAddress, err))
			return nil, err
		}

		// Return the vesting information in the response
		return &LogxTokenUserResponse{
			TotalAmount:      totalAmount,
			ClaimableAmount:  claimableAmount,
			HasVesting:       true,
			IsBlocked:        false,
			MinTimeRemaining: minTimeRemaining,
			NextUnlockAmount: nextUnlockAmount,
		}, nil
	}

	// If user is not blocked and has no vesting, return claimable and claimed amounts
	totalAmount := tokenUserData.TotalAmt.Copy()
	claimableAmount := tokenUserData.TotalAmt.Sub(tokenUserData.ClaimedAmt)

	return &LogxTokenUserResponse{
		TotalAmount:      totalAmount,
		ClaimableAmount:  claimableAmount,
		HasVesting:       false,
		IsBlocked:        false,
		MinTimeRemaining: 0,
		NextUnlockAmount: ctypes.NewBigInt(big.NewInt(0)),
	}, nil
}

// UpdateTokenUserDataByUserAddress updates the claimed amount for a user by their address
// Note 1: userAddress here is the checksummed address
// Note 2: This function should always be called in a locked context
func (utils *LogxTokenUtils) UpdateTokenUserDataByUserAddress(userAddress string, amount ctypes.BigInt) (_success bool, _hasVested bool, _err error) {
	// Fetch airdrop token data as well as flag to know if user has vesting
	tokenUserData, err := utils.TokenDB.GetLogxTokenUserByUserAddress(userAddress)
	if err != nil {
		xlog.Errorf(fmt.Sprintf("Error fetching token user data for userAddress %s: %v", userAddress, err))
		return false, false, err
	}

	// If no record is found, return an error
	if tokenUserData == nil {
		return false, false, fmt.Errorf("no user data found for userAddress: %s", userAddress)
	}

	// Check if the user is blocked
	if tokenUserData.IsBlocked && !tokenUserData.HasVesting {
		xlog.Infof(fmt.Sprintf("User is blocked for userAddress %s", userAddress))
		return false, false, fmt.Errorf("user is blocked")
	}

	// Check if the user is blocked by IP
	// ipBlocked, err := utils.IsIPBlocked(userAddress)
	// if err != nil {
	// 	xlog.Errorf(fmt.Sprintf("Error checking IP block status for userAddress %s: %v", userAddress, err))
	// 	return false, err
	// }

	// If user is blocked by IP, return an error
	// if ipBlocked && !tokenUserData.HasVesting {
	// 	xlog.Infof(fmt.Sprintf("User is blocked by IP for userAddress %s", userAddress))
	// 	return false, fmt.Errorf("user is blocked by IP")
	// }

	// Check if user has vesting
	if tokenUserData.HasVesting {
		// Fetch all vesting records for the given userAddress
		vestingData, err := utils.TokenDB.GetAllTokenVestingsByUserAddress(userAddress)
		if err != nil {
			xlog.Errorf(fmt.Sprintf("Error fetching vesting data for userAddress %s: %v", userAddress, err))
			return false, true, err
		}

		// Call a helper function to update the vesting data with the claimed amount
		if err := utils.UpdateClaimedAmtInVesting(userAddress, vestingData, amount, tokenUserData); err != nil {
			xlog.Errorf(fmt.Sprintf("Error updating vesting claimed amount for userAddress %s: %v", userAddress, err))
			return false, true, err
		}

		xlog.Infof(fmt.Sprintf("Successfully updated vesting claimed amount for userAddress %s", userAddress))
		return true, true, nil // Return true to indicate success
	}
	// AUDIT: @mananbordia - This logic can be removed as Airdrop claim should ideally not be allowed
	// Calculate remaining amount: TotalAmt - ClaimedAmt
	remainingAmt := tokenUserData.TotalAmt.Sub(tokenUserData.ClaimedAmt)

	// Check if the input amount is equal to the remaining claimable amount
	if remainingAmt.Cmp(amount) != 0 {
		return false, false, nil // Return false if the amounts are not equal
	}

	// If user does not have vesting, update the claimed amount in LogxTokenUserTable
	err = utils.TokenDB.UpdateClaimedAmtInLogxTokenUserTable(userAddress, amount)
	if err != nil {
		xlog.Errorf(fmt.Sprintf("Error updating claimed amount for userAddress %s: %v", userAddress, err))
		return false, false, err
	}

	xlog.Infof(fmt.Sprintf("Successfully updated claimed amount for userAddress %s", userAddress))
	return true, false, nil // Return true to indicate success
}

// UpdateClaimedAmtInVesting updates the claimed amount across the vesting records if the total claimable amount matches the input amount
func (utils *LogxTokenUtils) UpdateClaimedAmtInVesting(userAddress string, vestingData []db.TokenVestingTable, amount ctypes.BigInt, tokenUserData *db.LogxTokenUserTable) error {

	// Call ClaimableAmountifVesting to get the total claimable amount
	// Get the current time
	now := time.Now().UTC()

	totalClaimableAmount, _, _, _, vestingAmountMap, err := utils.ClaimableAmountifVesting(tokenUserData, vestingData, now)
	if err != nil {
		xlog.Errorf(fmt.Sprintf("Error calculating claimable amount for userAddress %s: %v", userAddress, err))
		return err
	}

	// Check if the input amount is equal to the total claimable amount
	if totalClaimableAmount.Cmp(amount) != 0 {
		return fmt.Errorf("input amount does not match the total claimable amount for userAddress: %s", userAddress)
	}

	// Calculate remaining amount: TotalAmt - ClaimedAmt
	remainingAmt := tokenUserData.TotalAmt.Sub(tokenUserData.ClaimedAmt)

	// Check if user not claimed from normal trading token
	if remainingAmt.Cmp(ctypes.NewBigInt(big.NewInt(0))) > 0 {
		// If user does not have vesting, update the claimed amount in LogxTokenUserTable
		err = utils.TokenDB.UpdateClaimedAmtInLogxTokenUserTable(userAddress, remainingAmt)
		if err != nil {
			xlog.Errorf(fmt.Sprintf("Error updating claimed amount for userAddress %s: %v", userAddress, err))
			return err
		}
	}

	// If the amounts match, proceed to update each vesting entry
	for vestingID, amountVested := range vestingAmountMap {
		// Update the claimed amount for each vesting entry
		err := utils.TokenDB.UpdateClaimedAmtInVestingTable(userAddress, vestingID, amountVested)
		if err != nil {
			xlog.Errorf(fmt.Sprintf("Error updating claimed amount for vesting ID %d, userAddress %s: %v", vestingID, userAddress, err))
			return err
		}
		xlog.Infof(fmt.Sprintf("Successfully updated claimed amount for vesting ID %d, userAddress %s", vestingID, userAddress))
	}

	return nil
}

// Return total claimable amount including Airdrop and Vesting
func (utils *LogxTokenUtils) ClaimableAmountifVesting(tokenUserData *db.LogxTokenUserTable, vestingData []db.TokenVestingTable, currentTime time.Time) (ctypes.BigInt, ctypes.BigInt, int64, ctypes.BigInt, map[uint]ctypes.BigInt, error) {
	// Airdrop claimable amount
	totalClaimableAmount := tokenUserData.TotalAmt.Sub(tokenUserData.ClaimedAmt)

	// Initialize the total amount including tokenUserData.TotalAmt and vesting amounts
	totalAmount := tokenUserData.TotalAmt.Copy()

	// Variables to track the minimum time remaining in minutes and the next unlock amount
	var minTimeRemaining int64 = -1 // Start with an invalid large value
	var nextUnlockAmountVal ctypes.BigInt = ctypes.NewBigInt(big.NewInt(0))

	// Map to store the vested amount for each vesting ID
	vestingAmountMap := make(map[uint]ctypes.BigInt)

	totalClaimedAmountTillNow := tokenUserData.ClaimedAmt
	// Loop through all vesting records and calculate unclaimed vested amounts
	for _, vesting := range vestingData {
		// Calculate the unclaimed (vested) amount for this vesting entry
		amountVested, timeRemainingDuration, nextUnlockAmount := utils.CalculateUnclaimedVestedAmount(vesting, currentTime)

		// Add the unclaimed vested amount to the total claimable amount
		totalClaimableAmount = totalClaimableAmount.Add(amountVested)

		// Add the vesting's TotalAmt to the total amount
		totalAmount = totalAmount.Add(vesting.TotalAmt)
		totalClaimedAmountTillNow = totalClaimedAmountTillNow.Add(vesting.ClaimedAmt)
		// Store the vested amount for this vesting ID in the map
		vestingAmountMap[vesting.ID] = amountVested

		if minTimeRemaining == -1 {
			minTimeRemaining = timeRemainingDuration
		}
		// Track the minimum time remaining in minutes and update nextUnlockAmountVal
		if timeRemainingDuration <= minTimeRemaining {
			if timeRemainingDuration == minTimeRemaining {
				nextUnlockAmountVal = nextUnlockAmountVal.Add(nextUnlockAmount)
			} else {
				nextUnlockAmountVal = nextUnlockAmount
			}
			minTimeRemaining = timeRemainingDuration

		}
	}
	// If no vesting records exist or claimable amount is <= 0, set next unlock info to default
	if totalClaimableAmount.Sign() < 0 {
		return ctypes.NewBigInt(big.NewInt(0)), totalAmount, 0, ctypes.NewBigInt(big.NewInt(0)), vestingAmountMap, nil
	}

	// Add totalClaimableAmount to the result
	combinedAmount := totalClaimableAmount.Add(nextUnlockAmountVal)
	combinedAmount = combinedAmount.Add(totalClaimedAmountTillNow)
	if (combinedAmount).Cmp(totalAmount) > 0 {
		return totalClaimableAmount, totalAmount, 0, ctypes.NewBigInt(big.NewInt(0)), vestingAmountMap, nil
	}
	if minTimeRemaining == -1 {
		minTimeRemaining = 0
	}
	// Return the total claimable amount, the total amount, the minimum time remaining in minutes, the next unlock amount, and the vesting map
	return totalClaimableAmount, totalAmount, minTimeRemaining, nextUnlockAmountVal, vestingAmountMap, nil
}

// CalculateUnlockedAmount calculates the unlocked amount and also returns next unlock info
func (utils *LogxTokenUtils) CalculateUnclaimedVestedAmount(vesting db.TokenVestingTable, currentTime time.Time) (ctypes.BigInt, int64, ctypes.BigInt) {
	// Calculate the total vesting amount that is available after AmtUnlockDay1
	totalVestingAmount := vesting.TotalAmt.Sub(vesting.AmtUnlockDay1)

	// Calculate the maximum claimable amount (TotalAmt - ClaimedAmt)
	maxClaimableAmount := vesting.TotalAmt.Sub(vesting.ClaimedAmt)

	// Check if there is any remaining amount to claim
	if maxClaimableAmount.Sign() <= 0 {
		// If no remaining amount, return 0 for unlocked amount, time remaining, and next unlock amount
		return ctypes.NewBigInt(big.NewInt(0)), 0, ctypes.NewBigInt(big.NewInt(0))
	}

	// Define the hardcoded date "2024-09-24T00:00:00Z"
	// Need to update
	tgeDate := time.Date(2024, 9, 23, 0, 0, 0, 0, time.UTC)

	/*
		Currently Three typw of vesting there -
		1. Vesting which started from September.
		2. Vesting which will start from March without cliff - INVESTOR
		-- so in this case amount unlock day1 will be 0
		3. Vesting which will start from March with cliff i.e on day 1 there will 10% unlock - INVESTORCLIFF
		-- so in this case amount unlock day1 will be 10% of total and remaing will vest at 3% rate from next month

	*/
	if cutils.SliceExists([]string{"INVESTOR", "TEAM"}, vesting.VestingType) {
		tgeDate = time.Date(2025, 2, 22, 0, 0, 0, 0, time.UTC)
	} else if vesting.VestingType == "INVESTORCLIFF" {
		tgeDate = time.Date(2025, 3, 23, 0, 0, 0, 0, time.UTC)
	}
	// Get the current date at 00:00 UTC
	now := currentTime
	currentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if cutils.SliceExists([]string{"INVESTOR", "TEAM"}, vesting.VestingType) {
		if currentDate.Before(tgeDate) {
			totalVestingAmount := vesting.TotalAmt.Sub(vesting.AmtUnlockDay1)

			unlockedAmountinEachInterval := totalVestingAmount.Mul(vesting.UnlockPercentageX100)

			nextUnlockAmount := unlockedAmountinEachInterval.Div(ctypes.NewBigInt(big.NewInt(10000)))

			// Calculate time remaining
			timeRemaining := tgeDate.AddDate(0, 1, 0).Sub(currentDate)

			// Return calculated time remaining as int64, along with the other values
			return ctypes.NewBigInt(big.NewInt(0)), int64(timeRemaining.Minutes()), nextUnlockAmount
		}
	} else if vesting.VestingType == "INVESTORCLIFF" {
		if currentDate.Before(tgeDate) {

			nextUnlockAmount := vesting.AmtUnlockDay1

			// Calculate time remaining
			timeRemaining := tgeDate.Sub(currentDate)

			// Return calculated time remaining as int64, along with the other values
			return ctypes.NewBigInt(big.NewInt(0)), int64(timeRemaining.Minutes()), nextUnlockAmount
		}
	} else {
		if currentDate.Before(tgeDate) {
			return ctypes.NewBigInt(big.NewInt(0)), 0, ctypes.NewBigInt(big.NewInt(0))
		}
	}
	// To handle case if current date >  last date of vesting
	vestingPercentage := ctypes.NewBigInt(big.NewInt(10000))
	vestingUnits := vestingPercentage.Div(vesting.UnlockPercentageX100)
	vestingUnitsInt := int(vestingUnits.Val.Int64())
	expectedVestingEndDate := tgeDate.AddDate(0, 0, vestingUnitsInt*int(vesting.Frequency))

	if currentDate.After(expectedVestingEndDate) {
		currentDate = expectedVestingEndDate
	}

	// Calculate the time difference in days from the hardcoded date
	duration := currentDate.Sub(tgeDate)

	daysPassed := int64(duration.Hours() / 24)
	// Determine the frequency (1 for daily, 30 for monthly)

	frequency := vesting.Frequency
	// Calculate the number of time units (floor value based on days)
	timeUnits := daysPassed / frequency

	// Calculate the unlocked percentage based on time units
	unlockedPercentage := vesting.UnlockPercentageX100.Mul(ctypes.NewBigInt(big.NewInt(timeUnits)))

	// Step 1: Multiply totalVestingAmount by the unlocked percentage
	unlockedAmountinEachInterval := totalVestingAmount.Mul(unlockedPercentage)

	// Step 2: Divide the result by 100 to get the final unlocked amount --- 100 for percentgae conversion into num
	unlockedAmountTillNowFromTge := unlockedAmountinEachInterval.Div(ctypes.NewBigInt(big.NewInt(10000)))

	claimedAmountFromVestingAmount := vesting.ClaimedAmt
	// If the "updated_at" column is less than the hardcoded date, add AmtUnlockDay1 to unlockedAmount
	if vesting.UpdatedAt.Before(tgeDate) {
		unlockedAmountTillNowFromTge = unlockedAmountTillNowFromTge.Add(vesting.AmtUnlockDay1)
	} else {
		claimedAmountFromVestingAmount = claimedAmountFromVestingAmount.Sub(vesting.AmtUnlockDay1)
	}

	// total vested amount - already claimed amount
	unlockedAmountTillNowFromTge = unlockedAmountTillNowFromTge.Sub(claimedAmountFromVestingAmount)

	// Calculate the time remaining and the next unlock amount using NextUnlockInfo
	timeRemaining, nextUnlockAmount := utils.NextUnlockInfo(vesting, currentTime, tgeDate)

	// Return the unlocked amount, time remaining, and the next unlock amount
	return unlockedAmountTillNowFromTge, timeRemaining, nextUnlockAmount
}

// NextUnlockInfo calculates the time remaining for the next unlock and the amount to be unlocked in the next iteration
func (utils *LogxTokenUtils) NextUnlockInfo(vesting db.TokenVestingTable, currentTime time.Time, tgeDate time.Time) (int64, ctypes.BigInt) {
	// Calculate the total vesting amount that is available after AmtUnlockDay1
	totalVestingAmount := vesting.TotalAmt.Sub(vesting.AmtUnlockDay1)

	// Step 1: Multiply totalVestingAmount by the unlocked percentage
	unlockedAmountinEachInterval := totalVestingAmount.Mul(vesting.UnlockPercentageX100)

	// Step 2: Divide the result by 100 to get the final unlocked amount
	nextUnlockAmount := unlockedAmountinEachInterval.Div(ctypes.NewBigInt(big.NewInt(10000)))

	// Define the reference day of the month (24th) for monthly unlocks
	refDay := 24

	now := currentTime

	var nextUnlockTime time.Time
	// Calculate the time for the next unlock based on the vesting frequency
	if vesting.Frequency == 1 {
		// Daily vesting: the next unlock is tomorrow at 00:00 UTC
		nextUnlockTime = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	} else if vesting.Frequency == 30 {
		// Monthly vesting: The next unlock should happen on the 24th of the current or next month

		// Check if today's date is past the 24th of the current month
		if now.Day() < refDay {
			// If today is before the 24th, set next unlock to the 24th of the current month
			nextUnlockTime = time.Date(now.Year(), now.Month(), refDay, 0, 0, 0, 0, time.UTC)
		} else {
			// If today is on or after the 24th, set next unlock to the 24th of the next month
			nextUnlockTime = time.Date(now.Year(), now.Month()+1, refDay, 0, 0, 0, 0, time.UTC)
		}
	} else if vesting.Frequency == 90 {

		// Calculate 3 months after TGE date
		nextUnlockTimeVal := tgeDate.AddDate(0, 3, 0)

		if now.Before(nextUnlockTimeVal) {
			// If the current date is before the next unlock (TGE + 3 months), set nextUnlockTime to nextUnlockTimeVal
			nextUnlockTime = nextUnlockTimeVal
		} else {
			// If the current date is after the next unlock (TGE + 3 months), set nextUnlockTime to the current time
			nextUnlockTime = now
		}
	}

	// Calculate the time remaining until the next unlock
	timeRemaining := nextUnlockTime.Sub(now)

	timeRemainingInMinutes := int64(timeRemaining.Minutes())

	// Return the time remaining and the next unlock amount
	return timeRemainingInMinutes, nextUnlockAmount
}

// IsIPBlocked is  to check if a user is blocked based on their IP
func (utils *LogxTokenUtils) IsIPBlocked(subaccountID string) (bool, error) {
	// List of restricted countries
	var notAllowedCountries = []string{
		"CU", "IR", "KP", "US", // List of countries where access is restricted
	}

	// Fetch the country code for the given subaccountID
	countryCodeRecord := utils.CountryCodeDB.GetCountryCodeBySubaccount(subaccountID)
	if countryCodeRecord == nil {
		xlog.Infof("country code not found for this Address %s", subaccountID)
		return false, nil ////////// Need to check if this required or return false
	}

	// Access the CountryCode field from the fetched record
	countryCode := countryCodeRecord.CountryCode

	// Check if the country is in the not allowed list
	isBlocked := false
	for _, notAllowedCountry := range notAllowedCountries {
		if countryCode == notAllowedCountry {
			// If the country is in the not allowed list, the user is blocked
			isBlocked = true
			break
		}
	}
	// Return the blocking status
	return isBlocked, nil
}

// GetTokenUserDataForChecker fetches and returns a structured response for a user by their address for the checker scenario
func (utils *LogxTokenUtils) GetTokenUserDataForChecker(userAddress string) (*LogxTokenUserResponseChecker, error) {
	// Fetch the user data from the database
	tokenUserData, err := utils.TokenDB.GetLogxTokenUserByUserAddress(userAddress)
	if err != nil {
		xlog.Errorf(fmt.Sprintf("Error fetching token user data for userAddress %s: %v", userAddress, err))
		return nil, err
	}
	// If no record is found, return a response with zero amounts and false flags
	if tokenUserData == nil {
		xlog.Errorf("tokenUserData for userAddress 2: %s, data: %v, error: %v", userAddress, tokenUserData, err)
		return &LogxTokenUserResponseChecker{
			AmountDay1:         ctypes.NewBigInt(big.NewInt(0)),
			AmountUnderVesting: ctypes.NewBigInt(big.NewInt(0)),
			HasVesting:         false,
			IsBlocked:          false,
		}, nil
	}

	// ipBlocked, err := utils.IsIPBlocked(userAddress)
	// if err != nil {
	// 	xlog.Errorf(fmt.Sprintf("Error checking IP block status for userAddress %s: %v", userAddress, err))
	// 	return nil, err
	// }

	// // If user is blocked by IP, return a blocked response
	// if ipBlocked {
	// 	// Update the IsBlocked status in the database based on the result
	// 	return &LogxTokenUserResponseChecker{
	// 		AmountDay1:         ctypes.NewBigInt(big.NewInt(0)),
	// 		AmountUnderVesting: ctypes.NewBigInt(big.NewInt(0)),
	// 		HasVesting:         false,
	// 		IsBlocked:          true,
	// 	}, nil
	// }
	// xlog.Errorf("tokenUserData for userAddress 3: %s, data: %v, error: %v", userAddress, tokenUserData, err)
	// Check if the user is blocked
	// if tokenUserData.IsBlocked {
	// 	xlog.Errorf("tokenUserData for userAddress 4: %s, data: %v, error: %v", userAddress, tokenUserData, err)
	// 	// xlog.Infof(fmt.Sprintf("User is blocked for userAddress %s", userAddress))
	// 	return &LogxTokenUserResponseChecker{
	// 		AmountDay1:         ctypes.NewBigInt(big.NewInt(0)),
	// 		AmountUnderVesting: ctypes.NewBigInt(big.NewInt(0)),
	// 		HasVesting:         false,
	// 		IsBlocked:          true,
	// 	}, nil
	// }
	// xlog.Errorf("tokenUserData for userAddress 5: %s, data: %v, error: %v", userAddress, tokenUserData, err)
	// Check if the user is blocked by IP
	// ipBlocked := false
	// startDate := time.Date(2024, 9, 22, 0, 0, 0, 0, time.UTC)
	// if(tokenUserData.UpdatedAt.Before(startDate)){
	// 	ipBlocked, err = utils.IsIPBlocked(userAddress)
	// 	if err != nil {
	// 		xlog.Errorf(fmt.Sprintf("Error checking IP block status for userAddress %s: %v", userAddress, err))
	// 		return nil, err
	// 	}
	// }
	// xlog.Errorf("tokenUserData for userAddress 6: %s, data: %v, error: %v", userAddress, tokenUserData, err)
	// // If user is blocked by IP, return a blocked response
	// if ipBlocked && !tokenUserData.HasVesting {
	// 	// Update the IsBlocked status in the database based on the result
	// 	err := utils.TokenDB.UpdateIsBlockedStatus(userAddress, ipBlocked)
	// 	if err != nil {
	// 		xlog.Errorf("tokenUserData for userAddress 7: %s, data: %v, error: %v", userAddress, tokenUserData, err)
	// 		return nil, fmt.Errorf("failed to update IsBlocked status for subaccount: %s, error: %v", userAddress, err)
	// 	}
	// 	xlog.Errorf("tokenUserData for userAddress 8: %s, data: %v, error: %v", userAddress, tokenUserData, err)
	// 	return &LogxTokenUserResponseChecker{
	// 		AmountDay1:         ctypes.NewBigInt(big.NewInt(0)),
	// 		AmountUnderVesting: ctypes.NewBigInt(big.NewInt(0)),
	// 		HasVesting:         false,
	// 		IsBlocked:          true,
	// 	}, nil
	// }
	// Check if user has vesting enabled
	if tokenUserData.HasVesting {
		// Fetch all vesting records for the given userAddress
		vestingData, err := utils.TokenDB.GetAllTokenVestingsByUserAddress(userAddress)
		if err != nil {
			xlog.Errorf(fmt.Sprintf("Error fetching vesting data for userAddress %s: %v", userAddress, err))
			return nil, err
		}

		// shouldBlock := true

		// // Loop through all vestingData for the user
		// for _, vesting := range vestingData {
		// 	if vesting.VestingType == "KOL COMISSION" ||
		// 		vesting.VestingType == "Referral" ||
		// 		vesting.VestingType == "KOL" ||
		// 		vesting.VestingType == "OTC DEAL" {
		// 		// If any vesting type matches, we should not block the user
		// 		shouldBlock = false
		// 		break
		// 	}
		// }

		// if ipBlocked && shouldBlock {
		// 	// Update the IsBlocked status in the database based on the result
		// 	err := utils.TokenDB.UpdateIsBlockedStatus(userAddress, ipBlocked)
		// 	if err != nil {
		// 		xlog.Errorf("tokenUserData for userAddress 11: %s, data: %v, error: %v", userAddress, tokenUserData, err)
		// 		return nil, fmt.Errorf("failed to update IsBlocked status for subaccount: %s, error: %v", userAddress, err)
		// 	}
		// 	xlog.Errorf("tokenUserData for userAddress 12: %s, data: %v, error: %v", userAddress, tokenUserData, err)
		// 	return &LogxTokenUserResponseChecker{
		// 		AmountDay1:         ctypes.NewBigInt(big.NewInt(0)),
		// 		AmountUnderVesting: ctypes.NewBigInt(big.NewInt(0)),
		// 		HasVesting:         false,
		// 		IsBlocked:          true,
		// 	}, nil
		// }

		// Calculate total claimable amount including vesting records
		totalClaimableAmount := tokenUserData.TotalAmt.Copy()
		amountUnderVesting := ctypes.NewBigInt(big.NewInt(0))
		amountDay1 := totalClaimableAmount

		// Sum up the vesting amount for all records
		for _, vesting := range vestingData {
			amountDay1 = amountDay1.Add(vesting.AmtUnlockDay1)
			amountUnderVesting = amountUnderVesting.Add(vesting.TotalAmt.Sub(vesting.AmtUnlockDay1))
		}
		xlog.Errorf("tokenUserData for userAddress 13: %s, data: %v, error: %v", userAddress, tokenUserData, err)
		// Return the response for vesting information
		return &LogxTokenUserResponseChecker{
			AmountDay1:         amountDay1,
			AmountUnderVesting: amountUnderVesting,
			HasVesting:         true,
			IsBlocked:          false,
		}, nil
	}
	xlog.Errorf("tokenUserData for userAddress 14: %s, data: %v, error: %v", userAddress, tokenUserData, err)
	// If user is not blocked and has no vesting, return total amount and set amounts to zero
	return &LogxTokenUserResponseChecker{
		AmountDay1:         tokenUserData.TotalAmt.Copy(),
		AmountUnderVesting: ctypes.NewBigInt(big.NewInt(0)),
		HasVesting:         false,
		IsBlocked:          false,
	}, nil
}

func (utils *LogxTokenUtils) GetTokenVestingDataForExternal(userAddress string) (*LogxTokenVestingResponseChecker, error) {
	// Fetch the user data from the database
	tokenUserData, err := utils.TokenDB.GetLogxTokenUserByUserAddress(userAddress)
	if err != nil {
		xlog.Errorf(fmt.Sprintf("Error fetching token user data for userAddress %s: %v", userAddress, err))
		return nil, err
	}
	// If no record is found, return a response with zero amounts and false flags
	if tokenUserData == nil {
		xlog.Errorf("tokenUserData for userAddress 2: %s, data: %v, error: %v", userAddress, tokenUserData, err)
		return &LogxTokenVestingResponseChecker{
			AmountDay1:         ctypes.NewBigInt(big.NewInt(0)),
			AmountUnderVesting: ctypes.NewBigInt(big.NewInt(0)),
			HasVesting:         false,
			IsBlocked:          false,
			ClaimedAmount:      ctypes.NewBigInt(big.NewInt(0)),
		}, nil
	}

	if tokenUserData.HasVesting {
		// Fetch all vesting records for the given userAddress
		vestingData, err := utils.TokenDB.GetAllTokenVestingsByUserAddress(userAddress)
		if err != nil {
			xlog.Errorf(fmt.Sprintf("Error fetching vesting data for userAddress %s: %v", userAddress, err))
			return nil, err
		}

	

		// Calculate total claimable amount including vesting records
		totalClaimableAmount := tokenUserData.TotalAmt.Copy()
		amountUnderVesting := ctypes.NewBigInt(big.NewInt(0))
		amountDay1 := totalClaimableAmount
		claimedAmount := tokenUserData.ClaimedAmt
		// Sum up the vesting amount for all records
		for _, vesting := range vestingData {
			amountDay1 = amountDay1.Add(vesting.AmtUnlockDay1)
			amountUnderVesting = amountUnderVesting.Add(vesting.TotalAmt.Sub(vesting.AmtUnlockDay1))
			claimedAmount = claimedAmount.Add(vesting.ClaimedAmt)
		}
		xlog.Errorf("tokenUserData for userAddress 13: %s, data: %v, error: %v", userAddress, tokenUserData, err)
		// Return the response for vesting information
		return &LogxTokenVestingResponseChecker{
			AmountDay1:         amountDay1,
			AmountUnderVesting: amountUnderVesting,
			HasVesting:         true,
			IsBlocked:          false,
			ClaimedAmount:      claimedAmount,
		}, nil
	}
	xlog.Errorf("tokenUserData for userAddress 14: %s, data: %v, error: %v", userAddress, tokenUserData, err)
	// If user is not blocked and has no vesting, return total amount and set amounts to zero
	return &LogxTokenVestingResponseChecker{
		AmountDay1:         tokenUserData.TotalAmt.Copy(),
		AmountUnderVesting: ctypes.NewBigInt(big.NewInt(0)),
		HasVesting:         false,
		IsBlocked:          false,
		ClaimedAmount:      ctypes.NewBigInt(big.NewInt(0)),
	}, nil
}