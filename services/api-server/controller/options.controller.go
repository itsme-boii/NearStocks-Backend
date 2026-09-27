package controller

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type OptionsController struct {
	balanceClient *xclient.BalanceClient
	optionsDb     *db.OptionsDB
}

func RegisterOptionsController(r *gin.RouterGroup) {
	optionsController := &OptionsController{
		balanceClient: xclient.GlobalBalanceClient,
		optionsDb:     &db.OptionsDB{},
	}

	rg := r.Group("/options")

	//Endpoints
	rg.POST("/placeBet", middleware.RequireAuth, middleware.RequireSequencer, middleware.RequireSessionKey, optionsController.PlaceBet)
	rg.POST("closeBetCronOnly", optionsController.CloseBetCronOnly)
	rg.GET("/positionsAndHistory", middleware.RequireAuth, optionsController.PositionsAndHistory)
	rg.GET("/marketData", optionsController.GetMarketData)
	rg.GET("/exposure", optionsController.GetExposure)
	rg.GET("/historicalData", optionsController.GetHistoricalData)
	rg.GET("/tradesBreakdown", optionsController.GetTradesBreakdown)
	rg.GET("/24HourStats", optionsController.Get24HourStats)
	rg.GET("/chartData", optionsController.Get30DayStats)
	rg.GET("/downloadHistory", middleware.RequireAuth, optionsController.DownloadHistory)
}

