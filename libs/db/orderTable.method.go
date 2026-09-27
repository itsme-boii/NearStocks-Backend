package db

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"math/big"
)

func (o *OrderTable) IsLimitOrder() bool {
	return o.Type == ctypes.ORDER_TYPE_LIMIT
}

func (o *OrderTable) IsMarketOrder() bool {
	return o.Type == ctypes.ORDER_TYPE_MARKET
}

func (o *OrderTable) IsLiquidationOrder() bool {
	return o.Type == ctypes.ORDER_TYPE_LIQUIDATION
}

func (o *OrderTable) RemainingAmount() ctypes.BigInt {
	if o.Amountx18.Val.Cmp(o.TotalFilledx18.Val) == -1 {
		panic("Remaining Amount cannot be greater than total amount")
	}
	return ctypes.BigInt{Val: new(big.Int).Sub(o.Amountx18.Val, o.TotalFilledx18.Val)}
}

// IOC - Immediate or Cancel. Currently, market and liquidation orders are IOC
func (o *OrderTable) IsIOCOrder() bool {
	return o.IsMarketOrder() || o.IsLiquidationOrder()
}

// Reduce only orders can only decrease the position keeping the side same
func (o *OrderTable) CanReduce(perpNetAmount *big.Int) bool {
	if !o.IsReduce {
		panic("CanReduce called on non-reduce order")
	}
	if perpNetAmount.Sign() > 0 {
		return o.Side == ctypes.ORDER_SIDE_SELL
	}
	if perpNetAmount.Sign() < 0 {
		return o.Side == ctypes.ORDER_SIDE_BUY
	}
	return false
}

// For non reduce only orders matchable amount is nothing but the remaining amount
// For reduce only orders, matchable amount is the minimum of remaining amount and open perp position
func (o *OrderTable) GetMatchableAmount(perpNetAmount *big.Int) *big.Int {
	if !o.IsReduce {
		return o.RemainingAmount().Val
	}
	if !o.CanReduce(perpNetAmount) {
		return cutils.GetBig0()
	}

	remainingAmount := o.RemainingAmount().Val
	perpAbsNetAmount := new(big.Int).Abs(perpNetAmount)
	return cutils.MinBigInt(perpAbsNetAmount, remainingAmount)
}

func (o *OrderTable) IsExpired() bool {
	return cutils.IsTimestampExpiredV2(o.ExpiryTs, 0)
}

func (o *OrderTable) RequireInitialMarginLock() bool {
	return o.IsLimitOrder() && !o.IsReduce
}

func (o *OrderTable) SyncStatusFromTotalFilled() {
	if o.IsMarketOrder() {
		if o.TotalFilledx18.Val.Cmp(o.Amountx18.Val) != 0 {
			o.Status = ctypes.ORDER_STATUS_CANCELLED
		} else {
			o.Status = ctypes.ORDER_STATUS_FILLED
		}
	} else {
		if o.TotalFilledx18.Val.Sign() == 0 {
			o.Status = ctypes.ORDER_STATUS_OPEN
		} else if o.TotalFilledx18.Val.Cmp(o.Amountx18.Val) == 0 {
			o.Status = ctypes.ORDER_STATUS_FILLED
		} else {
			o.Status = ctypes.ORDER_STATUS_PARTIAL
		}
	}
	// Else keep the status as it is (for cancel orders)
}

func (o *OrderTable) IsBuy() bool {
	return o.Side == ctypes.ORDER_SIDE_BUY
}

func (o *OrderTable) IsConditional() bool {
	return o.TriggerCondition == ctypes.TAKE_PROFIT || o.TriggerCondition == ctypes.STOP_LOSS
}

func (o *OrderTable) ConditionFullName() string {
	switch o.TriggerCondition {
	case ctypes.TAKE_PROFIT:
		return "Take Profit"
	case ctypes.STOP_LOSS:
		return "Stop Loss"
	}
	return ""
}

func (o *OrderTable) IsValidTriggerCondition() bool {
	return o.IsMarketOrder() && o.IsConditional() && o.IsReduce
}

func (o *OrderTable) GetTriggerDirection() ctypes.TriggerDirection {
	switch o.TriggerCondition {
	case ctypes.TAKE_PROFIT:
		if o.IsBuy() {
			return ctypes.TRIGGER_DIRECTION_DECR
		}
		return ctypes.TRIGGER_DIRECTION_INCR
	case ctypes.STOP_LOSS:
		if o.IsBuy() {
			return ctypes.TRIGGER_DIRECTION_INCR
		}
		return ctypes.TRIGGER_DIRECTION_DECR
	}
	return ""
}

func (o *OrderTable) IsOpen() bool {
	return o.Status == ctypes.ORDER_STATUS_OPEN
}
