package controller

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/services"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type BalanceController struct {
	balanceService       *services.BalanceService
	balanceLockerService *services.BalanceLockerService
	liquidationService   services.LiquidationService
}

func NewBalanceController() *BalanceController {
	service := services.NewBalanceService()
	balanceLockerService := services.NewBalanceLockerService()
	liquidationService := services.NewLiquidationService()
	return &BalanceController{
		balanceService:       service,
		balanceLockerService: balanceLockerService,
		liquidationService:   liquidationService,
	}
}

func (c *BalanceController) UpdateTokenBalanceHandler(ctx *gin.Context) {
	var req types.TokenBalanceRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.SubaccountID = strings.ToLower(req.SubaccountID)
	if err := c.balanceService.UpdateTokenBalance(req.SubaccountID, req.ProductId, req.TokenBalance); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *BalanceController) UpdatePreMarketBalanceHandler(ctx *gin.Context) {
	var req types.TokenBalanceRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.SubaccountID = strings.ToLower(req.SubaccountID)
	if err := c.balanceService.UpdatePreMarketBalance(req.SubaccountID, req.ProductId, req.TokenBalance); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *BalanceController) GetPreMarketBalanceHandler(ctx *gin.Context) {
	subaccountID := strings.ToLower(ctx.Query("subaccountID"))
	productIDStr := ctx.Query("productID")

	if subaccountID == "" || productIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID and productID are required"})
		return
	}

	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid productID"})
		return
	}

	balance, err := c.balanceService.GetPreMarketBalance(subaccountID, uint32(productID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (c *BalanceController) GetPreMarketBalancesHandler(ctx *gin.Context) {
	subaccountID := strings.ToLower(ctx.Query("subaccountID"))

	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
		return
	}

	balances, err := c.balanceService.GetPreMarketBalances(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"balances": balances})
}

func (c *BalanceController) UpdateSyntheticSpotBalanceHandler(ctx *gin.Context) {
	var req types.TokenBalanceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.SubaccountID = strings.ToLower(req.SubaccountID)
	if err := c.balanceService.UpdateSyntheticSpotBalance(req.SubaccountID, req.ProductId, req.TokenBalance); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *BalanceController) GetSyntheticSpotBalanceHandler(ctx *gin.Context) {
	subaccountID := strings.ToLower(ctx.Query("subaccountID"))
	productIDStr := ctx.Query("productID")

	if subaccountID == "" || productIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID and productID are required"})
		return
	}

	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid productID"})
		return
	}

	balance, err := c.balanceService.GetSyntheticSpotBalance(subaccountID, uint32(productID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (c *BalanceController) GetSyntheticSpotBalancesHandler(ctx *gin.Context) {
	subaccountID := strings.ToLower(ctx.Query("subaccountID"))

	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
		return
	}

	balances, err := c.balanceService.GetSyntheticSpotBalances(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"balances": balances})
}

func (c *BalanceController) PerformHealthCheck(ctx *gin.Context) {
	ctx.String(http.StatusOK, "Balance Health Check: OK")
}

func (c *BalanceController) UpdateMultiTokenBalanceHandler(ctx *gin.Context) {
	var req types.MultiTokenBalanceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.SubaccountID = strings.ToLower(req.SubaccountID)
	if err := c.balanceService.UpdateMultiTokenBalance(req.SubaccountID, req.ProductIds, req.TokenBalances); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *BalanceController) GetBalanceHandler(ctx *gin.Context) {
	subaccountID := ctx.Query("subaccountID")
	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
		return
	}
	subaccountID = strings.ToLower(subaccountID)

	balance, err := c.balanceService.GetBalance(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, balance)
}

func (c *BalanceController) WithdrawableTokenBalanceHandler(ctx *gin.Context) {
	subaccountID := ctx.Query("subaccountID")
	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID are required"})
		return
	}
	subaccountID = strings.ToLower(subaccountID)

	withdrawableBalance, err := c.balanceService.WithdrawableTokenBalancev2(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"withdrawableBalance": withdrawableBalance})
}

//todo: change function name check where is this being used

func (c *BalanceController) GetTotalBuyingPowerHandler(ctx *gin.Context) {
	subaccountID := ctx.Query("subaccountID")
	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
		return
	}
	subaccountID = strings.ToLower(subaccountID)

	totalSpotValuex18, err := c.balanceService.GetTotalSpotValuex18(subaccountID)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"totalBuyingPower": totalSpotValuex18.String()})
}

