package controller

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
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

type SyntheticSpotsController struct {
	balanceClient    *xclient.BalanceClient
	syntheticSpotsDb *db.SyntheticSpotDB
	preMarketDb      *db.PreMarketsDB
}

func RegisterSyntheticSpotsController(r *gin.RouterGroup) {
	syntheticSpotsController := &SyntheticSpotsController{
		balanceClient:    xclient.NewBalanceClient(),
		syntheticSpotsDb: &db.SyntheticSpotDB{},
		preMarketDb:      &db.PreMarketsDB{},
	}

	rg := r.Group("/syn-spot")

	//Endpoints
	rg.POST("/placeOrder", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, syntheticSpotsController.PlaceOrder)
	rg.GET("/userHistory", middleware.RequireAuth, syntheticSpotsController.UserHistory)
	rg.GET("/userBalances", middleware.RequireAuth, syntheticSpotsController.UserBalances)
	rg.GET("/marketData", syntheticSpotsController.GetMarketData)

	// Internal dashboard APIs
	rg.GET("/getProductWiseData", syntheticSpotsController.GetProductWiseData)
	rg.GET("/getOverallData", syntheticSpotsController.GetOverallData)
	rg.GET("/getChartData", syntheticSpotsController.GetChartsData)

	// New endpoint for downloading history
	rg.GET("/downloadHistory", middleware.RequireAuth, syntheticSpotsController.DownloadHistory)
}

func FetchPriceFromOracle(marketId uint32) (*big.Int, error) {
	// Initialize the app state
	appstateInstance := appstate.NewAppState()

	// Fetch oracle prices from the app state
	fetchExpiration := 500 * time.Millisecond
	oraclePricesMap, err := appstateInstance.GetAllOraclePrices(fetchExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to get app state: %v", err)
	}

	// Use marketId to get the symbol from the product ID map
	// Not removing _OSTRICH suffix here because in SyntheticSpotUserTable we will not be having those productIds
	symbol, ok := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[marketId]
	if !ok {
		return nil, fmt.Errorf("invalid market ID: %d", marketId)
	}

	// Fetch the price from the oraclePricesMap based on the symbol
	price, exists := oraclePricesMap[symbol]
	if !exists {
		return nil, fmt.Errorf("price not found for symbol: %s", symbol)
	}

	// Return the price as *big.Int
	return price.Pricex18, nil
}