func FetchOraclePrice(marketId uint32) (*big.Int, error) {
	// Initialize the app state
	appstateInstance := appstate.NewAppState()

	// Fetch oracle prices from the app state
	fetchExpiration := 500 * time.Millisecond
	oraclePricesMap, err := appstateInstance.GetAllOraclePrices(fetchExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to get app state: %v", err)
	}

	symbol, ok := marketutils.GetBaseSymbolForProduct(marketId)
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

func (oc *OptionsController) PlaceBet(ctx *gin.Context) {
	var req contractUtils.UserOptionBet

	// Bind request JSON and validate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		xlog.Errorf("Failed to bind request JSON for place bet - options : %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Check if options are disabled by getting from redis instead
	isEnabledValue, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetOptionsEnabledKey(), fmt.Sprintf("%d", req.ProductId)).Result()
	if err != nil {
		xlog.Errorf("Failed to fetch options enabled status from Redis: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch options status"})
		return
	}

	// Check if the value indicates options are disabled
	if isEnabledValue == "0" {
		xlog.Infof("Options are disabled as per Redis configuration")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Options are disabled as of now, please try again later"})
		return
	}

	// Check if the interval passed is among the allowed intervals
	if !contractUtils.OPTIONS_INTERVALS[req.Interval] {
		xlog.Warnf("Place bet - options Invalid interval: %d for subAccount: %s", req.Interval, req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid interval"})
		return
	}

	// Convert subaccount ID to hex
	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubAccountId)
	if err != nil {
		xlog.Errorf("Place bet - options, Invalid subaccount ID: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
		return
	}

	// Verify chain ID
	if req.ChainId != contractUtils.EndpointChainId() {
		xlog.Warnf("Place bet - options chain ID: %d for subAccount: %s", req.ChainId, req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid chain ID"})
		return
	}

	// Get signature and signer address from headers
	signature := ctx.GetHeader("Logx-Signature")
	signerAddress := ctx.GetHeader("Logx-Signer-Address")
	if signature == "" || signerAddress == "" {
		xlog.Warnf("Place bet - options Missing signature or signer address in headers for subAccount: %s", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing signature or signer address"})
		return
	}

	// Convert token amount from string to big.Int
	amountBigIntVal := new(big.Int)
	if _, success := amountBigIntVal.SetString(req.Amount, 10); !success {
		xlog.Errorf("Place bet - options Failed to convert token amount: %s to big.Int for subAccount: %s", req.Amount, req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to convert token amount to big.Int"})
		return
	}

	// Make sure token amount is not zero
	if amountBigIntVal.Sign() == 0 {
		xlog.Errorf("Place bet - options Token amount is zero for subAccount: %s", req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Token amount cannot be zero"})
		return
	}

	// Get entry price from the oracle
	entryPrice, err := FetchOraclePrice(req.ProductId)
	if err != nil || entryPrice == nil {
		xlog.Errorf("Place bet - options Failed to fetch oracle price for subaccount: %s, productId: %d, error: %v, oraclePrice: %+v", req.SubAccountId, req.ProductId, err, entryPrice)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch oracle price"})
	}

	quoteDelta := cutils.Divx18(new(big.Int).Mul(amountBigIntVal, entryPrice))
	quoteDeltAbs := new(big.Int).Abs(quoteDelta)

	// absoulte value of quote delta should be between minimum and maximum (fetch from redis)
	minimumQuote, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetOptionsMinimumAmountKey()).Result()
	if err != nil {
		xlog.Errorf("Place bet - options Failed to get minimum quote amount from Redis for subaccount: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get minimum quote amount"})
		return
	}

	maximumQuote, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetOptionsMaximumAmountKey()).Result()
	if err != nil {
		xlog.Errorf("Place bet - options Failed to get maximum quote amount from Redis for subaccount: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get maximum quote amount"})
		return
	}

	minQuote, ok := new(big.Int).SetString(minimumQuote, 10)
	if !ok {
		xlog.Errorf("Failed to parse minimumQuote: %s", minimumQuote)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid minimum quote value"})
		return
	}

	maxQuote, ok := new(big.Int).SetString(maximumQuote, 10)
	if !ok {
		xlog.Errorf("Failed to parse maximumQuote: %s", maximumQuote)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid maximum quote value"})
		return
	}

	// Check abs(quote Delta) with these values
	if quoteDeltAbs.Cmp(minQuote) < 0 || quoteDeltAbs.Cmp(maxQuote) > 0 {
		xlog.Errorf("Place bet - options Quote delta: %s is not within the allowed range for subAccount: %s", quoteDeltAbs.String(), req.SubAccountId)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Quote delta is not within the allowed range"})
		return
	}

	// Verify the place option bet using signature
	if errVerf := contractUtils.VerifyPlaceOptionsBet(req, signature, signerAddress); errVerf != nil {
		xlog.Warnf("Place bet - options Signature verification failed for subAccount: %s, error: %v", req.SubAccountId, errVerf)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Signature verification failed: %v", errVerf)})
		return
	}

	// Increment transaction counter
	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Place bet - options Failed to increment transaction counter for subAccount: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment transaction counter"})
		return
	}

	// Perform the action within a nonce lock
	response, err := services.WithNonceRedisLock(subaccountHex, func(currentNonce string) (*Response, error) {
		// Validate nonce
		nonceCheck := cutils.CheckNonce(req.Nonce, currentNonce)
		if !nonceCheck {
			xlog.Warnf("Place bet - options Nonce check failed for subAccount: %s, provided nonce: %d, currentNonce: %s", req.SubAccountId, req.Nonce, currentNonce)
			return &Response{status: http.StatusBadRequest, data: "Nonce check failed"}, nil
		}

		xlog.Infof("Placing option bet for subAccount: %s, amount: %s, productId: %d", req.SubAccountId, req.Amount, req.ProductId)

		// Check the available margin also
		availableMargin, err := oc.balanceClient.GetAvailableMargin(subaccountHex)

		if err != nil {
			xlog.Errorf("Place bet - options Failed to get available margin for subaccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to get available margin"}, err
		}

		// Check if the available margin is enough
		if availableMargin.Cmp(quoteDeltAbs) < 0 {
			xlog.Errorf("Place bet - options Insufficient margin for subAccount: %s, available margin: %s, quote delta: %s", req.SubAccountId, availableMargin.String(), quoteDeltAbs.String())
			return &Response{status: http.StatusBadRequest, data: "Insufficient margin"}, nil
		}

		// Add the option bet to the database
		var placeBetEntry db.OptionsTable

		// We need to get entry time, payout% and fees% when placing a bet

		// Get entry time
		entryTime := time.Now().Unix()

		// Get payout and fees percentage
		payout, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetOptionsPayoutKey(), xredis.GetOptionsProductIDField(req.ProductId, req.Interval)).Result()
		if err != nil {
			xlog.Errorf("Place bet - options Failed to get payout percentage for subaccount: %s, productId: %d, error: %v", req.SubAccountId, req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to get payout percentage"}, err
		}

		fees, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetOptionsFeesKey(), xredis.GetOptionsProductIDField(req.ProductId, req.Interval)).Result()
		// if fees not found, get the default (0) fees
		if err == redis.Nil {
			fees = "0"
		}
		if err != nil && err != redis.Nil {
			xlog.Errorf("Place bet - options Failed to get fees percentage for subaccount: %s, productId: %d, error: %v", req.SubAccountId, req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to get fees percentage"}, err
		}

		// convert to ctype bigint for db
		amountBigInt := ctypes.NewBigInt(amountBigIntVal)
		entryPriceBigInt := ctypes.NewBigInt(entryPrice)
		quoteDeltaBigInt := ctypes.NewBigInt(quoteDelta)
		// covert to uint8 from string
		payoutUint32, err := strconv.ParseUint(payout, 10, 32)
		if err != nil {
			xlog.Errorf("Place bet - options Failed to convert payout percentage to uint32 for subaccount: %s, productId: %d, error: %v", req.SubAccountId, req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to convert payout percentage"}, err
		}
		feesUint32, err := strconv.ParseUint(fees, 10, 32)
		if err != nil {
			xlog.Errorf("Place bet - options Failed to convert fees percentage to uint32 for subaccount: %s, productId: %d, error: %v", req.SubAccountId, req.ProductId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to convert fees percentage"}, err
		}

		placeBetEntry = db.OptionsTable{
			SubaccountId: req.SubAccountId,
			ProductId:    req.ProductId,
			Amount:       amountBigInt,
			Interval:     req.Interval,
			EntryPrice:   entryPriceBigInt,
			EntryTime:    entryTime,
			Payout:       uint32(payoutUint32),
			Fees:         uint32(feesUint32),
			QuoteDelta:   &quoteDeltaBigInt,
		}

		// Add the entry to the database
		optionsEntry, err := oc.optionsDb.InsertOptionBet(placeBetEntry)
		if err != nil {
			xlog.Errorf("Place bet - options Failed to insert option bet into the database for subAccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to insert option bet"}, err
		}

		// Write the option to the redis along with the id, convert interval to seconds
		closeTimestamp := entryTime + int64(req.Interval)*60

		placeBetRedis := contractUtils.PlaceOptionBetRedis{
			OrderId:      optionsEntry.ID,
			SubaccountId: req.SubAccountId,
			ProductId:    req.ProductId,
			QuoteDelta:   quoteDelta.String(),
			EntryPrice:   entryPriceBigInt.String(),
			Payout:       uint32(payoutUint32),
			Fees:         uint32(feesUint32),
		}

		// Marshal the struct to JSON
		placeBetRedisJSON, err := json.Marshal(placeBetRedis)
		if err != nil {
			xlog.Errorf("Place bet - options Failed to marshal option bet to JSON for subAccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to marshal option bet to JSON"}, err
		}
		// Add the job to the sorted set with closeTimestamp as the score
		zAddResult := xredis.GetRedisClient().ZAdd(context.Background(), xredis.GetOptionsCloseJobsKey(), redis.Z{Score: float64(closeTimestamp), Member: placeBetRedisJSON})
		if err := zAddResult.Err(); err != nil {
			xlog.Errorf("Place bet - options Failed to add job to redis for subAccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to add job to redis"}, err
		}

		// Change balances in balance server
		balanceToChange := new(big.Int).Neg(new(big.Int).Abs(quoteDelta))
		status, err := oc.balanceClient.UpdateTokenBalance(subaccountHex, contractUtils.QUOTE_TOKEN_PRODUCT_ID, balanceToChange.String())
		xlog.Infof("Subaccount balance change status: %v, error: %v", status, err)
		if err != nil || !status {
			xlog.Errorf("Place bet - options Failed to update token balance for subAccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to update token balance for the user"}, err
		}
		// Update options balance
		status2, err2 := oc.balanceClient.UpdateTokenBalance(contractUtils.OPTIONS_X_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, balanceToChange.Abs(balanceToChange).String())
		if err2 != nil || !status2 {
			xlog.Errorf("Place bet - options Failed to update token balance (OPTIONS ACCOUNT) for subAccount: %s, error: %v", req.SubAccountId, err2)
			return &Response{status: http.StatusInternalServerError, data: "Failed to update options subaccount balance"}, err2
		}

		xlog.Infof("Option bet balance changes done successfully for subAccount: %s, amount: %s, productId: %d", req.SubAccountId, req.Amount, req.ProductId)

		// Increment nonce
		if err := services.IncrementNonce(subaccountHex, currentNonce); err != nil {
			xlog.Errorf("Place bet - options Failed to increment nonce for subAccount: %s, error: %v", req.SubAccountId, err)
			return &Response{status: http.StatusInternalServerError, data: "Failed to increment nonce"}, err
		}

		// Add to the batching db
		status3, err3 := contract.GlobalContracts.EndpointContract.PlaceOptionBet(req, amountBigIntVal, entryPrice, optionsEntry.ID, uint32(payoutUint32), uint32(feesUint32), signature, transactionCounter)
		if err3 != nil || !status3 {
			xlog.Errorf("Place bet - options Failed to add to batching db for subAccount: %s, error: %v", req.SubAccountId, err3)
			return &Response{status: http.StatusInternalServerError, data: "Failed to add to batching db"}, err3
		}

		xlog.Infof("Option bet added to batching db successfully for subAccount: %s, amount: %s, productId: %d", req.SubAccountId, req.Amount, req.ProductId)

		return &Response{status: http.StatusOK, data: "Option bet placed successfully"}, nil
	})

	// Handle any errors that occurred within the nonce lock
	if err != nil {
		xlog.Errorf("Place bet - options Failed to place option bet for subAccount: %s, error: %v", req.SubAccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to place option bet, Internal Server Error"})
		return
	}

	// Respond with the response
	ctx.JSON(response.status, response.data)
}

func (oc *OptionsController) CloseBetCronOnly(ctx *gin.Context) {
	var req contractUtils.PlaceOptionBetRedis

	// Bind request JSON and validate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		xlog.Errorf("Failed to bind request JSON for close bet - options : %v", err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request: unable to bind JSON"})
		return
	}

	// Convert subaccount ID to hex
	subaccountHex, err := cutils.HackySubaccountIdToHex(req.SubaccountId)
	if err != nil {
		xlog.Errorf("Close bet - options: Invalid subaccount ID: %s, error: %v", req.SubaccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("invalid subaccount ID: %s, error: %v", req.SubaccountId, err)})
		return
	}

	// Increment transaction counter
	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Close bet - options: Failed to increment transaction counter for subAccount: %s, error: %v", req.SubaccountId, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to increment transaction counter for subAccount: %s, error: %v", req.SubaccountId, err)})
		return
	}

	// Use Redis lock to prevent race conditions
	response, errLock := xredis.WithRedisLock(xredis.GetOptionBetLockKey(req.OrderId), func() (*Response, error) {
		// First check if the bet exists and if it's already closed
		existingBet, err := oc.optionsDb.GetOptionBetById(req.OrderId)
		if err != nil {
			return &Response{status: http.StatusInternalServerError, data: "Failed to find bet with id"}, err
		}

		// If ExitPrice is not nil, the bet has already been closed
		if existingBet.ExitPrice != nil {
			xlog.Infof("Close bet - options, Bet with ID: %d has already been closed", req.OrderId)
			return &Response{status: http.StatusOK, data: "Bet already closed"}, nil
		}

		// Get current oracle price
		exitPrice, err := FetchOraclePrice(req.ProductId)
		if err != nil {
			return &Response{status: http.StatusInternalServerError, data: "Failed to fetch oracle price"}, err
		}

		// Check sign of quote delta
		quoteDeltaBigInt := new(big.Int)
		if _, success := quoteDeltaBigInt.SetString(req.QuoteDelta, 10); !success {
			return &Response{status: http.StatusInternalServerError, data: "Failed to convert quote delta"}, fmt.Errorf("failed to convert quote delta")
		}

		// convert entry price to big int
		entryPriceBigInt := new(big.Int)
		if _, success := entryPriceBigInt.SetString(req.EntryPrice, 10); !success {
			return &Response{status: http.StatusInternalServerError, data: "Failed to convert entry price"}, fmt.Errorf("failed to convert entry price")
		}

		userWin := false
		if quoteDeltaBigInt.Cmp(big.NewInt(0)) > 0 {
			userWin = exitPrice.Cmp(entryPriceBigInt) > 0
		} else {
			userWin = exitPrice.Cmp(entryPriceBigInt) < 0
		}

		// Calculate delta
		var payoutAmount *big.Int
		var feesAmount *big.Int
		if userWin {
			payoutAmount = new(big.Int).Div(new(big.Int).Mul(quoteDeltaBigInt.Abs(quoteDeltaBigInt), big.NewInt(int64(req.Payout))), big.NewInt(100))
			feesAmount = new(big.Int).Div(new(big.Int).Mul(quoteDeltaBigInt.Abs(quoteDeltaBigInt), big.NewInt(int64(req.Fees))), big.NewInt(100))
		} else {
			payoutAmount = big.NewInt(0)
			feesAmount = big.NewInt(0)
		}

		// add exit details to the db
		exitPriceBigInt := ctypes.NewBigInt(exitPrice)
		payoutAmountBigInt := ctypes.NewBigInt(payoutAmount.Abs(payoutAmount))
		feesAmountBigInt := ctypes.NewBigInt(feesAmount)
		// userpnl - subtract abs(quoteDelta) from abs(payoutAmount)
		userPnl := ctypes.NewBigInt(new(big.Int).Sub(payoutAmount.Abs(payoutAmount), quoteDeltaBigInt.Abs(quoteDeltaBigInt)))

		// Update the database
		_, err = oc.optionsDb.CloseOptionBetById(req.OrderId, &exitPriceBigInt, &payoutAmountBigInt, &feesAmountBigInt, &userPnl)
		if err != nil {
			return &Response{status: http.StatusInternalServerError, data: "Failed to add close option bet to the database"}, err
		}

		// Function to revert DB changes if Redis operations fail
		rollbackDbChanges := func() error {
			// Set all fields back to nil to revert to open state
			nullExitPrice := (*ctypes.BigInt)(nil)
			nullPayoutAmount := (*ctypes.BigInt)(nil)
			nullFeesAmount := (*ctypes.BigInt)(nil)
			nullUserPnl := (*ctypes.BigInt)(nil)

			// Revert to original state (which had these fields as nil)
			_, revertErr := oc.optionsDb.CloseOptionBetById(req.OrderId, nullExitPrice, nullPayoutAmount, nullFeesAmount, nullUserPnl)
			if revertErr != nil {
				xlog.Errorf("Failed to rollback DB changes for bet ID: %d, error: %v", req.OrderId, revertErr)
				return revertErr
			}
			return nil
		}

		if userWin {
			//subtract fees from payout and assign to combine delta
			combinedDelta := new(big.Int).Sub(payoutAmount, feesAmount)

			// Change balances in balance server

			// deduct from options account
			status2, err2 := oc.balanceClient.UpdateTokenBalance(contractUtils.OPTIONS_X_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, payoutAmount.Neg(payoutAmount).String())
			if err2 != nil || !status2 {
				rollbackErr := rollbackDbChanges()
				if rollbackErr != nil {
					xlog.Errorf("Failed to rollback after options account update failure: %v", rollbackErr)
				}
				return &Response{status: http.StatusInternalServerError, data: "Failed to update token balance (OPTIONS ACCOUNT) for subAccount"}, err2
			}

			status4, err4 := oc.balanceClient.UpdateTokenBalance(contractUtils.OPTIONS_FEES_SUBACCOUNT_ID, contractUtils.QUOTE_TOKEN_PRODUCT_ID, feesAmount.String())
			if err4 != nil || !status4 {
				rollbackErr := rollbackDbChanges()
				if rollbackErr != nil {
					xlog.Errorf("Failed to rollback after fees account update failure: %v", rollbackErr)
				}
				return &Response{status: http.StatusInternalServerError, data: "Failed to update token balance (OPTIONS FEES ACCOUNT) for subAccount"}, err4
			}

			status, err := oc.balanceClient.UpdateTokenBalance(subaccountHex, contractUtils.QUOTE_TOKEN_PRODUCT_ID, combinedDelta.String())
			if err != nil || !status {
				rollbackErr := rollbackDbChanges()
				if rollbackErr != nil {
					xlog.Errorf("Failed to rollback after balance update failure: %v", rollbackErr)
				}
				return &Response{status: http.StatusInternalServerError, data: "Failed to update token balance for subAccount"}, err
			}
		}

		// Add to the batching db
		status5, err5 := contract.GlobalContracts.EndpointContract.CloseOptionBet(req.OrderId, exitPrice, transactionCounter, subaccountHex)
		if err5 != nil || !status5 {
			return &Response{status: http.StatusInternalServerError, data: "Failed to add to batching db for subAccount"}, err5
		}

		return &Response{status: http.StatusOK, data: "Option bet closed successfully"}, nil
	})

	if errLock != nil {
		xlog.Errorf("Failed to close bet - options: %v", errLock)
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Close bet - options faced error for subAccount: %s, orderId: %d, please look into this: %v", req.SubaccountId, req.OrderId, errLock))
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to close bet - options, Internal Server Error: " + errLock.Error()})
		return
	}

	// Respond with the response
	ctx.JSON(response.status, response.data)
}

