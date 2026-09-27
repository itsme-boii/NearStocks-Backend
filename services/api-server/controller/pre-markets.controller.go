package controller

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type PreMarketsController struct {
	balanceClient       *xclient.BalanceClient
	preMarketsDb        *db.PreMarketsDB
	preMarketsCandlesDb *db.PreMarketCandleDB
	marketPricesWs      *MarketPricesWebsocket
}

// Add this new struct for the request
type GetOHLCRequest struct {
	ProductId uint32 `json:"productId" binding:"required"`
	StartTime int64  `json:"startTime" binding:"required"`
	EndTime   int64  `json:"endTime" binding:"required"`
	Interval  string `json:"interval" binding:"required"`
}

// Add this struct for the transformed response
type CandleResponse struct {
	T  int64  `json:"t"` // start time
	T2 int64  `json:"T"` // end time
	I  string `json:"i"` // interval
	O  string `json:"o"` // open
	C  string `json:"c"` // close
	H  string `json:"h"` // high
	L  string `json:"l"` // low
}

type ConvertToSpotRequest struct {
	ProductId     uint32 `json:"productId" binding:"required"`
	ProductIdSpot uint32 `json:"productIdSpot" binding:"required"`
	Factor        uint32 `json:"factor" binding:"required"`
}

func RegisterPreMarketsController(r *gin.RouterGroup, marketPricesWs *MarketPricesWebsocket) {
	preMarketsController := &PreMarketsController{
		balanceClient:       xclient.GlobalBalanceClient,
		preMarketsDb:        &db.PreMarketsDB{},
		preMarketsCandlesDb: &db.PreMarketCandleDB{},
		marketPricesWs:      marketPricesWs,
	}

	rg := r.Group("/pre-markets")

	// Endpoints
	rg.POST("/placeOrder", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, preMarketsController.PlaceOrder)

	// Add new endpoint
	rg.POST("/ohlc", preMarketsController.GetOHLC)
	rg.GET("/userHistory", middleware.RequireAuth, preMarketsController.UserHistory)
	rg.GET("/userBalances", middleware.RequireAuth, preMarketsController.UserBalances)
	rg.GET("/marketData", preMarketsController.GetMarketData)
	rg.GET("/maxSlippage", preMarketsController.GetMaxSlippage)
	rg.GET("/prices", preMarketsController.GetPrices)
	rg.GET("/24HourHighLowPrices", preMarketsController.Get24HourHighLowPrices)
	// Admin Endpoints (enable in redis to use them - key - "PRE_MARKET_ADMIN_APIS")
	rg.POST("/admin/addProduct", middleware.RequireSequencer, preMarketsController.AddProduct)
	rg.POST("/admin/closeProduct", middleware.RequireSequencer, preMarketsController.CloseProduct)
	rg.POST("/admin/convertToSpot", middleware.RequireSequencer, preMarketsController.ConvertToSpot)
	// Internal monitoring endpoints
	rg.GET("/productWiseVolume", preMarketsController.GetProductIdWiseData)
	rg.GET("/premarketProductWiseVolume", preMarketsController.GetModifiedProductIdWiseData)
	rg.GET("/getOverallData", preMarketsController.GetOverallData)
	rg.GET("/getChartData", preMarketsController.GetPreMarketChartData)
	rg.GET("/downloadHistory", middleware.RequireAuth, preMarketsController.DownloadHistory)
}

/* Helper functions */
// X is our token and Y is the quote token

func FetchCurrentPrice(productId uint32) (*big.Int, error) {
	// We have 18 decimals precision and x is our base token and y is the quote token (stable coin)
	// Fetch x and y from redis
	var price *big.Int

	var poolData contractUtils.PreMarketPricingRedis
	poolDataJSON, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetPreMarketPricingKey(), xredis.GetPreMarketPricingField(productId)).Result()

	if err != nil {
		xlog.Errorf("Failed to fetch premarket pricing data from redis for product ID: %d", productId)
		return nil, err
	}

	if err := json.Unmarshal([]byte(poolDataJSON), &poolData); err != nil {
		xlog.Errorf("Failed to unmarshal premarket pricing data from redis for product ID: %d", productId)
		return nil, err
	}

	// Convert to bigint
	xPoolBigInt := new(big.Int)
	if _, success := xPoolBigInt.SetString(poolData.XReserve, 10); !success {
		xlog.Errorf("Failed to convert X reserve to big.Int for product ID: %d", productId)
		return nil, fmt.Errorf("invalid X reserve format")
	}

	yPoolBigInt := new(big.Int)
	if _, success := yPoolBigInt.SetString(poolData.YReserve, 10); !success {
		xlog.Errorf("Failed to convert Y reserve to big.Int for product ID: %d", productId)
		return nil, fmt.Errorf("invalid Y reserve format")
	}

	// Multiply yPoolBigInt by 10^18 to get the correct price
	yPoolBigInt.Mul(yPoolBigInt, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	// Calculate price
	price = new(big.Int).Div(yPoolBigInt, xPoolBigInt)
	return price, nil
}