func (c *BalanceController) GetTotalTokenAmountHandler(ctx *gin.Context) {
	subaccountID := ctx.Query("subaccountID")
	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
		return
	}
	subaccountID = strings.ToLower(subaccountID)

	totalTokenAmount, err := c.balanceService.GetTokenBalanceMap(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"tokenBalances": totalTokenAmount})
}

func (c *BalanceController) GetTotalPnLHandler(ctx *gin.Context) {
	subaccountID := ctx.Query("subaccountID")
	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
		return
	}
	subaccountID = strings.ToLower(subaccountID)

	pnlPerToken, totalPnL, err := c.balanceService.GetTotalPnL(subaccountID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"pnlPerToken": pnlPerToken,
		"totalPnL":    totalPnL,
	})
}

func (c *BalanceController) AddMappingHandler(ctx *gin.Context) {
	var req types.MappingRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.balanceService.AddMapping(req.ProductId, "", req.Symbol); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

// NOTE: Currently, only USDC Quote Token is supported
func (c *BalanceController) LockBalanceHandler(ctx *gin.Context) {
	var req = types.LockBalanceRequest{}
	if err := ctx.BindJSON(&req); err != nil {
		fmt.Printf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}
	xlog.Debugf("Received request: %+v\n", req)

	if err := req.Validate(); err != nil {
		fmt.Printf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Convert subaccountID to lower case for safety
	subaccountID := strings.ToLower(req.SubaccountId)

	bigAvailableMarginx18, err := c.balanceService.GetAvailableMarginV2(subaccountID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Cannot get available buying power")
		return
	}

	dollarValueInQuoteTokenBigInt, err := c.balanceService.GetValueInQuoteTokenx18(req.ProductId, req.LockQuotex18)
	if err != nil {
		fmt.Printf("Error while fetching lockedAmount valueInQuoteToken: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error while fetching lockedAmount valueInQuoteToken")
		return
	}

	if bigAvailableMarginx18.Cmp(dollarValueInQuoteTokenBigInt) == -1 {
		xlog.Errorf("Insufficient available margin: %v | Required: %v\n", bigAvailableMarginx18, req.LockQuotex18)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Insufficient available buying power")
		return
	}

	err = c.balanceLockerService.AtomicLockBalance(req.SubaccountId, req.Entity, req.EntityId, req.ProductId, req.LockQuotex18)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "System fault: Unable to lock balance")
		return
	}

	cutils.ApiSuccess(ctx, nil, "Balance locked successfully")
}