func (oc *OptionsController) PositionsAndHistory(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	// Get positions and history from the database
	positionsAndHistory, err := oc.optionsDb.GetPositionsAndHistoryBySubaccount(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Failed to get positions and history for subaccount: %s, error: %v", currentSubaccount, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get positions and history")
		return
	}

	// Separate positions and history
	var positions []map[string]interface{}
	var history []map[string]interface{}

	for _, option := range positionsAndHistory {
		if option.ExitPrice == nil { // Position (ExitPrice is null)
			positions = append(positions, map[string]interface{}{
				"entrySize":  option.Amount.String(),     // Amount in string
				"entryPrice": option.EntryPrice.String(), // Entry price in string
				"entryTime":  option.EntryTime,
				"interval":   option.Interval,
				"payout_pnl": option.Payout,
				"productId":  option.ProductId,
			})
		} else { // History
			history = append(history, map[string]interface{}{
				"entrySize":  option.Amount.String(),     // Amount in string
				"entryPrice": option.EntryPrice.String(), // Entry price in string
				"entryTime":  option.EntryTime,
				"exitPrice":  option.ExitPrice.String(), // Exit price in string
				"interval":   option.Interval,
				"pnl":        option.UserPnl.String(),
				"productId":  option.ProductId,
			})
		}
	}

	//sort the arrays by entry time (descending) (write function)
	// Sort positions by descending entryTime
	sort.Slice(positions, func(i, j int) bool {
		return positions[i]["entryTime"].(int64) > positions[j]["entryTime"].(int64)
	})

	// Sort history by descending entryTime
	sort.Slice(history, func(i, j int) bool {
		return history[i]["entryTime"].(int64) > history[j]["entryTime"].(int64)
	})

	// Prepare the response
	response := gin.H{
		"positions": positions,
		"history":   history,
	}

	// Send the response
	ctx.JSON(http.StatusOK, response)
}

