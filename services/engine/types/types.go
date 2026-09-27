package types

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
)

type OrderOp string

const (
	ORDER_PLACE_OP                      OrderOp = "ORDER_PLACE"
	ORDER_CANCEL_BULK_OP                OrderOp = "ORDER_CANCEL_BULK"
	ORDER_UPDATE_OP                     OrderOp = "ORDER_UPDATE"
	ORDER_CANCEL_BULK_AND_PLACE_OP      OrderOp = "ORDER_CANCEL_BULK_AND_PLACE"
	ORDER_CANCEL_BULK_AND_PLACE_BULK_OP OrderOp = "ORDER_CANCEL_BULK_AND_PLACE_BULK"

	// Conditional order ops
	ORDER_PLACE_CONDITIONAL_OP       OrderOp = "ORDER_PLACE_CONDITIONAL"
	ORDER_CANCEL_CONDITIONAL_BULK_OP OrderOp = "ORDER_CANCEL_CONDITIONAL_BULK"
)

type EngineRequest struct {
	Party    ctypes.Party `json:"party" binding:"required"`
	MarketId uint         `json:"marketId" binding:"required"`
	OP       OrderOp      `json:"op" binding:"required"`
	Msg      interface{}  `json:"msg"`
}

func (e *EngineRequest) IsConditionalRequest() bool {
	return cutils.SliceExists([]OrderOp{ORDER_PLACE_CONDITIONAL_OP, ORDER_CANCEL_CONDITIONAL_BULK_OP}, e.OP)
}

func (e *EngineRequest) IsOrderRequest() bool {
	return cutils.SliceExists([]OrderOp{ORDER_PLACE_OP, ORDER_CANCEL_BULK_OP, ORDER_UPDATE_OP, ORDER_CANCEL_BULK_AND_PLACE_OP, ORDER_CANCEL_BULK_AND_PLACE_BULK_OP}, e.OP)
}
