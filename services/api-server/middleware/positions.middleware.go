package middleware

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/gin-gonic/gin"
)

func RequireValidPositionLimits(ctx *gin.Context) {
	var subaccountID string
	var marketId uint32
	var isBuy *bool
	var amountStr, priceStr, newVquote *big.Int
	ammSubaccountID := os.Getenv("AMM_SUBACCOUNT_ID")

	// Parse the incoming CreateOrderBody request
	createOrderReq := ctypes.CreateOrderBody{Party: ctypes.PARTY_TRADER}
	if err := ctx.ShouldBindJSON(&createOrderReq); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid order request body")
		return
	}

	ctx.Set("createOrderReq", createOrderReq)

	// Skip the check if it's a reducing order
	if createOrderReq.IsReduce != nil && *createOrderReq.IsReduce {
		ctx.Next()
		return
	}

	// Extract necessary fields from CreateOrderBody
	marketId = uint32(*createOrderReq.MarketId)
	isBuy = createOrderReq.IsBuy

	// If it is a market order, fetch prices from the app state
	if createOrderReq.OrderType == ctypes.ORDER_TYPE_MARKET {
		var err error
		priceStr, err = FetchOraclePrice(marketId)
		if err != nil {
			cutils.ApiAbort(ctx, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		if createOrderReq.PriceStr != "" {
			price, err := convertDecimalStringToBigInt(createOrderReq.PriceStr, 18)
			if err != nil {
				cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
				return
			}
			priceStr = price
		}
	}

	// Extract amount from CreateOrderBody
	amount, err := convertDecimalStringToBigInt(createOrderReq.AmountStr, 18)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}
	amountStr = amount

	// Get the current subaccount from the context
	currentSubaccount := getCurrentSubaccount(ctx)
	if currentSubaccount == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Subaccount is missing in the request context"})
		return
	}

	// Convert subaccount ID to bytes and get its hex representation
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(currentSubaccount.ID)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to convert subaccount ID")
		return
	}
	subaccountID = strings.ToLower("0x" + hex.EncodeToString(subaccountIdBytes[:]))

	// Check if the subaccount is the AMM subaccount, skip further checks if true
	if subaccountID == ammSubaccountID {
		ctx.Next()
		return
	}

	// Multiply amount by price to get total value
	amountXPrice := new(big.Int).Mul(amountStr, priceStr)
	amountXPrice = cutils.Divx18(amountXPrice)
	if *isBuy {
		newVquote = new(big.Int).Neg(amountXPrice)
	} else {
		newVquote = amountXPrice
	}

	// Check if the new position is allowed using the updated CheckSubaccountPositions function
	isPositionAllowed, currentVQuote, err := CheckSubaccountPositions(marketId, subaccountID, *isBuy, newVquote)
	if err != nil || !isPositionAllowed {
		if err != nil {
			cutils.ApiAbort(ctx, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
			return
		} else {
			cutils.ApiAbort(ctx, http.StatusForbidden, "The new position exceeds the allowed OI limit")
			return
		}
	}

	totalLongPosition, totalShortPosition, err := FetchTotalPosition(marketId, priceStr)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error fetching total position")
		return
	}

	// Set product ID limits based on marketId
	longLimit, shortLimit := getMarketLimit(marketId)

	// Add currentVQuote and newVquote to get newTotalVQuote
	newTotalVQuote := new(big.Int)

	// Check if currentVQuote and newVQuote have opposite signs
	if (currentVQuote.Sign() >= 0 && newVquote.Sign() < 0) || (currentVQuote.Sign() < 0 && newVquote.Sign() >= 0) {
		// If they have opposite signs, add the two values
		if new(big.Int).Abs(newVquote).Cmp(new(big.Int).Abs(currentVQuote)) < 0 {
			ctx.Next()
			return
		}
		newTotalVQuote.Add(currentVQuote, newVquote)
		if (newTotalVQuote.Sign() >= 0 && currentVQuote.Sign() > 0) || (newTotalVQuote.Sign() <= 0 && currentVQuote.Sign() < 0) {

			if newTotalVQuote.Sign() >= 0 {
				totalShortPosition = new(big.Int).Sub(totalShortPosition, new(big.Int).Abs(currentVQuote))
			} else {
				totalLongPosition = new(big.Int).Sub(totalLongPosition, new(big.Int).Abs(currentVQuote))
			}
		}
		// if newVquote postive remove from short else from long remove currentVQuote
	} else {
		// If they have the same sign, newTotalVQuote will just be the value of newVQuote
		newTotalVQuote.Set(newVquote)
	}
	// If newTotalVQuote is negative, compare with the total long position
	if newTotalVQuote.Sign() < 0 {
		newTotalPosition := new(big.Int).Add(totalLongPosition, new(big.Int).Abs(newTotalVQuote))
		// Check if the new total position exceeds the allowed limit
		if newTotalPosition.Cmp(longLimit) > 0 {
			cutils.ApiAbort(ctx, http.StatusInternalServerError, "The new position exceeds the allowed limit based on total long position")
			return
		}
	} else {
		// If newTotalVQuote is non-negative, compare with the total short position
		newTotalPosition := new(big.Int).Add(totalShortPosition, new(big.Int).Abs(newTotalVQuote))
		// Check if the new total position exceeds the allowed limit
		if newTotalPosition.Cmp(shortLimit) > 0 {
			cutils.ApiAbort(ctx, http.StatusInternalServerError, "The new position exceeds the allowed limit based on total short position")
			return
		}
	}

	// If all checks pass, proceed to the next middleware or handle the trade
	ctx.Next()
}

