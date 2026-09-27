package controller

import (
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/logxTokenUtils"
	"github/eugenix-io/logx-inf-backend/libs/metric"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

type UserController struct {
	balanceClient  *xclient.BalanceClient
	LogxTokenUtils *logxTokenUtils.LogxTokenUtils
}

func RegisterUserController(r *gin.RouterGroup) {
	userController := UserController{
		balanceClient:  xclient.GlobalBalanceClient, // Ensure the client is initialized
		LogxTokenUtils: logxTokenUtils.NewLogxTokenUtils(),
	}
	rg := r.Group("/user")

	// Endpoints
	rg.GET("", middleware.RequireAuth, userController.GetSubaccountData)
	rg.GET("/positions", middleware.RequireAuth, userController.GetPositions)
	rg.GET("/orders", middleware.RequireAuth, userController.GetOrders)
	rg.GET("/withdrawableTokenBalance", middleware.RequireAuth, userController.GetWithdrawableToken)
	rg.GET("/buyingPower", middleware.RequireAuth, userController.GetBuyingPower)
	rg.GET("/deposits", middleware.RequireAuth, userController.GetDeposits)
	rg.GET("/withdrawals", middleware.RequireAuth, userController.GetWithdrawals)
	rg.GET("/getLogxTokenData", middleware.RequireAuth, userController.GetTokenUserData)
	rg.GET("/airdropData", middleware.RequireAuth, userController.GetTokenUserDataForChecker)
	rg.GET("/health", middleware.RequireAuth, userController.GetHealth)
	rg.GET("/rewards", middleware.RequireAuth, userController.GetSubaccountRewards)
	rg.GET("/GlobalOICaps", userController.GetTotalAmmPositionsAndCaps)
	rg.GET("/MarketOICaps", userController.GetMarketDataOICaps)
	rg.GET("/majaAagayaXD", middleware.RequireAdminKey, userController.GetTokenUserDataByQueryParam) // returns any address's country code: admin only
	rg.GET("/lifeSortHai", userController.GetTokenUserAndClaimData)                                  // aggregate stats only; public staking page uses it
	rg.GET("/checkLotteryDailyClaim", middleware.RequireAuth, userController.CheckDailyClaim)
	rg.POST("/updateLotteryDailyClaim", middleware.RequireAuth, userController.CreateDailyClaim)
	rg.GET("/logxwithdrawdata", userController.GetLogxWithdrawData)
	rg.GET("/xfbClaimUdata", userController.GetTokenVestingDataForExternal)
	rg.GET("/subaccountsWithOtherTokens", userController.GetSubaccountsWithOtherTokens)
	rg.GET("/totalDepositsWithdrawals", userController.GetTotalDepositsAndWithdrawals)
	rg.GET("/downloadTradeHistory", middleware.RequireAuth, userController.DownloadTradeHistory)
}

func (uc *UserController) GetWithdrawals(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}
	subaccountID, err := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "")
		return
	}

	// Fetch all finalized withdrawals
	withdrawals := (&db.DepositWithdrawDB{}).GetWithdrawalsBySubaccountId(subaccountID)
	withdrawalResponses := []gin.H{}
	for _, withdrawal := range *withdrawals {
		withdrawalResponses = append(withdrawalResponses, gin.H{
			"amount":             withdrawal.Amount,
			"productId":          withdrawal.ProductId,
			"sourceChainId":      withdrawal.SourceChainID,
			"destinationChainId": withdrawal.DestinationChainID,
			"finalised":          withdrawal.Finalised,
			"createdAt":          withdrawal.CreatedAt,
			"txHash":             withdrawal.TxnHash,
		})
	}

	// Create response
	response := gin.H{
		"withdrawals": withdrawalResponses,
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetDeposits(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	subaccountID, err := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "")
		return
	}

	deposits := (&db.DepositWithdrawDB{}).GetDepositsBySubaccountId(subaccountID)
	depositResponses := []gin.H{}
	for _, deposit := range *deposits {
		depositResponses = append(depositResponses, gin.H{
			"amount":             deposit.Amount,
			"productId":          deposit.ProductId,
			"sourceChainId":      deposit.SourceChainID,
			"destinationChainId": deposit.DestinationChainID,
			"finalised":          deposit.Finalised,
			"createdAt":          deposit.CreatedAt,
			"txHash":             deposit.TxnHash,
		})
	}

	response := gin.H{
		"deposits": depositResponses,
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetWithdrawableToken(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}
	subaccountID, err := cutils.SubaccountIdToHex(currentSubaccount.ID)

	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to convert subaccount to hex")
		return
	}

	withdrawableTokenBalance, err := uc.balanceClient.GetWithdrawableTokenBalance(subaccountID)

	if err != nil {
		xlog.Errorf("UC - Error occurred while fetching withdrawable token balance from BC. Error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "An unexpected error occurred while processing your request. You can report issue on LogX discord"})
	}

	response := gin.H{
		"withdrawableBalance": withdrawableTokenBalance,
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetLogxWithdrawData(ctx *gin.Context) {
	withdrawals, err := (&db.DepositWithdrawDB{}).GetWithdrawalsWithConditions()
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to fetch withdrawal data")
		return
	}

	withdrawalResponses := make([]gin.H, 0)
	for _, withdrawal := range *withdrawals {
		withdrawalResponses = append(withdrawalResponses, gin.H{
			"created_at":   withdrawal.CreatedAt,
			"subaccountId": withdrawal.SubaccountId,
			"amount":       withdrawal.Amount,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"LogxWithdrawals": withdrawalResponses})
}

func (uc *UserController) GetBuyingPower(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}
	subaccountID, err := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "")
		return
	}

	buyingPower, err := uc.balanceClient.GetBuyingPower(subaccountID)

	if err != nil {
		xlog.Errorf("UC - Error occurred while fetching buying power from BC. Error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "An unexpected error occurred while processing your request. Sorry for the inconvience."})
	}

	response := gin.H{
		"totalBuyingPower": buyingPower,
	}
	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetSubaccountData(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}
	subaccountID, err := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "")
		return
	}

	// Fetch spot balance via API
	spotBalance, freeBalance, usedBalance, err := uc.balanceClient.GetSpotBalance(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create an instance of StakingDB
	stakingDB := &db.StakingDB{}

	// Get unstake sum in the last hour
	unstakeSum, err := stakingDB.GetUnstakeSumOfPendingUnstake(currentSubaccount.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Adjust spotBalance
	for i := range spotBalance {
		balance := &spotBalance[i]
		if balance.ProductId == 0 {
			tokenBalanceInt := big.NewInt(0)
			if _, ok := tokenBalanceInt.SetString(balance.TokenBalance, 10); !ok {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "invalid tokenBalance"})
				return
			}
			withdrawBalanceInt := big.NewInt(0)
			if _, ok := withdrawBalanceInt.SetString(balance.WithdrawBalance, 10); !ok {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "invalid withdrawBalance"})
				return
			}

			// Subtract unstakeSum
			tokenBalanceInt.Sub(tokenBalanceInt, unstakeSum)
			withdrawBalanceInt.Sub(withdrawBalanceInt, unstakeSum)

			// Update balances
			balance.TokenBalance = tokenBalanceInt.String()
			balance.WithdrawBalance = withdrawBalanceInt.String()
		}
	}

	// Create response
	response := gin.H{
		"subAccountData": gin.H{
			"spotBalance": gin.H{
				"freeBalance": freeBalance,
				"usedBalance": usedBalance,
				"balances":    spotBalance,
			},
		},
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetTokenUserData(ctx *gin.Context) {
	// Initialize LogxTokenUtils

	// Extract currentSubaccount from the request context
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}
	subaccountID := currentSubaccount.EthAddress

	// Fetch token user data using LogxTokenUtils
	tokenUserData, err := uc.LogxTokenUtils.GetTokenUserDataByUserAddress(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If no data is found for the user, return an empty response
	if tokenUserData == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no data found for the user address"})
		return
	}

	// Create response using the fields from LogxTokenUserResponse
	response := gin.H{
		"userTokenData": gin.H{
			"totalAmount":      tokenUserData.TotalAmount,
			"claimableAmount":  tokenUserData.ClaimableAmount,
			"hasVesting":       tokenUserData.HasVesting,
			"isBlocked":        tokenUserData.IsBlocked,
			"minTimeRemaining": tokenUserData.MinTimeRemaining,
			"nextUnlockAmount": tokenUserData.NextUnlockAmount,
		},
	}

	// Send the response back to the client
	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetTokenUserDataForChecker(ctx *gin.Context) {
	// Extract currentSubaccount from the request context
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}
	subaccountID := currentSubaccount.EthAddress

	// Fetch token user data for checker using LogxTokenUtils
	tokenUserDataChecker, err := uc.LogxTokenUtils.GetTokenUserDataForChecker(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If no data is found for the user, return an empty response
	if tokenUserDataChecker == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no data found for the user address"})
		return
	}

	// Create response using the fields from LogxTokenUserResponseChecker
	response := gin.H{
		"userTokenDataChecker": gin.H{
			"amountDay1":         tokenUserDataChecker.AmountDay1,
			"amountUnderVesting": tokenUserDataChecker.AmountUnderVesting,
			"hasVesting":         tokenUserDataChecker.HasVesting,
			"isBlocked":          tokenUserDataChecker.IsBlocked,
		},
	}

	// Send the response back to the client
	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetPositions(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}
	subaccountID, err := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "")
		return
	}

	// Fetch positions via API
	positions, err := uc.balanceClient.GetPerpPositions(subaccountID, []uint{})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, position := range positions {
		productIDFloat, ok := position["productID"].(float64)
		if !ok {
			xlog.Errorf("User Controller - Error: productID is not a float64\n")
			continue
		}
		productID := uint32(productIDFloat)

		// Fetch margin fractions from MarketTable
		marketDB := &db.MarketDB{}
		imfx18, err := marketDB.GetInitialMarginFractionByID(productID)
		if err != nil {
			xlog.Errorf("User Controller - Error fetching initial margin fraction for productID %d: %v\n", productID, err)
		}
		mmf, err := marketDB.GetMaintenanceMarginFractionByID(productID)
		if err != nil {
			xlog.Errorf("User Controller - Error fetching maintenance margin fraction for productID %d: %v\n", productID, err)
		}

		// Store the margins as strings in the position map
		position["InitialMarginFraction"] = cutils.X18ToFloatStr(imfx18.Val)
		position["MaintenanceMarginFraction"] = cutils.X18ToFloatStr(mmf.Val)
	}

	// Fetch trade history from database
	tradeHistoryResponses := []gin.H{}

	tradeHistory := (&db.FillDB{}).GetAllBySubaccountId(currentSubaccount.ID)

	// Fetch all fee rewards in one query for the current subaccount
	rewardsMap := (&db.UserRewardsDB{}).GetAllArbRewardsBySubaccount(currentSubaccount.ID)

	for _, trade := range *tradeHistory {
		feeBonusReward := "0"

		// If there's a corresponding entry in the rewards map, use it
		if reward, exists := rewardsMap[trade.ID]; exists {
			feeBonusReward = reward.Val.String() // Override the default with the value from the UserRewards table
		}

		tradeHistoryResponses = append(tradeHistoryResponses, gin.H{
			"productId":     trade.MarketId,               // Assuming MarketId maps to ProductID in your original structure
			"type":          trade.Side,                   // Convert Order side to string, assuming `trade.Side` has a String() method
			"amountx18":     trade.Amountx18.Val.String(), // Convert Amount in x18 format to string
			"pricex18":      trade.Pricex18.Val.String(),  // Convert Price in x18 format to string
			"timestamp":     trade.CreatedAt,
			"feex18":        trade.Feex18.Val.String(),         // Convert Fee in x18 format to string
			"pnl":           trade.RealizedPnlx18.Val.String(), // Use the calculated pnl value, "0" if nil
			"feebonusx18":   feeBonusReward,
			"isLiquidation": trade.Type == ctypes.FILL_LIQUIDATION,
		})

	}

	// Create response
	response := gin.H{
		"positions":    positions,
		"tradeHistory": tradeHistoryResponses,
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetOrders(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	orders := (&db.OrderDB{}).GetAllOpenOrders(currentSubaccount.ID)
	orderResponses := []gin.H{}
	for _, order := range *orders {
		// Fetch market details
		market := (&db.MarketDB{}).GetById(order.MarketId)
		if market == nil {
			xlog.Errorf("User Controller - Error fetching market details for marketID: %d\n", order.MarketId)
			continue
		}

		var orderType string
		var price string
		var isTPSL bool
		if order.IsConditional() {
			orderType = order.ConditionFullName()
			price = order.TriggerPricex18.Val.String()
			isTPSL = true
		} else {
			orderType = string(order.Type)
			price = order.Pricex18.Val.String()
			isTPSL = false
		}

		orderResponses = append(orderResponses, gin.H{
			"orderID":         order.BaseTable.ID,
			"productID":       order.MarketId, // need to update productID
			"pricex18":        price,
			"amountx18":       order.Amountx18.Val.String(),
			"operation":       order.Side,
			"OrderType":       orderType,
			"filledAmountx18": order.TotalFilledx18.Val.String(),
			"isTPSL":          isTPSL,
			"isReduce":        order.IsReduce,
		})
	}

	response := gin.H{
		"orders": orderResponses,
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetHealth(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	subaccountHex, err := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Error converting subaccount to hex: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Invalid subaccount")
		return
	}

	health, err := uc.balanceClient.GetHealth(subaccountHex)
	if err != nil {
		xlog.Errorf("Error fetching health from BC: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Something went wrong while fetching health")
	}
	cutils.ApiSuccess(ctx, health, "successfuly fetched health")
}

func (uc *UserController) GetSubaccountRewards(ctx *gin.Context) {
	// Get the current subaccount
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	// Fetch total fee bonuses for ARB and LOGX from UserRewardsDB
	rewardsDB := &db.UserRewardsDB{}
	arbSymbol := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.ARB_MARKET]
	logxSymbol := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.LOGX]

	totalArbFeeBonus, err := rewardsDB.GetTotalBonusForSubaccount(currentSubaccount.ID, arbSymbol)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "error fetching ARB fee bonus from rewards table")
		return
	}

	totalLogxFeeBonus, err := rewardsDB.GetTotalBonusForSubaccount(currentSubaccount.ID, logxSymbol)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "error fetching LOGX fee bonus from rewards table")
		return
	}

	response := gin.H{
		"totalArbFeeBonus":  totalArbFeeBonus.Val.String(),
		"totalLogxFeeBonus": totalLogxFeeBonus.Val.String(),
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetTotalAmmPositionsAndCaps(ctx *gin.Context) {
	// Get AMM Subaccount ID from environment variable
	// ammSubaccountID := os.Getenv("AMM_SUBACCOUNT_ID")
	perputils := perputils.NewPerpUtils()
	// if ammSubaccountID == "" {
	// 	cutils.ApiAbort(ctx, http.StatusInternalServerError, "AMM Subaccount ID not set in environment variables")
	// 	return
	// }

	// // Fetch all perp positions for the AMM subaccount
	// positions, err := uc.balanceClient.GetPerpPositions(ammSubaccountID, []uint{})
	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error fetching positions", "details": err.Error()})
	// 	return
	// }

	// // Initialize total sum of vQuote
	// totalVQuote := big.NewInt(0)

	// // Filtered positions to only include productID and vQuoteBalance
	// var filteredPositions []map[string]interface{}

	// // Iterate through the positions and sum vQuotes
	// for _, position := range positions {
	// 	productIDFloat, ok := position["productID"].(float64)
	// 	if !ok {
	// 		xlog.Errorf("User Controller - Error: productID is not a float64\n")
	// 		continue
	// 	}

	// 	// Get vQuote balance from position
	// 	vQuoteBalanceStr, ok := position["vQuoteBalance"].(string)
	// 	if !ok {
	// 		xlog.Errorf("User Controller - Error: vQuoteBalance is not a string for productID %v\n", productIDFloat)
	// 		continue
	// 	}

	// 	vQuoteBalance := new(big.Int)
	// 	vQuoteBalance.SetString(vQuoteBalanceStr, 10)

	// 	// Add to total vQuote
	// 	totalVQuote.Add(totalVQuote, new(big.Int).Abs(vQuoteBalance))

	// 	// Append filtered position with only productID and vQuoteBalance
	// 	filteredPositions = append(filteredPositions, map[string]interface{}{
	// 		"productID":     productIDFloat,
	// 		"vQuoteBalance": vQuoteBalanceStr,
	// 	})
	// }

	// // Fetch Global OI Cap
	// globalOICap, err := perputils.GetGlobalNegativeOICap()
	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error fetching global OI cap", "details": err.Error()})
	// 	return
	// }

	// Fetch Subaccount OI Cap for AMM
	subaccountOICap, err := perputils.GetSubaccountOICap()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error fetching subaccount OI cap", "details": err.Error()})
		return
	}

	// Create the response with the filtered positions
	response := gin.H{
		// "totalVQuote":     totalVQuote.String(),
		// "positions":       filteredPositions, // Return only filtered positions
		// "globalOICap":     globalOICap.String(),
		"subaccountOICap": subaccountOICap.String(),
	}

	// Return the response
	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetMarketDataOICaps(ctx *gin.Context) {
	// Initialize a response map to hold the results
	response := gin.H{}
	perputils := perputils.NewPerpUtils()

	// Iterate through contractUtils..PRODUCT_ID_SYMBOL_TO_MAP
	for _, productID := range contractUtils.ALL_PERPS_ON_CONTRACT {
		// Fetch Market OI Cap for the given productID (marketId)
		marketLongOICap, marketShortOICap, err := perputils.GetMarketOICap(productID)
		if err != nil {
			xlog.Errorf("Error fetching market OI cap for productID %d: %v", productID, err)
			continue
		}

		// Fetch Total Long Position for the given productID
		totalLongPosition, err := perputils.GetTotalLongPosition(productID)
		if err != nil {
			xlog.Errorf("Error fetching total long position for productID %d: %v", productID, err)
			continue
		}

		// Fetch Total Short Position for the given productID
		totalShortPosition, err := perputils.GetTotalShortPosition(productID)
		if err != nil {
			xlog.Errorf("Error fetching total short position for productID %d: %v", productID, err)
			continue
		}

		marketSubaccountCap, err := perputils.GetSubaccountMarketCap(strconv.FormatUint(uint64(productID), 10))
		if err != nil {
			xlog.Errorf("Error fetching subaccount market level cap for productID %d: %v", productID, err)
		}

		// Add the data for this productID to the response
		response[fmt.Sprintf("%d", productID)] = gin.H{
			// ToDo - to prevent any downtime during launch on FE, we will send market OI Cap with long OI Cap.
			//		this key, (marketOICap) will be removed shortly after launch once everything is working smoothly
			"marketOICap":        marketLongOICap.String(),
			"marketLongOICap":    marketLongOICap.String(),
			"marketShortOICap":   marketShortOICap.String(),
			"subaccountCap":      marketSubaccountCap.String(),
			"totalLongPosition":  totalLongPosition.String(),
			"totalShortPosition": totalShortPosition.String(),
		}
	}

	// Return the response
	ctx.JSON(http.StatusOK, response)
}

// //// For temporary use to debug . Handle carefully cannot outsource
func (uc *UserController) GetTokenUserDataByQueryParam(ctx *gin.Context) {

	defer func() {
		if r := recover(); r != nil {
			// Add a discord alert here if needed
			xlog.Errorf("Recovered from panic in job - %v: %v \n Stack Trace", "", r, string(debug.Stack()))
		}
	}()

	subaccountID := ctx.Query("subaccountID")

	if subaccountID == "" {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "subaccountID query parameter is required")
		return
	}

	checksumAddress := common.HexToAddress(subaccountID).Hex()

	// Fetch the country code for the given subaccountID
	countryCodeRecord := (&db.CountryCodeDB{}).GetCountryCodeBySubaccount(checksumAddress)
	countryCode := ""
	if countryCodeRecord == nil {
		xlog.Infof("Country code not found for Address %s", subaccountID)
		countryCode = "Not found"
	} else {
		countryCode = countryCodeRecord.CountryCode // Assuming 'Code' holds the country code
	}

	// Fetch token user data using LogxTokenUtils
	tokenUserDataChecker, err := uc.LogxTokenUtils.GetTokenUserDataForChecker(checksumAddress)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dbUserData, err := (&db.LogxTokenDB{}).GetLogxTokenUserByUserAddress(checksumAddress)
	if err != nil {
		xlog.Errorf("Error fetching token user data for userAddress %s: %v", checksumAddress, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If no token user data is found for the user, return an empty response
	if tokenUserDataChecker == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no data found for the user address"})
		return
	}

	// Create response
	response := gin.H{
		"countryCode":   countryCode,
		"userTokenData": tokenUserDataChecker,
		"dbUserData":    dbUserData,
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetTokenVestingDataForExternal(ctx *gin.Context) {

	defer func() {
		if r := recover(); r != nil {
			xlog.Errorf("Recovered from panic in job - %v: %v \n Stack Trace", "", r, string(debug.Stack()))
		}
	}()

	subaccountID := ctx.Query("subaccountID")

	if subaccountID == "" {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "subaccountID query parameter is required")
		return
	}

	checksumAddress := common.HexToAddress(subaccountID).Hex()

	tokenVestingData, err := uc.LogxTokenUtils.GetTokenVestingDataForExternal(checksumAddress)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"tokenVestingData": tokenVestingData,
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetTokenUserAndClaimData(ctx *gin.Context) {

	defer func() {
		if r := recover(); r != nil {
			xlog.Errorf("Recovered from panic in job - %v: %v \n Stack Trace", "", r, string(debug.Stack()))
		}
	}()

	// Calculate total stake amount across all subaccounts
	totalStakeAmount, last24HourUnstake, last24HourStake, err := (&db.StakingDB{}).CalculateTotalStakeAmount()
	if err != nil {
		xlog.Errorf("Error calculating total stake amount: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating total stake amount"})
		return
	}

	// Call the function to calculate total claimed amounts and distinct users
	logxUserTotal, vestingTotal, distinctUserCount, err := (&db.LogxTokenDB{}).CalculateTotalClaimedAmounts()
	if err != nil {
		xlog.Errorf("Error calculating total claimed amounts: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating total claimed amounts"})
		return
	}

	// Call the CountDistinctStakers function to get the stakers count
	stakerCount, err := (&db.StakingDB{}).CountDistinctStakers()
	if err != nil {
		xlog.Errorf("Error counting distinct stakers: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting distinct stakers"})
		return
	}

	// Calculate deposit and withdrawal amounts
	totalWithdrawals, totalDeposits, total24hrWithdrawals, total24hrDeposits, err := (&db.DepositWithdrawDB{}).CalculateDepositWithdrawAmounts()
	if err != nil {
		xlog.Errorf("Error calculating deposit and withdrawal amounts: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error calculating deposit and withdrawal amounts"})
		return
	}

	totalToken := new(big.Int)
	totalToken.Add(logxUserTotal, vestingTotal)

	// Create the response, including stakedAmount for product 0
	response := gin.H{
		"NormalTokenClaimed":    logxUserTotal,             // Total claimed amount from logx_token_user_tables
		"VestingTokenClaimed":   vestingTotal,              // Total claimed amount from token_vesting_tables
		"TokenClaimedUserCount": distinctUserCount,         // Total number of distinct users who claimed tokens
		"DistinctStakers":       stakerCount,               // Total number of distinct stakers
		"StakedAmount":          totalStakeAmount.String(), // Staked amount for product 0 (from spot balance)
		"totalClaimed":          totalToken,
		"TotalWithdrawals":      totalWithdrawals.String(),     // Total withdrawals
		"TotalDeposits":         totalDeposits.String(),        // Total deposits
		"24hrWithdrawals":       total24hrWithdrawals.String(), // Withdrawals in the last 24 hours
		"24hrDeposits":          total24hrDeposits.String(),
		"24hrStake":             last24HourStake.String(),
		"24hrUnstake":           last24HourUnstake.String(),
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) isValidFreeBalance(subaccountID string) (bool, error) {
	_, freeBalance, _, err := uc.balanceClient.GetSpotBalance(subaccountID)
	if err != nil {
		return false, err
	}

	freeBalanceInt := big.NewInt(0)
	if _, ok := freeBalanceInt.SetString(freeBalance, 10); !ok {
		return false, fmt.Errorf("invalid freeBalance")
	}

	threshold := big.NewInt(1e18) // 1e18 represents the threshold for comparison
	if freeBalanceInt.Cmp(threshold) < 0 {
		return false, nil
	}

	return true, nil
}

