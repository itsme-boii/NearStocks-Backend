// TODO: Move some logic into service file
// NOTE: For now orders are assumed to be perp orders
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"os"

	"github/eugenix-io/logx-inf-backend/services/api-server/client"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"github/eugenix-io/logx-inf-backend/services/api-server/types"
	"github/eugenix-io/logx-inf-backend/services/cron-server/settlepnl"
	engineTypes "github/eugenix-io/logx-inf-backend/services/engine/types"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var marketStatusService = services.NewMarketStatusService()

type OrderController struct {
	orderService services.OrderService
}

func NewOrderController() *OrderController {
	return &OrderController{
		orderService: services.NewOrderService(),
	}
}

func RegisterOrderController(
	r *gin.RouterGroup,
) {
	OrderController := NewOrderController()

	rg := r.Group("/order")

	// Endpoints
	rg.GET("/:id", middleware.RequireAuth, OrderController.GetOrder)
	rg.GET("", middleware.RequireAuth, OrderController.GetAllOrders)
	rg.GET("/quotes", middleware.RequireAuth, OrderController.GetAllQuotes)
	rg.POST("", middleware.RequireAuth, middleware.RequireSignature, middleware.RequireSequencer, middleware.RequireValidPositionLimits, OrderController.CreateOrder)
	rg.POST("/cancel-all-and-place", middleware.RequireAuth, middleware.RequireSignature, middleware.RequireSequencer, middleware.RequireValidPositionLimits, OrderController.CancelAllAndPlaceOrder)
	rg.PUT("/:id", middleware.RequireAuth, OrderController.UpdateOrder)
	rg.DELETE("/:id", middleware.RequireAuth, OrderController.CancelOrder)
	rg.DELETE("/market/:marketId", middleware.RequireAuth, OrderController.CancelAllOrders)
	rg.POST("/settle-pnl", middleware.RequireSequencer, OrderController.SettlePnlSubaccount)

	// TODO: Add auth check and verify that liquidation engine sent this order
	rg.POST("/liquidation", middleware.RequireSequencer, OrderController.PlaceLiquidationOrder)
	rg.POST("/conditional", middleware.RequireSequencer, OrderController.PlaceConditionalOrder)

	// internal
	rg.GET("/internalQuotes", OrderController.GetAllQuotes)
	rg.GET("/internalCombinedQuotes", OrderController.GetCombinedQuotes)
}

func (*OrderController) GetAllOrders(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	orders := (&db.OrderDB{}).GetAllBySubaccount(currentSubaccount.ID)
	// Get unique markets
	uniqueMarkets := map[uint]*db.MarketTable{}
	for _, order := range *orders {
		if _, ok := uniqueMarkets[order.MarketId]; !ok {
			uniqueMarkets[order.MarketId] = (&db.MarketDB{}).GetById(order.MarketId)
			if uniqueMarkets[order.MarketId] == nil {
				xlog.Errorf("Market not found for id: %v", order.MarketId)
				cutils.ApiAbort(ctx, http.StatusInternalServerError, fmt.Sprintf("Market with order id: %v not found for the order", order.MarketId))
				return
			}
		}
	}

	var ordersResponse []gin.H = []gin.H{}
	for _, order := range *orders {
		ordersResponse = append(ordersResponse, getOrderResponse(&order, uniqueMarkets[order.MarketId]))
	}

	cutils.ApiSuccess(ctx, gin.H{"orders": ordersResponse}, "")
}

type GetOrderRequest struct {
	OrderId uint `uri:"id" binding:"required"`
}

func (*OrderController) GetOrder(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	var requestObj GetOrderRequest
	if err := ctx.BindUri(&requestObj); err != nil {
		xlog.Infof("Order Controller Error: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid url params")
		return
	}

	order := (&db.OrderDB{}).GetBySubaccount_Id(currentSubaccount.ID, requestObj.OrderId)

	if order == nil {
		fmt.Println("Order not found for id: ", requestObj.OrderId)
		cutils.ApiAbort(ctx, http.StatusNotFound, "Order not found for the given id")
		return
	}

	if order.SubaccountId != currentSubaccount.ID {
		cutils.ApiAbort(ctx, http.StatusForbidden, "User does not have access to this order")
		return
	}

	cutils.ApiSuccess(ctx, getOrderResponse(order, (&db.MarketDB{}).GetById(order.MarketId)), "")
}

