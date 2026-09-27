package controller

import (
	"math/big"
	"net/http"
	"strconv"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/referralUtils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"

	"github.com/gin-gonic/gin"
)

type ReferralController struct {
	referralDB        db.ReferralDB
	referralRewardDB  db.ReferralRewardDB
	referralHistoryDB db.ReferralHistoryDB
	ReferralUtils     *referralUtils.ReferralUtils
}

func RegisterReferralController(r *gin.RouterGroup) {
	referralController := ReferralController{
		ReferralUtils: referralUtils.NewReferralUtils(),
	}
	rg := r.Group("/referral")

	// add a new referral
	rg.POST("/addReferrer", middleware.RequireAuth, referralController.AddReferral)

	// Get applied referral code
	rg.GET("/getAppliedCode", middleware.RequireAuth, referralController.GetReferralCode)

	// Get userReferralDetails
	rg.GET("/userReferralDetails", middleware.RequireAuth, referralController.GetUserReferralDetails)

	// Get userReferralHistoryDetails
	rg.GET("/referralHistoryDetails", middleware.RequireAuth, referralController.GetUserReferralHistory)

	// for testing purpose
	rg.POST("/createUserTesting", referralController.CreateReferralUser)

	// for migrating purpose
	rg.GET("/updateReferralCodeWithUserName", referralController.UpdateReferralCodesWithUsernamesByIDRange)

	// for growth team
	rg.POST("/UpdateReferralRebateMapping", referralController.UpdateReferralRebateMapping)

	// Add referee for Affiliate
	rg.POST("/addReferee", middleware.RequireAuth, referralController.AddReferee)

	// Get affiliate stats
	rg.GET("/affiliateStats", middleware.RequireAuth, referralController.GetAffiliateStats)
}

// AddReferral handles the POST request to add a new referral
func (rc *ReferralController) AddReferral(c *gin.Context) {
	var request struct {
		ReferralCode   string `json:"referral_code"`
		ReferredUserID string `json:"referred_user_id"`
	}

	// Validate the incoming JSON payload
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Fetch the referrer user based on the referral code
	referrerUser, err := rc.referralDB.GetReferralUserByCode(request.ReferralCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Referral code not found"})
		return
	}

	// Check if the referred user has already placed a trade (assuming this is done via ActivateReferralUser)
	status, _ := rc.referralDB.ActivateReferralUser(request.ReferredUserID)

	if status {
		// If status is true, it means the referred user has already been activated (already placed a trade)
		c.JSON(http.StatusConflict, gin.H{"error": "Referred user has already placed a trade"})
		return
	}

	// Map the referred user to the referrer by adding the referrer mapping
	referralData := db.ReferralUserTable{
		ReferrerUserID: referrerUser.UserAddress, // Use referrer's user address
		UserAddress:    request.ReferredUserID,   // The referred user's ID
	}

	// Add the referrer mapping for the referred user
	if _, err := rc.referralDB.AddReferrerMapping(referralData); err != nil {
		// Handle the case where the referred user already has a referrer
		if err.Error() == "referrer user ID already present for this user" {
			c.JSON(http.StatusConflict, gin.H{"error": "Referred user already has a referrer"})
			return
		}
		// Handle other errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add referrer mapping"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Referral created successfully"})
}

// api to get applied referral code for given userAddress
func (rc *ReferralController) GetReferralCode(c *gin.Context) {
	referredUserID := c.Query("referred_user_id")

	if referredUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Referred User ID is required"})
		return
	}

	// Fetch the referral entry based on the referred user ID
	referral, err := rc.referralDB.GetReferralByReferredUserID(referredUserID)
	if err != nil || referral.ReferrerUserID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Referral not found"})
		return
	}

	// Fetch the referrer user based on the referrer user ID
	referrerUser, _, err := rc.referralDB.GetReferralUserByAddress(referral.ReferrerUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Referrer user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"referral_code": referrerUser.ReferralCode})
}

// generateDefaultReferralResponse generates a default response with a given referral code
func generateDefaultReferralResponse(referralCode string) gin.H {
	productIDStr := strconv.Itoa(int(contractUtils.REFERRER_REWARD_ID))

	return gin.H{
		"referral_code":    referralCode, // Use the passed referral code
		"is_active":        false,
		"referrer_user_id": "",
		"fee_percentage":   strconv.Itoa(contractUtils.REFERRER_REBATE_PERCENTAGE), // Default fee percentage
		"rewards": gin.H{
			productIDStr: gin.H{
				"claimed_amount":   "0",
				"unclaimed_amount": "0",
			},
		},
	}
}