func (uc *UserController) CheckDailyClaim(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}
	subaccountID := currentSubaccount.ID

	isValid, err := uc.isValidFreeBalance(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !isValid {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "You need to deposit at least $1 before claiming the lottery."})
		return
	}

	var startTime, endTime time.Time
	now := time.Now().UTC()
	// Determine if the current time is before or after 12:00 UTC
	if now.Hour() < 12 {
		// If before 12:00 UTC, set startTime to 12:00 UTC of the previous day
		startTime = time.Date(now.Year(), now.Month(), now.Day()-1, 12, 0, 0, 0, time.UTC)
		endTime = startTime.Add(24 * time.Hour) // End time is 12:00 UTC of the current day
	} else {
		// If after 12:00 UTC, set startTime to 12:00 UTC of the current day
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
		endTime = startTime.Add(24 * time.Hour) // End time is 12:00 UTC of the next day
	}

	// Check if there is an existing entry for the given day
	exists, lotteryCode, err := (&db.LotteryFlowDB{}).GetLotteryCodeIfExists(subaccountID, startTime, endTime)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error checking for daily claim")
		return
	}

	if exists {
		ctx.JSON(http.StatusOK, gin.H{
			"message":     "You have already claimed for today and cannot claim more.",
			"lotteryCode": lotteryCode,
		})
	} else {
		ctx.JSON(http.StatusOK, gin.H{"message": "You can proceed with claiming for today."})
	}
}

