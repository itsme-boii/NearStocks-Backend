package controller

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"github/eugenix-io/logx-inf-backend/services/engine"
	"github/eugenix-io/logx-inf-backend/services/engine/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MessageContoller struct {
}

func RegisterMessageController(
	r *gin.RouterGroup,
) {
	messageController := MessageContoller{}
	r.POST("/message", messageController.HandleMessage)
}

const (
	INVALID_MSG_FORMAT = "INVALID_MSG_FORMAT"
	INVALID_MARKET_ID  = "INVALID_MARKET_ID"
	INVALID_OP         = "INVALID_OP"
	INVALID_TYPE       = "INVALID_TYPE"
)

// TODO: Add proper error handling
func (mc *MessageContoller) HandleMessage(ctx *gin.Context) {
	var requestObj types.EngineRequest
	if err := ctx.BindJSON(&requestObj); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, INVALID_MSG_FORMAT)
		return
	}

	var res types.EngineResponse

	if requestObj.IsConditionalRequest() {
		res = mc.handleConditionalOrderMessage(ctx, requestObj)
	} else if requestObj.IsOrderRequest() {
		res = mc.handleOrderMessage(ctx, requestObj)
	}

	if res == nil {
		xlog.Errorf("Cannot execute OP: %v for party: %s\n", requestObj.OP, requestObj.Party)
		cutils.ApiAbort(ctx, http.StatusBadRequest, INVALID_OP)
	}

	if res.Err() != nil {
		xlog.Errorf("Cannot execute OP: %v for party: %s\n", requestObj.OP, requestObj.Party)
		cutils.ApiAbort(ctx, http.StatusBadRequest, res.Err().Error())
		return
	}

	cutils.ApiSuccess(ctx, res, "")
}

func (*MessageContoller) handleOrderMessage(ctx *gin.Context, requestObj types.EngineRequest) (res types.EngineResponse) {

	orderbook := engine.GlobalEngine.GetOrderbook(requestObj.MarketId)
	if orderbook == nil {
		xlog.Errorf("Orderbook not found for marketId: %d\n", requestObj.MarketId)
		cutils.ApiAbort(ctx, http.StatusBadRequest, INVALID_MARKET_ID)
		return
	}

	cutils.LogByParty(requestObj.Party, "Executing OP: %v for party: %s\n", requestObj.OP, requestObj.Party)

	_msg := requestObj.Msg.(map[string]interface{})

	orderbook.Lock()
	defer orderbook.Unlock()

	switch requestObj.OP {
	case types.ORDER_PLACE_OP:
		var modMsg types.PlaceOrderMsg
		cutils.ConvertMapToStructViaJson(_msg, &modMsg)
		res = orderbook.PlaceOrder(&modMsg)
	case types.ORDER_CANCEL_BULK_OP:
		var modMsg types.CancelOrdersBulkMsg
		cutils.ConvertMapToStructViaJson(_msg, &modMsg)
		res = orderbook.CancelOrdersBulk(&modMsg)
	case types.ORDER_CANCEL_BULK_AND_PLACE_OP:
		var modMsg types.CancelBulkAndPlaceOrderMsg
		cutils.ConvertMapToStructViaJson(_msg, &modMsg)
		res = orderbook.CancelBulkAndPlaceOrder(&modMsg)
	case types.ORDER_CANCEL_BULK_AND_PLACE_BULK_OP:
		var modMsg types.CancelBulkAndPlaceBulkOrderMsg
		cutils.ConvertMapToStructViaJson(_msg, &modMsg)
		res = orderbook.CancelBulkAndPlaceBulkOrder(&modMsg)
	// case types.ORDER_UPDATE_OP:
	// 	var modMsg types.UpdateOrderMsg
	// 	cutils.ConvertMapToStructViaJson(_msg, &modMsg)
	// 	err = orderbook.UpdateOrder(&modMsg)
	default:
		// this will ideally never happen
		xlog.Errorf("Invalid operation. Op: %v", requestObj.OP)
		return nil
	}

	return res
}

func (*MessageContoller) handleConditionalOrderMessage(ctx *gin.Context, requestObj types.EngineRequest) (res types.EngineResponse) {
	orderbook := engine.GlobalEngine.GetTriggerOrderbook(requestObj.MarketId)
	if orderbook == nil {
		xlog.Errorf("Orderbook not found for marketId: %d\n", requestObj.MarketId)
		cutils.ApiAbort(ctx, http.StatusBadRequest, INVALID_MARKET_ID)
		return
	}

	cutils.LogByParty(requestObj.Party, "Executing OP: %v for party: %s\n", requestObj.OP, requestObj.Party)

	_msg := requestObj.Msg.(map[string]interface{})

	orderbook.Lock()
	defer orderbook.Unlock()

	switch requestObj.OP {
	case types.ORDER_PLACE_CONDITIONAL_OP:
		var modMsg types.PlaceConditionalOrderMsg
		cutils.ConvertMapToStructViaJson(_msg, &modMsg)
		res = orderbook.PlaceConditionalOrder(&modMsg)
	case types.ORDER_CANCEL_CONDITIONAL_BULK_OP:
		var modMsg types.CancelConditonalOrdersBulkMsg
		cutils.ConvertMapToStructViaJson(_msg, &modMsg)
		res = orderbook.CancelConditionalOrdersBulk(&modMsg)
	// case types.ORDER_UPDATE_OP:
	// 	var modMsg types.UpdateOrderMsg
	// 	cutils.ConvertMapToStructViaJson(_msg, &modMsg)
	// 	err = orderbook.UpdateOrder(&modMsg)
	default:
		// this will ideally never happen
		xlog.Errorf("Invalid operation. Op: %v", requestObj.OP)
		return nil
	}

	return res
}
