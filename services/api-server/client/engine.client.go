package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/engine"
	engineTypes "github/eugenix-io/logx-inf-backend/services/engine/types"
	"net/http"
	"os"
)

type EngineClient struct {
	orderbooks map[uint]*engine.Orderbook
}

var GlobalEngineClient *EngineClient

func InitEngineClient() {
	GlobalEngineClient = &EngineClient{}
	xlog.Infof("Engine Client - Initializing lite engine")
	markets := (&db.MarketDB{}).GetAllPerpPairs()
	obs := map[uint]*engine.Orderbook{}
	for _, market := range *markets {
		obs[market.ID] = engine.NewOrderbook(&market)
	}
	GlobalEngineClient = &EngineClient{orderbooks: obs}
	xlog.Infof("Engine client - Initialized")
}

type ReadableOrderbookLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

type ReadableOrderbookLevels struct {
	Asks      *[]ReadableOrderbookLevel `json:"asks"`
	Bids      *[]ReadableOrderbookLevel `json:"bids"`
	Timestamp uint64                    `json:"timestamp"`
	MarketId  uint                      `json:"marketId"`
}

type PlaceOrderResponseObject struct {
	Body    *engineTypes.EnginePlaceOrderResponse `json:"body"`
	Message string                                `json:"message"`
	Status  int                                   `json:"status"`
}

type PlaceConditionalOrderResponseObject struct {
	Body    *engineTypes.EnginePlaceConditionalOrderResponse `json:"body"`
	Message string                                           `json:"message"`
	Status  int                                              `json:"status"`
}

type CancelOrderResponseObject struct {
	Body    *engineTypes.EngineCancelOrdersBulkResponse `json:"body"`
	Message string                                      `json:"message"`
	Status  int                                         `json:"status"`
}

type UpdateOrderResponseObject struct {
	Body    *engineTypes.EngineUpdateOrderResponse `json:"body"`
	Message string                                 `json:"message"`
	Status  int                                    `json:"status"`
}

type CancelOrderBulkAndPlaceOrderResponseObject struct {
	Body    *engineTypes.EngineCancelBulkAndPlaceOrderResponse `json:"body"`
	Message string                                             `json:"message"`
	Status  int                                                `json:"status"`
}

type CancelOrderBulkAndPlaceOrderBulkResponseObject struct {
	Body    *engineTypes.EngineCancelBulkAndPlaceBulkOrderResponse `json:"body"`
	Message string                                                 `json:"message"`
	Status  int                                                    `json:"status"`
}

// This will return human readable prices and amounts
func (e *EngineClient) GetOrderbookLevels(party ctypes.Party, marketId uint, depth uint32) *ReadableOrderbookLevels {
	// Get orderbook levels from engine
	orderbook, ok := GlobalEngineClient.orderbooks[marketId]
	if !ok {
		xlog.Warnf("engine client - orderbook not found for market=%d", marketId)
		return nil
	}

	fob, err := orderbook.GetFullOrderBookLevels(party, depth)
	if err != nil {
		xlog.Errorf("Engine Client - Error getting orderbook levels: ", err)
		return nil
	}

	market := orderbook.GetMarket()
	return ConvertFullOrderbookToReadable(fob, &market)
}

func (e *EngineClient) GetImpactPrices(marketId uint) *engine.ImpactPrices {
	orderbook, ok := GlobalEngineClient.orderbooks[marketId]
	if !ok {
		xlog.Warnf("engine client - orderbook not found for market=%d", marketId)
		return nil
	}
	// FIXME: ImpactNotional should come from db
	return orderbook.GetImpactPrices(ctypes.PARTY_TRADER, "10000")
}

func (e *EngineClient) PlaceOrder(order *db.OrderTable) *engineTypes.EnginePlaceOrderResponse {
	// Place order in engine
	cutils.LogByParty(order.Party, "Sending PlaceOrder message to matching engine for marketId %d and party %s", order.MarketId, order.Party)
	var response PlaceOrderResponseObject
	err := syncSendMessage(order.MarketId, order.Party, engineTypes.PlaceOrderMsg{Order: order}, engineTypes.ORDER_PLACE_OP, &response)
	if err != nil {
		return &engineTypes.EnginePlaceOrderResponse{Error: err}
	}
	// Listify the responses body fields for consistency
	response.Body.CancelledOrders = cutils.Listify(response.Body.CancelledOrders)
	response.Body.MatchedMakerOrders = cutils.Listify(response.Body.MatchedMakerOrders)
	return response.Body
}

