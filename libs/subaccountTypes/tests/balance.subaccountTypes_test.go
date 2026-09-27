package tests

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/testutils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubaccountBalances_GetMaxAbsValuePerpBalance(t *testing.T) {
	testutils.SetMainnetEnv()
	defer testutils.ResetEnv()

	tests := []struct {
		name                string
		subaccount          subaccountTypes.SubaccountBalances
		expectedPerpBalance *subaccountTypes.PerpBalance
	}{
		{
			name: "Case 1: Both positive and negative vQuotes exist",
			subaccount: subaccountTypes.SubaccountBalances{
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{
					contractUtils.DOGE_MARKET: {
						ProductId:        contractUtils.DOGE_MARKET,
						Amountx18:        cutils.FloatStrToX18("0"),
						VQuoteBalancex18: cutils.FloatStrToX18("0"),
					},
					contractUtils.ETH_MARKET: {
						ProductId:        contractUtils.ETH_MARKET,
						Amountx18:        cutils.FloatStrToX18("1"),
						VQuoteBalancex18: cutils.FloatStrToX18("-100"),
					},
					contractUtils.BTC_MARKET: {
						ProductId:        contractUtils.BTC_MARKET,
						Amountx18:        cutils.FloatStrToX18("1"),
						VQuoteBalancex18: cutils.FloatStrToX18("-200"),
					},
					contractUtils.LINK_MARKET: {
						ProductId:        contractUtils.LINK_MARKET,
						Amountx18:        cutils.FloatStrToX18("-1"),
						VQuoteBalancex18: cutils.FloatStrToX18("150"),
					},
				},
			},
			expectedPerpBalance: &subaccountTypes.PerpBalance{
				ProductId:        contractUtils.BTC_MARKET,
				Amountx18:        cutils.FloatStrToX18("1"),
				VQuoteBalancex18: cutils.FloatStrToX18("-200"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPerpBalance := tt.subaccount.GetMaxAbsValuePerpBalance()
			assert.EqualValuesf(t, *tt.expectedPerpBalance, *actualPerpBalance, "Expected and actual perp balance do not match")
		})
	}

}
