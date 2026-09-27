package engine

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/engine/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Orderbook struct {
	market *db.MarketTable
	mutex  *sync.Mutex
	// Separate redis connection
	redisClient   *redis.Client
	balanceClient *xclient.BalanceClient
}

func NewOrderbook(market *db.MarketTable) *Orderbook {
	return &Orderbook{
		market:        market,
		mutex:         &sync.Mutex{},
		redisClient:   xredis.GetRedisClient(),
		balanceClient: xclient.NewBalanceClient(),
	}
}

func (ob *Orderbook) Lock() {
	ob.mutex.Lock()
}

func (ob *Orderbook) Unlock() {
	ob.mutex.Unlock()
}

func (ob *Orderbook) GetMarket() db.MarketTable {
	return *ob.market
}

func (ob *Orderbook) PlaceOrder(msg *types.PlaceOrderMsg) *types.EnginePlaceOrderResponse {
	takerOrder, matchedMakerOrders, cancelledOrders, err := ob.placeOrder(msg.Order)
	if err != nil {
		xlog.Errorf("Error while placing order in engine. Err: %v", err)
	}

	return &types.EnginePlaceOrderResponse{
		TakerOrder:         takerOrder,
		MatchedMakerOrders: &matchedMakerOrders,
		CancelledOrders:    &cancelledOrders,
		Error:              nil,
	}
}

func (ob *Orderbook) CancelOrdersBulk(msg *types.CancelOrdersBulkMsg) *types.EngineCancelOrdersBulkResponse {
	var cancelledOrderIds []uint
	var skippedOrderWithReason []types.SkipCancelOrderRespData
	for _, order := range *msg.Orders {
		err := ob.RemoveOrder(&order)
		if err != nil {
			xlog.Errorf("Error cancelling order: %v | error: %v", order.ID, err)
			skippedOrderWithReason = append(skippedOrderWithReason, types.SkipCancelOrderRespData{ID: order.ID, Reason: "Order not found in orderbook"})
		} else {
			cancelledOrderIds = append(cancelledOrderIds, order.ID)
		}
	}

	return &types.EngineCancelOrdersBulkResponse{
		CancelledOrders:         &cancelledOrderIds,
		SkippedOrdersWithReason: &skippedOrderWithReason,
		Error:                   nil,
	}
}

func (ob *Orderbook) UpdateOrder(msg *types.UpdateOrderMsg) error {
	return ob.UpdateRedisOrder(&msg.OldOrder, &msg.NewOrder)
}

func (ob *Orderbook) CancelBulkAndPlaceOrder(msg *types.CancelBulkAndPlaceOrderMsg) *types.EngineCancelBulkAndPlaceOrderResponse {
	// Cancel orders
	cancelResponse := &types.EngineCancelOrdersBulkResponse{}
	if len(*msg.CancelOrders) > 0 {
		cancelResponse = ob.CancelOrdersBulk(&types.CancelOrdersBulkMsg{Orders: msg.CancelOrders})
	}
	if cancelResponse.Error != nil {
		// Do not Skip place order if cancel orders failed
		xlog.Warnf("Error cancelling orders: %v", cancelResponse.Error)
	}
	cancelResponse.CancelledOrders = cutils.Listify(cancelResponse.CancelledOrders)
	cancelResponse.SkippedOrdersWithReason = cutils.Listify(cancelResponse.SkippedOrdersWithReason)

	placeOrderResponse := ob.PlaceOrder(&types.PlaceOrderMsg{Order: msg.PlaceOrder})

	// Merge cancelled orders from both responses
	// Ideally we should get different order ids from cancel request and place order request
	mergeCancelledOrdersWithReason := make([]types.CancelTakerOrderRespData, 0)

	for _, cancelOrder := range *cancelResponse.CancelledOrders {
		mergeCancelledOrdersWithReason = append(mergeCancelledOrdersWithReason, types.CancelTakerOrderRespData{ID: cancelOrder, CancelReason: "User request"})
	}

	// TODO: Keep a standard error msg type which tells api server to cancel orders even if place order failed
	if placeOrderResponse.Error != nil {
		xlog.Errorf("Place Order Response %v\n", placeOrderResponse)
		return &types.EngineCancelBulkAndPlaceOrderResponse{
			Error:                         placeOrderResponse.Error,
			TakerOrder:                    placeOrderResponse.TakerOrder,
			CancelledOrders:               &mergeCancelledOrdersWithReason,
			SkippedCancelOrdersWithReason: cancelResponse.SkippedOrdersWithReason,
		}
	}
	if placeOrderResponse.CancelledOrders != nil {
		mergeCancelledOrdersWithReason = append(mergeCancelledOrdersWithReason, *placeOrderResponse.CancelledOrders...)
	}

	return &types.EngineCancelBulkAndPlaceOrderResponse{
		SkippedCancelOrdersWithReason: cancelResponse.SkippedOrdersWithReason,
		TakerOrder:                    placeOrderResponse.TakerOrder,
		CancelledOrders:               &mergeCancelledOrdersWithReason,
		MatchedMakerOrders:            placeOrderResponse.MatchedMakerOrders,
		Error:                         nil,
	}
}

// Cancel bulk - Get cancelled orders and skip cancelled orders
// Place bulk - Get maker orders with taker orders and cancelled orders
// Cancelled orders list doesn't include cancelled from place orders
func (ob *Orderbook) CancelBulkAndPlaceBulkOrder(msg *types.CancelBulkAndPlaceBulkOrderMsg) *types.EngineCancelBulkAndPlaceBulkOrderResponse {
	// Cancel orders
	cancelResponse := &types.EngineCancelOrdersBulkResponse{}
	if len(*msg.CancelOrders) > 0 {
		cancelResponse = ob.CancelOrdersBulk(&types.CancelOrdersBulkMsg{Orders: msg.CancelOrders})
	}
	if cancelResponse.Error != nil {
		// Do not Skip place order if cancel orders failed
		xlog.Warnf("Error cancelling orders: %v", cancelResponse.Error)
	}
	cancelResponse.CancelledOrders = cutils.Listify(cancelResponse.CancelledOrders)
	cancelResponse.SkippedOrdersWithReason = cutils.Listify(cancelResponse.SkippedOrdersWithReason)

	// Merge cancelled orders from both responses
	// Ideally we should get different order ids from cancel request and place order request
	cancelledOrdersWithReason := make([]types.CancelTakerOrderRespData, 0)
	for _, cancelOrder := range *cancelResponse.CancelledOrders {
		cancelledOrdersWithReason = append(cancelledOrdersWithReason, types.CancelTakerOrderRespData{ID: cancelOrder, CancelReason: "User request"})
	}

	placeOrderResponses := make([]types.EnginePlaceOrderResponse, 0)
	for _, order := range *msg.PlaceOrders {
		resp := ob.PlaceOrder(&types.PlaceOrderMsg{Order: &order})
		if resp.Error != nil {
			xlog.Errorf("Error placing order: %v | error: %v", order.ID, resp.Error)
			continue
		}
		placeOrderResponses = append(placeOrderResponses, *resp)
	}

	return &types.EngineCancelBulkAndPlaceBulkOrderResponse{
		SkippedCancelOrdersWithReason: cancelResponse.SkippedOrdersWithReason,
		CancelledOrders:               &cancelledOrdersWithReason,
		PlaceOrderResponses:           &placeOrderResponses,
		Error:                         nil,
	}
}