func (e *EngineClient) PlaceConditionalOrder(order *db.OrderTable) *engineTypes.EnginePlaceConditionalOrderResponse {
	// Place order in engine
	cutils.LogByParty(order.Party, "Sending PlaceConditionalOrder message to matching engine for marketId %d and party %s", order.MarketId, order.Party)
	var response PlaceConditionalOrderResponseObject
	err := syncSendMessage(order.MarketId, order.Party, engineTypes.PlaceConditionalOrderMsg{Order: order}, engineTypes.ORDER_PLACE_CONDITIONAL_OP, &response)
	if err != nil {
		return &engineTypes.EnginePlaceConditionalOrderResponse{Error: err}
	}
	return response.Body
}

func (e *EngineClient) CancelOrdersBulk(orders *[]db.OrderTable) *engineTypes.EngineCancelOrdersBulkResponse {
	if len(*orders) == 0 {
		return &engineTypes.EngineCancelOrdersBulkResponse{Error: fmt.Errorf("engine client - no orders to cancel")}
	}
	xlog.Debugf("Sending CancelOrdersBulk message to matching engine for marketId %d and party %s", (*orders)[0].MarketId, (*orders)[0].Party)
	var response CancelOrderResponseObject
	err := syncSendMessage((*orders)[0].MarketId, (*orders)[0].Party, engineTypes.CancelOrdersBulkMsg{Orders: orders}, engineTypes.ORDER_CANCEL_BULK_OP, &response)
	if err != nil {
		return &engineTypes.EngineCancelOrdersBulkResponse{Error: err}
	}

	// Listify the responses body fields for consistency
	response.Body.CancelledOrders = cutils.Listify(response.Body.CancelledOrders)
	response.Body.SkippedOrdersWithReason = cutils.Listify(response.Body.SkippedOrdersWithReason)
	return response.Body
}

func (e *EngineClient) CancelConditionalOrdersBulk(orders *[]db.OrderTable) *engineTypes.EngineCancelConditionalOrdersBulkResponse {
	if len(*orders) == 0 {
		return &engineTypes.EngineCancelConditionalOrdersBulkResponse{Error: fmt.Errorf("engine client - no orders to cancel")}
	}
	xlog.Debugf("Sending CancelConditionalOrdersBulk message to matching engine for marketId %d and party %s", (*orders)[0].MarketId, (*orders)[0].Party)
	var response CancelOrderResponseObject
	err := syncSendMessage((*orders)[0].MarketId, (*orders)[0].Party, engineTypes.CancelConditonalOrdersBulkMsg{Orders: orders}, engineTypes.ORDER_CANCEL_CONDITIONAL_BULK_OP, &response)
	if err != nil {
		return &engineTypes.EngineCancelConditionalOrdersBulkResponse{Error: err}
	}

	// Listify the responses body fields for consistency
	response.Body.CancelledOrders = cutils.Listify(response.Body.CancelledOrders)
	response.Body.SkippedOrdersWithReason = cutils.Listify(response.Body.SkippedOrdersWithReason)
	return response.Body
}

func (e *EngineClient) CancelBulkAndPlaceOrder(cancelOrders *[]db.OrderTable, placeOrder *db.OrderTable) *engineTypes.EngineCancelBulkAndPlaceOrderResponse {
	xlog.Debugf("Sending CancelBulkAndPlaceOrder message to matching engine for marketId %d and party %s", placeOrder.MarketId, placeOrder.Party)
	var response CancelOrderBulkAndPlaceOrderResponseObject
	err := syncSendMessage(placeOrder.MarketId, placeOrder.Party, engineTypes.CancelBulkAndPlaceOrderMsg{CancelOrders: cancelOrders, PlaceOrder: placeOrder}, engineTypes.ORDER_CANCEL_BULK_AND_PLACE_OP, &response)
	if err != nil {
		return &engineTypes.EngineCancelBulkAndPlaceOrderResponse{Error: err}
	}

	// Listify the responses body fields for consistency
	response.Body.CancelledOrders = cutils.Listify(response.Body.CancelledOrders)
	response.Body.SkippedCancelOrdersWithReason = cutils.Listify(response.Body.SkippedCancelOrdersWithReason)
	response.Body.MatchedMakerOrders = cutils.Listify(response.Body.MatchedMakerOrders)
	return response.Body
}