func (*OrderController) GetAllQuotes(ctx *gin.Context) {
	marketIds := *(&db.MarketDB{}).GetAllActiveMarketIDs()
	var traderOrderBookCache []*client.ReadableOrderbookLevels
	depth := uint32(10) // considering it as harcoded depth for now can be changes later
	for _, id := range marketIds {
		traderOrderBookCache = append(traderOrderBookCache, client.GlobalEngineClient.GetOrderbookLevels("SOLVER", id, depth))
	}

	cutils.ApiSuccess(ctx, traderOrderBookCache, "")
}

func (*OrderController) GetCombinedQuotes(ctx *gin.Context) {
	marketIds := *(&db.MarketDB{}).GetAllActiveMarketIDs()
	var combinedOrderBookCache []*client.ReadableOrderbookLevels
	depth := uint32(50) // Depth set to 50

	for _, id := range marketIds {
		// Fetch SOLVER order book levels
		solverOrderBook := client.GlobalEngineClient.GetOrderbookLevels(ctypes.PARTY_SOLVER, id, depth)
		if solverOrderBook == nil {
			continue
		}

		// Fetch Trader order book levels
		traderOrderBook := client.GlobalEngineClient.GetOrderbookLevels(ctypes.PARTY_TRADER, id, depth)
		if traderOrderBook == nil {
			continue
		}

		// Combine asks and bids from both SOLVER and Trader
		combinedAsks := append(*solverOrderBook.Asks, *traderOrderBook.Asks...)
		combinedBids := append(*solverOrderBook.Bids, *traderOrderBook.Bids...)

		// Add the combined order book to the cache
		combinedOrderBookCache = append(combinedOrderBookCache, &client.ReadableOrderbookLevels{
			MarketId:  id,
			Timestamp: cutils.TimestampMilliNow(),
			Asks:      &combinedAsks,
			Bids:      &combinedBids,
		})
	}

	// Send the combined order book cache as the response
	cutils.ApiSuccess(ctx, combinedOrderBookCache, "")
}

type CreateOrderHeader struct {
	Signature           string `header:"Logx-Signature" `
	SignerAddress       string `header:"Logx-Signer-Address"`
	BrokerSignature     string `header:"Broker-Signature"`
	BrokerSignerAddress string `header:"Broker-Signer-Address"`
	BrokerId            uint   `header:"Broker-Id" binding:"required"`
}

type CreateOrderRequest struct {
	*CreateOrderHeader
	*ctypes.CreateOrderBody
}