func (oc *OptionsController) GetMarketData(ctx *gin.Context) {
	// Get the payout and fees from Redis
	payoutMap, err := xredis.GetRedisClient().HGetAll(context.Background(), xredis.GetOptionsPayoutKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get payout from Redis, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get payout")
		return
	}

	// Restructure the response
	restructuredResponse := make(map[string]map[string]int)
	for key, value := range payoutMap {
		// Split the key into productId and interval
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			xlog.Warnf("Invalid key format in Redis: %s", key)
			continue
		}

		productId := parts[0]
		interval := parts[1]

		// Parse the payout value into an integer
		payoutValue, err := strconv.Atoi(value)
		if err != nil {
			xlog.Warnf("Invalid payout value in Redis for key %s: %v", key, err)
			continue
		}
		adjustedPayoutValue := payoutValue - 100

		// Initialize the nested map for the productId if not present
		if _, exists := restructuredResponse[productId]; !exists {
			restructuredResponse[productId] = make(map[string]int)
		}

		// Add the interval and payout value
		restructuredResponse[productId][interval] = adjustedPayoutValue
	}

	minimumQuote, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetOptionsMinimumAmountKey()).Result()
	if err != nil {
		xlog.Errorf("Options - Cache Data - Failed to get minimum quote amount from Redis, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get minimum quote amount"})
		return
	}

	maximumQuote, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetOptionsMaximumAmountKey()).Result()
	if err != nil {
		xlog.Errorf("Options - Cache Data - Failed to get maximum quote amount from Redis, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get maximum quote amount"})
		return
	}

	isEnabled, err := xredis.GetRedisClient().HGetAll(context.Background(), xredis.GetOptionsEnabledKey()).Result()
	if err != nil {
		xlog.Errorf("Options - Cache Data - Failed to get isEnabled from Redis, error: %v", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get isEnabled"})
		return
	}

	// Restructure isEnabled map
	isEnabledBool := make(map[int]bool)
	for key, value := range isEnabled {
		productId, err := strconv.Atoi(key)
		if err != nil {
			xlog.Errorf("Options - Cache Data - Failed to convert productId to int, error: %v", err)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert productId to int"})
			return
		}

		isEnabledBool[productId] = value == "1"
	}
	// add the restructuredResponse as a key in the final response and others keys being minimum and maximum quote
	finalResponse := gin.H{
		"payout":        restructuredResponse,
		"minimumAmount": minimumQuote,
		"maximumAmount": maximumQuote,
		"isEnabled":     isEnabledBool,
	}

	// Send the response
	ctx.JSON(http.StatusOK, finalResponse)
}

