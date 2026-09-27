package routes

import (
	"services/external-campaigns/controllers"
	"services/external-campaigns/services"

	"github.com/gin-gonic/gin"
)

func PublicRoutesSetup(router *gin.Engine) {
	// Initialize the PublicApi instance
	publicApi := services.NewPublicApi()

	// Initialize the controller with PublicApi
	publicApiController := controllers.NewPublicApiController(publicApi)

	// Define public routes
	publicController := router.Group("/public/v1/market")
	{
		publicController.GET("/depth", publicApiController.OrderBookHandler)           // API to return orderbook depth for given market
		publicController.GET("/positions", publicApiController.MarketDetailsByProductId)
		publicController.GET("/liquidationTrades", publicApiController.LiquidationTradesHandler) // API to return latest liquidation trades
	}

	// Define user dashboard routes
	dashboardController := router.Group("/v1/dashboard")
	{
		dashboardController.GET("/users", publicApiController.UserDashboardHandler)
		dashboardController.GET("/user", publicApiController.UserDashboardByAddressHandler)
		dashboardController.GET("/user/dashboardDetails", publicApiController.UserDashboardDetailsBySubaccountIdHandler)
	}
}
