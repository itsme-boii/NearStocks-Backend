package utils

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"math/big"
)

func CalcFundingFeesx18(perpBalance types.PerpBalance, curFundingRatex18 *big.Int) *big.Int {
	lastFundingRatex18 := ctypes.NewBigIntFromString(perpBalance.LastFundingRate).Val
	lastVquoteBalancex18 := ctypes.NewBigIntFromString(perpBalance.VQuoteBalance).Val
	diffInFundingRatex18 := new(big.Int).Sub(curFundingRatex18, lastFundingRatex18)
	fundingFeex36 := new(big.Int).Mul(lastVquoteBalancex18, diffInFundingRatex18)
	// Funding fee is negative of the calculated value
	fundingFeex36 = new(big.Int).Neg(fundingFeex36)
	return cutils.Divx18(fundingFeex36)
}

func CalcRealisedPnlAndQuoteUpdate(updateAmount string, amount string, updateQuote string, quote string) (*big.Int, *big.Int) {
	if amount == "0" || quote == "0" {
		if amount == "0" && quote == "0" {
			return ctypes.NewBigIntFromString("0").Val, ctypes.NewBigIntFromString(updateQuote).Val
		} else {
			xlog.Errorf("High severity error! amount:%s , vQuote: %s", amount, quote)
			panic("Application panicked while calulcation realisedPnl and quoteUpdate")
		}
	}

	updateAmountInt := ctypes.NewBigIntFromString(updateAmount)
	amountInt := ctypes.NewBigIntFromString(amount)
	updateQuoteInt := ctypes.NewBigIntFromString(updateQuote)
	quoteInt := ctypes.NewBigIntFromString(quote)
	if amountInt.Mul(updateAmountInt).Sign() > 0 {
		return big.NewInt(0), updateQuoteInt.Add(quoteInt).Val
	}
	absAmountInt := new(big.Int).Abs(amountInt.Val)
	absUpdateAmountInt := new(big.Int).Abs(updateAmountInt.Val)
	if absAmountInt.Cmp(absUpdateAmountInt) > 0 {
		x := new(big.Int).Mul(updateAmountInt.Val, quoteInt.Val)
		x = cutils.Divx18(x)
		x = cutils.Mulx18(x)
		y := ctypes.NewBigInt(x)
		proportionalQuote := y.Div(amountInt)
		realisedPnl := updateQuoteInt.Sub(proportionalQuote)
		updatedVquote := quoteInt.Add(proportionalQuote)
		return realisedPnl.Val, updatedVquote.Val
	} else {
		x := new(big.Int).Mul(amountInt.Val, updateQuoteInt.Val)
		x = cutils.Divx18(x)
		x = cutils.Mulx18(x)
		y := ctypes.NewBigInt(x)
		proportionalQuote := y.Div(updateAmountInt)
		realisedPnl := quoteInt.Sub(proportionalQuote)
		updatedVquote := updateQuoteInt.Add(proportionalQuote)
		return realisedPnl.Val, updatedVquote.Val
	}
}
