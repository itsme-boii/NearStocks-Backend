package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"services/external-campaigns/services"

	"github.com/gin-gonic/gin"
)

type PublicApiController struct {
	PublicApi *services.PublicApi
}

func NewPublicApiController(publicApi *services.PublicApi) *PublicApiController {
	return &PublicApiController{
		PublicApi: publicApi,
	}
}

func (pac *PublicApiController) OrderBookHandler(ctx *gin.Context) {
	ticker := ctx.Query("ticker")
	if ticker == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Ticker is required",
		})
		return
	}

	orderBook, err := pac.PublicApi.FetchOrderBookByTicker(ctx, ticker)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch order book",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":   "Order book fetched successfully",
		"orderBook": orderBook,
	})
}

func (pac *PublicApiController) UserDashboardHandler(ctx *gin.Context) {
	startTime := time.Now()

	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid page parameter",
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid limit parameter (must be between 1 and 100)",
		})
		return
	}

	dashboardData, err := pac.PublicApi.FetchUserDashboardData(page, limit)
	if err != nil {
		xlog.Errorf("Error fetching user dashboard data: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch user dashboard data",
			"error":   err.Error(),
		})
		return
	}

	ctx.Header("X-Response-Time", time.Since(startTime).String())

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User dashboard data fetched successfully",
		"data":    dashboardData,
	})
}

func (pac *PublicApiController) UserDashboardByAddressHandler(ctx *gin.Context) {
	userAddress := ctx.Query("address")
	if userAddress == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "User address is required",
		})
		return
	}

	dashboardData, err := pac.PublicApi.FetchUserDashboardByAddress(userAddress)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch user dashboard data",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User dashboard data fetched successfully",
		"data":    dashboardData,
	})
}

// UserDashboardDetailsBySubaccountHandler handles GET /user/dashboard-details
func (pac *PublicApiController) UserDashboardDetailsBySubaccountIdHandler(ctx *gin.Context) {
	subaccountId := ctx.Query("subaccount_id")
	if subaccountId == "" {
		xlog.Errorf("Missing subaccount_id parameter in request")
		ctx.JSON(400, gin.H{"error": "subaccount_id is required"})
		return
	}

	// Get pagination parameters for trade history
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid page parameter",
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid limit parameter (must be between 1 and 100)",
		})
		return
	}

	dashboardDataBySubAccountId, err := pac.PublicApi.FetchUserDashBoardDetailsBySubAccountId(subaccountId, page, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch user dashboard data by SubAccountId",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User dashboard data fetched successfully",
		"data":    dashboardDataBySubAccountId,
	})
}

func (pac *PublicApiController) MarketDetailsByProductId(ctx *gin.Context) {
	marketId := ctx.Query("market_id")
	marketIdUint, err := strconv.ParseUint(marketId, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch market Id",
			"error":   err.Error(),
		})
		return
	}
	openPositions, err := pac.PublicApi.GetPositionsByMarketId(uint(marketIdUint), ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch market Positions",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "Market contracts fetched successfully",
		"openPositions": openPositions,
	})
}

// LiquidationTradesHandler handles the request to fetch liquidation trades
func (pac *PublicApiController) LiquidationTradesHandler(ctx *gin.Context) {
	startTime := time.Now()

	liquidationTrades, err := pac.PublicApi.FetchLiquidationTrades()
	if err != nil {
		xlog.Errorf("Error fetching liquidation trades: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch liquidation trades",
			"error":   err.Error(),
		})
		return
	}

	ctx.Header("X-Response-Time", time.Since(startTime).String())

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Liquidation trades fetched successfully",
		"data": gin.H{
			"liquidation_trades": liquidationTrades,
			"count":              len(*liquidationTrades),
			"broker_id":          2,
		},
	})
}