func (cor *CreateOrderRequest) ValidateAndGetOrder(subaccount *db.SubaccountTable, market *db.MarketTable) (newOrder *db.OrderTable, status int, err error) {
	if cor.Party == ctypes.PARTY_SOLVER && subaccount.ID != contractUtils.AMM_SUBACCOUNT_ID_1 {
		return nil, http.StatusBadRequest, fmt.Errorf("solver party can only be used by AMM account")
	}

	// Check timestamp
	if cutils.IsTimestampExpiredV2(*cor.ExpiryTs, 0) {
		xlog.Errorf("Timestamp expired - ExpiryTs: %d, CurrentTime: %d", *cor.ExpiryTs, cutils.TimestampMilliNow())
		return nil, http.StatusBadRequest, fmt.Errorf("timestamp expired")
	}

	if cutils.IsTimestampExceeding(*cor.ExpiryTs, db.ORDER_EXPIRY_DURATION+cutils.ALLOWED_NETWORK_DELAY_MILLI) {
		xlog.Errorf("Timestamp exceeding allowed value - ExpiryTs: %d, MaxAllowed: %d", *cor.ExpiryTs, cutils.TimestampMilliNow()+db.ORDER_EXPIRY_DURATION+cutils.ALLOWED_NETWORK_DELAY_MILLI)
		return nil, http.StatusBadRequest, fmt.Errorf("timestamp exceeding allowed value")
	}

	// Conditional order validations
	if cor.TriggerCondition != "" {
		if cor.TriggerCondition != ctypes.TAKE_PROFIT && cor.TriggerCondition != ctypes.STOP_LOSS {
			return nil, http.StatusBadRequest, fmt.Errorf("trigger condition is invalid for conditional order: current trigger condition: %v", cor.TriggerCondition)
		}
		if cor.TriggerPriceStr == "" || cutils.FloatStrToX18(cor.TriggerPriceStr).Sign() <= 0 {
			return nil, http.StatusBadRequest, fmt.Errorf("trigger price is invalid for conditional order: current trigger price: %v", cor.TriggerPriceStr)
		}
		if cor.OrderType != ctypes.ORDER_TYPE_MARKET {
			return nil, http.StatusBadRequest, fmt.Errorf("conditional order can be only of market types: current order type: %v", cor.OrderType)
		}
		if cor.IsReduce == nil || !*cor.IsReduce {
			return nil, http.StatusBadRequest, fmt.Errorf("conditional order should be reduce only: current isReduce: %v", cor.IsReduce)
		}
		// Check if trigger price is multiple of quantums
		if err := cutils.QuantumPrecisionCheck(cor.TriggerPriceStr, market.PriceToQtmConversionExpo); err != nil {
			return nil, http.StatusBadRequest, err
		}
	} else if cor.TriggerPriceStr != "" {
		return nil, http.StatusBadRequest, fmt.Errorf("trigger price is invalid for non conditional order: current trigger price: %v", cor.TriggerPriceStr)
	}

	// Basic validations
	// Check if amount is multiple of quantums
	if err := cutils.QuantumPrecisionCheck(cor.AmountStr, market.AmtToQtmConversionExpo); err != nil {
		return nil, http.StatusBadRequest, err
	}
	// Check if price is multiple of quantums
	if err := cutils.QuantumPrecisionCheck(cor.PriceStr, market.PriceToQtmConversionExpo); err != nil {
		return nil, http.StatusBadRequest, err
	}

	bigIntPricex18 := ctypes.NewBigInt(cutils.FloatStrToX18(cor.PriceStr))
	bigIntAmountx18 := ctypes.NewBigInt(cutils.FloatStrToX18(cor.AmountStr))

	if bigIntPricex18.Sign() < 0 || bigIntAmountx18.Sign() < 0 {
		return newOrder, http.StatusBadRequest, fmt.Errorf("price and amount should be positive")
	}

	if cor.OrderType == ctypes.ORDER_TYPE_LIMIT && bigIntPricex18.Sign() == 0 {
		return newOrder, http.StatusBadRequest, fmt.Errorf("price cannot be 0 for limit order")
	} else if cor.OrderType == ctypes.ORDER_TYPE_MARKET && bigIntPricex18.Sign() != 0 {
		return newOrder, http.StatusBadRequest, fmt.Errorf("price should be 0 for market order")
	}

	// For non reduce only orders amount should be greater than minimum amount
	if bigIntAmountx18.Cmp(market.MinAmountx18) == -1 && (cor.IsReduce == nil || !*cor.IsReduce) {
		return newOrder, http.StatusBadRequest, fmt.Errorf("amount is less than minimum amount. Amountx18: %v, MinAmountx18: %v, ProductId: %v", bigIntAmountx18, market.MinAmountx18, *cor.MarketId)
	}

	if bigIntAmountx18.Sign() <= 0 {
		return newOrder, http.StatusBadRequest, fmt.Errorf("amount should be positive. Current Amounx18t: %v", bigIntAmountx18)
	}

	// NOTE: Loose check
	// TODO: Add proper check for market order
	if cor.Party != ctypes.PARTY_SOLVER {
		notionalx18 := cutils.Divx18(new(big.Int).Mul(bigIntAmountx18.Val, bigIntPricex18.Val))
		if notionalx18.Cmp(market.MaxPositionValuex18.Val) == 1 {
			return newOrder, http.StatusBadRequest, fmt.Errorf("position notional: %v is more than maximum position value: %v", notionalx18, market.MaxPositionValuex18.Val)
		}
	}
	marketOrdersEnabled, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetMarketOrdersEnabledKey()).Result()
	if err == redis.Nil {
		return newOrder, http.StatusBadRequest, fmt.Errorf("market orders are not enabled")
	} else if err != nil {
		return newOrder, http.StatusBadRequest, fmt.Errorf("failed to get market orders enabled status")
	}
	// Check if market is open for US market IDs
	if cor.Party != ctypes.PARTY_SOLVER && contractUtils.US_MARKET_IDS[*cor.MarketId] && os.Getenv("ENV") == "MAINNET" && marketOrdersEnabled == "1" {
		isOpen := marketStatusService.IsMarketOpen()
		if !isOpen {
			return newOrder, http.StatusBadRequest, fmt.Errorf("US market is currently closed. Orders for US market instruments are only allowed during market hours")
		}
	}

	amountx18ForSignature := bigIntAmountx18.Copy().Val
	// If it is a sell order, negate the amount
	if !*cor.IsBuy {
		amountx18ForSignature = amountx18ForSignature.Neg(amountx18ForSignature)
	}

	// Verify signature
	if err := contractUtils.VerifyOrderSignature(contractUtils.OrderSigRequest{
		SubAccountId: subaccount.ID,
		PriceX18:     bigIntPricex18.Val,
		Amount:       amountx18ForSignature,
		Expiration:   *cor.ExpiryTs,
		IsReduce:     *cor.IsReduce,
		SessionKey:   cor.SignerAddress,
		ChainId:      contractUtils.SessionKeyChainId(),
		Signature:    cor.Signature,
		ProductId:    big.NewInt(int64(*cor.MarketId)),
	}); err != nil {
		xlog.Warnf("Signature verification failed - Subaccount: %s, Market: %d, Signature: %s, SignerAddress: %s, Error: %v",
			subaccount.ID, *cor.MarketId, cor.Signature, cor.SignerAddress, err)
		return newOrder, http.StatusBadRequest, fmt.Errorf("invalid signature: %v", err)
	}

	// Check signature not already used
	if (&db.OrderDB{}).IsSignatureUsed(cor.Signature) {
		xlog.Warnf("Signature already used - Signature: %s", cor.Signature)
		return newOrder, http.StatusBadRequest, fmt.Errorf("signature already used")
	}

	newOrder = &db.OrderTable{
		SubaccountId:     subaccount.ID,
		BrokerId:         subaccount.BrokerId,
		MarketId:         *cor.MarketId,
		Side:             ctypes.NewOrderSide(*cor.IsBuy),
		Signature:        cor.Signature,
		Type:             cor.OrderType,
		Amountx18:        bigIntAmountx18,
		Pricex18:         bigIntPricex18,
		TotalFilledx18:   ctypes.NewBigInt(big.NewInt(0)),
		ExpiryTs:         *cor.ExpiryTs,
		Status:           ctypes.ORDER_STATUS_OPEN,
		Party:            cor.Party,
		IsReduce:         *cor.IsReduce,
		SessionKey:       cor.SignerAddress,
		Timestamp:        cutils.TimestampMilliNow(),
		TriggerCondition: cor.TriggerCondition,
		TriggerPricex18:  ctypes.NewBigInt(cutils.FloatStrToX18(cor.TriggerPriceStr)),
	}

	return newOrder, http.StatusOK, nil
}