// APIs for internal monitoring

// 1. Get exposure by productId
func (oc *OptionsController) GetExposure(ctx *gin.Context) {
	// Get all open options
	options, err := oc.optionsDb.GetCurrentOI()
	if err != nil {
		xlog.Errorf("Failed to get open options from the database, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get open options")
		return
	}

	// Group by ProductId and calculate the required fields
	groupedData := make(map[uint32]map[string]string)

	for _, option := range options {
		// Ensure the group exists
		if _, exists := groupedData[option.ProductId]; !exists {
			groupedData[option.ProductId] = map[string]string{
				"down_quote_total": "0",
				"up_quote_total":   "0",
			}
		}

		// If quoteDelta is null, continue
		if option.QuoteDelta == nil || option.QuoteDelta.Val == nil {
			xlog.Warnf("QuoteDelta is null for option ID: %d", option.ID)
			continue
		}

		if option.Amount.Val.Sign() < 0 {
			// Add to down quote total
			currentDownTotal, _ := new(big.Int).SetString(groupedData[option.ProductId]["down_quote_total"], 10)
			groupedData[option.ProductId]["down_quote_total"] = new(big.Int).Add(currentDownTotal, option.QuoteDelta.Val).String()
		} else {
			// Add to up quote total
			currentUpTotal, _ := new(big.Int).SetString(groupedData[option.ProductId]["up_quote_total"], 10)
			groupedData[option.ProductId]["up_quote_total"] = new(big.Int).Add(currentUpTotal, option.QuoteDelta.Val).String()
		}
	}

	// Prepare the response
	response := gin.H{
		"oi": groupedData,
	}

	// Send the response
	ctx.JSON(http.StatusOK, response)
}