func BuyTokens(productId uint32, amount *big.Int) (*big.Int, error) {
	// Fetch current pool reserves
	var amountX *big.Int
	_, err := xredis.WithRedisLock(xredis.GetPreMarketLockKey(productId), func() (*xredis.NOOP, error) {
		// Fetch current reserves
		var poolData contractUtils.PreMarketPricingRedis
		poolDataStr, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetPreMarketPricingKey(), xredis.GetPreMarketPricingField(productId)).Result()

		if err != nil {
			xlog.Errorf("Failed to fetch premarket pricing data from redis for product ID: %d", productId)
			return nil, err
		}

		if err := json.Unmarshal([]byte(poolDataStr), &poolData); err != nil {
			xlog.Errorf("Failed to unmarshal premarket pricing data from redis for product ID: %d", productId)
			return nil, err
		}

		// Convert to bigint
		xPoolBigInt := new(big.Int)
		if _, success := xPoolBigInt.SetString(poolData.XReserve, 10); !success {
			xlog.Errorf("Failed to convert X reserve to big.Int for product ID: %d", productId)
			return nil, fmt.Errorf("invalid X reserve format")
		}

		yPoolBigInt := new(big.Int)
		if _, success := yPoolBigInt.SetString(poolData.YReserve, 10); !success {
			xlog.Errorf("Failed to convert Y reserve to big.Int for product ID: %d", productId)
			return nil, fmt.Errorf("invalid Y reserve format")
		}

		newReserveY := new(big.Int).Add(yPoolBigInt, amount)
		newReserveX := new(big.Int).Div(new(big.Int).Mul(xPoolBigInt, yPoolBigInt), newReserveY)

		// Calculate output amount for X
		amountX = new(big.Int).Sub(xPoolBigInt, newReserveX)

		// Update the pool in redis
		poolData.XReserve = newReserveX.String()
		poolData.YReserve = newReserveY.String()

		poolDataJSON, err := json.Marshal(poolData)
		if err != nil {
			xlog.Errorf("Failed to marshal premarket pricing data for product ID: %d", productId)
			return nil, err
		}

		// Update the reserves in redis
		err2 := xredis.GetRedisClient().HSet(context.Background(), xredis.GetPreMarketPricingKey(), xredis.GetPreMarketPricingField(productId), poolDataJSON).Err()

		if err2 != nil {
			xlog.Errorf("Failed to update premarket pricing data in redis for product ID: %d", productId)
			return nil, err2
		}

		return &xredis.NOOP{}, nil
	})

	if err != nil {
		xlog.Errorf("Failed to acquire lock for pre-market pools: %v", err)
		return nil, err
	}

	return amountX, nil
}

func SellTokens(productId uint32, amount *big.Int) (*big.Int, error) {
	// Fetch current pool reserves
	var amountY *big.Int
	_, err := xredis.WithRedisLock(xredis.GetPreMarketLockKey(productId), func() (*xredis.NOOP, error) {
		// Fetch current reserves
		var poolData contractUtils.PreMarketPricingRedis
		poolDataStr, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetPreMarketPricingKey(), xredis.GetPreMarketPricingField(productId)).Result()

		if err != nil {
			xlog.Errorf("Failed to fetch premarket pricing data from redis for product ID: %d", productId)
			return nil, err
		}

		if err := json.Unmarshal([]byte(poolDataStr), &poolData); err != nil {
			xlog.Errorf("Failed to unmarshal premarket pricing data from redis for product ID: %d", productId)
			return nil, err
		}

		// Convert to bigint
		xPoolBigInt := new(big.Int)
		if _, success := xPoolBigInt.SetString(poolData.XReserve, 10); !success {
			xlog.Errorf("Failed to convert X reserve to big.Int for product ID: %d", productId)
			return nil, fmt.Errorf("invalid X reserve format")
		}

		yPoolBigInt := new(big.Int)
		if _, success := yPoolBigInt.SetString(poolData.YReserve, 10); !success {
			xlog.Errorf("Failed to convert Y reserve to big.Int for product ID: %d", productId)
			return nil, fmt.Errorf("invalid Y reserve format")
		}

		newReserveX := new(big.Int).Add(xPoolBigInt, amount)
		newReserveY := new(big.Int).Div(new(big.Int).Mul(xPoolBigInt, yPoolBigInt), newReserveX)

		// Calculate output amount for Y
		amountY = new(big.Int).Sub(yPoolBigInt, newReserveY)

		// Update the pool in redis
		poolData.XReserve = newReserveX.String()
		poolData.YReserve = newReserveY.String()

		poolDataJSON, err := json.Marshal(poolData)
		if err != nil {
			xlog.Errorf("Failed to marshal premarket pricing data for product ID: %d", productId)
			return nil, err
		}

		// Update the reserves
		err2 := xredis.GetRedisClient().HSet(context.Background(), xredis.GetPreMarketPricingKey(), xredis.GetPreMarketPricingField(productId), poolDataJSON).Err()

		if err2 != nil {
			xlog.Errorf("Failed to update premarket pricing data in redis for product ID: %d", productId)
			return nil, err2
		}

		return &xredis.NOOP{}, nil
	})

	if err != nil {
		xlog.Errorf("Failed to acquire lock for pre-market pools: %v", err)
		return nil, err
	}

	return amountY, nil
}