func (uc *UserController) CreateDailyClaim(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}
	subaccountID := currentSubaccount.ID

	isValid, err := uc.isValidFreeBalance(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !isValid {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "You need to deposit at least $1 before claiming the lottery."})
		return
	}

	now := time.Now().UTC()

	var startTime, endTime time.Time

	// Determine if the current time is before or after 12:00 UTC
	if now.Hour() < 12 {
		// If before 12:00 UTC, set startTime to 12:00 UTC of the previous day
		startTime = time.Date(now.Year(), now.Month(), now.Day()-1, 12, 0, 0, 0, time.UTC)
		endTime = startTime.Add(24 * time.Hour) // End time is 12:00 UTC of the current day
	} else {
		// If after 12:00 UTC, set startTime to 12:00 UTC of the current day
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
		endTime = startTime.Add(24 * time.Hour) // End time is 12:00 UTC of the next day
	}

	// Check if an entry already exists for today
	exists, lotteryCodeval, err := (&db.LotteryFlowDB{}).GetLotteryCodeIfExists(subaccountID, startTime, endTime)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error checking for existing claim")
		return
	}

	if exists {
		ctx.JSON(http.StatusConflict, gin.H{
			"message":     "You have already claimed for today and cannot claim more.",
			"lotteryCode": lotteryCodeval,
		})
		return
	}

	// Fetch existing lottery codes for today
	existingCodes, err := (&db.LotteryFlowDB{}).GetLotteryCodesForDateRange(startTime, endTime)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error fetching existing lottery codes")
		return
	}
	existingCodeSet := make(map[string]bool)
	for _, code := range existingCodes {
		existingCodeSet[code] = true
	}

	// Generate a unique 4-character alphanumeric code
	lotteryCode, err := generateUniqueCode(subaccountID, existingCodeSet, 10)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error generating unique lottery code")
		return
	}

	// Create a new entry for the daily claim with the generated lottery code
	newEntry := &db.LotteryFlowTable{
		SubaccountId: subaccountID,
		LotteryCode:  lotteryCode, // Store the 4-character code
	}

	if createdEntry := (&db.LotteryFlowDB{}).Create(newEntry); createdEntry == nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error creating daily claim entry")
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message":     "Claim entry created successfully for today.",
		"lotteryCode": lotteryCode,
	})
}

