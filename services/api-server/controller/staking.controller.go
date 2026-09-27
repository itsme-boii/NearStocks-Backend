package controller

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type StakingController struct {
	stakingService    *services.StakingService
	stakingDb         *db.StakingDB
	depositWithdrawDb *db.DepositWithdrawDB
}

type StakingGetUri struct {
	SubAccountId string `uri:"subAccountId" binding:"required"`
}

func RegisterStakingController(
	r *gin.RouterGroup,
) {
	stakingController := StakingController{
		stakingService:    services.NewStakingService(),
		stakingDb:         &db.StakingDB{},
		depositWithdrawDb: &db.DepositWithdrawDB{},
	}
	rg := r.Group("/stake")

	//Endpoints
	rg.GET("/stakes", middleware.RequireAuth, stakingController.GetStakes)
	rg.GET("/totalEarnings", middleware.RequireAuth, stakingController.GetTotalEarnings)
	rg.GET("/unclaimedEarnings", middleware.RequireAuth, stakingController.GetUnclaimedEarnings)
	rg.GET("/history", middleware.RequireAuth, stakingController.GetStakingHistory)
	rg.GET("/unclaimedEarningsOpen/:subaccountId", stakingController.GetUnclaimedEarningsOpen)
}

func (sc *StakingController) GetStakes(ctx *gin.Context) {
	// Get the current subaccount
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	subaccountIDHex, err := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "error in converting into hex format")
		return
	}

	// Fetch staking information
	stakes, err := sc.stakingDb.GetAllBySubaccountId(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Staking Controller - error fetching staking information for the subaccount ID %v", currentSubaccount.ID)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "no stake information found for the user")
		return
	}

	// Fetch deposit/withdrawal information (both finalized and unfinalized)
	depositsWithdrawals, err := sc.depositWithdrawDb.GetAllBySubaccountId(subaccountIDHex)
	if err != nil {
		xlog.Errorf("Staking Controller - error fetching deposit/withdrawal information for the subaccount ID %v", currentSubaccount.ID)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "no deposit/withdrawal information found for the user")
		return
	}

	// Combine the stakes and deposits/withdrawals into a response
	response := []gin.H{}

	// Add staking records
	earningsFromLastClaimx18 := big.NewInt(0)
	// Revering stakes to caculate earnings from last claim
	for _, stake := range cutils.ReverseSlice(stakes) {
		isPending := false

		// Fetch the number of minutes from the .env file
		minuteStr := os.Getenv("UNSTAKE_MINUTES")
		minutes, err := strconv.Atoi(minuteStr)
		if err != nil {
			xlog.Errorf("Invalid UNSTAKE_MINUTES value: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		// Check if action is 'unstake' and created within the last X minutes
		if stake.Action == "unstake" && time.Since(stake.CreatedAt) <= time.Duration(minutes)*time.Minute {
			isPending = true
		}

		earningsFromLastClaimx18.Add(earningsFromLastClaimx18, cutils.StrToBigInt(stake.Earnings))
		if stake.Action == "claim" {
			response = append(response, gin.H{
				"action":     stake.Action,
				"amount":     earningsFromLastClaimx18.String(),
				"created_at": stake.CreatedAt,
				"isPending":  isPending,
			})
			earningsFromLastClaimx18 = big.NewInt(0)
		} else {
			response = append(response, gin.H{
				"action":     stake.Action,
				"amount":     stake.Amount,
				"created_at": stake.CreatedAt,
				"isPending":  isPending,
			})
		}
	}

	// Add deposit/withdrawal records with additional fields, filtering by productId == 0 or 2
	for _, dw := range depositsWithdrawals {
		if dw.ProductId == 0 || dw.ProductId == 2 { // Only include productId 0 or 2
			action := "withdraw"
			if dw.IsDeposit {
				action = "deposit"
			}

			// Determine if it's pending based on the 'finalised' field
			isPending := !dw.Finalised

			response = append(response, gin.H{
				"action":             action,
				"amount":             dw.Amount,
				"created_at":         dw.CreatedAt,
				"isPending":          isPending,
				"txHash":             dw.TxnHash,            // Assuming dw contains the TxnHash field
				"destinationChainId": dw.DestinationChainID, // Assuming dw contains the DestinationChainID field
				"sourceChainId":      dw.SourceChainID,      // Assuming dw contains the SourceChainID field
			})
		}
	}

	// Respond with the combined staking and deposit/withdrawal information
	cutils.ApiResponse(ctx, gin.H{"records": response}, http.StatusOK, "success")
}

func (sc *StakingController) GetTotalEarnings(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	totalEarnings, err := sc.stakingDb.GetTotalEarnings(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Staking Controller - error fetching total earnings information for subaccount ID %v", currentSubaccount.ID)
	}
	cutils.ApiResponse(ctx, gin.H{"totalEarnings": totalEarnings.String()}, http.StatusOK, "")
}

func (sc *StakingController) GetUnclaimedEarnings(ctx *gin.Context) {
	currentSubaccount, err1 := getCurrentSubaccount(ctx)
	if err1 != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	// Fetch the total claimable earnings
	earnings := sc.stakingService.GetClaimableEarnings(currentSubaccount.ID)
	if earnings == nil {
		xlog.Errorf("Staking Controller - error fetching subaccount %v's earnings", currentSubaccount.ID)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error while fetching user earnings")
		return
	}

	cutils.ApiResponse(ctx, gin.H{"unclaimedEarnings": earnings.TotalClaimableEarningsx18.String(), "currentApyFraction": cutils.X18ToFloatStr(earnings.EarningMetadata.EarningApyfx18)}, http.StatusOK, "")
}


func (sc *StakingController) GetUnclaimedEarningsOpen(ctx *gin.Context) {
	subAccountId := ctx.Param("subaccountId")
	if subAccountId == "" {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "subAccountId is required")
		return
	}

	earnings := sc.stakingService.GetClaimableEarnings(subAccountId)
	if earnings == nil {
		xlog.Errorf("Staking Controller - error fetching subaccount %v's earnings", subAccountId)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error while fetching user earnings")
		return
	}

	cutils.ApiResponse(ctx, gin.H{"unclaimedEarnings": earnings.TotalClaimableEarningsx18.String(), "currentApyFraction": cutils.X18ToFloatStr(earnings.EarningMetadata.EarningApyfx18)}, http.StatusOK, "")
}

func (sc *StakingController) GetStakingHistory(ctx *gin.Context) {
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

	// Fetch and filter withdrawals by productId 0 and 2
	withdrawals := (&db.DepositWithdrawDB{}).GetWithdrawalsBySubaccountId(subaccountID)
	withdrawalResponses := []gin.H{}
	for _, withdrawal := range *withdrawals {
		if withdrawal.ProductId == 0 || withdrawal.ProductId == 2 {
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
	}

	// Fetch and filter deposits by productId 0 and 2
	deposits := (&db.DepositWithdrawDB{}).GetDepositsBySubaccountId(subaccountID)
	depositResponses := []gin.H{}
	for _, deposit := range *deposits {
		if deposit.ProductId == 0 || deposit.ProductId == 2 {
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
	}

	// Combine both filtered responses into one
	response := gin.H{
		"deposits":    depositResponses,
		"withdrawals": withdrawalResponses,
	}

	// Send the combined response
	ctx.JSON(http.StatusOK, response)
}