// 2. Get historical pnl by productId further grouped by interval
func (oc *OptionsController) GetHistoricalData(ctx *gin.Context) {
	historicalData, err := oc.optionsDb.GetHistoricalData()
	if err != nil {
		xlog.Errorf("Failed to get historical data from the database, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get historical data")
		return
	}

	response := gin.H{
		"historicalData": historicalData,
	}

	// Send the response
	ctx.JSON(http.StatusOK, response)
}

// 3.
// Number of trades grouped via product and interval wise with (won and lost)
func (oc *OptionsController) GetTradesBreakdown(ctx *gin.Context) {
	tradesBreakdown, err := oc.optionsDb.GetTradesBreakdown()
	if err != nil {
		xlog.Errorf("Failed to get trades breakdown from the database, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get trades breakdown")
		return
	}

	response := gin.H{
		"tradesBreakdown": tradesBreakdown,
	}

	// Send the response
	ctx.JSON(http.StatusOK, response)
}

// 4. 24 hour data
func (oc *OptionsController) Get24HourStats(ctx *gin.Context) {
	// Fetch the last 24-hour stats from the database
	numTrades, numUsers, totalQuoteDelta, totalUniqueSubaccounts, err := oc.optionsDb.GetLast24HourStats()
	if err != nil {
		xlog.Errorf("Failed to get 24-hour stats from the database, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get 24-hour stats")
		return
	}

	// Prepare the response data
	response := gin.H{
		"trades_count_24hour":          numTrades,
		"unique_users_count_24hour":    numUsers,
		"total_quote_delta_sum_24hour": totalQuoteDelta.String(),
		"total_unique_subaccounts":     totalUniqueSubaccounts,
	}

	// Send the response
	ctx.JSON(http.StatusOK, response)
}

