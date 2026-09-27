package funding

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"math/big"
)

// Returns a map of all funding rates for all perpetual markets
func GetAllCumulativeFundingRates() map[string]*big.Int {
	cumulativeFundingRateMap := make(map[string]*big.Int)
	for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {

		symbol, exists := marketutils.GetFundingSymbolForProduct(productId)
		if !exists {
			xlog.Errorf("Error fetching symbol for product id: %v", productId)
			continue
		}

		cumulativeFundingRate, err := xredis.GetCumulativeFundingRateForSymbol(xredis.GetRedisClient(), symbol)
		if err != nil {
			cumulativeFundingRate = big.NewInt(0)
			xlog.Errorf("Error fetching funding rate for symbol %s, assigning 0 value: %v", symbol, err)
		}

		cumulativeFundingRateMap[symbol] = cumulativeFundingRate
	}
	return cumulativeFundingRateMap
}

func CalcFundingFeesx36(perpBalance subaccountTypes.PerpBalance, curFundingRatex18 *big.Int) *big.Int {
	diffInFundingRatex18 := new(big.Int).Sub(curFundingRatex18, perpBalance.LastCumFundingRatex18)
	fundingFeex36 := new(big.Int).Mul(perpBalance.VQuoteBalancex18, diffInFundingRatex18)
	// Funding fee is negative of the calculated value
	fundingFeex36 = new(big.Int).Neg(fundingFeex36)
	return fundingFeex36
}