// Function to generate a random 4-character alphanumeric code
func generateUniqueCode(subaccountID string, existingCodeSet map[string]bool, maxRetries int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	now := time.Now().UTC()
	dateTimeSeed := now.Year()*100000000 + int(now.Month())*1000000 + now.Day()*10000 + now.Hour()*100 + now.Minute()
	var code string

	for i := 0; i < maxRetries; i++ {
		// Generate a unique integer by hashing subaccountID (string) and dateSeed with an incremental value
		hashSeed := fmt.Sprintf("%s-%d-%d", subaccountID, dateTimeSeed, i)
		hash := sha256.Sum256([]byte(hashSeed))

		// Generate each character in the 4-character code individually
		code = ""
		for j := 0; j < 4; j++ {
			// Use the hash bytes and mod them to get an index within the charset length
			charIndex := int(hash[j]) % len(charset)
			code += string(charset[charIndex])
		}

		// Check for uniqueness in the existing set
		if !existingCodeSet[code] {
			return code, nil
		}
	}
	return "", fmt.Errorf("could not generate a unique code after %d attempts", maxRetries)
}

func (uc *UserController) GetSubaccountsWithOtherTokens(ctx *gin.Context) {
	subaccounts, err := (&db.DepositWithdrawDB{}).GetSubaccountsWithOtherTokens()
	if err != nil {
		xlog.Errorf("Error fetching subaccounts with other tokens: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error fetching subaccounts data")
		return
	}

	response := gin.H{
		"subaccounts": subaccounts,
	}

	ctx.JSON(http.StatusOK, response)
}