func (oc *OrderController) CreateOrder(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}
	_reqBody := ctypes.CreateOrderBody{Party: ctypes.PARTY_TRADER}
	_reqHeader := CreateOrderHeader{}

	createOrderReqVal, exists := ctx.Get("createOrderReq")
	if !exists {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Order request body is missing")
		return
	}

	// Type assert the request to the proper struct
	reqBody, ok := createOrderReqVal.(ctypes.CreateOrderBody)
	if !ok {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Failed to cast order request body")
		return
	}

	// Assign the type-asserted value to the existing _reqBody
	_reqBody = reqBody

	if err := ctx.BindHeader(&_reqHeader); err != nil {
		xlog.Infof("Order Controller Error in request header: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request header")
		return
	}

	if _reqHeader.BrokerSignature != "" && _reqHeader.BrokerSignerAddress != "" {
		_reqHeader.Signature = _reqHeader.BrokerSignature
		_reqHeader.SignerAddress = _reqHeader.BrokerSignerAddress
	}

	if _reqHeader.Signature == "" || _reqHeader.SignerAddress == "" {
		xlog.Infof("Order Controller Error: Missing signature or signer address")
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Missing signature or signer address")
		return
	}

	requestObj := CreateOrderRequest{
		CreateOrderHeader: &_reqHeader,
		CreateOrderBody:   &_reqBody,
	}

	order, market, feeBonus, statusCode, err := oc.ProcessCreateOrder(&requestObj, currentSubaccount)
	if err != nil {
		cutils.ApiAbort(ctx, statusCode, err.Error())
		return
	}

	// Build the response with miscellaneous field
	response := getOrderResponseWithMisc(order, market, feeBonus)

	subaccountHex, _ := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if os.Getenv("STOP_SUBACCOUNT_TRACKING") != "1" && cutils.IsSubaccountRegisteredForTracking(subaccountHex) {
		jsonResp, _ := json.Marshal(response)
		go xclient.GlobalDiscordClient.SendWebhookMessage(subaccountHex + " Placed order: " + string(jsonResp))
	}

	cutils.ApiSuccess(ctx, response, "Order created successfully")
}

