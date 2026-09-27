package tests

import (
	"testing"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/spotUtils"
	"github/eugenix-io/logx-inf-backend/testutils"
	"github/eugenix-io/logx-inf-backend/testutils/mocks"
	"math/big"

	"github.com/stretchr/testify/assert"
)

func TestGetSpotPricesX18SliceForContract(t *testing.T) {
	testutils.SetMainnetEnv()

	// Create a map of spot oracle prices
	spotOraclePricesX18 := make(map[uint32]*big.Int)
	for _, productId := range contractUtils.ALL_SPOTS_ON_CONTRACT {
		spotOraclePricesX18[productId] = big.NewInt(10)
	}

	// Call the function under test
	result := spotUtils.GetSpotPricesX18SliceForContract(spotOraclePricesX18)

	// Assert the expected result
	assert.Equal(t, len(contractUtils.ALL_SPOTS_ON_CONTRACT), len(result), "Expected length of result to be %d, but got %d", len(contractUtils.ALL_SPOTS_ON_CONTRACT), len(result))
	for i := 0; i < len(result); i++ {
		if contractUtils.PRODUCT_MARKET_WEIGHTS[contractUtils.ALL_SPOTS_ON_CONTRACT[i]] == 0 {
			assert.Equal(t, big.NewInt(0), result[i], "Expected result[%d] to be 0, but got %s", i, result[i].String())
		} else {
			assert.Equal(t, big.NewInt(10), result[i], "Expected result[%d] to be 10, but got %s", i, result[i].String())
		}
	}
}

func TestGetQuoteValuex36(t *testing.T) {
	testutils.SetMainnetEnv()

	// Create a map of oracle prices
	oraclePricesMap := mocks.GetMockOraclePrices()
	for _, productId := range contractUtils.ALL_SPOTS_ON_CONTRACT {
		quoteValuex18, err := spotUtils.GetQuoteValuex36(productId, big.NewInt(10), oraclePricesMap)
		assert.Nil(t, err, "Expected error to be nil, but got %v", err)
		if productId == contractUtils.LOGX || productId == contractUtils.ST_LOGX {
			assert.Equal(t, quoteValuex18.Sign(), 0, "Expected quote value to be 0, but got %v", quoteValuex18.Sign())
		} else {
			assert.Equal(t, quoteValuex18.Sign(), 1, "Expected quote value to be 100, but got %s", quoteValuex18.String())
		}
	}
}