func (c *SyntheticSpotsController) PlaceOrder(ctx *gin.Context) {
	var req contractUtils.PlacePreMarketOrderRequest

	// Bind request JSON and validate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		xlog.Errorf("Failed to bind request JSON for synthetic spot order : %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Check if the product is enabled or not
	isEnabledValue, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetSyntheticSpotsEnabledKey(), fmt.Sprintf("%d", req.ProductId)).Result()
	if err != nil {
		xlog.Errorf("Failed to fetch synthetic spots enabled status from Redis: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch syntheic spots status"})
		return
	}

	// Check if the value indicates synthetic spots are enabled
	if isEnabledValue != "1" {
		xlog.Infof("Synthetic spots are disabled as per Redis configuration")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Synthetic Spots are disabled as of now, please try again later"})
		return
	}

	// Convert subaccount ID to hex
	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Place synthetic spot order, Invalid subaccount ID: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
		return
	}

	// Verify chain ID
	if req.ChainId != contractUtils.EndpointChainId() {
		xlog.Warnf("Place synthetic spot order - invalid chain ID: %d for subAccount: %s", req.ChainId, req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	// Get signature and signer address from headers
	signature := ctx.GetHeader("Logx-Signature")
	signerAddress := ctx.GetHeader("Logx-Signer-Address")
	if signature == "" || signerAddress == "" {
		xlog.Warnf("Place synthetic spot order - Missing signature or signer address in headers for subAccount: %s", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing signature or signer address"})
		return
	}

	// Convert token amount from string to big.Int
	amountBigIntVal := new(big.Int)
	if _, success := amountBigIntVal.SetString(req.Amount, 10); !success {
		xlog.Errorf("Place synthetic spot order Failed to convert token amount: %s to big.Int for subAccount: %s", req.Amount, req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to convert token amount to big.Int"})
		return
	}

	// Make sure token amount is greater than 0
	if amountBigIntVal.Cmp(big.NewInt(0)) <= 0 {
		xlog.Errorf("Place synthetic spot order Token amount is less than or equal to zero for subAccount: %s", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Token amount cannot be less than or equal to zero"})
		return
	}

	// Verify the signature
	if errVerf := contractUtils.VerifyPlaceSyntheticSpotOrder(req, signature, signerAddress); errVerf != nil {
		xlog.Warnf("Place synthetic spot order Signature verification failed for subAccount: %s, error: %v", req.SubAccountId, errVerf)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Signature verification failed: %v", errVerf)})
		return
	}

	// Increment transaction counter
	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Place synthetic spot order Failed to increment transaction counter for subAccount: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment transaction counter"})
		return
	}

	// Perform the action within a nonce lock
	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
		// Validate nonce
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			xlog.Warnf("Place synthetic spot order Nonce check failed for subAccount: %s, provided nonce: %d, currentNonce: %s", req.SubAccountId, req.Nonce, currentNonce)
			return &Response{status: http.StatusBadRequest, data: "Nonce check failed"}, nil
		}

		xlog.Infof("Placing synthetic spot order for subAccount: %s, product ID: %d, amount: %s, isBuy: %t", req.SubAccountId, req.ProductId, req.Amount, req.IsBuy)

		// Get entry price from the oracle
		currentPrice, err := FetchPriceFromOracle(req.ProductId)
		if err != nil {
			xlog.Errorf("Place order - synthetic spot Failed to fetch oracle price for subaccount: %s, productId: %d, error: %v", req.SubAccountId, req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to fetch oracle price"}, err
		}

		// Check net position cap
		// Get the current net position from Redis using HGET
		netPositionStr, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetSyntheticSpotsNetPositionKey(), xredis.GetPreMarketPricingField(req.ProductId)).Result()
		if err != nil && err != redis.Nil {
			xlog.Errorf("Place synthetic spot order Failed to get net position from Redis for productId: %d, error: %v", req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to get net position"}, err
		}

		// If the key doesn't exist, initialize it to 0
		if err == redis.Nil {
			netPositionStr = "0"
		}

		// Convert net position to big.Int
		netPosition := new(big.Int)
		if _, success := netPosition.SetString(netPositionStr, 10); !success {
			xlog.Errorf("Place synthetic spot order Failed to convert net position: %s to big.Int for productId: %d", netPositionStr, req.ProductId)
			return &Response{status: http.StatusInternalServerError, data: "Failed to convert net position to big.Int"}, fmt.Errorf("invalid net position format")
		}

		// Get the position cap from Redis using HGET
		positionCapStr, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetSyntheticSpotsPositionCapKey(), xredis.GetPreMarketPricingField(req.ProductId)).Result()
		if err != nil && err != redis.Nil {
			xlog.Errorf("Place synthetic spot order Failed to get position cap from Redis for productId: %d, error: %v", req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to get position cap"}, err
		}

		// If the key doesn't exist, use a default value (10000 USD in this example)
		if err == redis.Nil {
			positionCapStr = "10000000000000000000000" // 10,000 * 10^18
		}

		// Convert position cap to big.Int
		positionCap := new(big.Int)
		if _, success := positionCap.SetString(positionCapStr, 10); !success {
			xlog.Errorf("Place synthetic spot order Failed to convert position cap: %s to big.Int for productId: %d", positionCapStr, req.ProductId)
			return &Response{status: http.StatusInternalServerError, data: "Failed to convert position cap to big.Int"}, fmt.Errorf("invalid position cap format")
		}

		// Calculate the USD value of the current net position
		netPositionUSD := new(big.Int).Mul(netPosition, currentPrice)
		netPositionUSD = cutils.Divx18(netPositionUSD) // Divide by 10^18 to get the USD value

		// Calculate the potential new position after this trade
		newPosition := new(big.Int).Set(netPosition)

		// For sell orders, we'll subtract amountBigIntVal from the net position immediately
		if !req.IsBuy {
			// For sell orders, we'll subtract amountBigIntVal from the net position
			newPosition.Sub(newPosition, amountBigIntVal)
		}

		// If it's a buy order, check if the user has enough available margin, if sell, check if user has enough balance
		if req.IsBuy {
			// Check the available margin also
			availableMargin, err := c.balanceClient.GetAvailableMargin(subaccountHex)

			if err != nil {
				xlog.Errorf("Place synthetic spot order Failed to get available margin for subaccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to get available margin"}, err
			}

			// Check if the available margin is enough
			if availableMargin.Cmp(amountBigIntVal) < 0 {
				xlog.Errorf("Place synthetic spot order Insufficient margin for subAccount: %s, available margin: %s, quote delta: %s", req.SubAccountId, availableMargin.String(), amountBigIntVal.String())
				return &Response{status: http.StatusBadRequest, data: "Insufficient margin"}, nil
			}
		} else {
			// Check if the user has enough balance
			balance, err := c.balanceClient.GetSyntheticSpotBalance(subaccountHex, req.ProductId)
			if err != nil {
				xlog.Errorf("Place synthetic spot order Failed to get balance for subaccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to get balance"}, err
			}

			// convert balance to big.Int
			balanceBigIntVal := new(big.Int)
			if _, success := balanceBigIntVal.SetString(balance, 10); !success {
				xlog.Errorf("Place synthetic spot order Failed to convert balance: %s to big.Int for subAccount: %s", balance, req.SubAccountId)
				return &Response{status: http.StatusBadRequest, data: "Unable to convert balance to big.Int"}, nil
			}

			// Check if the balance is enough
			if balanceBigIntVal.Cmp(amountBigIntVal) < 0 {
				xlog.Errorf("Place synthetic spot order Insufficient balance for subAccount: %s, balance: %s, quote delta: %s", req.SubAccountId, balanceBigIntVal.String(), amountBigIntVal.String())
				return &Response{status: http.StatusBadRequest, data: "Insufficient balance"}, nil
			}
		}

		// Deduct fees from the initial amount
		feesPercentageStr, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetSyntheticSpotsFeesKey(), xredis.GetPreMarketPricingField(req.ProductId)).Result()

		// If fees not found, set to default (0)
		if err == redis.Nil {
			feesPercentageStr = "0"
		}
		if err != nil && err != redis.Nil {
			xlog.Errorf("Place synthetic spot order: Failed to get fees percentage for subaccount: %s, productId: %d, error: %v", req.SubAccountId, req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to get fees percentage"}, err
		}

		// Parse fees percentage as an integer
		feesPercentage, ok := new(big.Int).SetString(feesPercentageStr, 10)
		if !ok {
			xlog.Errorf("Place synthetic spot order: Invalid fees percentage format for productId: %d, fees: %s", req.ProductId, feesPercentageStr)
			return &Response{status: http.StatusInternalServerError, data: "Invalid fees percentage format"}, err
		}

		// Fetch the slippage from the redis
		slippageStr, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetSyntheticSpotsSlippageKey(), xredis.GetPreMarketPricingField(req.ProductId)).Result()
		if err != nil {
			xlog.Errorf("Place synthetic spot order Failed to get slippage for subaccount: %s, productId: %d, error: %v", req.SubAccountId, req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to get slippage"}, err
		}

		// Change the price according to the slippage (increase the price by slippage percentage)
		slippage, ok := new(big.Int).SetString(slippageStr, 10)
		if !ok {
			xlog.Errorf("Place synthetic spot order Invalid slippage format for productId: %d, slippage: %s", req.ProductId, slippageStr)
			return &Response{status: http.StatusInternalServerError, data: "Invalid slippage format"}, err
		}

		// slippage is in basis points of the oracle price (50 = 0.5%): 10000 or more would make the
		// sell price zero or negative (the contract refuses such a quote and pauses the subaccount)
		if slippage.Sign() < 0 || slippage.Cmp(big.NewInt(10000)) >= 0 {
			xlog.Errorf("Place synthetic spot order: slippage %s bps out of range for productId %d", slippageStr, req.ProductId)
			return &Response{status: http.StatusInternalServerError, data: "Invalid slippage configuration"}, fmt.Errorf("slippage out of range")
		}
		newPrice := new(big.Int).Mul(currentPrice, slippage)
		newPrice.Div(newPrice, new(big.Int).SetInt64(10000))

		var quoteDeltaBatchDb *big.Int
		var feesAmount *big.Int

		if req.IsBuy {
			// Calculate the fees amount
			feesAmount = new(big.Int).Mul(amountBigIntVal, feesPercentage)
			feesAmount.Div(feesAmount, new(big.Int).SetInt64(10000))

			// Deduct the fees from the initial amount
			amountBigIntVal.Sub(amountBigIntVal, feesAmount)

			// Calculate the new price by multiplying the current price with the slippage and dividing by 10000 and adding the current price
			currentPrice.Add(currentPrice, newPrice)

			// Calculate the amountX by dividing the amount with the current price (first multiply the amount with 10^18)
			amountX := new(big.Int).Mul(amountBigIntVal, new(big.Int).SetInt64(1e18))
			amountX.Div(amountX, currentPrice)
			if amountX.Sign() <= 0 {
				return &Response{status: http.StatusBadRequest, data: "Order too small: rounds to zero tokens"}, nil
			}

			// For buy orders, update the position with amountX and check the position cap
			// For buy orders, we'll add amountX to the net position
			newPosition.Add(newPosition, amountX)

			// Calculate the USD value of the new position
			newPositionUSD := new(big.Int).Mul(newPosition, currentPrice)
			newPositionUSD = cutils.Divx18(newPositionUSD) // Divide by 10^18 to get the USD value

			// Check if the new position exceeds the cap
			if newPositionUSD.Cmp(positionCap) > 0 {
				xlog.Errorf("Place synthetic spot order Net position cap exceeded for productId: %d, current: %s, new: %s, cap: %s",
					req.ProductId, netPositionUSD.String(), newPositionUSD.String(), positionCap.String())
				return &Response{status: http.StatusBadRequest, data: "Net position cap exceeded"}, nil
			}

			// Deduct the amount and fees from the user's available margin
			combinedAmount := new(big.Int).Add(amountBigIntVal, feesAmount)

			// Add the entry to the database
			entry := db.SyntheticSpotUserTable{
				SubaccountId: req.SubAccountId,
				ProductID:    req.ProductId,
				Amount:       ctypes.NewBigInt(amountBigIntVal),
				Fees:         ctypes.NewBigInt(feesAmount),
				Price:        ctypes.NewBigInt(currentPrice),
				QuoteDelta:   ctypes.NewBigInt(amountX),
				IsBuy:        true,
			}

			// Insert the entry into the database
			if _, err := c.syntheticSpotsDb.InsertSyntheticSpotOrder(entry); err != nil {
				xlog.Errorf("Place  synthetic spot order Failed to insert order into database for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to insert order into database"}, err
			}

			status, err := c.balanceClient.UpdateTokenBalance(subaccountHex, contractUtils.QUOTE_TOKEN_PRODUCT_ID, combinedAmount.Neg(combinedAmount).String())
			if err != nil || !status {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			// Send the margin to x account and fees to the fees account
			status1, err1 := c.balanceClient.UpdateTokenBalance(contractUtils.SYN_SPOTS_X_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, amountBigIntVal.String())
			if err1 != nil || !status1 {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status2, err2 := c.balanceClient.UpdateTokenBalance(contractUtils.SYN_SPOTS_FEES_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, feesAmount.String())
			if err2 != nil || !status2 {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status3, err3 := c.balanceClient.UpdateSyntheticSpotBalance(contractUtils.SYN_SPOTS_X_SUBACCOUNT_ID, req.ProductId, new(big.Int).Neg(amountX).String())
			if err3 != nil || !status3 {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status4, err4 := c.balanceClient.UpdateSyntheticSpotBalance(subaccountHex, req.ProductId, amountX.String())
			if err4 != nil || !status4 {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			quoteDeltaBatchDb = amountX
		} else {
			// Get the slippage price (decrease the price by slippage percentage)
			currentPrice.Sub(currentPrice, newPrice)

			// sell token amount (amountY) = amount * currentPrice
			amountY := cutils.Divx18(new(big.Int).Mul(amountBigIntVal, currentPrice))

			// Calculate the fees amount
			feesAmount = new(big.Int).Mul(amountY, feesPercentage)
			feesAmount.Div(feesAmount, new(big.Int).SetInt64(10000))

			// Deduct the fees from the initial amount
			amountY.Sub(amountY, feesAmount)
			if amountY.Sign() <= 0 {
				return &Response{status: http.StatusBadRequest, data: "Order too small: nothing left after fees"}, nil
			}

			// Deduct the amount and fees from the user's balance
			combinedAmount := new(big.Int).Add(amountY, feesAmount)

			// Add the entry to the database
			entry := db.SyntheticSpotUserTable{
				SubaccountId: req.SubAccountId,
				ProductID:    req.ProductId,
				Amount:       ctypes.NewBigInt(amountBigIntVal),
				Fees:         ctypes.NewBigInt(feesAmount),
				Price:        ctypes.NewBigInt(currentPrice),
				QuoteDelta:   ctypes.NewBigInt(amountY),
				IsBuy:        false,
			}

			// Insert the entry into the database
			if _, err := c.syntheticSpotsDb.InsertSyntheticSpotOrder(entry); err != nil {
				xlog.Errorf("Place synthetic spot order Failed to insert order into database for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to insert order into database"}, err
			}

			status, err := c.balanceClient.UpdateSyntheticSpotBalance(subaccountHex, req.ProductId, new(big.Int).Neg(amountBigIntVal).String())
			if err != nil || !status {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status1, err1 := c.balanceClient.UpdateSyntheticSpotBalance(contractUtils.SYN_SPOTS_X_SUBACCOUNT_ID, req.ProductId, amountBigIntVal.String())
			if err1 != nil || !status1 {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			// Send the fees to the fees account
			status2, err2 := c.balanceClient.UpdateTokenBalance(contractUtils.SYN_SPOTS_FEES_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, feesAmount.String())
			if err2 != nil || !status2 {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status3, err3 := c.balanceClient.UpdateTokenBalance(contractUtils.SYN_SPOTS_X_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(combinedAmount).String())
			if err3 != nil || !status3 {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			status4, err4 := c.balanceClient.UpdateTokenBalance(subaccountHex, contractUtils.QUOTE_TOKEN_PRODUCT_ID, amountY.String())
			if err4 != nil || !status4 {
				xlog.Errorf("Place synthetic spot order Failed to update balance for subAccount: %s, error: %v", req.SubAccountId, err)
				return &Response{status: http.StatusInternalServerError, data: "Failed to update balance"}, err
			}

			quoteDeltaBatchDb = amountY
		}

		// Increment nonce
		if err := services.IncrementNonce(subaccountHex, currentNonce); err != nil {
			xlog.Errorf("Place synthetic spot order Failed to increment nonce for subAccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to increment nonce"}, err
		}

		// Add to the batching db
		status2, err2 := contract.GlobalContracts.EndpointContract.SyntheticSpotOrderRequest(req, quoteDeltaBatchDb, feesAmount, signature, transactionCounter)
		if err2 != nil || !status2 {
			xlog.Errorf("Place synthetic spot order Failed to add to batching db for subAccount: %s, error: %v", req.SubAccountId, err2)
			return &Response{status: http.StatusInternalServerError, data: "Failed to add to batching db"}, err2
		}

		xlog.Infof("Place synthetic spot order added to batching db successfully for subAccount: %s, amount: %s, productId: %d", req.SubAccountId, req.Amount, req.ProductId)

		// Update the net position in Redis using HSET
		_, err = xredis.GetRedisClient().HSet(context.Background(), xredis.GetSyntheticSpotsNetPositionKey(), xredis.GetPreMarketPricingField(req.ProductId), newPosition.String()).Result()
		if err != nil {
			xlog.Errorf("Place synthetic spot order Failed to update net position in Redis for productId: %d, error: %v", req.ProductId, err)
			// Send a discord notification with new position, position before transaction and error
			xclient.GlobalDiscordClient.SendWebhookMessageWithMentions(fmt.Sprintf("Redis update failure for net position for productId: %d, old net position: %s, new position: %s, error: %v", req.ProductId, netPosition.String(), newPosition.String(), err), true)
			// We don't want to fail the order if this update fails, just log the error
			xlog.Warnf("Continuing despite Redis update failure for net position")
		}

		return &Response{status: http.StatusOK, data: "Order placed successfully"}, nil
	})

	// Handle any errors that occurred within the nonce lock
	if err != nil {
		xlog.Errorf("Failed to place synthetic spot order for subAccount: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to place synthetic spot order, Internal Server Error"})
		return
	}

	// Respond with the response
	ctx.JSON(response.status, response.data)
}

func (c *SyntheticSpotsController) UserHistory(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	userHistory, err := c.syntheticSpotsDb.GetUserHistory(currentSubaccount.ID)
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

func (c *SyntheticSpotsController) UserBalances(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	// Convert subaccount ID to hex
	subaccountHex, err := cutils.HackySubaccountIdToHex(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Place synthetic spot order, Invalid subaccount ID: %s, error: %v", currentSubaccount.ID, err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
		return
	}

	synSpotBalances, err := c.balanceClient.GetSyntheticSpotBalances(subaccountHex)
	if err != nil {
		xlog.Errorf("Failed to get synthetic spot balances for subaccount: %s, error: %v", currentSubaccount, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get user balances")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"balances": synSpotBalances})
}

func (c *SyntheticSpotsController) GetMarketData(ctx *gin.Context) {
	// Get enabled status, fees and slippage from Redis
	enabledStatus, err := xredis.GetRedisClient().HGetAll(context.Background(), xredis.GetSyntheticSpotsEnabledKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get synthetic spots enabled status from Redis: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get synthetic spots enabled status"})
		return
	}

	fees, err := xredis.GetRedisClient().HGetAll(context.Background(), xredis.GetSyntheticSpotsFeesKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get synthetic spots fees from Redis: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get synthetic spots fees"})
		return
	}

	slippage, err := xredis.GetRedisClient().HGetAll(context.Background(), xredis.GetSyntheticSpotsSlippageKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get synthetic spots slippage from Redis: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get synthetic spots slippage"})
		return
	}

	// Categorise the data into a map (product ID -> enabled status, fees, slippage)
	marketData := make(map[string]map[string]interface{})
	// For enabled if value is 1 then make it true else false

	for key, value := range enabledStatus {
		marketData[key] = map[string]interface{}{
			"is_enabled": value == "1",
			"fees":       fees[key],
			"slippage":   slippage[key],
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"market_data": marketData})
}

// APIs for internal monitoring
// 1. Total and 24 hour - volume, fees, unique users, total buy volume, total sell volume and net open positions - productid wise
// 2. Total and 24 hour - volume, fees, unique users - overall
// 3. Charts for last 30 days total volume and total users
func (c *SyntheticSpotsController) GetProductWiseData(ctx *gin.Context) {
	stats, err := c.syntheticSpotsDb.GetSyntheticSpotStatsByProduct()
	if err != nil {
		xlog.Errorf("Failed to get synthetic spot stats by product: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get synthetic spot stats"})
		return
	}

	// multiply the net open positions by the current price
	for i := range stats {
		// we need to add tokens sold at the pre-market to the net open positions
		// to get the actual net open positions
		stat := &stats[i] // Take a reference to modify the slice element in place
		preMarketId := contractUtils.SPOT_TO_PRE_MARKET_MAP[stat.ProductID]
		// only add the pre-market token sold if we get back a valid pre-market ID
		finalUsdDeposited := big.NewInt(0)
		if preMarketId != 0 {
			premarketTokenSold, err := c.preMarketDb.GetTokensSoldForProductForSpot(preMarketId)
			if err != nil {
				xlog.Errorf("Failed to get pre-market token sold for product ID: %d, error: %v", stat.ProductID, err)
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pre-market token sold"})
				return
			}
			stat.NetOpenPositions.Val.Add(stat.NetOpenPositions.Val, &premarketTokenSold)

			// Get final usd deposited in the pre-market for pnl calculation
			val, err := c.preMarketDb.GetFinalUsdDepositedForProduct(preMarketId)
			if err != nil {
				xlog.Errorf("Failed to get final USD deposited for product ID: %d, error: %v", stat.ProductID, err)
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get final USD deposited"})
				return
			}
			finalUsdDeposited.Set(&val)
		}

		// Multiply the net open positions by the current price
		if stat.NetOpenPositions.Val.Cmp(big.NewInt(0)) != 0 {
			price, err := FetchPriceFromOracle(stat.ProductID)
			if err != nil {
				xlog.Errorf("Failed to fetch oracle price for product ID: %d, error: %v", stat.ProductID, err)
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch oracle price"})
				return
			}

			stat.NetOpenPositions.Val.Mul(stat.NetOpenPositions.Val, price)
			stat.NetOpenPositions.Val.Div(stat.NetOpenPositions.Val, big.NewInt(1e18))
		}

		stat.NetPnl.Val = big.NewInt(0)

		// Start with finalUsdDeposited if you want to include that
		stat.NetPnl.Val.Add(stat.NetPnl.Val, finalUsdDeposited)

		// Add your (boughtVolume - soldVolume)
		stat.NetPnl.Val.Add(stat.NetPnl.Val, stat.BoughtVolumeUSD.Val)
		stat.NetPnl.Val.Sub(stat.NetPnl.Val, stat.SoldVolumeUSD.Val)

		// Subtract netOpenPositions (the cost or value of current positions)
		stat.NetPnl.Val.Sub(stat.NetPnl.Val, stat.NetOpenPositions.Val)
	}

	ctx.JSON(http.StatusOK, gin.H{"stats": stats})
}

// 2. Total and 24 hour - volume, fees, unique users - overall
func (c *SyntheticSpotsController) GetOverallData(ctx *gin.Context) {
	// Get overall stats
	overallStats, err := c.syntheticSpotsDb.GetOverallSyntheticSpotStats()
	if err != nil {
		xlog.Errorf("Failed to get overall synthetic spot stats: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get overall synthetic spot stats"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"stats": overallStats})
}

// 3. Charts for last 30 days total volume and total users
func (c *SyntheticSpotsController) GetChartsData(ctx *gin.Context) {
	// Get charts data
	chartsData, err := c.syntheticSpotsDb.GetLast30DaysSyntheticSpotStats()
	if err != nil {
		xlog.Errorf("Failed to get synthetic spot charts data: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get synthetic spot charts data"})
		return
	}

	ctx.JSON(http.StatusOK, chartsData)
}

// DownloadHistory handles API request to download synthetic spot trading history
func (c *SyntheticSpotsController) DownloadHistory(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	// Get trading history from the database (already sorted by timestamp)
	userHistory, err := c.syntheticSpotsDb.GetUserHistorySorted(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Failed to get history for subaccount: %s, error: %v", currentSubaccount.ID, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get user history")
		return
	}

	// Create CSV data
	csvData := [][]string{
		{"Timestamp", "Product", "Side", "Amount", "Price", "Value in USD", "Fees"},
	}

	for _, entry := range userHistory {
		// Get product symbol using the map
		productSymbol, exists := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[entry.ProductID]
		if !exists {
			productSymbol = fmt.Sprintf("Product-%d", entry.ProductID)
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
		valueFloat := cutils.X18ToFloatStr(entry.QuoteDelta.Val)
		if valueVal, parseErr := strconv.ParseFloat(valueFloat, 64); parseErr == nil {
			valueFloat = fmt.Sprintf("%.6f", math.Abs(valueVal))
		}

		// Convert Fees to float and take absolute value
		feesFloat := cutils.X18ToFloatStr(entry.Fees.Val)
		if feesVal, parseErr := strconv.ParseFloat(feesFloat, 64); parseErr == nil {
			feesFloat = fmt.Sprintf("%.6f", math.Abs(feesVal))
		}

		csvData = append(csvData, []string{
			timestamp,
			productSymbol,
			side,
			amount,
			price,
			valueFloat,
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
	filename := fmt.Sprintf("synthetic_spot_history_%s.csv", timestamp)

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
