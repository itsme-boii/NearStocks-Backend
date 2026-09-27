package marketutils

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"log"
	"math/big"
)

// Create a function which will provide spot prices in correct order
// The order of spot prices should be in the same order as the product ids
func GetSpotPricesX18Map(oraclePrices map[string]*big.Int, includeWeights bool) (spotPricesX18 map[uint32]*big.Int) {
	// Iterate over all the spots and get the symbol. Then get the oracle price for that symbol
	spotPricesX18 = make(map[uint32]*big.Int)
	for _, productId := range contractUtils.ALL_SPOTS_IN_ORDER {
		weight := 1
		if includeWeights {
			// Get the weight of the product id
			weight = contractUtils.PRODUCT_MARKET_WEIGHTS[productId]

		}
		if weight == 0 {
			spotPricesX18[productId] = cutils.GetBig0()
		} else {
			// Assume weight to be 1
			spotPricesX18[productId] = GetOraclePriceForProduct(productId, oraclePrices)
		}
	}

	return spotPricesX18
}

func GetPerpPricesx18Map(oraclePrices map[string]*big.Int) (perpOraclePricesX18 map[uint32]*big.Int) {
	perpOraclePricesX18 = make(map[uint32]*big.Int)
	for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
		perpOraclePricesX18[productId] = GetOraclePriceForProduct(productId, oraclePrices)
	}

	return perpOraclePricesX18
}

func GetBaseSymbolForProduct(productId uint32) (string, bool) {
	symbol, exists := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[productId]
	return symbol, exists
}
func GetFundingSymbolForProduct(productId uint32) (string, bool) {
	symbol, exists := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[productId]
	return symbol, exists
}
func GetOraclePriceForProduct(productId uint32, oraclePrices map[string]*big.Int) *big.Int {
	symbol, _ := GetBaseSymbolForProduct(productId)

	spotPrice, ok := oraclePrices[symbol]
	if !ok {
		log.Panicf("Oracle price not found for product id: %v", productId)
	}

	return spotPrice
}

func GetLiquidationFractionx18(productId uint32) *big.Int {
	liquidationFractionStr, ok := contractUtils.LIQUIDATION_FRACTION_STR[productId]
	if !ok {
		liquidationFractionStr = "0.015" // 1.5%
	}

	return cutils.FloatStrToX18(liquidationFractionStr)
}
