package spotUtils

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
)

func GetSpotPricesX18SliceForContract(spotOraclePricesX18 map[uint32]*big.Int) []*big.Int {
	spotPricesX18 := make([]*big.Int, 0)
	for _, productId := range contractUtils.ALL_SPOTS_ON_CONTRACT {
		if spotOraclePricesX18[productId] == nil || contractUtils.PRODUCT_MARKET_WEIGHTS[productId] == 0 {
			spotPricesX18 = append(spotPricesX18, big.NewInt(0))
		} else {
			spotPricesX18 = append(spotPricesX18, spotOraclePricesX18[productId])
		}
	}
	return spotPricesX18
}

// Returns dollar value of the spot product that you are providing
func GetQuoteValuex36(productId uint32, tokenAmountx18 *big.Int, oraclePricesMap map[string]ctypes.OraclePrice) (*big.Int, error) {
	if contractUtils.PRODUCT_MARKET_WEIGHTS[productId] == 0 {
		return big.NewInt(0), nil
	}

	oracleSymbol, exists := marketutils.GetBaseSymbolForProduct(productId)
	if !exists {
		xlog.Errorf("Oracle symbol not found for product id: %d. This means there is a bug in contractUtils.PRODUCT_ID_SYMBOL_TO_MAP", productId)
		return nil, fmt.Errorf("oracle symbol not found for product id: %d. This means there is a bug in contractUtils.PRODUCT_ID_SYMBOL_TO_MAP", productId)
	}

	if oraclePrice, ok := oraclePricesMap[oracleSymbol]; ok {
		return new(big.Int).Mul(tokenAmountx18, oraclePrice.Pricex18), nil
	}
	return nil, fmt.Errorf("oracle price not found for symbol: %s. This means there is a bug in oracle", oracleSymbol)
}