// TODO: Make sure to take care of MMP within maching engine
// 0. Perform some validations on order - Orderside etc
// 1. Signature verification
// 2. Check Market or Perp pair exists ✅
// 3. Use perp pair data for order object
// 4. Validate user has enough balance to complete the trade: include fee as well
// 5. Update user available balance in web2
// 6. Create the order in db ✅
// 7. Publish new order message for maching engine
func (oc *OrderController) ProcessCreateOrder(requestObj *CreateOrderRequest, currentSubaccount *db.SubaccountTable) (*db.OrderTable, *db.MarketTable, ctypes.BigInt, int, error) {
	market := (&db.MarketDB{}).GetById(*requestObj.MarketId)
	if market == nil {
		return nil, nil, ctypes.NewBigInt(big.NewInt(0)), http.StatusBadRequest, fmt.Errorf(fmt.Sprintf("market %d not found", *requestObj.MarketId))
	}

	var newOrder *db.OrderTable
	newOrder, status, err := requestObj.ValidateAndGetOrder(currentSubaccount, market)
	if err != nil {
		return nil, nil, ctypes.NewBigInt(big.NewInt(0)), status, err
	}

	// Capture FeeBonus along with takerOrder and status
	takerOrder, FeeBonus, status, err := oc.orderService.PlaceOrder(newOrder, market)

	return takerOrder, market, FeeBonus, status, err
}

type UpdateOrderUri struct {
	OrderId uint `uri:"id"`
}
type UpdateOrderBody struct {
	Signature    string `json:"signature"`
	AmtQuantum   uint64 `json:"amtQuantum"`
	PriceQuantum uint64 `json:"priceQuantum"`
	Timestamp    uint   `json:"timestamp"`
}
type UpdateOrderRequest struct {
	*UpdateOrderUri
	*UpdateOrderBody
}

// 1. Get order from db
// 2. Perform checks on order: Status should be open
// 3. Async Send update request to maching engine
// 4. Async Update db
func (*OrderController) UpdateOrder(ctx *gin.Context) {
	var _requestUri UpdateOrderUri
	var _requestBody UpdateOrderBody
	if err := ctx.BindHeader(&_requestUri); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
	}
	if err := ctx.BindJSON(&_requestBody); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
	}

	requestObj := UpdateOrderRequest{
		UpdateOrderUri:  &_requestUri,
		UpdateOrderBody: &_requestBody,
	}

	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	// Get order from db
	order := (&db.OrderDB{}).GetBySubaccount_Id(currentSubaccount.ID, requestObj.OrderId)
	if order == nil {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("order not found"))
		return
	}
	if order.Status != ctypes.ORDER_STATUS_OPEN {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("only open orders can be updated"))
		return
	}

	// TODO: Send message to maching engine
	// TODO: Update db after receiving response from maching engine

}

type CancelOrderRequest struct {
	OrderId uint `uri:"id" binding:"required"`
}

