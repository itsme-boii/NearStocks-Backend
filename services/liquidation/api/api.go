package api

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/services/liquidation/stats"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.GET("/", HealthCheck)
	router.GET("/stats", GetStats)
	router.GET("/withdrawableFundsStats", GetWithdrawableFundsStats)
}

func HealthCheck(c *gin.Context) {
	if os.Getenv("STOP_SERVICE") == "1" {
		c.String(http.StatusOK, "Liquidation service is stopped using env variable")
	} else {
		c.String(http.StatusOK, "Liquidation Server is running")
	}
}

func GetStats(c *gin.Context) {
	res, err := stats.GlobalLiquidationStats.Results()
	if err != nil {
		cutils.ApiAbort(c, http.StatusInternalServerError, "Error getting liquidation stats")
		return
	}
	cutils.ApiSuccess(c, res, "Liquidation Stats")
}

func GetWithdrawableFundsStats(c *gin.Context) {
	res, err := stats.GlobalFundsMonitorStats.FundsResults()
	if err != nil {
		cutils.ApiAbort(c, http.StatusInternalServerError, "Error getting funds stats")
		return
	}
	cutils.ApiSuccess(c, res, "Funds Stats")
}