func (uc *UserController) GetTotalDepositsAndWithdrawals(ctx *gin.Context) {
	totals, err := (&db.DepositWithdrawDB{}).GetTotalDepositsAndWithdrawals()
	if err != nil {
		xlog.Errorf("Error fetching total deposits and withdrawals: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error fetching totals")
		return
	}

	ctx.JSON(http.StatusOK, totals)
}

func (uc *UserController) DownloadTradeHistory(ctx *gin.Context) {
	// Get current subaccount and start timer
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Subaccount not found for the API key")
		return
	}
	startTime := time.Now()
	defer cutils.LogTime(startTime, metric.DOWNLOAD_TRADE_HISTORY)

	// Fetch trade history
	tradeHistory, err := (&db.FillDB{}).GetTradeHistoryWithMarketSymbols(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Error fetching trade history: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error fetching trade history")
		return
	}

	// Set up response headers for CSV download
	timestamp := time.Now().Format("20060102-150405")
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=trade_history_%s.csv", timestamp))
	ctx.Header("Content-Type", "text/csv")

	// Create CSV writer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header row
	if err := writer.Write([]string{"Timestamp", "Market", "Side", "Amount", "Value", "Price", "Fee", "PnL", "Fee Bonus"}); err != nil {
		xlog.Errorf("Error writing CSV headers: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error generating CSV file")
		return
	}

	// Process each trade and write to CSV
	for _, tradeData := range *tradeHistory {
		record, err := formatTradeRecord(tradeData)
		if err != nil {
			xlog.Errorf("Error formatting trade record: %v", err)
			continue
		}
		if err := writer.Write(record); err != nil {
			xlog.Errorf("Error writing CSV record: %v", err)
			cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error generating CSV file")
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		xlog.Errorf("Error flushing CSV writer: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error generating CSV file")
		return
	}

	// Send the CSV file
	ctx.Data(http.StatusOK, "text/csv", buf.Bytes())
}

// Simplified helper function for formatting trade records
func formatTradeRecord(trade map[string]interface{}) ([]string, error) {
	// Get market symbol with fallback
	marketSymbol, _ := trade["market_symbol"].(string)
	if marketSymbol == "" {
		marketSymbol = fmt.Sprintf("Market-%v", trade["market_id"])
	}

	// Format timestamp
	createdAt, ok := trade["created_at"].(time.Time)
	if !ok {
		createdAt = time.Now()
	}
	timestamp := createdAt.Format("2006-01-02 15:04:05")

	// Convert BigInt values to human-readable format
	convertToHuman := func(fieldName string) string {
		strVal, _ := trade[fieldName].(string)
		bigVal := new(big.Int)
		bigVal.SetString(strVal, 10)
		return cutils.X18ToFloatStr(bigVal)
	}

	return []string{
		timestamp,
		marketSymbol,
		fmt.Sprintf("%v", trade["side"]),
		convertToHuman("amountx18"),
		convertToHuman("total_value_x18"),
		convertToHuman("pricex18"),
		convertToHuman("feex18"),
		convertToHuman("realized_pnlx18"),
		convertToHuman("fee_bonus_x18"),
	}, nil
}
