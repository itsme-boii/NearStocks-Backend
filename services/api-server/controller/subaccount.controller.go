package controller

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"github/eugenix-io/logx-inf-backend/xclient"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SubAccountController struct {
	accountService *services.AccountService
	balanceClient  *xclient.BalanceClient
}

func RegisterSubaccountController(
	r *gin.RouterGroup,
) {
	subaccountController := SubAccountController{
		accountService: services.NewAccountService(),
		balanceClient:  xclient.GlobalBalanceClient,
	}
	rg := r.Group("/subaccount")

	// Private Endpoints
	rg.GET("/:id", middleware.RequireAuth, subaccountController.GetById)
	rg.GET("", middleware.RequireAuth, subaccountController.GetAll)
	rg.GET("/nonce/:id", subaccountController.GetNonce)
	rg.GET("/get-username/:id", middleware.RequireAuth, subaccountController.GetUserName)
	/// to create username
	rg.POST("/update-username", middleware.RequireAuth, subaccountController.UpdateUserName)
	rg.GET("/check-username", subaccountController.CheckUserNameAvailability)
	// for migration
	rg.POST("/create-username", subaccountController.CreateUserName)
	// rg.POST("/unpause-subaccount/:subaccountID", subaccountController.UnpauseSubaccount)

}

type SubaccountGetByIdRequest struct {
	Id string `uri:"id" binding:"required"`
}

// FIXME: @snehilms - Add check for case whether subaccount exists or not
func (s *SubAccountController) GetNonce(ctx *gin.Context) {
	var requestObj SubaccountGetByIdRequest
	if err := ctx.BindUri(&requestObj); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	nonce, err := services.WithNonceRedisLock(requestObj.Id, func(nonce string) (*string, error) {
		return &nonce, nil
	})

	if err != nil {
		xlog.Errorf("Failed to fetch nonce with err: %v", err)
		ctx.AbortWithError(http.StatusInternalServerError, gin.Error{})
	}

	cutils.ApiResponse(ctx, gin.H{"nonce": *nonce}, http.StatusOK, "")
}

func (*SubAccountController) GetById(ctx *gin.Context) {
	var requestObj SubaccountGetByIdRequest
	if err := ctx.BindUri(&requestObj); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	curSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if curSubaccount.ID != requestObj.Id {
		ctx.AbortWithError(http.StatusForbidden, fmt.Errorf("user with id: %v does not have access to id: %v", curSubaccount.ID, requestObj.Id))
		return
	}

	cutils.ApiResponse(ctx, gin.H{"subaccount": curSubaccount}, http.StatusOK, "")
}

func (*SubAccountController) GetAll(ctx *gin.Context) {
	curSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	subaccounts := (&db.SubaccountDB{}).GetAllByAddressBrokerId(curSubaccount.EthAddress, curSubaccount.BrokerId)

	cutils.ApiResponse(ctx, gin.H{"subaccounts": subaccounts}, http.StatusOK, "")
}

type SettlePnlBody struct {
	Nonce   *int64 `json:"nonce" binding:"required"`
	ChainId int64  `json:"chainId" binding:"required"`
}
type SettlePnlHeader struct {
	Signature     string `header:"Logx-Signature" binding:"required"`
	SignerAddress string `header:"Logx-Signer-Address" binding:"required"`
}
type SettlePnlRequest struct {
	*SettlePnlBody
	*SettlePnlHeader
}

func (*SubAccountController) GetUserName(ctx *gin.Context) {
	id := ctx.Param("id")
	userName, err := (&db.SubaccountDB{}).GetUserName(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if userName == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "username not found"})
		return
	}

	cutils.ApiResponse(ctx, gin.H{"userName": *userName}, http.StatusOK, "")
}

func (*SubAccountController) CheckUserNameAvailability(ctx *gin.Context) {
	username := ctx.Query("username")
	if username == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	isAvailable, err := (&db.SubaccountDB{}).CheckUserNameAvailability(username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"available": isAvailable})
}

func (*SubAccountController) UpdateUserName(ctx *gin.Context) {
	var req struct {
		ID       string `json:"id" binding:"required"`
		UserName string `json:"userName" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err := (&db.SubaccountDB{}).UpdateUserName(req.ID, req.UserName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Username updated successfully"})
}

func (*SubAccountController) CreateUserName(ctx *gin.Context) {
	var req struct {
		ID         []string `json:"id" binding:"required"`
		UserName   []string `json:"userName" binding:"required"`
		EthAddress []string `json:"ethAddress" binding:"required"`
		BatchSize  uint     `json:"batchSize" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request payload")
		return
	}

	for i := range req.BatchSize {
		(&db.SubaccountDB{}).CreateUserName(req.ID[i], req.UserName[i], req.EthAddress[i])
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Username created successfully"})
}

// func (sc *SubAccountController) UnpauseSubaccount(ctx *gin.Context) {
// 	// Here subaccountID will be in hex
// 	subaccountID := ctx.Param("subaccountID")
// 	if subaccountID == "" {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountID is required"})
// 		return
// 	}

// 	// Check if the subaccount is even paused in Redis.
// 	if !cutils.IsSubAccountPaused(subaccountID) {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccount is not paused"})
// 		return
// 	}

// 	// Update the batch entries to unpause the subaccount
// 	if err := (&db.BatchDB{}).UpdatePausedEntries(subaccountID); err != nil {
// 		ctx.JSON(http.StatusInternalServerError, gin.H{
// 			"error": fmt.Sprintf("failed to unpause subaccount in DB: %v", err),
// 		})
// 		return
// 	}

// 	// Call SyncSubaccountBalances to synchronize the subaccount’s balance
// 	msg, err := sc.balanceClient.SyncSubaccountBalances([]string{subaccountID})
// 	if err != nil {
// 		ctx.JSON(http.StatusInternalServerError, gin.H{
// 			"error": fmt.Sprintf("failed to sync subaccount balances: %v", err),
// 		})
// 		return
// 	}

// 	redisKey := xredis.GetSubacountPauseSequencerKey()

// 	if _, err := xredis.GetRedisClient().HDel(context.Background(), redisKey, subaccountID).Result(); err != nil {
// 		ctx.JSON(http.StatusInternalServerError, gin.H{
// 			"error": fmt.Sprintf("failed to remove subaccount from Redis: %v", err),
// 		})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, gin.H{
// 		"message":      "Subaccount unpaused successfully",
// 		"syncMessage":  msg,
// 		"subaccountID": subaccountID,
// 	})
// }
