package controller

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xsocket"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/services/api-server/types"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type QuoteWebsocket struct {
	websocket *xsocket.Websocket
}

type SocketHeader struct {
	SignerAddress string `header:"Logx-Signer-Address" binding:"required"`
}

type OrderResponse struct {
	Order                   interface{} `json:"order"`                      // Replace interface{} with the actual type if known
	SkippedOrdersWithReason interface{} `json:"skipped_orders_with_reason"` // Replace interface{} with the actual type if known
	Market                  interface{} `json:"market"`                     // Replace interface{} with the actual type if known
}

func RegisterQuoteWebsocket(rg *gin.RouterGroup, wsUpgrader *websocket.Upgrader) {
	quoteWebsocket := QuoteWebsocket{
		websocket: xsocket.NewWebsocket(wsUpgrader, nil),
	}

	//ToDo - update require signature middleware check to match websocket messages
	rg.GET("/quote", middleware.RequireAuth, middleware.RequireSequencer, quoteWebsocket.RegisterQuoteClient)
}

func (q *QuoteWebsocket) RegisterQuoteClient(ctx *gin.Context) {
	_conn, err := q.websocket.UpgradeWS(ctx)
	xlog.Infof("Quote Websocket - Connection established")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Could not upgrade connection to WebSocket"})
		return
	}

	xConn := xsocket.NewXConn(_conn)
	defer xConn.Close()

	q.websocket.RegisterClient(xConn)

	socketHeader := &SocketHeader{}
	if err := ctx.ShouldBindHeader(socketHeader); err != nil {
		xConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
		return
	}

	orderController := NewOrderController()

	handleMessage := func(message []byte) {
		if isBulkOrderRequest(message) {
			// Bulk order processing
			handleBulkOrderMessage(message, xConn, socketHeader, orderController, ctx)
		} else {
			// Individual order processing
			handleSingleOrderMessage(message, xConn, socketHeader, orderController, ctx)
		}
	}

	for {
		messageType, message, err := xConn.ReadMessage()
		if err != nil {
			q.websocket.UnregisterClient(xConn)
			break
		}

		if messageType == websocket.CloseMessage {
			q.websocket.UnregisterClient(xConn)
			break
		}

		go handleMessage(message)
	}
}

func handleBulkOrderMessage(message []byte, xConn xsocket.XConn, socketHeader *SocketHeader, orderController *OrderController, ctx *gin.Context) {
	// Parse the bulk order request using the same type as the order controller
	var bulkOrderReq CreateBulkOrderRequest
	if err := json.Unmarshal(message, &bulkOrderReq); err != nil {
		xlog.Errorf("Quote Websocket - Error unmarshalling bulk order message: %v\n", err)
		xConn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Invalid bulk order message format"}`))
		return
	}

	// Validate the bulk order structure
	if bulkOrderReq.CreateBulkOrderBody == nil || bulkOrderReq.CreateBulkOrderHeader == nil {
		xConn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Invalid bulk order structure"}`))
		return
	}
	// Set the signer address from the socket header
	if bulkOrderReq.CreateBulkOrderHeader == nil {
		bulkOrderReq.CreateBulkOrderHeader = &types.CreateBulkOrderHeader{}
	}
	bulkOrderReq.SignerAddress = socketHeader.SignerAddress

	// Retrieve the current subaccount
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		xlog.Errorf("Quote Websocket - Error retrieving subaccount: %v\n", err)
		xConn.WriteMessage(websocket.TextMessage, []byte(`{"error":"subaccount not found for the api key"}`))
		return
	}

	response, _, _, err := orderController.ProcessCancellAllAndPlaceBulkOrder(&bulkOrderReq, currentSubaccount)
	if err != nil {
		xlog.Errorf("Quote Websocket - Error processing bulk order: %v\n", err)
		xConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
		return
	}

	// Send the response back to the WebSocket client
	if err := xConn.WriteJSON(response); err != nil {
		xlog.Errorf("Quote Websocket - Error writing JSON response: %v\n", err)
		return
	}
}

func handleSingleOrderMessage(message []byte, xConn xsocket.XConn, socketHeader *SocketHeader, orderController *OrderController, ctx *gin.Context) {
	// Handle message
	var createOrderReq CreateOrderRequest
	createOrderReq.CreateOrderBody = &ctypes.CreateOrderBody{} // Use pointer here

	// Unmarshal message into a temporary map to extract Signature
	var temp map[string]interface{}
	if err := json.Unmarshal(message, &temp); err != nil {
		xlog.Errorf("Quote Websocket - Error unmarshalling message: %v\n", err)
		xConn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Invalid message format"}`))
		return
	}
	// Flag in quote order body to check if we want to place direct solver orders
	var useCancelAll bool
	if val, ok := temp["processCancel"].(bool); ok {
		useCancelAll = val
	}
	delete(temp, "processCancel")

	// Extract Signature and manually set it in CreateOrderHeader
	if sig, ok := temp["signature"].(string); ok {
		createOrderReq.CreateOrderHeader = &CreateOrderHeader{
			Signature:     sig,
			SignerAddress: socketHeader.SignerAddress,
		}
		delete(temp, "signature")
	} else {
		xConn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Signature is required"}`))
		return
	}

	// Re-marshal temp map without signature and unmarshal into CreateOrderBody
	bodyBytes, _ := json.Marshal(temp)
	if err := json.Unmarshal(bodyBytes, createOrderReq.CreateOrderBody); err != nil {
		xlog.Errorf("Quote Websocket - Error unmarshalling message body: %v\n", err)
		xConn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Invalid message format"}`))
		return
	}

	// Retrieve the current subaccount (you might need to adjust this function based on your actual implementation)
	currentSubaccount, err := getCurrentSubaccount(ctx)
	if err != nil {
		xlog.Errorf("Quote Websocket - Error retrieving subaccount: %v\n", err)
		xConn.WriteMessage(websocket.TextMessage, []byte(`{"error":"subaccount not found for the api key"}`))
		return
	}

	var response interface{}
	if useCancelAll {
		order, skippedOrdersWithReason, market, _, err := orderController.ProcessCancellAllAndPlaceOrder(
			&createOrderReq,
			currentSubaccount,
		)
		if err != nil {
			xlog.Errorf("Quote Websocket - Error processing CancelAll: %v\n", err)
			xConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
			return
		}
		response = OrderResponse{
			Order:                   order,
			SkippedOrdersWithReason: skippedOrdersWithReason,
			Market:                  market,
		}
	} else {
		// This is for directly placing the orders for rest of quotes
		order, market, _, _, err := orderController.ProcessCreateOrder(
			&createOrderReq,
			currentSubaccount,
		)
		if err != nil {
			xConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
			return
		}
		response = OrderResponse{
			Order:                   order,
			SkippedOrdersWithReason: nil,
			Market:                  market,
		}
	}

	// Send the response back to the WebSocket client
	if err := xConn.WriteJSON(response); err != nil {
		xlog.Errorf("Quote Websocket - Error writing JSON response: %v\n", err)
		return
	}
}

func isBulkOrderRequest(message []byte) bool {
	var temp map[string]interface{}
	if err := json.Unmarshal(message, &temp); err != nil {
		return false
	}

	// Check if the "orders" field exists and is an array
	if orders, exists := temp["orders"]; exists {
		if ordersArray, ok := orders.([]interface{}); ok && len(ordersArray) > 0 {
			return true
		}
	}

	return false
}