/*
	User APIs

1. Place pre-market order
2. Get user history
3. Get user balances for pre-market
4. Market Data for active tokens
5. Return max slippage
6. Return prices for all active tokens
*/
func (c *PreMarketsController) PlaceOrder(ctx *gin.Context) {
	var req contractUtils.PlacePreMarketOrderRequest

	// Bind request JSON and validate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		xlog.Errorf("Failed to bind request JSON for place pre-market order : %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Check if the product id is active and enabled
	isEnabled, closingTimestamp, err := c.preMarketsDb.IsProductEnabled(req.ProductId)
	if err != nil {
		xlog.Errorf("Failed to check if product is enabled : %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to check if product is enabled"})
		return
	}

	if !isEnabled {
		xlog.Infof("Product Id %d is not enabled", req.ProductId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Product is not enabled"})
		return
	}

	// Check if current timestamp is within the pre-market window

	// Get current timestamp
	currentTimestamp := time.Now().Unix()

	if currentTimestamp > closingTimestamp {
		xlog.Infof("Current timestamp %d is greater than closing timestamp %d", currentTimestamp, closingTimestamp)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Pre-market window is closed"})
		return
	}

	// Convert subaccount ID to hex
	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Place pre-market order, Invalid subaccount ID: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
		return
	}

	// Verify chain ID
	if req.ChainId != contractUtils.EndpointChainId() {
		xlog.Warnf("Place pre-market order - invalid chain ID: %d for subAccount: %s", req.ChainId, req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	// Get signature and signer address from headers
	signature := ctx.GetHeader("Logx-Signature")
	signerAddress := ctx.GetHeader("Logx-Signer-Address")
	if signature == "" || signerAddress == "" {
		xlog.Warnf("Place pre-market order - Missing signature or signer address in headers for subAccount: %s", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing signature or signer address"})
		return
	}

	// Convert token amount from string to big.Int
	amountBigIntVal := new(big.Int)
	if _, success := amountBigIntVal.SetString(req.Amount, 10); !success {
		xlog.Errorf("Place pre-market order Failed to convert token amount: %s to big.Int for subAccount: %s", req.Amount, req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to convert token amount to big.Int"})
		return
	}

	// Make sure token amount is greater than 0
	if amountBigIntVal.Cmp(big.NewInt(0)) <= 0 {
		xlog.Errorf("Place pre-market order Token amount is zero for subAccount: %s", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Token amount cannot be zero"})
		return
	}

	// Verify the signature
	if errVerf := contractUtils.VerifyPlacePreMarketOrder(req, signature, signerAddress); errVerf != nil {
		xlog.Warnf("Place pre-market order Signature verification failed for subAccount: %s, error: %v", req.SubAccountId, errVerf)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Signature verification failed: %v", errVerf)})
		return
	}

	// Increment transaction counter
	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Place pre-market order Failed to increment transaction counter for subAccount: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment transaction counter"})
		return
	}

	// Perform the action within a nonce lock
	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
		// Validate nonce
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			xlog.Warnf("Place pre-market order Nonce check failed for subAccount: %s, provided nonce: %d, currentNonce: %s", req.SubAccountId, req.Nonce, currentNonce)
			return &Response{status: http.StatusBadRequest, data: "Nonce check failed"}, nil
		}

		xlog.Infof("Placing pre-market order for subAccount: %s, product ID: %d, amount: %s, isBuy: %t", req.SubAccountId, req.ProductId, req.Amount, req.IsBuy)

		// If it's a buy order, check if the user has enough available margin, if sell, check if user has enough balance
		if req.IsBuy {
			// Check the available margin also
			availableMargin, err := c.balanceClient.GetAvailableMargin(subaccountHex)

			if err != nil {
				xlog.Errorf("Place pre-market order Failed to get available margin for subaccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to get available margin"}, err
			}

			// Check if the available margin is enough
			if availableMargin.Cmp(amountBigIntVal) < 0 {
				xlog.Errorf("Place pre-market order Insufficient margin for subAccount: %s, available margin: %s, quote delta: %s", req.SubAccountId, availableMargin.String(), amountBigIntVal.String())
				return &Response{status: http.StatusBadRequest, data: "Insufficient margin"}, nil
			}
		} else {
			// Check if the user has enough balance
			balance, err := c.balanceClient.GetPreMarketBalance(subaccountHex, req.ProductId)
			if err != nil {
				xlog.Errorf("Place pre-market order Failed to get balance for subaccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to get balance"}, err
			}

			// convert balance to big.Int
			balanceBigIntVal := new(big.Int)
			if _, success := balanceBigIntVal.SetString(balance, 10); !success {
				xlog.Errorf("Place pre-market order Failed to convert balance: %s to big.Int for subAccount: %s", balance, req.SubAccountId)
				return &Response{status: http.StatusBadRequest, data: "Unable to convert balance to big.Int"}, nil
			}

			// Check if the balance is enough
			if balanceBigIntVal.Cmp(amountBigIntVal) < 0 {
				xlog.Errorf("Place pre-market order Insufficient balance for subAccount: %s, balance: %s, quote delta: %s", req.SubAccountId, balanceBigIntVal.String(), amountBigIntVal.String())
				return &Response{status: http.StatusBadRequest, data: "Insufficient balance"}, nil
			}
		}

		// Deduct fees from the initial amount
		feesPercentageStr, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetPreMarketFeesKey(), xredis.GetPreMarketPricingField(req.ProductId)).Result()

		// If fees not found, set to default (0)
		if err == redis.Nil {
			feesPercentageStr = "0"
		}
		if err != nil && err != redis.Nil {
			xlog.Errorf("Place pre-market order: Failed to get fees percentage for subaccount: %s, productId: %d, error: %v", req.SubAccountId, req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to get fees percentage"}, err
		}

		// Parse fees percentage as an integer
		feesPercentage, ok := new(big.Int).SetString(feesPercentageStr, 10)
		if !ok {
			xlog.Errorf("Place pre-market order: Invalid fees percentage format for productId: %d, fees: %s", req.ProductId, feesPercentageStr)
			return &Response{status: http.StatusInternalServerError, data: "Invalid fees percentage format"}, err
		}

		// Fetch current price
		currentPrice, err := FetchCurrentPrice(req.ProductId)
		if err != nil {
			xlog.Errorf("Place pre-market order Failed to fetch current price for subAccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to fetch current price"}, err
		}

		var quoteDeltaBatchDb *big.Int
		var feesAmount *big.Int

		if req.IsBuy {
			// Calculate the fees amount
			feesAmount = new(big.Int).Mul(amountBigIntVal, feesPercentage)
			feesAmount.Div(feesAmount, new(big.Int).SetInt64(10000))

			// Deduct the fees from the initial amount
			amountBigIntVal.Sub(amountBigIntVal, feesAmount)
			// Buy tokens
			amountX, err := BuyTokens(req.ProductId, amountBigIntVal)
			if err != nil {
				xlog.Errorf("Place pre-market order Failed to buy tokens for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to buy tokens"}, err
			}

			// Deduct the amount and fees from the user's available margin
			combinedAmount := new(big.Int).Add(amountBigIntVal, feesAmount)

			// Add the entry to the database
			entry := db.PreMarketUserTable{
				SubaccountId: req.SubAccountId,
				ProductID:    req.ProductId,
				Amount:       ctypes.NewBigInt(amountBigIntVal),
				Fees:         ctypes.NewBigInt(feesAmount),
				Price:        ctypes.NewBigInt(currentPrice),
				QuoteDelta:   ctypes.NewBigInt(amountX),
				IsBuy:        true,
			}

			// Insert the entry into the database
			if _, err := c.preMarketsDb.InsertPreMarketOrder(entry); err != nil {
				xlog.Errorf("Place pre-market order Failed to insert order into database for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to insert order into database"}, err
			}

			status, err := c.balanceClient.UpdateTokenBalance(subaccountHex, contractUtils.QUOTE_TOKEN_PRODUCT_ID, combinedAmount.Neg(combinedAmount).String())
			if err != nil || !status {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			// Send the margin to premarket x account and fees to the fees account
			status1, err1 := c.balanceClient.UpdateTokenBalance(contractUtils.PRE_MARKETS_X_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, amountBigIntVal.String())
			if err1 != nil || !status1 {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status2, err2 := c.balanceClient.UpdateTokenBalance(contractUtils.PRE_MARKETS_FEES_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, feesAmount.String())
			if err2 != nil || !status2 {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			// Dedeuct from x account the pre-market tokens
			status3, err3 := c.balanceClient.UpdatePreMarketBalance(contractUtils.PRE_MARKETS_X_SUBACCOUNT_ID, req.ProductId, new(big.Int).Neg(amountX).String())
			if err3 != nil || !status3 {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			// Update the user pre-market balance
			status4, err4 := c.balanceClient.UpdatePreMarketBalance(subaccountHex, req.ProductId, amountX.String())
			if err4 != nil || !status4 {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			quoteDeltaBatchDb = amountX
		} else {
			// Sell tokens
			amountY, err := SellTokens(req.ProductId, amountBigIntVal)
			if err != nil {
				xlog.Errorf("Place pre-market order Failed to sell tokens for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to sell tokens"}, err
			}

			// Calculate the fees amount
			feesAmount = new(big.Int).Mul(amountY, feesPercentage)
			feesAmount.Div(feesAmount, new(big.Int).SetInt64(10000))

			// Deduct the fees from the final amount
			amountY.Sub(amountY, feesAmount)

			// Deduct the amount and fees from the user's balance
			combinedAmount := new(big.Int).Add(amountY, feesAmount)

			// Add the entry to the database
			entry := db.PreMarketUserTable{
				SubaccountId: req.SubAccountId,
				ProductID:    req.ProductId,
				Amount:       ctypes.NewBigInt(amountBigIntVal),
				Fees:         ctypes.NewBigInt(feesAmount),
				Price:        ctypes.NewBigInt(currentPrice),
				QuoteDelta:   ctypes.NewBigInt(amountY),
				IsBuy:        false,
			}

			// Insert the entry into the database
			if _, err := c.preMarketsDb.InsertPreMarketOrder(entry); err != nil {
				xlog.Errorf("Place pre-market order Failed to insert order into database for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to insert order into database"}, err
			}

			status, err := c.balanceClient.UpdatePreMarketBalance(subaccountHex, req.ProductId, new(big.Int).Neg(amountBigIntVal).String())
			if err != nil || !status {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			// Send the amount to the user's pre-market account
			status1, err1 := c.balanceClient.UpdatePreMarketBalance(contractUtils.PRE_MARKETS_X_SUBACCOUNT_ID, req.ProductId, amountBigIntVal.String())
			if err1 != nil || !status1 {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			// Send the fees to the fees account
			// Fees are quote tokens, so they go to the fees account's token balance (same as the buy path
			// and synthetic spot). They were previously credited to its pre-market ledger (behavior-spec H-4).
			status2, err2 := c.balanceClient.UpdateTokenBalance(contractUtils.PRE_MARKETS_FEES_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, feesAmount.String())
			if err2 != nil || !status2 {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status3, err3 := c.balanceClient.UpdateTokenBalance(contractUtils.PRE_MARKETS_X_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(combinedAmount).String())
			if err3 != nil || !status3 {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status4, err4 := c.balanceClient.UpdateTokenBalance(subaccountHex, contractUtils.QUOTE_TOKEN_PRODUCT_ID, amountY.String())
			if err4 != nil || !status4 {
				xlog.Errorf("Place pre-market order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			quoteDeltaBatchDb = amountY
		}

		// Increment nonce
		if err := services.IncrementNonce(subaccountHex, currentNonce); err != nil {
			xlog.Errorf("Place pre-market order Failed to increment nonce for subAccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to increment nonce"}, err
		}

		// Add to the batching db
		status2, err2 := contract.GlobalContracts.EndpointContract.PreMarketOrderRequest(req, quoteDeltaBatchDb, feesAmount, signature, transactionCounter)
		if err2 != nil || !status2 {
			xlog.Errorf("Place pre-market order Failed to add to batching db for subAccount: %s, error: %v", req.SubAccountId, err2)
			return &Response{status: http.StatusInternalServerError, data: "Failed to add to batching db"}, err2
		}

		xlog.Infof("Place pre-market order added to batching db successfully for subAccount: %s, amount: %s, productId: %d", req.SubAccountId, req.Amount, req.ProductId)

		return &Response{status: http.StatusOK, data: "Order placed successfully"}, nil
	})

	// Handle any errors that occurred within the nonce lock
	if err != nil {
		xlog.Errorf("Failed to place pre-market order for subAccount: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to place pre-market order, Internal Server Error"})
		return
	}

	// Respond with the response
	ctx.JSON(response.status, response.data)
}

// Add this new handler
func (c *PreMarketsController) GetOHLC(ctx *gin.Context) {
	var req GetOHLCRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		xlog.Errorf("Failed to bind request JSON for OHLC data: %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Validate time range
	if req.EndTime <= req.StartTime {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "End time must be greater than start time"})
		return
	}

	// Get OHLC data using the existing function
	candles, err := c.preMarketsCandlesDb.GetPreMarketCandles(req.ProductId, req.StartTime, req.EndTime, req.Interval)
	if err != nil {
		xlog.Errorf("Failed to fetch OHLC data for product %d: %v", req.ProductId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch OHLC data"})
		return
	}

	// Transform the candles into the desired format
	response := make([]CandleResponse, 0, len(candles))
	for _, candle := range candles {
		response = append(response, CandleResponse{
			T:  candle.StartTime,
			T2: candle.EndTime,
			I:  candle.Interval,
			O:  candle.OpenPrice,
			C:  candle.ClosePrice,
			H:  candle.HighPrice,
			L:  candle.LowPrice,
		})
	}

	if c.marketPricesWs != nil {
		liveCandle, ok := c.marketPricesWs.currentOHLC[req.Interval][req.ProductId]
		if ok && liveCandle != nil {
			// Only include if it intersects the requested time range
			if liveCandle.StartTime < req.EndTime && liveCandle.EndTime > req.StartTime {
				response = append(response, CandleResponse{
					T:  liveCandle.StartTime,
					T2: liveCandle.EndTime,
					I:  req.Interval,
					O:  liveCandle.Open,
					C:  liveCandle.Close,
					H:  liveCandle.High,
					L:  liveCandle.Low,
				})
			}
		}
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *PreMarketsController) UserHistory(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	userHistory, err := c.preMarketsDb.GetUserHistory(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Failed to get history for subaccount: %s, error: %v", currentSubaccount, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get user history")
		return
	}

	var history []map[string]interface{}

	for _, entry := range userHistory {
		history = append(history, map[string]interface{}{
			"timestamp":   entry.CreatedAt,
			"product_id":  entry.ProductID,
			"amount":      entry.Amount.String(),
			"fees":        entry.Fees.String(),
			"price":       entry.Price.String(),
			"quote_delta": entry.QuoteDelta.String(),
			"is_buy":      entry.IsBuy,
		})
	}

	// sory array in descending order
	sort.Slice(history, func(i, j int) bool {
		return history[i]["timestamp"].(time.Time).After(history[j]["timestamp"].(time.Time))
	})

	ctx.JSON(http.StatusOK, gin.H{"history": history})
}

func (c *PreMarketsController) UserBalances(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	// Convert subaccount ID to hex
	subaccountHex, err := cutils.HackySubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Place pre-market order, Invalid subaccount ID: %s, error: %v", currentSubaccount.ID, err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
		return
	}

	// Get the user's pre-market balances
	preMarketBalances, err := c.balanceClient.GetPreMarketBalances(subaccountHex)
	if err != nil {
		xlog.Errorf("Failed to get pre-market balances for subaccount: %s, error: %v", currentSubaccount, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get user balances")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"balances": preMarketBalances})
}

func (c *PreMarketsController) GetMarketData(ctx *gin.Context) {
	// Fetch all active pre-market products
	preMarketData, err := c.preMarketsDb.GetMarketData()
	if err != nil {
		xlog.Errorf("Failed to get pre-market data, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get pre-market data")
		return
	}

	// Fetch fees from the redis
	fees, err := xredis.GetRedisClient().HGetAll(context.Background(), xredis.GetPreMarketFeesKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get fees from redis, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get fees")
		return
	}

	// Prepare the response map where product_id is the key
	marketData := make(map[uint32]map[string]interface{})

	for _, entry := range preMarketData {
		productIDStr := fmt.Sprintf("%d", entry.ProductID)

		// Get fee for the current product ID
		fee, ok := fees[productIDStr]
		if !ok {
			fee = "0" // Default fee if not found in Redis
		}

		marketData[entry.ProductID] = map[string]interface{}{
			"closing_timestamp":  entry.ClosingTimestamp,
			"delivery_timestamp": entry.DeliveryTimestamp,
			"is_enabled":         entry.IsEnabled,
			"details":            entry.Details,
			"fee":                fee,
		}
	}

	ctx.JSON(http.StatusOK, marketData)
}

func (c *PreMarketsController) GetMaxSlippage(ctx *gin.Context) {
	// Get productId from query parameter
	productIdStr := ctx.Query("productId")
	if productIdStr == "" {
		xlog.Errorf("Missing productId in query parameters")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: productId is required as a query parameter"})
		return
	}

	// Parse productId into uint32
	productId, err := strconv.ParseUint(productIdStr, 10, 32)
	if err != nil {
		xlog.Errorf("Failed to parse productId as uint32: %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid productId format"})
		return
	}

	// Get current x and y reserve values from redis
	poolDataStr, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetPreMarketPricingKey(), xredis.GetPreMarketPricingField(uint32(productId))).Result()
	if err != nil {
		xlog.Errorf("Failed to get premarket pricing data from redis for product ID: %d, error: %v", uint32(productId), err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get premarket pricing data"})
		return
	}

	var poolData contractUtils.PreMarketPricingRedis

	if err := json.Unmarshal([]byte(poolDataStr), &poolData); err != nil {
		xlog.Errorf("Failed to unmarshal premarket pricing data from redis for product ID: %d, error: %v", uint32(productId), err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal premarket pricing data"})
		return
	}

	// return x and y as string
	ctx.JSON(http.StatusOK, gin.H{
		"x": poolData.XReserve,
		"y": poolData.YReserve,
	})
}

func (c *PreMarketsController) GetPrices(ctx *gin.Context) {
	// Fetch all unique product ids and get their prices
	uints, err := (&db.PreMarketsDB{}).GetUniqueProductIDs()
	if err != nil {
		xlog.Errorf("Failed to get unique product IDs, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get unique product IDs")
		return
	}

	// Prepare the response map where product_id is the key
	prices := make(map[uint32]string)
	// call fetchprice function to get price for each product id
	for _, productID := range uints {
		price, err := FetchCurrentPrice(productID)
		if err != nil {
			xlog.Errorf("Failed to get price for product ID: %d, error: %v", productID, err)
			cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get price")
			return
		}
		prices[productID] = price.String()
	}

	ctx.JSON(http.StatusOK, prices)
}

/* Admin APIs
1. Add pre-market product
2. Disable pre-market product
3. Convert to Spot by a factor
*/

func (c *PreMarketsController) AddProduct(ctx *gin.Context) {
	var req contractUtils.AddPreMarketProductRequest

	// Bind request JSON and validate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		xlog.Errorf("Failed to bind request JSON for add pre-market product: %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Check redis to see if the admin APIs are enabled
	// get from redis
	adminApisEnabled, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetPreMarketAdminKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get admin APIs status from redis, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get admin APIs status"})
		return
	}

	if adminApisEnabled != "1" {
		xlog.Errorf("Admin APIs are disabled")
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin APIs are disabled"})
		return
	}

	// Send the max supply to the pre-market account
	status, err := c.balanceClient.UpdatePreMarketBalance(contractUtils.PRE_MARKETS_X_SUBACCOUNT_ID, req.ProductId, req.MaxSupply)
	if err != nil || !status {
		xlog.Errorf("Failed to send max supply to the pre-market account, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to send max supply to the pre-market account"})
		return
	}

	// Initialise the pool and send the initial tokens to the pre-market account with redis lock
	_, err = xredis.WithRedisLock(xredis.GetPreMarketLockKey(req.ProductId), func() (*xredis.NOOP, error) {
		// Initialise the pool
		// xReserve would be equal to the max supply
		// yReserve would be equal to the starting price * max supply

		// Convert max supply to big.Int
		maxSupplyBigIntVal := new(big.Int)
		if _, success := maxSupplyBigIntVal.SetString(req.MaxSupply, 10); !success {
			xlog.Errorf("Failed to convert max supply to big.Int for product ID: %d", req.ProductId)
			return nil, fmt.Errorf("invalid max supply format")
		}
		// Convert starting price to big.Int
		startingPriceBigIntVal := new(big.Int)
		if _, success := startingPriceBigIntVal.SetString(req.StartingPrice, 10); !success {
			xlog.Errorf("Failed to convert starting price to big.Int for product ID: %d", req.ProductId)
			return nil, fmt.Errorf("invalid starting price format")
		}
		// Calculate yReserve
		yReserveBigIntVal := cutils.Divx18(new(big.Int).Mul(startingPriceBigIntVal, maxSupplyBigIntVal))

		// Create the pool data
		poolData := contractUtils.PreMarketPricingRedis{
			XReserve: maxSupplyBigIntVal.String(),
			YReserve: yReserveBigIntVal.String(),
		}

		// Marshal the pool data
		poolDataJSON, err := json.Marshal(poolData)
		if err != nil {
			xlog.Errorf("Failed to marshal premarket pricing data for product ID: %d", req.ProductId)
			return nil, err
		}

		// Set the pool data in redis
		err = xredis.GetRedisClient().HSet(context.Background(), xredis.GetPreMarketPricingKey(), xredis.GetPreMarketPricingField(req.ProductId), poolDataJSON).Err()
		if err != nil {
			xlog.Errorf("Failed to set premarket pricing data in redis for product ID: %d", req.ProductId)
			return nil, err
		}

		return &xredis.NOOP{}, nil
	})

	if err != nil {
		xlog.Errorf("Failed to acquire lock for pre-market pools: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to acquire lock for pre-market pools"})
		return
	}

	// convert to ctypes big.Int
	maxSupplyBigInt := ctypes.NewBigIntFromString(req.MaxSupply)
	startingPriceBigInt := ctypes.NewBigIntFromString(req.StartingPrice)

	// Convert ToBeDelivered to ctypes.BigInt if provided (i.e., not nil)
	var toBeDeliveredBigInt *ctypes.BigInt
	if req.ToBeDelivered != nil {
		toBeDelivered := ctypes.NewBigIntFromString(*req.ToBeDelivered)
		toBeDeliveredBigInt = &toBeDelivered
	} else {
		toBeDeliveredBigInt = nil
	}

	// Add the product to the database
	entry := db.PreMarketTable{
		ProductID:         req.ProductId,
		MaxSupply:         maxSupplyBigInt,
		ClosingTimestamp:  req.ClosingTimestamp,
		DeliveryTimestamp: req.DeliveryTimestamp, // Can handle nil values
		StartingPrice:     startingPriceBigInt,
		IsEnabled:         req.IsEnabled,
		ToBeDelivered:     toBeDeliveredBigInt, // Can handle nil values
		Details:           req.Details,
	}

	// Insert the new product into the database
	if _, err := c.preMarketsDb.AddPreMarket(entry); err != nil {
		xlog.Errorf("Failed to add pre-market product, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to add pre-market product"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "Product added successfully"})
}

func (c *PreMarketsController) CloseProduct(ctx *gin.Context) {
	var req contractUtils.ClosePreMarketProductRequest

	// Bind request JSON and validate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		xlog.Errorf("Failed to bind request JSON for close pre-market product: %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Check redis to see if the admin APIs are enabled
	// get from redis
	adminApisEnabled, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetPreMarketAdminKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get admin APIs status from redis, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get admin APIs status"})
		return
	}

	if adminApisEnabled != "1" {
		xlog.Errorf("Admin APIs are disabled")
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin APIs are disabled"})
		return
	}

	// Check if the product is enabled
	isEnabled, _, err := c.preMarketsDb.IsProductEnabled(req.ProductId)
	if err != nil {
		xlog.Errorf("Failed to check if product is enabled : %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to check if product is enabled"})
		return
	}

	if !isEnabled {
		xlog.Infof("Product Id %d is not enabled", req.ProductId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Product is not enabled"})
		return
	}

	// get pre-market balance of PRE_MARKETS_X_SUBACCOUNT_ID
	preMarketBalance, err := c.balanceClient.GetPreMarketBalance(contractUtils.PRE_MARKETS_X_SUBACCOUNT_ID, req.ProductId)
	if err != nil {
		xlog.Errorf("Failed to get pre-market balance for product ID: %d, error: %v", req.ProductId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pre-market balance"})
		return
	}

	// convert balance to ctypes big.Int
	preMarketBalanceBigInt := ctypes.NewBigIntFromString(preMarketBalance)

	// Close the product
	if err := c.preMarketsDb.CloseProduct(req.ProductId, preMarketBalanceBigInt); err != nil {
		xlog.Errorf("Failed to close pre-market product, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to close pre-market product"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "Product closed successfully"})
}

func (c *PreMarketsController) ConvertToSpot(ctx *gin.Context) {
	var req ConvertToSpotRequest

	// Bind request JSON and validate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		xlog.Errorf("Failed to bind request JSON for convert to spot: %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Check redis to see if the admin APIs are enabled
	// get from redis
	adminApisEnabled, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetPreMarketAdminKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get admin APIs status from redis, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get admin APIs status"})
		return
	}

	if adminApisEnabled != "1" {
		xlog.Errorf("Admin APIs are disabled")
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin APIs are disabled"})
		return
	}

	// Get the holdings of the pre-market product id
	subaccountWiseHoldings, err := c.preMarketsDb.GetHoldingsForProduct(req.ProductId)
	if err != nil {
		xlog.Errorf("Failed to get holdings for product ID: %d, error: %v", req.ProductId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get holdings"})
		return
	}

	// For all subaccount with holdings greater than 0, update spot balance by multiplying holdings with conversion factor
	for _, holding := range subaccountWiseHoldings {
		holdingBigIntVal := holding.Holdings.Val
		// if zero, continue
		if holdingBigIntVal.Cmp(new(big.Int)) == 0 {
			continue
		}
		// Convert conversion factor to big.Int
		factorBigInt := new(big.Int).SetInt64(int64(req.Factor))

		// Multiply holdings with conversion factor
		holdingBigIntVal.Mul(holdingBigIntVal, factorBigInt)
		// divide by 100
		holdingBigIntVal.Div(holdingBigIntVal, new(big.Int).SetInt64(100))

		// Convert subaccount ID to hex
		subaccountHex, err := cutils.HackySubaccountIdToHex(holding.SubaccountID)
		if err != nil {
			xlog.Errorf("Failed to convert subaccount ID to hex: %v", err)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert subaccount ID to hex"})
			return
		}

		// Update synthetic spot balance
		status, err := c.balanceClient.UpdateSyntheticSpotBalance(subaccountHex, req.ProductIdSpot, holdingBigIntVal.String())
		if err != nil || !status {
			xlog.Errorf("Failed to update synthetic spot balance for subaccount: %s, error: %v", holding.SubaccountID, err)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to update synthetic spot balance"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "Converted to spot successfully"})
}

// Write an api to get 24 hour high/low prices
func (c *PreMarketsController) Get24HourHighLowPrices(ctx *gin.Context) {
	// Fetch 24 hour high/low prices
	highLowPrices, err := c.preMarketsDb.Get24HourHighLowPrices()
	if err != nil {
		xlog.Errorf("Failed to get 24 hour high/low prices, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get 24 hour high/low prices")
		return
	}

	// Send the response
	ctx.JSON(http.StatusOK, highLowPrices)
}

/* Internal Dashboard APIs
1. Total buy, sell, volume, average price, fees and unique users product id wise.
3. Total and 24 hour data product level - Volume, fees, unique users and total usdc deposited.
4. Chart data - volume and and unique users - product level.
*/

func (c *PreMarketsController) GetProductIdWiseData(ctx *gin.Context) {
	// Fetch all unique product ids
	productDataBreakdown, err := c.preMarketsDb.GetProductWiseData()
	if err != nil {
		xlog.Errorf("Failed to get product ID wise data, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get product ID wise data")
		return
	}

	response := gin.H{
		"productWiseBreakDown": productDataBreakdown,
	}

	// Send the response
	ctx.JSON(http.StatusOK, response)
}

func (c *PreMarketsController) GetModifiedProductIdWiseData(ctx *gin.Context) {
	productDataBreakdown, err := c.preMarketsDb.GetProductWiseVolumeOnly()
	if err != nil {
		xlog.Errorf("Failed to get product ID wise data, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get product ID wise data")
		return
	}
	for _, data := range productDataBreakdown {
		if totalVolumeStr, exists := data["total_volume"]; exists {
			totalVolume := new(big.Int)
			totalVolume.SetString(totalVolumeStr, 10)
			multiplier := big.NewInt(35)
			scalingFactor := big.NewInt(10)
			modifiedVolume := new(big.Int).Mul(totalVolume, multiplier)
			modifiedVolume = modifiedVolume.Div(modifiedVolume, scalingFactor)
			data["premarket_total_volume"] = modifiedVolume.String()
			delete(data, "total_volume")
		}
	}
	response := gin.H{
		"productWiseBreakDown": productDataBreakdown,
	}
	ctx.JSON(http.StatusOK, response)
} // Move to stats controller

func (c *PreMarketsController) GetOverallData(ctx *gin.Context) {
	// Fetch total data
	overallStats, err := c.preMarketsDb.GetOverAllPreMarketsStats()
	if err != nil {
		xlog.Errorf("Failed to get overall pre-market stats, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get overall pre-market stats")
		return
	}

	// Send the response
	ctx.JSON(http.StatusOK, overallStats)
}

func (c *PreMarketsController) GetPreMarketChartData(ctx *gin.Context) {
	// Fetch chart data
	chartData, err := c.preMarketsDb.GetLast30DaysPreMarketStats()
	if err != nil {
		xlog.Errorf("Failed to get pre-market chart data, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get pre-market chart data")
		return
	}

	// Send the response
	ctx.JSON(http.StatusOK, chartData)
}

// DownloadHistory handles API request to download pre-market trading history
func (c *PreMarketsController) DownloadHistory(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	// Get trading history from the database (already sorted by timestamp)
	userHistory, err := c.preMarketsDb.GetUserHistorySorted(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Failed to get history for subaccount: %s, error: %v", currentSubaccount.ID, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get user history")
		return
	}

	// Get market data to use details instead of product IDs
	marketData, err := c.preMarketsDb.GetMarketData()
	if err != nil {
		xlog.Errorf("Failed to get market data: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get market data")
		return
	}

	// Create a map of product ID to details for quick lookup
	productDetails := make(map[uint32]string)
	for _, market := range marketData {
		productDetails[market.ProductID] = market.Details
	}

	// Create CSV data
	csvData := [][]string{
		{"Timestamp", "Product", "Side", "Amount", "Price", "Value in USD", "Fees"},
	}

	for _, entry := range userHistory {
		// Get product details from the map, fall back to ID and symbol if not found
		productInfo := fmt.Sprintf("Product-%d", entry.ProductID)
		if details, exists := productDetails[entry.ProductID]; exists && details != "" {
			productInfo = details
		} else if symbol, exists := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[entry.ProductID]; exists {
			productInfo = symbol
		}

		// Format timestamp
		timestamp := entry.CreatedAt.Format("2006-01-02 15:04:05")

		// Determine if buy or sell
		side := "SELL"
		if entry.IsBuy {
			side = "BUY"
		}

		// Convert BigInt values to human-readable format using X18ToFloatStr
		amount := cutils.X18ToFloatStr(entry.Amount.Val)
		price := cutils.X18ToFloatStr(entry.Price.Val)

		// Convert QuoteDelta to float and take absolute value
		quoteDeltaFloat := cutils.X18ToFloatStr(entry.QuoteDelta.Val)
		if quoteDeltaVal, parseErr := strconv.ParseFloat(quoteDeltaFloat, 64); parseErr == nil {
			quoteDeltaFloat = fmt.Sprintf("%.6f", math.Abs(quoteDeltaVal))
		}

		// Convert Fees to float and take absolute value
		feesFloat := cutils.X18ToFloatStr(entry.Fees.Val)
		if feesVal, parseErr := strconv.ParseFloat(feesFloat, 64); parseErr == nil {
			feesFloat = fmt.Sprintf("%.6f", math.Abs(feesVal))
		}

		csvData = append(csvData, []string{
			timestamp,
			productInfo,
			side,
			amount,
			price,
			quoteDeltaFloat,
			feesFloat,
		})
	}

	// Generate the CSV content
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	err = writer.WriteAll(csvData)
	if err != nil {
		xlog.Errorf("Failed to write CSV data: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to generate CSV file")
		return
	}
	writer.Flush()

	// Set appropriate headers for file download
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("pre_market_history_%s.csv", timestamp)
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	ctx.Header("Content-Type", "text/csv")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Expires", "0")
	ctx.Header("Cache-Control", "must-revalidate")
	ctx.Header("Pragma", "public")

	// Write the CSV content to the response
	ctx.Data(http.StatusOK, "text/csv", buffer.Bytes())
}
