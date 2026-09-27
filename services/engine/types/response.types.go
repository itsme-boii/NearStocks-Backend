package types

import (
	"math/big"
)

type EngineResponse interface {
	Err() error // Fatal errors only
	Self() interface{}
}

type TakerOrderRespData struct {
	ID             uint     `json:"id"`
	TotalFilledx18 *big.Int `json:"totalFilled"`
}

type MakerOrderRespData struct {
	ID               uint     `json:"id"`
	TotalFilledx18   *big.Int `json:"totalFilled"`
	Pricex18         *big.Int `json:"priceQuantum"`
	MakerRealizedPnl *big.Int `json:"makerRealizedPnl"`
	TakerRealizedPnl *big.Int `json:"takerRealizedPnl"`
	MakerFundingFees *big.Int `json:"makerFundingFees"`
	TakerFundingFees *big.Int `json:"takerFundingFees"`
	TxnCounter       uint     `json:"txnCounter"`
	MatchedAmountx18 *big.Int `json:"matchedAmount"`
}

type CancelTakerOrderRespData struct {
	ID           uint   `json:"id"`
	CancelReason string `json:"cancelReason"`
}

type SkipCancelOrderRespData struct {
	ID     uint   `json:"id"`
	Reason string `json:"cancelReason"`
}

type EnginePlaceOrderResponse struct {
	TakerOrder         *TakerOrderRespData
	MatchedMakerOrders *[]MakerOrderRespData
	CancelledOrders    *[]CancelTakerOrderRespData
	Error              error
}

// Validate EnginePlaceOrderResponse implements EngineResponse
var _ EngineResponse = &EnginePlaceOrderResponse{}

func (e *EnginePlaceOrderResponse) Err() error {
	return e.Error
}

func (e *EnginePlaceOrderResponse) Self() interface{} {
	return e
}

type EngineCancelOrdersBulkResponse struct {
	SkippedOrdersWithReason *[]SkipCancelOrderRespData
	CancelledOrders         *[]uint
	Error                   error
}

// Validate EngineCancelOrdersBulkResponse implements EngineResponse
var _ EngineResponse = &EngineCancelOrdersBulkResponse{}

func (e *EngineCancelOrdersBulkResponse) Err() error {
	return e.Error
}

func (e *EngineCancelOrdersBulkResponse) Self() interface{} {
	return e
}

type EngineUpdateOrderResponse struct {
	Error error
}

func (e *EngineUpdateOrderResponse) Err() error {
	return e.Error
}

func (e *EngineUpdateOrderResponse) Self() interface{} {
	return e
}

type EngineCancelBulkAndPlaceOrderResponse struct {
	SkippedCancelOrdersWithReason *[]SkipCancelOrderRespData
	CancelledOrders               *[]CancelTakerOrderRespData
	TakerOrder                    *TakerOrderRespData
	MatchedMakerOrders            *[]MakerOrderRespData
	Error                         error
}

// Validate EngineCancelBulkAndPlaceOrderResponse implements EngineResponse
var _ EngineResponse = &EngineCancelBulkAndPlaceOrderResponse{}

func (e *EngineCancelBulkAndPlaceOrderResponse) Err() error {
	return e.Error
}

func (e *EngineCancelBulkAndPlaceOrderResponse) Self() interface{} {
	return e
}

type EngineCancelBulkAndPlaceBulkOrderResponse struct {
	SkippedCancelOrdersWithReason *[]SkipCancelOrderRespData
	CancelledOrders               *[]CancelTakerOrderRespData
	PlaceOrderResponses           *[]EnginePlaceOrderResponse
	Error                         error
}

// Validate EngineCancelBulkAndPlaceBulkOrderResponse implements EngineResponse
var _ EngineResponse = &EngineCancelBulkAndPlaceBulkOrderResponse{}

func (e *EngineCancelBulkAndPlaceBulkOrderResponse) Err() error {
	return e.Error
}
func (e *EngineCancelBulkAndPlaceBulkOrderResponse) Self() interface{} {
	return e
}

// Conditional order responses
type EnginePlaceConditionalOrderResponse struct {
	Error error
}

func (e *EnginePlaceConditionalOrderResponse) Err() error {
	return e.Error
}

func (e *EnginePlaceConditionalOrderResponse) Self() interface{} {
	return e
}

type EngineCancelConditionalOrdersBulkResponse = EngineCancelOrdersBulkResponse