func (oc *OptionsController) Get30DayStats(ctx *gin.Context) {
	// Fetch the last 30-day stats from the database
	stats, err := oc.optionsDb.GetLast30DaysStats()
	if err != nil {
		xlog.Errorf("Failed to get 30-day stats from the database, error: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get 30-day stats")
		return
	}

	// Send the response
	ctx.JSON(http.StatusOK, stats)
}

// DownloadHistory handles API request to download options trading history
func (oc *OptionsController) DownloadHistory(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the API key")
		return
	}

	// Get only history (completed trades) from the database
	history, err := oc.optionsDb.GetHistoryBySubaccount(currentSubaccount.ID)
	if err != nil {
		xlog.Errorf("Failed to get history for subaccount: %s, error: %v", currentSubaccount.ID, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to get trading history")
		return
	}

	// Create CSV data
	csvData := [][]string{
		{"Timestamp", "Product", "Duration", "Entry Price", "Entry Size in USD", "Exit Price", "PnL", "Payout Amount", "Fees Amount", "Outcome"},
	}

	for _, option := range history {
		// Get product symbol using the map
		productSymbol, exists := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[option.ProductId]
		if !exists {
			productSymbol = fmt.Sprintf("Product-%d", option.ProductId)
		}

		// Format timestamp
		timestamp := time.Unix(option.EntryTime, 0).Format("2006-01-02 15:04:05")

		// Format duration
		duration := fmt.Sprintf("%d Minutes", option.Interval)

		// Convert BigInt values to human-readable format
		entryPrice := cutils.X18ToFloatStr(option.EntryPrice.Val)
		entrySize := cutils.X18ToFloatStr(option.QuoteDelta.Val)
		if entrySizeVal, parseErr := strconv.ParseFloat(entrySize, 64); parseErr == nil {
			entrySize = fmt.Sprintf("%.6f", math.Abs(entrySizeVal))
		}
		exitPrice := cutils.X18ToFloatStr(option.ExitPrice.Val)
		pnl := cutils.X18ToFloatStr(option.UserPnl.Val)
		payoutAmount := cutils.X18ToFloatStr(option.PayoutAmount.Val)
		feesAmount := cutils.X18ToFloatStr(option.FeesAmount.Val)

		outcome := "LOSS"
		if option.UserPnl != nil && option.UserPnl.Val != nil && option.UserPnl.Val.Sign() >= 0 {
			outcome = "WIN"
		}

		csvData = append(csvData, []string{
			timestamp,
			productSymbol,
			duration,
			entryPrice,
			entrySize,
			exitPrice,
			pnl,
			payoutAmount,
			feesAmount,
			outcome,
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

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("options_history_%s.csv", timestamp)

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	ctx.Header("Content-Type", "text/csv")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Expires", "0")
	ctx.Header("Cache-Control", "must-revalidate")
	ctx.Header("Pragma", "public")

	ctx.Data(http.StatusOK, "text/csv", buffer.Bytes())
}