// referral user details for a given user address
func (rc *ReferralController) GetUserReferralDetails(c *gin.Context) {
	userAddress := c.Query("user_address")

	if userAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User address is required"})
		return
	}

	// Fetch the referral user based on the user address
	referralUser, status, err := rc.referralDB.GetReferralUserByAddress(userAddress)
	if err != nil {
		// If the error is due to the user not being found or any other issue
		c.JSON(http.StatusOK, generateDefaultReferralResponse(""))
		return
	}

	var referralCode string

	// Check if referralUser is not nil before accessing its fields
	if referralUser != nil {
		referralCode = referralUser.ReferralCode
	} else {
		// If referralUser is nil, consider the referral code as empty
		referralCode = ""
	}

	// If status is false, it means the user was not activated
	if !status {
		// Handle the case where the user was fetched but not activated
		c.JSON(http.StatusOK, generateDefaultReferralResponse(referralCode))
		return
	}

	// If the user is not active, return default values but include the referral user info
	if !referralUser.IsActive {
		defaultResponse := generateDefaultReferralResponse(referralCode)
		c.JSON(http.StatusOK, defaultResponse)
		return
	}

	// Fetch the reward details for the product contractUtils.ARB_USDC
	rewardEntry, err := rc.referralRewardDB.GetReferralReward(userAddress, uint(contractUtils.REFERRE_REWARD_ID))
	rewardAmount := ctypes.NewBigIntFromString("0")
	claimedAmount := ctypes.NewBigIntFromString("0")
	if err == nil {
		rewardAmount = rewardEntry.RewardAmount
		claimedAmount = rewardEntry.ClaimedAmount
	}

	// Calculate unclaimed amount: reward_amount - claimed_amount
	unclaimedAmount := rewardAmount.Sub(claimedAmount)

	// Fetch the fee percentage from Redis or return the default value "10" if not found
	feePercentage, err := rc.ReferralUtils.GetReferralFeePercentage(userAddress)
	if err != nil || feePercentage == "" {
		feePercentage = strconv.Itoa(contractUtils.REFERRER_REBATE_PERCENTAGE) // Default to "10" if not found or an error occurred
	}

	// Convert contractUtils.ARB_USDC to string for map key
	productIDStr := strconv.Itoa(int(contractUtils.REFERRER_REWARD_ID))

	var username string
	// Call the function to get the username based on ReferrerUserID (you can adjust this according to your structure)
	usernameMapping, err := (&db.SubaccountDB{}).GetUserNamesByEthAddresses([]string{referralUser.ReferrerUserID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Check if the username exists in the mapping
	if val, exists := usernameMapping[referralUser.ReferrerUserID]; exists && val != "" {
		username = val
	} else {
		username = referralUser.ReferrerUserID // Fallback to ReferrerUserID if no username is found
	}

	// Return the response with unclaimed amount, claimed amount, fee percentage, and other user details
	c.JSON(http.StatusOK, gin.H{
		"referral_code":    referralCode, // Use the actual referral code if active
		"is_active":        referralUser.IsActive,
		"referrer_user_id": username,
		"fee_percentage":   feePercentage,
		"rewards": gin.H{
			productIDStr: gin.H{
				"claimed_amount":   claimedAmount.String(),
				"unclaimed_amount": unclaimedAmount.String(),
			},
		},
	})
}

func (rc *ReferralController) GetUserReferralHistory(c *gin.Context) {
	referrerUserID := c.Query("referrer_user_id")

	if referrerUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Referrer user ID is required"})
		return
	}

	// Calculate referral stats for the given referrerUserID
	stats, referees, err := rc.referralHistoryDB.CalculateReferralStats(referrerUserID)
	if err != nil {
		xlog.Errorf("Failed to calculate referral stats for user %s: %v", referrerUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate referral stats"})
		return
	}
	// Extract all referee user IDs from the referees slice
	refereeUserIDs := []string{}
	for _, referee := range referees {
		refereeUserIDs = append(refereeUserIDs, referee.RefereeUserID)
	}
	// Call the function to get the usernames for the referee user IDs
	usernameMapping, err := (&db.SubaccountDB{}).GetUserNamesByEthAddresses(refereeUserIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format the referee details for the response
	formattedReferees := []gin.H{} // Initialize as an empty slice
	for _, referee := range referees {
		refereeIDOrUsername := referee.RefereeUserID
		if username, exists := usernameMapping[referee.RefereeUserID]; exists && username != "" {
			refereeIDOrUsername = username
		}

		formattedReferees = append(formattedReferees, gin.H{
			"referee_user_id":        refereeIDOrUsername,
			"total_fees":             referee.TotalFees.String(),            // Convert to string in 10^18 format
			"total_volume":           referee.TotalVolume.String(),          // Convert to string in 10^18 format
			"total_referrer_rewards": referee.TotalReferrerRewards.String(), // Convert to string in 10^18 format
		})
	}

	// Return the calculated referral statistics with the formatted referees array
	c.JSON(http.StatusOK, gin.H{
		"net_volume":              stats.NetVolume.String(), // Convert to string in 10^18 format
		"total_distinct_referees": stats.TotalDistinctReferees,
		"total_fees":              stats.TotalRefereeFeesOverall.String(), // Convert to string in 10^18 format
		"referees":                formattedReferees,                      // Return an empty array if no referees
	})
}

// POST request to create a new referral user -- tesing
func (rc *ReferralController) CreateReferralUser(c *gin.Context) {
	var request struct {
		UserAddress  string `json:"user_address"`
		ReferralCode string `json:"referral_code,omitempty"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Create referral user data
	referralUserData := db.ReferralUserTable{
		UserAddress:  request.UserAddress,
		ReferralCode: request.ReferralCode,
	}

	// Call CreateReferralUser function
	createdUser, err := rc.referralDB.CreateReferralUser(referralUserData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create referral user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_address":  createdUser.UserAddress,
		"referral_code": createdUser.ReferralCode,
		"is_active":     createdUser.IsActive,
	})
}

// for one time use only to migrate username into referral code
func (rc *ReferralController) UpdateReferralCodesWithUsernamesByIDRange(c *gin.Context) {
	// Get startID and endID from query parameters
	startIDParam := c.Query("start_id")
	endIDParam := c.Query("end_id")

	if startIDParam == "" || endIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_id and end_id are required"})
		return
	}

	// Convert query parameters to uint
	startID, err := strconv.ParseUint(startIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_id"})
		return
	}
	endID, err := strconv.ParseUint(endIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_id"})
		return
	}

	// Call the function to update referral codes within the given ID range
	lastProcessedID, err := rc.referralDB.UpdateReferralCodesWithUsernamesByIDRange(uint(startID), uint(endID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "last_processed_id": lastProcessedID})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"message":           "Referral codes updated successfully",
		"start_id":          startID,
		"last_processed_id": lastProcessedID,
		"end_id":            endID,
	})
}

// UpdateReferralRebateMapping updates the rebate mapping for a given user address ----- for growth team
func (rc *ReferralController) UpdateReferralRebateMapping(c *gin.Context) {
	var request struct {
		UserAddress   string `json:"user_address"`
		FeePercentage string `json:"fee_percentage"`
	}

	// Bind the JSON request to the struct
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Validate that user address and fee percentage are provided
	if request.UserAddress == "" || request.FeePercentage == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User address and valid fee percentage are required"})
		return
	}

	// Call the utility function to add or update the rebate mapping in Redis
	err := rc.ReferralUtils.AddOrUpdateReferralRebateMapping(request.UserAddress, request.FeePercentage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update referral rebate mapping"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Referral rebate mapping updated successfully"})
}

func (rc *ReferralController) AddReferee(c *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current subaccount"})
		return
	}

	var request struct {
		ReferralCode string `json:"referral_code"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	referralCodeExists, affiliateSubaccountId, err := rc.ReferralUtils.CheckReferralCodeExists(request.ReferralCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking referral code"})
		return
	}
	if !referralCodeExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid referral code"})
		return
	}

	// Prevent self referral (self-referral)
	if affiliateSubaccountId == currentSubaccount.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot use your own referral code"})
		return
	}

	affiliateDB := db.AffiliateDB{}
	affiliateExists, err := affiliateDB.CheckAffiliateExists(currentSubaccount.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking affiliate"})
		return
	}
	if affiliateExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Already a user"})
		return
	}

	_, err = affiliateDB.AddAffiliate(currentSubaccount.ID, request.ReferralCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add referee"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Referee added successfully"})
}

