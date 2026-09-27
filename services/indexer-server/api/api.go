package api

import (
	"github/eugenix-io/logx-inf-backend/services/indexer-server/indexer"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.GET("/", HealthCheck)
	router.POST("/runSourceDeposit", RunSourceDepositTracker)
}

func HealthCheck(c *gin.Context) {
	c.String(http.StatusOK, "Indexer Server is running")
}

type RunSourceDepositTrackerRequest struct {
	ProductId  uint32 `json:"productId"`
	StartBlock uint64 `json:"startBlock"`
}

func RunSourceDepositTracker(c *gin.Context) {
	var request RunSourceDepositTrackerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	depositsTracker := indexer.NewDepositSourceTracker()
	err := depositsTracker.ExecuteSingle(request.ProductId, request.StartBlock)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.String(http.StatusOK, "Running Source Deposit Tracker")
}