func convertDecimalStringToBigInt(decimalStr string, decimals int) (*big.Int, error) {
	// Convert string to big.Float
	amountFloat, _, err := new(big.Float).Parse(decimalStr, 10)
	if err != nil {
		return nil, fmt.Errorf("invalid decimal format: %v", err)
	}

	// Scale the big.Float by 10^decimals
	scaleFactor := new(big.Float).SetFloat64(math.Pow10(decimals))
	scaledAmount := new(big.Float).Mul(amountFloat, scaleFactor)

	// Convert the scaled big.Float to a big.Int
	amountInt := new(big.Int)
	scaledAmount.Int(amountInt) // Drop any fractional part
	return amountInt, nil
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

	// Use marketId to get the symbol from the product ID map
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

// Validate the new position based on subaccount and AMM limits
// func ValidateNewPosition(marketId uint32, newVquote *big.Int, subaccountID, ammSubaccountID string, IsBuy bool) error {
// 	subaccountAvailable, err := CheckSubaccountPositions(marketId, subaccountID, IsBuy)
// 	if err != nil {
// 		return fmt.Errorf("error in subaccount position check: %v", err)
// 	}

// 	ammAvailable, err := GetAMMOpenInterestForMarket(marketId, ammSubaccountID, IsBuy)
// 	if err != nil {
// 		return fmt.Errorf("error in amm open interest check: %v", err)
// 	}

// 	// Compare subaccountAvailable and ammAvailable to find the minimum
// 	minAvailable := subaccountAvailable
// 	if ammAvailable.Cmp(subaccountAvailable) < 0 {
// 		minAvailable = ammAvailable
// 	}
// 	// Validate the new position
// 	if newVquote.Cmp(minAvailable) > 0 {
// 		return fmt.Errorf("the new position amount exceeds the allowed limit, maximum allowed is %v", minAvailable.String())
// 	}

// 	return nil
// }

// Function to fetch total long or short position based on marketId and isBuy flag
func FetchTotalPosition(marketId uint32, priceStr *big.Int) (*big.Int, *big.Int, error) {
	perputils := perputils.NewPerpUtils()

	// Fetch total long position
	totalLongPosition, err := perputils.GetTotalLongPosition(marketId)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching total long position: %v", err)
	}

	// Fetch total short position
	totalShortPosition, err := perputils.GetTotalShortPosition(marketId)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching total short position: %v", err)
	}

	// Multiply both positions by priceStr
	totalLongPositionWithPrice := cutils.Divx18(new(big.Int).Mul(totalLongPosition, priceStr))
	totalShortPositionWithPrice := cutils.Divx18(new(big.Int).Mul(totalShortPosition, priceStr))

	// Return both multiplied positions
	return totalLongPositionWithPrice, totalShortPositionWithPrice, nil
}

