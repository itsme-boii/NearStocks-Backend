package tests

import (
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/testutils"
	"math/big"
	"testing"
)

func setupEndpoint() {
	testutils.SetupContractEnv()
	contract.Init()
}

func TestGetPerpPricesX18SliceForContract(t *testing.T) {
	perpOraclePricesX18 := map[uint32]*big.Int{}
	for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
		perpOraclePricesX18[productId] = big.NewInt(10)
	}

	result := contract.GetPerpPricesX18SliceForContract(perpOraclePricesX18)

	if len(result) != len(contractUtils.ALL_PERPS_ON_CONTRACT) {
		t.Errorf("Expected length of result to be %d, but got %d", len(contractUtils.ALL_PERPS_ON_CONTRACT), len(result))
	}

	for i := 0; i < len(result); i++ {
		if result[i].Cmp(big.NewInt(10)) != 0 {
			t.Errorf("Expected result[%d] to be 10, but got %s", i, result[i].String())
		}
	}
}
