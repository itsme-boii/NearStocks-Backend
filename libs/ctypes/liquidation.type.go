package ctypes

import "math/big"

type LiquidationStatsField string

const (
	FULL_LIQUIDATION           LiquidationStatsField = "FULL_LIQUIDATION"
	PARTIAL_LIQUIDATION        LiquidationStatsField = "PARTIAL_LIQUIDATION"
	FAILED_LIQUIDATION         LiquidationStatsField = "FAILED_LIQUIDATION"
	BELOW_MM_ERRORS            LiquidationStatsField = "BELOW_MM_ERRORS"
	BELOW_IM_ERRORS            LiquidationStatsField = "BELOW_IM_ERRORS"
	NEW_SUBACCOUNTS_ERRORS     LiquidationStatsField = "NEW_SUBACCOUNTS_ERRORS"
	HEALTHY_SUBACCOUNTS_ERRORS LiquidationStatsField = "HEALTHY_SUBACCOUNTS_ERRORS"
	NEGATIVE_BALANCE_ERRORS    LiquidationStatsField = "NEGATIVE_BALANCE_ERRORS"
)

var ALL_LIQUIDATION_STATS_FIELDS = []LiquidationStatsField{
	FULL_LIQUIDATION,
	PARTIAL_LIQUIDATION,
	FAILED_LIQUIDATION,
	BELOW_MM_ERRORS,
	BELOW_IM_ERRORS,
	NEW_SUBACCOUNTS_ERRORS,
	HEALTHY_SUBACCOUNTS_ERRORS,
	NEGATIVE_BALANCE_ERRORS,
}

type LiquidationOrderRequest struct {
	SubaccountId  string              `json:"subaccountId" binding:"required"`     // This is the subaccount of trader getting liquididated. It should be in string format <BROKER_ID>_<CHECKSUM_ETH_ADDRESS>_<SUBCCOUNT_NUMBER>
	Party         Party               `json:"party" binding:"oneof=TRADER SOLVER"` // Currently we will be setting this to TRADER as default value
	AmountStr     string              `json:"amount" binding:"required"`           // Amount float string
	MarketId      uint                `json:"marketId" binding:"required"`         // This is same as ProductId. Keeping MarketId for consistency
	IsBuy         *bool               `json:"isBuy" binding:"required"`
	SpotPricesx18 map[uint32]*big.Int `json:"spotPricesx18" binding:"required"`
	PerpPricesx18 map[uint32]*big.Int `json:"perpPricesx18" binding:"required"`
}
