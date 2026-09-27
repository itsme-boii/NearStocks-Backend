package api

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/services/balance-server/controller"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

func RegisterRoutes(router *gin.Engine, balanceController *controller.BalanceController) {
	// CORS middleware configuration to allow all origins and headers
	corsConfig := cors.New(cors.Options{
		AllowedOrigins: cutils.CORSAllowedOrigins(),
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		//ToDo - Add only allowed headers
		AllowedHeaders:   []string{"*"}, // Allow all headers
		AllowCredentials: false,         // clients authenticate with headers, never cookies
	})

	// Apply the CORS middleware to the router
	// Apply the CORS middleware to the router using gin's middleware functionality
	router.Use(func(c *gin.Context) {
		corsConfig.HandlerFunc(c.Writer, c.Request)
		c.Next()
	})
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Service is healthy")
	})
	balanceGroup := router.Group("/balance")
	{
		balanceGroup.GET("/", balanceController.PerformHealthCheck)
		balanceGroup.POST("/update-token-balance", balanceController.UpdateTokenBalanceHandler)
		balanceGroup.POST("/update-pre-market-balance", balanceController.UpdatePreMarketBalanceHandler)
		balanceGroup.GET("/get-pre-market-balance", balanceController.GetPreMarketBalanceHandler)
		balanceGroup.GET("/get-pre-market-balances", balanceController.GetPreMarketBalancesHandler)
		balanceGroup.POST("/update-synthetic-spot-balance", balanceController.UpdateSyntheticSpotBalanceHandler)
		balanceGroup.GET("/get-synthetic-spot-balance", balanceController.GetSyntheticSpotBalanceHandler)
		balanceGroup.GET("/get-synthetic-spot-balances", balanceController.GetSyntheticSpotBalancesHandler)
		balanceGroup.GET("/get-user-entity", balanceController.GetBalanceHandler)
		balanceGroup.GET("/getTotalBuyingPower", balanceController.GetTotalBuyingPowerHandler)
		balanceGroup.GET("/getTotalTokenAmount", balanceController.GetTotalTokenAmountHandler)
		balanceGroup.GET("/getTotalPnL", balanceController.GetTotalPnLHandler)
		balanceGroup.POST("/add-mapping", balanceController.AddMappingHandler)
		balanceGroup.POST("/lock-balance", balanceController.LockBalanceHandler)
		balanceGroup.POST("/unlock-balance", balanceController.UnlockBalanceHandler)
		balanceGroup.POST("/subaccount-match", balanceController.UpdateSubaccountForMatchHandler)
		balanceGroup.POST("/subaccount-liquidation-match", balanceController.UpdateSubaccountsForLiquidationHandler)
		balanceGroup.POST("/settle-with-insurance", balanceController.SettleUsingInsuranceHandler)
		balanceGroup.GET("/getAvailableMargin", balanceController.GetAvailableMarginHandler)
		balanceGroup.GET("/withdrawableTokenBalance", balanceController.WithdrawableTokenBalanceHandler)
		balanceGroup.GET("/getPerpPositions", balanceController.GetPerpPositionsHandler)
		balanceGroup.GET("/getSpotBalance", balanceController.GetSpotBalanceHandler)
		balanceGroup.POST("/update-multi-token-balance", balanceController.UpdateMultiTokenBalanceHandler)
		balanceGroup.POST("/syncBalances", balanceController.SyncSubaccountsHandler)
		balanceGroup.GET("/syncFundingRates", balanceController.SyncCumulativeFundingRatesHandler)
		balanceGroup.POST("/syncOi", balanceController.SyncOiHandler)
		balanceGroup.GET("/getFullBalance/:subaccountID", balanceController.GetFullBalanceHandler)
		balanceGroup.GET("/health/:subaccountID", balanceController.GetSubaccountHealthHandler)
		balanceGroup.POST("/settlePnl", balanceController.SettlePnLForSubaccountsHandler)
		balanceGroup.POST("/settleBroker2Pnl/:subaccountID", balanceController.SettleBroker2PnLHandler)
		balanceGroup.POST("/shiftKromaFunds/:subaccountID", balanceController.ShiftKromaFundsHandler)
		balanceGroup.POST("/burnKromaFunds/:subaccountID", balanceController.BurnKromaFundsHandler)

		// Only for testing
		balanceGroup.POST("/unlock-all-balance", balanceController.UnlockAllLockedBalanceHandler)
	}
}
