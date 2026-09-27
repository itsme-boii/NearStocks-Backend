package liquidator

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	apiserverTypes "github/eugenix-io/logx-inf-backend/services/api-server/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
)

// TODO: Move to utils
func simplifyOraclePriceMap(oraclePricesMap map[string]ctypes.OraclePrice) map[string]*big.Int {
	simplifiedOraclePricesMap := make(map[string]*big.Int, len(oraclePricesMap))
	for symbol, oraclePrice := range oraclePricesMap {
		simplifiedOraclePricesMap[symbol] = oraclePrice.Pricex18
	}
	return simplifiedOraclePricesMap
}

// TODO: This can be optimised
// Pick the first Perp from the map and create a liquidation order
func createLiquidationOrder(subaccount subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice) (*apiserverTypes.LiquidationOrderRequest, error) {
	// Pick the first Perp from the map and create a liquidation order
	perpBalance := subaccount.GetMaxAbsValuePerpBalance()

	// If there are no open positions, nothing can be done. Ideally this should not happen and negative balances should be compensated by insurance funds
	if perpBalance == nil {
		return nil, fmt.Errorf("no open positions found for liquidation. Ideally this means the subaccount is having negative spot balances and insurance funds / develeraging is not working as expected")
	}

	// We need to place opposite order to liquidate the position
	isBuy := false
	if perpBalance.Amountx18.Sign() == -1 {
		isBuy = true
	}
	// Amount str is the absolute value of the net long position in float
	amountStr := cutils.X18ToFloatStr(new(big.Int).Abs(perpBalance.Amountx18))

	// Get spot prices
	spotPricesx18 := marketutils.GetSpotPricesX18Map(simplifyOraclePriceMap(oraclePricesMap), true)
	// Get perp prices
	perpOraclePricesx18 := marketutils.GetPerpPricesx18Map(simplifyOraclePriceMap(oraclePricesMap))

	return &apiserverTypes.LiquidationOrderRequest{
		MarketId:      uint(perpBalance.ProductId),
		SubaccountId:  subaccount.SubaccountId, // Convert hex subaccount to string version
		AmountStr:     amountStr,
		IsBuy:         &isBuy,
		Party:         ctypes.PARTY_TRADER, // Current assumption is that only traders will be liquidated
		SpotPricesx18: spotPricesx18,
		PerpPricesx18: perpOraclePricesx18,
	}, nil
}

// Call api server and return the amount filled
func PlaceLiquidationOrder(subaccount subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice) (amountFilledx18 *big.Int, fillStatus ctypes.OrderStatus, err error) {
	liquidationOrder, err := createLiquidationOrder(subaccount, oraclePricesMap)
	if err != nil {
		return nil, ctypes.ORDER_STATUS_CANCELLED, err
	}

	var resp *apiserverTypes.PlaceOrderResponse
	resp, err = xclient.GlobalApiServerClient.PlaceLiquidationOrder(*liquidationOrder)
	if err != nil {
		xlog.Errorf("Error placing liquidation order for subaccount: %v. Ideally this means there is some issue with api server or matching engine.", subaccount.SubaccountId)
		return nil, ctypes.ORDER_STATUS_CANCELLED, err
	}

	amountFilledx18, ok := new(big.Int).SetString(resp.TotalFilledx18, 10)
	if !ok {
		return nil, ctypes.ORDER_STATUS_CANCELLED, fmt.Errorf("error converting amount filled. This should never happen and means there is a bug in engine response")
	}

	amountRequestedx18 := cutils.FloatStrToX18(liquidationOrder.AmountStr)
	if amountFilledx18.Cmp(amountRequestedx18) == 0 {
		return amountFilledx18, ctypes.ORDER_STATUS_FILLED, nil
	} else if amountFilledx18.Sign() == 0 {
		return amountFilledx18, ctypes.ORDER_STATUS_CANCELLED, nil
	} else {
		return amountFilledx18, ctypes.ORDER_STATUS_PARTIAL, nil
	}
}

func SettleWithInsurance(subaccountId string) (bool, error) {
	err := xclient.GlobalApiServerClient.SettleWithInsurance(apiserverTypes.SettleWithInsuranceRequest{SubaccountId: subaccountId})
	if err != nil {
		xlog.Errorf("Error settling with insurance for subaccount: %v. Ideally this means there is some issue with api server or matching engine.", subaccountId)
		return false, err
	}
	return true, nil
}