// Check available position in subaccount
func CheckSubaccountPositions(marketId uint32, subaccountID string, isBuy bool, newVQuote *big.Int) (bool, *big.Int, error) {

	positions, err := xclient.GlobalBalanceClient.GetPerpPositions(subaccountID, []uint{})
	if err != nil {
		return false, nil, fmt.Errorf("error fetching perp positions: %v", err)
	}
	perputils := perputils.NewPerpUtils()
	limit, err := perputils.GetSubaccountOICap()
	if err != nil {
		return false, nil, fmt.Errorf("error fetching subaccount OI cap: %v", err)
	}
	limit = cutils.Mulx18(limit)
	totalVQuoteBalance := big.NewInt(0)

	// Initialize variables for the current position's vQuoteBalance
	currentVQuote := big.NewInt(0)
	for _, positionData := range positions {
		// Assert that productID is a uint32
		productIDFloat, ok := positionData["productID"].(float64)
		if !ok {
			xlog.Errorf("User Controller - Error: productID is not a float64\n")
			continue
		}
		productID := uint32(productIDFloat)

		// Extract vQuoteBalance as a string
		vQuoteBalanceStr, ok := positionData["vQuoteBalance"].(string)
		if !ok {
			xlog.Errorf("Error: vQuoteBalance is not a string in position data: %v", positionData)
			continue
		}

		// Convert vQuoteBalance string to *big.Int
		vQuoteBalance := new(big.Int)
		vQuoteBalance, success := vQuoteBalance.SetString(vQuoteBalanceStr, 10)
		if !success {
			xlog.Errorf("Error: unable to convert vQuoteBalance string to big.Int: %v", vQuoteBalanceStr)
			continue
		}
		// If the posMarketID matches the given marketId, store the vQuoteBalance for comparison
		if productID == marketId {
			currentVQuote = vQuoteBalance
		} else {
			// Add the absolute value of vQuoteBalance to totalVQuoteBalance
			totalVQuoteBalance.Add(totalVQuoteBalance, new(big.Int).Abs(vQuoteBalance))
		}
	}

	// Now handle the vQuote for the given marketId
	delta := big.NewInt(0)
	delta.Add(currentVQuote, newVQuote)

	// Check the delta against per user per market cap to
	productIdStr := strconv.FormatUint(uint64(marketId), 10)
	subaccountMarketLimit, err := perputils.GetSubaccountMarketCap(productIdStr)
	if err != nil {
		return false, currentVQuote, fmt.Errorf("error fetching subaccount market cap")
	}
	subaccountMarketLimit = cutils.Mulx18(subaccountMarketLimit)

	// Add delta to totalVQuoteBalance
	totalVQuoteBalance.Add(totalVQuoteBalance, new(big.Int).Abs(delta))

	// Check if the total vQuote balance exceeds the limit
	if (!isBuy && currentVQuote.Sign() <= 0) || (isBuy && currentVQuote.Sign() >= 0) {
		// Create absolute values for newVQuote and currentVQuote
		absNewVQuote := new(big.Int).Abs(newVQuote)
		absCurrentVQuote := new(big.Int).Abs(currentVQuote)
		// Compare the absolute values
		if absNewVQuote.Cmp(absCurrentVQuote) <= 0 {
			return true, currentVQuote, nil
		}
	}

	if delta.Abs(delta).Cmp(subaccountMarketLimit) > 0 {
		return false, currentVQuote, fmt.Errorf("new position exceeds subaccount market limit")
	}

	if totalVQuoteBalance.Abs(totalVQuoteBalance).Cmp(limit) > 0 {
		return false, currentVQuote, fmt.Errorf("new position exceeds the OI limit")
	}
	return true, currentVQuote, nil
}

