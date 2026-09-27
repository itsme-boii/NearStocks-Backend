package openinterest

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
	"time"
)

type OpenInterest struct {
	TotalOIX18Map map[uint32]*big.Int
}

func (o *OpenInterest) MustGetByProductId(productId uint32) *big.Int {
	oi, exists := o.TotalOIX18Map[productId]
	if !exists {
		return big.NewInt(0)
	}
	return oi
}

func (o *OpenInterest) NetSumX18() *big.Int {
	netSum := big.NewInt(0)
	for _, oi := range o.TotalOIX18Map {
		netSum = new(big.Int).Add(netSum, oi)
	}
	return netSum
}

func FetchTotalOIFromRedis() (*OpenInterest, error) {
	oiMap, err := fetchOIFromRedis()
	if err != nil {
		return nil, err
	}

	return &OpenInterest{
		TotalOIX18Map: oiMap,
	}, nil
}

// Return OI of users which is negative of AMM OI
func FetchTotalOIFromAMM() (*OpenInterest, error) {
	oiMap, err := fetchOIFromAMM()
	if err != nil {
		return nil, err
	}

	return &OpenInterest{
		TotalOIX18Map: oiMap,
	}, nil
}

func getOraclePriceFromProductId(productId uint32, oraclePricesMap map[string]ctypes.OraclePrice) (*big.Int, bool) {
	symbol, ok := marketutils.GetBaseSymbolForProduct(productId)
	if !ok {
		xlog.Errorf("Symbol not found for ProductId %d", productId)
		return nil, false
	}
	// Get the current oracle price
	oraclePrice, ok := oraclePricesMap[symbol]
	if !ok {
		xlog.Errorf("Oracle price not found for symbol %s", symbol)
		return nil, false
	}
	return oraclePrice.Pricex18, true
}

func fetchOIFromRedis() (map[uint32]*big.Int, error) {
	oraclePricesMap, err := appstate.NewAppState().GetAllOraclePrices(500 * time.Millisecond)
	if err != nil {
		xlog.Errorf("Error fetching oracle prices: %v", err)
		return nil, err
	}

	perpUtils := perputils.NewPerpUtils()
	totalOI := make(map[uint32]*big.Int)

	for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
		priceX18, ok := getOraclePriceFromProductId(productId, oraclePricesMap)
		if !ok {
			continue
		}

		totalLongAmtx18, err := perpUtils.GetTotalLongPosition(productId)
		if err != nil {
			return nil, err
		}
		totalShortAmtx18, err := perpUtils.GetTotalShortPosition(productId)
		if err != nil {
			return nil, err
		}

		netOI := cutils.Divx18(new(big.Int).Mul(new(big.Int).Sub(totalLongAmtx18, totalShortAmtx18), priceX18))
		totalOI[productId] = netOI
	}

	return totalOI, nil
}

func fetchOIFromAMM() (map[uint32]*big.Int, error) {
	// Fetch Oracle Prices
	// Fetch AMM Perp balances
	// OI = -AMM position amount * oracle price
	oraclePricesMap, err := appstate.NewAppState().GetAllOraclePrices(500 * time.Millisecond)
	if err != nil {
		xlog.Errorf("Error fetching oracle prices: %v", err)
		return nil, err
	}

	totalOI := make(map[uint32]*big.Int)
	ammBalances := subaccount.NewSubaccountBalanceImpl().MustGetSubaccountBalancesFromIds([]string{contractUtils.AMM_SUBACCOUNT_ID})[0]
	for productId, perpBalance := range ammBalances.PerpBalances {
		priceX18, ok := getOraclePriceFromProductId(productId, oraclePricesMap)
		if !ok {
			continue
		}

		oi := cutils.Divx18(new(big.Int).Mul(new(big.Int).Neg(perpBalance.Amountx18), priceX18))
		totalOI[productId] = oi
	}

	return totalOI, nil
}
