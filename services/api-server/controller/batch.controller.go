package controller

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type BatchController struct {
	RedisClient *redis.Client
}

type BatchTransaction struct {
	Transaction []byte `json:"transaction"`
	Signature1  []byte `json:"signature1"` // Signature <or> Taker Signature
	Signature2  []byte `json:"signature2"` // nil <or> Maker Signature
}

func RegisterBatchingController(r *gin.RouterGroup) {
	batchController := BatchController{
		RedisClient: xredis.GetRedisClient(), // Initialize RedisClient
	}
	rg := r.Group("/batch")
	rg.GET("/status", middleware.RequireAuth, middleware.RequireSequencer, batchController.GetBatchingServiceStatus)
	rg.GET("/check-undermaintenance/:subaccountId", batchController.CheckSubaccountUnderMaintenance)

}

func (b *BatchController) GetBatchingServiceStatus(ctx *gin.Context) {
	if ctx.IsAborted() {
		cutils.ApiSuccessWithoutMessage(ctx, false, http.StatusOK)
		return
	}

	cutils.ApiSuccessWithoutMessage(ctx, "true", http.StatusOK)
}
func (b *BatchController) CheckSubaccountUnderMaintenance(ctx *gin.Context) {
	subaccountId := ctx.Param("subaccountId")
	if subaccountId == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "subaccountId is required"})
		return
	}

	isUnderMaintenance := cutils.IsSubAccountPaused(subaccountId)

	response := gin.H{
		"isUnderMaintenance": isUnderMaintenance,
		"subaccountId":       subaccountId,
	}

	ctx.JSON(http.StatusOK, response)
}