// Get AMM open interest for the market
// func GetAMMOpenInterestForMarket(marketId uint32, ammSubaccountID string, IsBuy bool) (*big.Int, error) {
// 	positions, err := xclient.GlobalBalanceClient.GetPerpPositions(ammSubaccountID, []uint{})
// 	if err != nil {
// 		return nil, fmt.Errorf("error fetching perp positions: %v", err)
// 	}
// 	perputils := perputils.NewPerpUtils()
// 	globalCap, err := perputils.GetGlobalNegativeOICap()
// 	if err != nil {
// 		return new(big.Int), fmt.Errorf("error fetching global OI cap: %v", err)
// 	}

// 	limit := cutils.Mulx18(globalCap)
// 	totalVQuote := new(big.Int)  // Total vQuote for all positions
// 	marketVQuote := new(big.Int) // vQuote for the specific marketId
// 	isBuy := IsBuy               // Track if the position is buy or sell for marketId

// 	// Calculate the total vQuote across all positions and store the specific marketId vQuote
// 	for _, position := range positions {
// 		vQuoteBalanceStr, ok := position["vQuoteBalance"].(string)
// 		if !ok {
// 			xlog.Errorf("Error: vQuoteBalance is not a string in position data: %v", position)
// 			continue
// 		}

// 		// Convert vQuoteBalance string to *big.Int
// 		vQuoteBalance := new(big.Int)
// 		vQuoteBalance, success := vQuoteBalance.SetString(vQuoteBalanceStr, 10)
// 		if !success {
// 			xlog.Errorf("Error: unable to convert vQuoteBalance string to big.Int: %v", vQuoteBalanceStr)
// 			continue
// 		}
// 		totalVQuote.Add(totalVQuote, new(big.Int).Abs(vQuoteBalance))

// 		// Check if this position is for the specified marketId
// 		productID, _ := strconv.ParseUint(fmt.Sprintf("%v", position["productID"]), 10, 64)
// 		if productID == uint64(marketId) {
// 			marketVQuote.Set(vQuoteBalance)

// 		}
// 	}

// 	// Now handle the cases based on isBuy and vQuoteBalance sign for the specific marketId
// 	switch {
// 	// Case 1: isBuy and vQuoteBalance is positive
// 	case isBuy && marketVQuote.Sign() < 0:
// 		// Return limit - total vQuote except for the given productID
// 		excludedVQuote := new(big.Int).Sub(totalVQuote, marketVQuote)
// 		return new(big.Int).Sub(limit, excludedVQuote), nil

// 	// Case 2: isBuy and vQuoteBalance is negative
// 	case isBuy && marketVQuote.Sign() > 0:
// 		// Return limit - total vQuote
// 		return new(big.Int).Sub(limit, totalVQuote), nil

// 	// Case 3: !isBuy and vQuoteBalance is positive
// 	case !isBuy && marketVQuote.Sign() < 0:
// 		// Return limit - total vQuote
// 		return new(big.Int).Sub(limit, totalVQuote), nil

// 	// Case 4: !isBuy and vQuoteBalance is negative
// 	case !isBuy && marketVQuote.Sign() > 0:
// 		// Return limit - total vQuote except for the given productID
// 		excludedVQuote := new(big.Int).Sub(totalVQuote, marketVQuote)
// 		return new(big.Int).Sub(limit, excludedVQuote), nil
// 	}

// 	return limit, nil
// }

func getMarketLimit(marketId uint32) (*big.Int, *big.Int) {
	perputils := perputils.NewPerpUtils()
	longOiCap, shortOiCap, err := perputils.GetMarketOICap(marketId)
	if err != nil {
		// Log error and return default
		fmt.Printf("Error fetching market OI cap: %v\n", err)
		return cutils.Mulx18(perputils.GetDefaultMarketLimit(marketId)), cutils.Mulx18(perputils.GetDefaultMarketLimit(marketId))
	}
	return cutils.Mulx18(longOiCap), cutils.Mulx18(shortOiCap)
}