func (e *EngineClient) CancelBulkAndPlaceBulkOrder(cancelOrders *[]db.OrderTable, placeOrders *[]db.OrderTable) *engineTypes.EngineCancelBulkAndPlaceBulkOrderResponse {
	xlog.Debugf("Sending CancelBulkAndPlaceBulkOrder message to matching engine for marketId %d and party %s", (*placeOrders)[0].MarketId, (*placeOrders)[0].Party)
	var response CancelOrderBulkAndPlaceOrderBulkResponseObject
	err := syncSendMessage((*placeOrders)[0].MarketId, (*placeOrders)[0].Party, engineTypes.CancelBulkAndPlaceBulkOrderMsg{CancelOrders: cancelOrders, PlaceOrders: placeOrders}, engineTypes.ORDER_CANCEL_BULK_AND_PLACE_BULK_OP, &response)
	if err != nil {
		return &engineTypes.EngineCancelBulkAndPlaceBulkOrderResponse{Error: err}
	}
	// Listify the responses body fields for consistency
	response.Body.CancelledOrders = cutils.Listify(response.Body.CancelledOrders)
	response.Body.SkippedCancelOrdersWithReason = cutils.Listify(response.Body.SkippedCancelOrdersWithReason)
	response.Body.PlaceOrderResponses = cutils.Listify(response.Body.PlaceOrderResponses)
	return response.Body
}

func (e *EngineClient) UpdateOrder(oldOrder *db.OrderTable, newOrder *db.OrderTable) *engineTypes.EngineUpdateOrderResponse {
	// Update order in engine
	if oldOrder.Party != newOrder.Party {
		return &engineTypes.EngineUpdateOrderResponse{Error: fmt.Errorf("engine client - party mismatch")}
	}
	cutils.LogByParty(oldOrder.Party, "Sending UpdateOrder message to matching engine for marketId %d and party %s", newOrder.MarketId, oldOrder.Party)
	var response UpdateOrderResponseObject
	err := syncSendMessage(newOrder.MarketId, oldOrder.Party, engineTypes.UpdateOrderMsg{OldOrder: *oldOrder, NewOrder: *newOrder}, engineTypes.ORDER_UPDATE_OP, &response)
	if err != nil {
		return &engineTypes.EngineUpdateOrderResponse{Error: err}
	}
	return response.Body
}

func syncSendMessage[T interface{}](marketId uint, party ctypes.Party, msg interface{}, op engineTypes.OrderOp, target *T) error {
	msgBytes, err := json.Marshal(engineTypes.EngineRequest{Msg: msg, OP: op, Party: party, MarketId: marketId})
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf("%v/api/v1/message", os.Getenv("ENGINE_URL")), "application/json", bytes.NewBuffer(msgBytes))

	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("engine client - matching engine POST response code: %d", res.StatusCode)
	}

	cutils.LogByParty(party, "Engine Client - matching engine POST response code : %d", res.StatusCode)

	err = json.NewDecoder(res.Body).Decode(target)
	if err != nil {
		return err
	}
	return nil
}

func ConvertFullOrderbookToReadable(fob *engine.FullOrderbook, market *db.MarketTable) *ReadableOrderbookLevels {
	asks := make([]ReadableOrderbookLevel, len(fob.Asks))
	for i, ask := range fob.Asks {
		priceFloatStr := cutils.QuantumToFloatStr(ask.PriceQuantum, market.PriceToQtmConversionExpo)
		quantityFloatStr := cutils.QuantumToFloatStr(ask.QuantityQuantum, market.AmtToQtmConversionExpo)

		asks[i] = ReadableOrderbookLevel{
			Price:    priceFloatStr,
			Quantity: quantityFloatStr,
		}
	}

	bids := make([]ReadableOrderbookLevel, len(fob.Bids))
	for i, bid := range fob.Bids {
		priceFloatStr := cutils.QuantumToFloatStr(bid.PriceQuantum, market.PriceToQtmConversionExpo)
		quantityFloatStr := cutils.QuantumToFloatStr(bid.QuantityQuantum, market.AmtToQtmConversionExpo)

		bids[i] = ReadableOrderbookLevel{
			Price:    priceFloatStr,
			Quantity: quantityFloatStr,
		}
	}

	return &ReadableOrderbookLevels{
		Asks:      &asks,
		Bids:      &bids,
		Timestamp: fob.Timestamp,
		MarketId:  fob.MarketId,
	}
}
