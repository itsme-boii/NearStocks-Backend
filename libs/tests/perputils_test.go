package tests

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"math/big"
	"testing"
)

func TestGetBigLockQuotex18(t *testing.T) {
	// New market
	market := &db.MarketTable{
		TakerFeeFractionx18: ctypes.BigInt{Val: new(big.Int).Mul(new(big.Int).SetInt64(500), cutils.GetBigx12())},
		InitialMarginFractionx18: ctypes.BigInt{Val: new(big.Int).Mul(new(big.Int).SetInt64(50000), cutils.GetBigx12())},
	}

	// 1 mn base amount
	baseAmtx18 := new(big.Int).Mul(new(big.Int).SetInt64(1_000_000), cutils.GetBigx18())

	// 100 k price
	pricex18 := new(big.Int).Mul(new(big.Int).SetInt64(100_000), cutils.GetBigx18())

	ret := perputils.GetBigLockQuotex18(baseAmtx18, pricex18, market)

	// Quote big 18 = 1 mn * 100 k * 1e18 = 1e29
	// Taker Fee = 0.0005 * 1e29
	// InitialMarginFee = 0.05 * 1e29
	expected, _ := new(big.Int).SetString("5050000000000000000000000000", 10)

	if ret.Cmp(expected) != 0 {
		t.Errorf("Expected price: %v, got %v", expected, ret)
	}
}