func (c *BalanceController) UnlockBalanceHandler(ctx *gin.Context) {
	var req types.UnlockBalanceRequest

	if err := ctx.BindJSON(&req); err != nil {
		fmt.Printf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	} else if err := req.Validate(); err != nil {
		fmt.Printf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if req.Partial {
		// TODO: Need to implement different error types

		//Locking the balance-lock key here but not in the else condition because "UnlockFullBalance" function has a lock inside the function
		//We are having to acquire the keyr UnlockPartialbalance here because of "TxPipelined". This is not the case with rest of the functions in balance service
		mutex, err := xredis.AcquireLockWithRetry(fmt.Sprintf("balance-lock-%s", req.SubaccountId))
		if err != nil {
			xlog.Errorf("Failed to acquire lock:", err)
			return
		}
		defer func() {
			// Release lock
			if err := xredis.ReleaseLock(mutex); err != nil {
				xlog.Errorf("Failed to release lock:", err)
			}
		}()

		_, err = c.balanceLockerService.RedisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return c.balanceLockerService.UnlockPartialBalanceWithoutLock(pipe, req.SubaccountId, req.Entity, req.EntityId, *&req.LockQuotex18)
		})
		if err != nil {
			fmt.Printf("Cannot unlock partial balance: %v\n", err)
			cutils.ApiAbort(ctx, http.StatusInternalServerError, "Cannot unlock partial balance")
			return
		}
	} else {
		err := c.balanceLockerService.UnlockFullBalance(req.SubaccountId, req.Entity, req.EntityId)
		if err != nil {
			fmt.Printf("Cannot unlock full balance: %v\n", err)
			cutils.ApiAbort(ctx, http.StatusInternalServerError, "Cannot unlock full balance")
			return
		}
	}

	cutils.ApiSuccess(ctx, nil, "Balance unlocked successfully")
}
func (c *BalanceController) GetPerpPositionsHandler(ctx *gin.Context) {
	var req types.PerpPositionsRequest
	if err := ctx.BindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.SubaccountID = strings.ToLower(req.SubaccountID)

	var productIdsStr []string
	if req.ProductIds != "" {
		productIdsStr = strings.Split(req.ProductIds, ",")
	} else {
		productIdsStr = []string{}
	}

	productIds := make([]uint32, len(productIdsStr))

	for i, productIdStr := range productIdsStr {
		productId, err := strconv.ParseUint(productIdStr, 10, 32)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if productId%2 == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid productID: %d", productId)})
			return
		}
		productIds[i] = uint32(productId)
	}

	perpPositions, err := c.balanceService.GetPerpPositions(req.SubaccountID, productIds)
	if err != nil {
		fmt.Printf("Error getting perp positions: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"PerpPositions": perpPositions})
}

func (c *BalanceController) UnlockAllLockedBalanceHandler(ctx *gin.Context) {
	var req types.UnlockAllLockedBalanceRequest

	if err := ctx.BindJSON(&req); err != nil {
		fmt.Printf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	err := c.balanceLockerService.UnlockAllLockedBalance(req.SubaccountId)
	if err != nil {
		fmt.Printf("Cannot unlock all locked balance: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Cannot unlock all locked balance")
		return
	}

	cutils.ApiSuccess(ctx, nil, "Balance unlocked successfully")
}

// We will unlock: Initial Price * MatchQuantity + (1 + Taker Fee fraction)
// This value will come as unlockQuotex18
// NOTE: We are not taking maker fee into consideration for now
func (c *BalanceController) UpdateSubaccountForMatchHandler(ctx *gin.Context) {
	var req types.UpdateSubaccountForMatchRequest
	if err := ctx.BindJSON(&req); err != nil {
		xlog.Errorf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		xlog.Errorf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	unlockErr := c.balanceLockerService.AtomicUpdateSubaccountForMatch(req)
	if unlockErr != nil {
		cutils.ApiSuccess(ctx, gin.H{
			"maker_success":      !unlockErr.MakerError,
			"taker_success":      !unlockErr.TakerError,
			"maker_realized_pnl": "0",
			"taker_realized_pnl": "0",
		}, unlockErr.Error())
		return
	}

	// NOTE NOTE NOTE: We are not updating the perp balance here because for liquidation orders balance update will occur from api service
	// TODO: Move the balance update logic here itself from liqudiation.service.go
	if req.IsLiquidation {
		cutils.ApiSuccess(ctx, gin.H{
			"maker_success":      true,
			"taker_success":      true,
			"maker_realized_pnl": "0",
			"taker_realized_pnl": "0",
		}, "")
		return
	}

	// Since, there was no issue unlocking the balance we will update the perp balance of both the accounts
	updateErr, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees := c.balanceService.AtomicUpdateBalanceForOrderMatch(req)
	makerRealizedPnlStr := "0"
	takerRealizedPnlStr := "0"
	makerFundingFeesStr := "0"
	takerFundingFeesStr := "0"

	if makerRealizedPnl != nil {
		makerRealizedPnlStr = makerRealizedPnl.String()
	}
	if takerRealizedPnl != nil {
		takerRealizedPnlStr = takerRealizedPnl.String()
	}
	if makerFundingFees != nil {
		makerFundingFeesStr = makerFundingFees.String()
	}
	if takerFundingFees != nil {
		takerFundingFeesStr = takerFundingFees.String()
	}

	if updateErr != nil {
		cutils.ApiSuccess(ctx, gin.H{
			"maker_success":      !updateErr.MakerError,
			"taker_success":      !updateErr.TakerError,
			"maker_realized_pnl": makerRealizedPnlStr,
			"taker_realized_pnl": takerRealizedPnlStr,
			"maker_funding_fees": makerFundingFeesStr,
			"taker_funding_fees": takerFundingFeesStr,
		}, updateErr.Error())
		return
	}

	cutils.ApiSuccess(ctx, gin.H{
		"maker_success":      true,
		"taker_success":      true,
		"maker_realized_pnl": makerRealizedPnlStr,
		"taker_realized_pnl": takerRealizedPnlStr,
		"maker_funding_fees": makerFundingFeesStr,
		"taker_funding_fees": takerFundingFeesStr,
	}, "")
}

func (c *BalanceController) GetAvailableMarginHandler(ctx *gin.Context) {
	subaccountID := ctx.Query("subaccountID")
	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
		return
	}
	subaccountID = strings.ToLower(subaccountID)

	availableMargin, err := c.balanceService.GetAvailableMarginV2(subaccountID)
	if err != nil {
		// Log the error
		fmt.Printf("Error calculating available margin: %v\n", err)
	}

	ctx.JSON(http.StatusOK, gin.H{"availableMargin": availableMargin.String()})
}

func (c *BalanceController) SyncSubaccountsHandler(ctx *gin.Context) {
	var request struct {
		SubaccountIDs []string `json:"subaccountIDs"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if len(request.SubaccountIDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountIDs are required"})
		return
	}

	// Convert subaccount IDs to lowercase
	for i, subaccountID := range request.SubaccountIDs {
		request.SubaccountIDs[i] = strings.ToLower(subaccountID)
	}

	err := c.balanceService.SyncSubaccounts(request.SubaccountIDs)
	if err != nil {
		xlog.Infof("Error syncing subaccounts: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Subaccounts synced successfully"})
}

func (c *BalanceController) SyncCumulativeFundingRatesHandler(ctx *gin.Context) {
	// Check for pending transactions in batch_tables
	exists, err := (&db.BatchDB{}).CheckPerpTickExists()
	if err != nil {
		xlog.Infof("Error checking PerpTick existence: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If there are pending transactions, log the message and return
	if !exists {
		xlog.Infof("There are pending transactions in the batch_tables.")
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "There are pending transactions in the batch_tables."})
		return
	}

	// Call the syncCumulativeFundingRates function from the BalanceService
	err = c.balanceService.SyncCumulativeFundingRates()
	if err != nil {
		xlog.Infof("Error syncing cumulative funding rates: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return success response
	ctx.JSON(http.StatusOK, gin.H{"message": "Cumulative funding rates synced successfully"})
}

func (c *BalanceController) SyncOiHandler(ctx *gin.Context) {
	err := c.balanceService.SyncOi()
	if err != nil {
		xlog.Infof("Error syncing Oi : %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "OI synced successfully"})
}

// func (c *BalanceController) GetSubaccountSpotContractData(ctx *gin.Context) {
// 	subaccountID := ctx.Query("subaccountID")

// 	if subaccountID == "" {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
// 		return
// 	}

// 	// Fetch spot positions using the BalanceService
// 	positions, err := c.balanceService.GetAccountSpotPositionsContract(subaccountID)
// 	if err != nil {
// 		xlog.Infof("Error fetching contract data: %v\n", err)
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// Fetch nonce from Redis
// 	redisNonce, err := c.balanceService.GetNonce(subaccountID)
// 	if err != nil {
// 		xlog.Infof("Error fetching nonce from Redis: %v\n", err)
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// Fetch nonce from the contract directly
// 	contractNonceUint, err := contract.GlobalContracts.EndpointContract.GetNonceForSubaccount(subaccountID)
// 	if err != nil {
// 		xlog.Infof("Error fetching nonce from contract: %v\n", err)
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	contractNonce := fmt.Sprintf("%d", contractNonceUint)

// 	// Prepare the response
// 	response := gin.H{
// 		"positions":      positions,
// 		"redis_nonce":    redisNonce,
// 		"contract_nonce": contractNonce,
// 	}

// 	ctx.JSON(http.StatusOK, response)
// }

func (c *BalanceController) GetSpotBalanceHandler(ctx *gin.Context) {
	subaccountID := ctx.Query("subaccountID")
	if subaccountID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
		return
	}
	subaccountID = strings.ToLower(subaccountID)

	spotBalances, freeBalance, usedBalance, err := c.balanceService.GetSpotBalance(subaccountID)
	if err != nil {
		fmt.Printf("Error getting spot balances: %v\n", err)
		ctx.JSON(http.StatusOK, gin.H{"spotBalance": []ctypes.SpotBalance{}})
		return
	}

	response := gin.H{
		"freeBalance": freeBalance,
		"usedBalance": usedBalance,
		"spotBalance": spotBalances,
	}

	ctx.JSON(http.StatusOK, response)
}

// ----------------------------- Liquidation related handlers -----------------------------
// Todo: Add custom validations
func (c *BalanceController) UpdateSubaccountsForLiquidationHandler(ctx *gin.Context) {
	var req types.FinaliseLiquidationRequest
	if err := ctx.BindJSON(&req); err != nil {
		xlog.Errorf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	response, err := c.liquidationService.FinaliseLiquidation(&req)
	if err != nil {
		xlog.Errorf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, gin.H{
		"maker_realized_pnl": (*response.RealisedLiquidatorPnlx18).String(),
		"taker_realized_pnl": (*response.RealisedLiquidateePnlx18).String(),
		"maker_funding_fees": (*response.LiquidatorFundingFeesx18).String(),
		"taker_funding_fees": (*response.LiquidateeFundingFeesx18).String(),
	}, "Liquidation finalised successfully")
}

func (c *BalanceController) SettleUsingInsuranceHandler(ctx *gin.Context) {
	var req types.SettleWithInsuranceRequest
	if err := ctx.BindJSON(&req); err != nil {
		xlog.Errorf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.liquidationService.SettleUsingInsurance(&req); err != nil {
		xlog.Errorf("Error: %v\n", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, nil, "Settlement using insurance funds successful")
}

func (c *BalanceController) SettlePnLForSubaccountsHandler(ctx *gin.Context) {
	// Define a struct to capture the subaccountIDs from the request body
	var requestBody struct {
		SubaccountIDs []string `json:"subaccountIDs"`
	}

	// Parse the request body into the struct
	if err := ctx.ShouldBindJSON(&requestBody); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Call the service to settle PnL for the provided subaccountIDs
	tokenPrices, failedSubAccounts, err := c.balanceService.SettlePnLForSubaccounts(requestBody.SubaccountIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the result as JSON
	ctx.JSON(http.StatusOK, gin.H{
		"tokenPrices":       tokenPrices,
		"failedSubAccounts": failedSubAccounts,
	})
}

type FullBalanceRequest struct {
	SubaccountID string `uri:"subaccountID" binding:"required"`
}

func (c *BalanceController) GetFullBalanceHandler(ctx *gin.Context) {
	var req FullBalanceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		cutils.ApiAbort(ctx, 400, err.Error())
		return
	}

	subaccountID := strings.ToLower(req.SubaccountID)

	spotBalances, freeBalance, usedBalance, err := c.balanceService.GetSpotBalance(subaccountID)
	if err != nil {
		fmt.Printf("Error getting spot balances: %v\n", err)
		ctx.JSON(http.StatusOK, gin.H{"spotBalance": []ctypes.SpotBalance{}})
		return
	}

	web2Spot := gin.H{
		"freeBalance": freeBalance,
		"usedBalance": usedBalance,
		"spotBalance": spotBalances,
	}


	perpPositions, err := c.balanceService.GetPerpPositions(req.SubaccountID, []uint32{})
	if err != nil {
		fmt.Printf("Error getting perp positions: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	web2Perp := gin.H{"PerpPositions": perpPositions}

	cutils.ApiSuccess(ctx, gin.H{
		"web2Spot": web2Spot,
		"web2Perp": web2Perp,
	}, "")
}

// TODO: Move to utils
type HeathRequest struct {
	SubaccountID string `uri:"subaccountID" binding:"required"`
}

func (bc *BalanceController) GetSubaccountHealthHandler(ctx *gin.Context) {
	var req HeathRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		cutils.ApiAbort(ctx, 400, err.Error())
		return
	}

	subaccountID := strings.ToLower(req.SubaccountID)

	health, err := bc.balanceService.GetHealth(subaccountID)
	if err != nil {
		cutils.ApiAbort(ctx, 500, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, health, "")
}

// ----------------------------- End of Liquidation related handlers -----------------------------

func (bc *BalanceController) SettleBroker2PnLHandler(ctx *gin.Context) {
	var req HeathRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		cutils.ApiAbort(ctx, 400, err.Error())
		return
	}

	subaccountID := strings.ToLower(req.SubaccountID)

	err := bc.balanceService.SettleBroker2PnL(subaccountID)
	if err != nil {
		cutils.ApiAbort(ctx, 500, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, nil, "Ostrich PnL settled successfully")
}

func (bc *BalanceController) ShiftKromaFundsHandler(ctx *gin.Context) {
	var req HeathRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		cutils.ApiAbort(ctx, 400, err.Error())
		return
	}

	subaccountID := strings.ToLower(req.SubaccountID)

	err := bc.balanceService.ShiftKromaFunds(subaccountID)
	if err != nil {
		cutils.ApiAbort(ctx, 500, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, nil, "Kroma funds shifted to Arbitrum successfully")
}

func (bc *BalanceController) BurnKromaFundsHandler(ctx *gin.Context) {
	var req HeathRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		cutils.ApiAbort(ctx, 400, err.Error())
		return
	}

	subaccountID := strings.ToLower(req.SubaccountID)

	err := bc.balanceService.BurnKromaFunds(subaccountID)
	if err != nil {
		cutils.ApiAbort(ctx, 500, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, nil, "Kroma funds burned successfully")
}