// GetAffiliateStats returns affiliate statistics for a given subaccount ID
func (rc *ReferralController) GetAffiliateStats(c *gin.Context) {
	// Get subaccount ID from query parameter
	subaccountId := c.Query("subaccount_id")
	if subaccountId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subaccount_id query parameter is required"})
		return
	}

	// Initialize AffiliateDB
	affiliateDB := db.AffiliateDB{}

	// Get affiliate code for the given subaccount from Redis
	affiliateCode, err := rc.ReferralUtils.GetAffiliateCodeBySubaccountId(subaccountId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Affiliate code not found for this subaccount"})
		return
	}

	// Get all referrers that used this affiliate code
	referrerSubaccountIds, err := affiliateDB.GetReferrersByAffiliateCode(affiliateCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get referrers"})
		return
	}

	// Calculate total volume generated by referrers
	fillDB := db.FillDB{}
	totalVolume, err := fillDB.GetTotalVolumeForSubaccounts(referrerSubaccountIds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate total volume"})
		return
	}

	volumeBigInt := cutils.FloatStrToX18(totalVolume)

	// 0.06% = 60 * 10^13 / 10^18
	feeFraction := cutils.MulxCust(big.NewInt(60), 13)
	feesBigInt := ctypes.NewBigInt(cutils.Divx18(new(big.Int).Mul(volumeBigInt, feeFraction)))
	totalFees := cutils.X18ToFloatStr(feesBigInt.Val)

	// Return the response
	c.JSON(http.StatusOK, gin.H{
		"affiliate_code":      affiliateCode,
		"number_of_referrers": len(referrerSubaccountIds),
		"total_volume":        totalVolume,
		"total_fees":          totalFees,
	})
}
