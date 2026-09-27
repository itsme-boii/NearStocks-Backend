package types

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"math/big"
)

type TokenBalanceRequest struct {
	SubaccountID string `json:"subaccountID"`
	ProductId    uint32 `json:"productId"`
	TokenBalance string `json:"tokenBalance"`
}

type MultiTokenBalanceRequest struct {
	SubaccountID  string   `json:"subaccountID" binding:"required"`
	ProductIds    []uint32 `json:"productIds" binding:"required"`
	TokenBalances []string `json:"tokenBalances" binding:"required"`
}

type PerpBalanceRequest struct {
	TakerSubaccountID  string `json:"takerSubaccountID" binding:"required"`
	MakerSubaccountID  string `json:"makerSubaccountID" binding:"required"`
	ProductId          uint32 `json:"productId" binding:"required"`
	TakerAmount        string `json:"takerAmount" binding:"required"`
	MakerAmount        string `json:"makerAmount" binding:"required"`
	TakerVQuoteBalance string `json:"takerVQuoteBalance" binding:"required"`
	MakerVQuoteBalance string `json:"makerVQuoteBalance" binding:"required"`
}

type PerpPositionsRequest struct {
	SubaccountID string `form:"subaccountID" binding:"required"`
	ProductIds   string `form:"productIds"`
}

type MappingRequest struct {
	ProductId uint32 `json:"productId" binding:"required"`
	Symbol    string `json:"symbol" binding:"required"`
}

type LockBalanceRequest struct {
	SubaccountId string              `json:"subaccount_id" binding:"required"`
	Entity       ctypes.LockerEntity `json:"entity" binding:"required,oneof=ORDER FUNDING WITHDRAW"`
	EntityId     string              `json:"entity_id" binding:"required"`
	ProductId    uint32              `json:"product_id" binding:"required"`
	LockQuotex18 *big.Int            `json:"lock_quote_x18" binding:"required"`
}

func (r *LockBalanceRequest) Validate() error {
	if r.LockQuotex18 == nil || r.LockQuotex18.Sign() < 0 {
		return fmt.Errorf("amount must be greater than or equal to 0")
	}
	return nil
}

type UnlockAllLockedBalanceRequest struct {
	SubaccountId string `json:"subaccount_id" binding:"required"`
}

type UnlockBalanceRequest struct {
	SubaccountId string              `json:"subaccount_id" binding:"required"`
	Entity       ctypes.LockerEntity `json:"entity" binding:"required,oneof=ORDER FUNDING WITHDRAW"`
	EntityId     string              `json:"entity_id" binding:"required"`
	Partial      bool                `json:"partial"`
	LockQuotex18 *big.Int            `json:"lock_quote_x18"`
}

func (r *UnlockBalanceRequest) Validate() error {
	if r.Partial && r.LockQuotex18 == nil {
		return fmt.Errorf("amount must be provided for partial unlock")
	}

	if r.Partial && r.LockQuotex18.Sign() == -1 {
		return fmt.Errorf("amount must be greater than 0")
	} else if !r.Partial && (r.LockQuotex18 != nil && r.LockQuotex18.Sign() != 0) {
		return fmt.Errorf("amount must be 0 for full unlock")
	}
	return nil
}

type BalancePerpPayload struct {
	SubaccountId     string   `json:"subaccount_id" binding:"required"`
	OrderId          uint     `json:"order_id" binding:"required"`
	UnlockQuotex18   *big.Int `json:"unlock_quote_x18" binding:"required"`
	Amountx18        *big.Int `json:"amount_x18" binding:"required"`
	VQuoteBalancex18 *big.Int `json:"vquote_balance_x18" binding:"required"`
	SkipUnlock       bool     `json:"skip_unlock"`
}

type Balance struct {
	Available string `json:"available"`
	Locked    string `json:"locked"`
}

type PreMarketBalance struct {
	Available string `json:"available"`
}

type SyntheticSpotBalance struct {
	Available string `json:"available"`
}

type UpdateSubaccountForMatchRequest struct {
	TxnCounter    *uint `json:"txn_counter" binding:"required"`
	MarketId      uint  `json:"market_id" binding:"required"`
	IsLiquidation bool  `json:"is_liquidation"`
	Maker         BalancePerpPayload
	Taker         BalancePerpPayload
}

type PerpBalance struct {
	Amount          string `json:"amount"`
	VQuoteBalance   string `json:"vQuoteBalance"`
	LastFundingRate string `json:"lastFundingRate"`
}

type TokenBalance struct {
	Available string `json:"available"`
	Locked    string `json:"locked"`
}

type FinaliseLiquidationResponse struct {
	RealisedLiquidatorPnlx18 *big.Int
	RealisedLiquidateePnlx18 *big.Int
	LiquidatorFundingFeesx18 *big.Int
	LiquidateeFundingFeesx18 *big.Int
}

type FinaliseLiquidationRequest struct {
	TxnCounter             *uint               `json:"txnCounter" binding:"required"`
	LiquidateeOrderId      uint                `json:"liquidateeOrderId"`
	LiquidatorSubaccountId string              `json:"liquidatorSubaccountId"`
	LiquidateeSubaccountId string              `json:"liquidateeSubaccountId"`
	ProductId              uint32              `json:"productId"`
	AmountX18              *big.Int            `json:"amountX18"` // Amount is with respect to liquidatee
	MatchPriceX18          *big.Int            `json:"matchPriceX18"`
	PerpOraclePricesX18    map[uint32]*big.Int `json:"perpOraclePricesX18"`
	SpotOraclePricesX18    map[uint32]*big.Int `json:"spotOraclePricesX18"`
}

type SettleWithInsuranceRequest struct {
	SubaccountId        string              `json:"subaccountId" binding:"required"`
	SpotOraclePricesX18 map[uint32]*big.Int `json:"spotOraclePricesX18" binding:"required"`
}

func (usmr *UpdateSubaccountForMatchRequest) Validate() error {
	if (!usmr.Maker.SkipUnlock && usmr.Maker.UnlockQuotex18.Sign() == -1) || (!usmr.Taker.SkipUnlock && usmr.Taker.UnlockQuotex18.Sign() == -1) {
		return fmt.Errorf("quantity delta must be greater than 0")
	}
	if new(big.Int).Add(usmr.Taker.Amountx18, usmr.Maker.Amountx18).Sign() != 0 {
		return fmt.Errorf("amount should be of same magnitude and opposite sign")
	}
	if new(big.Int).Add(usmr.Taker.VQuoteBalancex18, usmr.Maker.VQuoteBalancex18).Sign() != 0 {
		return fmt.Errorf("vquote balance must be of same magnitude and opposite sign")
	}
	return nil
}

type Health struct {
	BelowInitialMargin     bool `json:"belowInitialMargin"`
	BelowMaintenanceMargin bool `json:"belowMaintenanceMargin"`
	RequireInsurance       bool `json:"requireInsurance"`
}