// 1. Get order from db. If not found return 404
// 2. Check if order is open or partially filled. If no return 400
// 3. Send cancel request to matching engine. If error return 500
// 4. If matching engine returns success, update db and return 200
// 5. If matching engine could not cancel the order return 400
func (oc *OrderController) CancelOrder(ctx *gin.Context) {
	var requestObj CancelOrderRequest
	if err := ctx.ShouldBindUri(&requestObj); err != nil {
		xlog.Errorf("Issue in parsing uri variables: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid url params")
		return
	}

	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		xlog.Errorf("Error getting subaccount: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	status, err := oc.orderService.CancelOrderById(currentSubaccount.ID, requestObj.OrderId)
	if err != nil {
		cutils.ApiAbort(ctx, status, err.Error())
		return
	}

	subaccountHex, _ := cutils.SubaccountIdToHex(currentSubaccount.ID)
	if os.Getenv("STOP_SUBACCOUNT_TRACKING") != "1" && cutils.IsSubaccountRegisteredForTracking(subaccountHex) {
		go xclient.GlobalDiscordClient.SendWebhookMessage(subaccountHex + " Cancelled order: " + fmt.Sprintf("%+v", requestObj.OrderId))
	}

	cutils.ApiSuccess(ctx, nil, "Order cancelled successfully")
}

type CancelAllOrdersRequest struct {
	MarketId uint `uri:"marketId" binding:"required"`
}

func (oc *OrderController) CancelAllOrders(ctx *gin.Context) {
	var requestObj CancelAllOrdersRequest
	if err := ctx.ShouldBindUri(&requestObj); err != nil {
		xlog.Errorf("Issue in parsing uri variables: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid url params")
		return
	}

	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	resp, status, err := oc.orderService.CancelAllOrders(currentSubaccount.ID, requestObj.MarketId)
	if err != nil {
		cutils.ApiAbort(ctx, status, err.Error())
		return
	}

	// TODO: Perform error handling
	cutils.ApiSuccess(ctx, gin.H{"cancelledOrderIds": resp.CancelledOrders, "skippedOrdersWithReason": resp.SkippedOrdersWithReason}, "")
}

func (oc *OrderController) CancelAllAndPlaceOrder(ctx *gin.Context) {
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "subaccount not found for the api key")
		return
	}

	_reqBody := ctypes.CreateOrderBody{Party: ctypes.PARTY_TRADER}
	_reqHeader := CreateOrderHeader{}

	if err := ctx.BindJSON(&_reqBody); err != nil {
		xlog.Errorf("Error in request body: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := ctx.BindHeader(&_reqHeader); err != nil {
		xlog.Errorf("Error in request header: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request header")
		return
	}

	if _reqHeader.BrokerSignature != "" && _reqHeader.BrokerSignerAddress != "" {
		_reqHeader.Signature = _reqHeader.BrokerSignature
		_reqHeader.SignerAddress = _reqHeader.BrokerSignerAddress
	}

	requestObj := CreateOrderRequest{
		CreateOrderHeader: &_reqHeader,
		CreateOrderBody:   &_reqBody,
	}

	takerOrder, skippedOrdersWithReason, market, status, err := oc.ProcessCancellAllAndPlaceOrder(&requestObj, currentSubaccount)
	if err != nil {
		cutils.ApiAbort(ctx, status, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, gin.H{"takerOrder": getOrderResponse(takerOrder, market), "skippedCancelOrdersWithReason": skippedOrdersWithReason}, "Orders cancelled and new order placed successfully")
}

func (oc *OrderController) ProcessCancellAllAndPlaceOrder(requestObj *CreateOrderRequest, currentSubaccount *db.SubaccountTable) (*db.OrderTable, *[]engineTypes.SkipCancelOrderRespData, *db.MarketTable, int, error) {
	market := (&db.MarketDB{}).GetById(*requestObj.MarketId)
	if market == nil {
		return nil, nil, nil, http.StatusBadRequest, fmt.Errorf("market not found")
	}

	var newOrder *db.OrderTable
	newOrder, status, err := requestObj.ValidateAndGetOrder(currentSubaccount, market)
	if err != nil {
		xlog.Errorf("Error validating order: %v", err)
		return nil, nil, nil, status, err
	}

	takerOrder, skippedOrdersWithReason, status, err := oc.orderService.CancelAllAndPlaceOrder(currentSubaccount.ID, newOrder, market)
	if err != nil {
		return nil, nil, nil, status, err
	}

	return takerOrder, skippedOrdersWithReason, market, status, err
}

type CreateBulkOrderRequest struct {
	*types.CreateBulkOrderHeader
	*types.CreateBulkOrderBody
}

func getCreateOrderBodyToDbOrderTable(createOrderBody *ctypes.CreateOrderBody, subaccount *db.SubaccountTable, market *db.MarketTable) db.OrderTable {
	if createOrderBody == nil || subaccount == nil || market == nil {
		return db.OrderTable{}
	}

	order := db.OrderTable{
		SubaccountId:   subaccount.ID,
		BrokerId:       subaccount.BrokerId,
		MarketId:       *createOrderBody.MarketId,
		Side:           ctypes.NewOrderSide(*createOrderBody.IsBuy),
		Type:           createOrderBody.OrderType,
		Amountx18:      ctypes.NewBigInt(cutils.FloatStrToX18(createOrderBody.AmountStr)),
		Pricex18:       ctypes.NewBigInt(cutils.FloatStrToX18(createOrderBody.PriceStr)),
		TotalFilledx18: ctypes.NewBigInt(big.NewInt(0)),
		ExpiryTs:       *createOrderBody.ExpiryTs,
		Status:         ctypes.ORDER_STATUS_OPEN,
		Timestamp:      cutils.TimestampMilliNow(),
		IsReduce:       *createOrderBody.IsReduce,
		SessionKey:     "",
	}

	return order
}

// Iterate through all the orders and validate them
func (cor *CreateBulkOrderRequest) ValidateAndGetOrder(subaccount *db.SubaccountTable, market *db.MarketTable) (validOrders []db.OrderTable, rejectedOrders []types.RejectedOrder, status int, err error) {
	validOrders = make([]db.OrderTable, 0)
	rejectedOrders = make([]types.RejectedOrder, 0)

	for _, order := range cor.Orders {
		reqOrder := CreateOrderRequest{
			CreateOrderHeader: &CreateOrderHeader{
				Signature:     order.Signature,
				SignerAddress: cor.SignerAddress,
			},
			CreateOrderBody: order.Details,
		}

		// check if order has valid market id
		if market.ID != *order.Details.MarketId {
			rejectedOrders = append(rejectedOrders, types.RejectedOrder{
				Order:  getCreateOrderBodyToDbOrderTable(order.Details, subaccount, market),
				Reason: fmt.Sprintf("MarketId mismatch. Expected: %d, Got: %d", market.ID, *order.Details.MarketId),
			})
			continue
		}

		// If order side is different then reject all orders
		if order.Details.IsBuy != nil && *order.Details.IsBuy != *cor.IsBuy {
			rejectedOrders = append(rejectedOrders, types.RejectedOrder{
				Order:  getCreateOrderBodyToDbOrderTable(order.Details, subaccount, market),
				Reason: fmt.Sprintf("Order side mismatch. Expected: %v, Got: %v", cor.IsBuy, *order.Details.IsBuy),
			})
			continue
		}

		newOrder, status, err := reqOrder.ValidateAndGetOrder(subaccount, market)
		if err != nil {
			xlog.Warnf("Bulk order validation failed - Market: %d, Side: %v, Price: %s, Amount: %s, ExpiryTs: %d, Error: %v",
				*order.Details.MarketId, *order.Details.IsBuy, order.Details.PriceStr, order.Details.AmountStr, *order.Details.ExpiryTs, err)
			rejectedOrders = append(rejectedOrders, types.RejectedOrder{
				Order:  getCreateOrderBodyToDbOrderTable(order.Details, subaccount, market),
				Reason: fmt.Sprintf("err: %v | status: %d", err.Error(), status),
			})
		} else {
			validOrders = append(validOrders, *newOrder)
		}
	}
	if len(validOrders) == 0 {
		return nil, nil, http.StatusBadRequest, fmt.Errorf("All invalid orders")
	}
	return validOrders, rejectedOrders, http.StatusOK, nil
}

// OrderRequest will contain all the place orders
// OrderResponse will return all the orders, cancelled orders and skipped cancelled orders.
// Currently restricted to AMM only
// Always return status ok as long as matching engine is able to process the request
func (oc *OrderController) ProcessCancellAllAndPlaceBulkOrder(requestObj *CreateBulkOrderRequest, currentSubaccount *db.SubaccountTable) (_resp *types.CreateBulkOrderResponse, _market *db.MarketTable, _statusCode int, _err error) {
	market := (&db.MarketDB{}).GetById(*requestObj.MarketId)
	if market == nil {
		return nil, nil, http.StatusBadRequest, fmt.Errorf("market not found")
	}

	validOrders, rejectedOrders, status, err := requestObj.ValidateAndGetOrder(currentSubaccount, market)
	if err != nil {
		xlog.Errorf("Error validating order: %v | Valid orders: %d | Rejected orders: %d | Market: %v", err, len(validOrders), len(rejectedOrders), market.ID)
		return &types.CreateBulkOrderResponse{
			Orders:         &validOrders, // This will be empty
			RejectedOrders: rejectedOrders,
		}, market, status, err
	}

	takerOrders, rejectedOrders, status, err := oc.orderService.CancelAllAndPlaceBulkOrder(currentSubaccount.ID, &validOrders, market, *requestObj.IsBuy)
	resp := &types.CreateBulkOrderResponse{
		Orders:         takerOrders,
		RejectedOrders: rejectedOrders,
	}
	return resp, market, status, err
}

func (oc *OrderController) SettlePnlSubaccount(ctx *gin.Context) {

	requestObj := types.SettleSubaccountPnl{}

	if err := ctx.BindJSON(&requestObj); err != nil {
		xlog.Errorf("Error in request body: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	var subAccounts []string
	subAccounts = append(subAccounts, requestObj.SubaccountId)

	settlepnl.SettlePnlJob(subAccounts)
	xlog.Infof("Settled pnl succesfully for subaccount: %v", requestObj.SubaccountId)
	cutils.ApiSuccess(ctx, gin.H{"response": "settled pnl succesfully"}, "")
}

func (oc *OrderController) PlaceLiquidationOrder(ctx *gin.Context) {
	// Setting default value for party
	requestObj := types.LiquidationOrderRequest{Party: ctypes.PARTY_TRADER}

	if err := ctx.BindJSON(&requestObj); err != nil {
		xlog.Errorf("Error in request body: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	subacountHex, err := cutils.SubaccountIdToHex(requestObj.SubaccountId)
	if err != nil {
		xlog.Errorf("Error converting subaccount id to hex: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid subaccount id")
		return
	}

	if contractUtils.IsAMMAccount(subacountHex) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "AMM account cannot be liquidated")
		return
	}

	if cutils.IsSubAccountPaused(subacountHex) {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Subaccount is paused")
		return
	}

	// Get Subaccount to verify that it exists
	subaccount := (&db.SubaccountDB{}).GetById(requestObj.SubaccountId)
	if subaccount == nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Subaccount with id: %v not found", requestObj.SubaccountId))
		return
	}

	// Get Market to verify that it exists
	market := (&db.MarketDB{}).GetById(requestObj.MarketId)
	if market == nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, fmt.Sprintf("Market with id: %v not found", requestObj.MarketId))
		return
	}

	liquidationOrder, status, err := requestObj.ValidateAndGetOrder(subaccount, market)
	if err != nil {
		xlog.Errorf("Error validating liquidation order: %v", err)
		cutils.ApiAbort(ctx, status, err.Error())
		return
	}

	taker, status, err := oc.orderService.PlaceLiquidationOrder(liquidationOrder, market, requestObj.SpotPricesx18, requestObj.PerpPricesx18)
	if err != nil {
		cutils.ApiAbort(ctx, status, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, getOrderResponsev2(taker, market), "Liquidation order placed successfully")
}

func (oc *OrderController) PlaceConditionalOrder(ctx *gin.Context) {
	var requestObj types.ConditionalOrderRequest

	if err := ctx.BindJSON(&requestObj); err != nil {
		xlog.Errorf("Error in request body for PlaceConditionalOrder: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get order from db
	conditionalOrder := (&db.OrderDB{}).GetById(requestObj.Id)
	if conditionalOrder == nil {
		xlog.Errorf("Order not found for id for placing conditional order: %v", requestObj.Id)
		cutils.ApiAbort(ctx, http.StatusNotFound, "Order not found for the given id")
		return
	}

	if !conditionalOrder.IsValidTriggerCondition() {
		xlog.Errorf("Order is not a conditional order")
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Order is not a conditional order")
		return
	}

	// Currently, we are only allowing conditional orders to be placed if they are open
	if !conditionalOrder.IsOpen() {
		xlog.Errorf("Only open conditional orders can be placed")
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Only open conditional orders can be placed")
		return
	}

	subacountHex, err := cutils.SubaccountIdToHex(conditionalOrder.SubaccountId)
	if err != nil {
		xlog.Errorf("Error converting subaccount id to hex: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid subaccount id")
		return
	}

	if cutils.IsSubAccountPaused(subacountHex) {
		xlog.Errorf("Subaccount is paused. Cannot place conditional order")
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Subaccount is paused")
		return
	}

	market := (&db.MarketDB{}).GetById(conditionalOrder.MarketId)
	if market == nil {
		xlog.Errorf("Market not found for id: %v", conditionalOrder.MarketId)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, fmt.Sprintf("Market with order id: %v not found for the order", conditionalOrder.MarketId))
		return
	}

	taker, status, err := oc.orderService.PlaceConditionalOrder(conditionalOrder, market)
	if err != nil {
		cutils.ApiAbort(ctx, status, err.Error())
		return
	}

	cutils.ApiSuccess(ctx, getOrderResponsev2(taker, market), "Liquidation order placed successfully")
}
