package types

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
)

type SubpoolTransferStats struct {
	HealthySubaccounts                uint
	BelowInitialMarginSubaccounts     uint
	BelowMaintenanceMarginSubaccounts uint
	NegativeBalanceSubaccounts        uint
}

type SettleWithInsuranceStats struct {
	TotalSettledWithInsurance   uint
	TotalUnsettledWithInsurance uint
}

func (swis *SettleWithInsuranceStats) TotalRecorded() uint {
	return swis.TotalSettledWithInsurance + swis.TotalUnsettledWithInsurance
}

func (swis *SettleWithInsuranceStats) AtleastOneRecorded() bool {
	return swis.TotalSettledWithInsurance > 0 || swis.TotalUnsettledWithInsurance > 0
}

func (ls *LiquidationStats) AtleastOneRecorded() bool {
	return ls.TotalFullyLiquidated > 0 || ls.TotalPartiallyLiquidated > 0 || ls.TotalUnliquidated > 0
}

type LiquidationStats struct {
	TotalFullyLiquidated     uint
	TotalPartiallyLiquidated uint
	TotalUnliquidated        uint
	TotalSkipped             uint
}

func (ls *LiquidationStats) TotalRecorded() uint {
	return ls.TotalFullyLiquidated + ls.TotalPartiallyLiquidated + ls.TotalUnliquidated
}

func (ls *LiquidationStats) Record(fillStatus ctypes.OrderStatus) {
	switch fillStatus {
	case ctypes.ORDER_STATUS_FILLED:
		ls.TotalFullyLiquidated++
	case ctypes.ORDER_STATUS_PARTIAL:
		ls.TotalPartiallyLiquidated++
	case ctypes.ORDER_STATUS_CANCELLED:
		ls.TotalUnliquidated++
	default:
		// This should never happen
		panic("Invalid fill status")
	}
}

func (ls *SettleWithInsuranceStats) Record(success bool) {
	if success {
		ls.TotalSettledWithInsurance++
	} else {
		ls.TotalUnsettledWithInsurance++
	}
}
