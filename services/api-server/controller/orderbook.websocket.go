package controller

import (
	"encoding/json"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xsocket"
	"github/eugenix-io/logx-inf-backend/services/api-server/client"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	traderOrderBookCache = make(map[uint]*client.ReadableOrderbookLevels)
	solverOrderBookCache = make(map[uint]*client.ReadableOrderbookLevels)
	cacheMutex           = sync.RWMutex{}
)

type OrderBookClient struct {
	marketId uint
	party    ctypes.Party
}
type TradeClient struct {
	marketId uint `uri:"marketId" binding:"required"`
}

type OrderBookWebsocket struct {
	websocket                *xsocket.Websocket
	clientDetails            map[xsocket.XConn]OrderBookClient
	marketPricesClientStatus map[xsocket.XConn]bool
	marketIds                []uint
	tradesClientDetails      map[xsocket.XConn]TradeClient
	clientsMutex             sync.RWMutex
	// Optimization: Add caching fields
	tradesCache      map[uint]*CachedTradeData
	tradesCacheMutex sync.RWMutex
}

type GetOrderbookUri struct {
	MarketId uint `uri:"marketId" binding:"required"`
}

type GetOrderbookQuery struct {
	Party ctypes.Party `form:"party" binding:"oneof=TRADER SOLVER"`
}
type TradeResponse struct {
	Price     string    `json:"price"`
	Size      string    `json:"size"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
}
type GetOrderbookRequest struct {
	*GetOrderbookUri
	*GetOrderbookQuery
}

type CachedTradeData struct {
	Trades    []TradeResponse `json:"trades"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func RegisterOrderbookWebsocket(rg *gin.RouterGroup, wsUpgrader *websocket.Upgrader) {
	orderBookWebsocket := OrderBookWebsocket{
		websocket:                xsocket.NewWebsocket(wsUpgrader, nil),
		clientDetails:            make(map[xsocket.XConn]OrderBookClient),
		marketPricesClientStatus: make(map[xsocket.XConn]bool),
		marketIds:                *(&db.MarketDB{}).GetAllActiveMarketIDs(),
		tradesClientDetails:      make(map[xsocket.XConn]TradeClient),
		tradesCache:              make(map[uint]*CachedTradeData),
	}

	//Initialise tickers
	go orderBookWebsocket.CacheOrderbookForTrader()
	go orderBookWebsocket.BroadcastOrderbook()
	go orderBookWebsocket.BroadcastMarketPrices()
	go orderBookWebsocket.BroadcastTrades()
	// Background data fetcher - updates cache every 5 minutes
	go orderBookWebsocket.BackgroundTradeDataFetcher()
	// Connection cleanup - removes stale connections every 60 seconds
	go orderBookWebsocket.cleanupStaleConnections()

	rg.GET("/orderbook/:marketId", orderBookWebsocket.RegisterOrderbookClient)
	rg.GET("/order/quotes", orderBookWebsocket.RegisterMarketPricesClient)
	rg.GET("/trades/:marketId", orderBookWebsocket.RegisterTradesClient)

	// Register a websocket endpoint
}

// NOTE: We will not upgrade the connect to websocket if validation fail
// TODO: We need to send error reason to client. Currently websocket just closes with unexpected server response
// By default orderbook containing solver data will be broadcasted
func (o *OrderBookWebsocket) RegisterOrderbookClient(ctx *gin.Context) {
	var _requestUri GetOrderbookUri
	var _requestQuery GetOrderbookQuery = GetOrderbookQuery{Party: "SOLVER"}

	if err := ctx.BindUri(&_requestUri); err != nil {
		xlog.Errorf("Orderbook Websocket Error binding orderbook websocket URI: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctx.BindQuery(&_requestQuery); err != nil {
		xlog.Errorf("Orderbook Websocket Error binding orderbook websocket query param: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, err.Error())
		return
	}

	requestObj := GetOrderbookRequest{
		GetOrderbookUri:   &_requestUri,
		GetOrderbookQuery: &_requestQuery,
	}

	if !o.IsValidMarketId(requestObj.MarketId) {
		xlog.Errorf("Orderbook Websocket Invalid market ID: %d", requestObj.MarketId)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid market ID")
		return
	}

	_conn, err := o.websocket.UpgradeWS(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Could not upgrade connection to WebSocket"})
		return
	}

	xConn := xsocket.NewXConn(_conn)
	defer xConn.Close()

	o.websocket.RegisterClient(xConn)
	o.clientsMutex.Lock()
	o.clientDetails[xConn] = OrderBookClient{
		marketId: requestObj.MarketId,
		party:    requestObj.Party,
	}
	o.clientsMutex.Unlock()

	//This code will end up creating one read channel for each use who subscribes to order book
	//	We need to figure out if there is a more graceful way of handling this
	for {
		messageType, _, err := xConn.ReadMessage()
		if err != nil || messageType == websocket.CloseMessage {
			o.UnregisterOrderbookClient(xConn)
			break
		}
	}
}

// NOTE: We will not upgrade the connect to websocket if validation fail
// TODO: We need to send error reason to client. Currently websocket just closes with unexpected server response
// By default orderbook containing solver data will be broadcasted
func (o *OrderBookWebsocket) RegisterMarketPricesClient(ctx *gin.Context) {
	_conn, err := o.websocket.UpgradeWS(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Could not upgrade connection to WebSocket"})
		return
	}
	xConn := xsocket.NewXConn(_conn)
	defer xConn.Close()
	o.websocket.RegisterClient(xConn)
	o.clientsMutex.Lock()
	o.marketPricesClientStatus[xConn] = true
	o.clientsMutex.Unlock()

	//This code will end up creating one read channel for each use who subscribes to order book
	//	We need to figure out if there is a more graceful way of handling this
	for {
		messageType, _, err := xConn.ReadMessage()
		if err != nil || messageType == websocket.CloseMessage {
			o.UnregisterMarketPricesClient(xConn)
			break
		}
	}
}

func (o *OrderBookWebsocket) RegisterTradesClient(ctx *gin.Context) {
	marketIdStr := ctx.Param("marketId")
	if marketIdStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing marketId"})
		return
	}

	marketId64, err := strconv.ParseUint(marketIdStr, 10, 32)
	if err != nil {
		xlog.Errorf("Trades Websocket: Invalid market ID: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid marketId"})
		return
	}

	mkt := uint(marketId64)
	if !o.IsValidMarketId(mkt) {
		xlog.Errorf("Trades Websocket: Invalid market ID: %d", mkt)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid market ID"})
		return
	}

	_conn, err := o.websocket.UpgradeWS(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Could not upgrade connection to WebSocket"})
		return
	}

	xConn := xsocket.NewXConn(_conn)
	defer xConn.Close()

	o.websocket.RegisterClient(xConn)
	o.clientsMutex.Lock()
	o.tradesClientDetails[xConn] = TradeClient{marketId: mkt}
	o.clientsMutex.Unlock()

	for {
		messageType, _, err := xConn.ReadMessage()
		if err != nil || messageType == websocket.CloseMessage {
			o.UnregisterTradesClient(xConn)
			break
		}
	}
}

func (o *OrderBookWebsocket) UnregisterTradesClient(xConn xsocket.XConn) {
	o.websocket.UnregisterClient(xConn)
	o.clientsMutex.Lock()
	delete(o.tradesClientDetails, xConn)
	o.clientsMutex.Unlock()
}

func (o *OrderBookWebsocket) BroadcastTrades() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	heartBeatCnt := 0

	for range ticker.C {
		heartBeatCnt++

		// Create a snapshot of clients to avoid race conditions
		o.clientsMutex.RLock()
		marketConnMap := make(map[uint][]xsocket.XConn)
		for xConn, tClient := range o.tradesClientDetails {
			marketConnMap[tClient.marketId] = append(marketConnMap[tClient.marketId], xConn)
		}
		o.clientsMutex.RUnlock()

		if len(marketConnMap) == 0 {
			continue
		}

		for marketId, conns := range marketConnMap {
			if len(conns) == 0 {
				continue
			}

			// Get trade data from cache
			cachedData := o.getCachedTradeData(marketId)
			if cachedData == nil || len(cachedData.Trades) == 0 {
				continue
			}

			// Broadcast cached data to all connected clients
			o.broadcastToMarketClients(conns, cachedData.Trades)
		}

		// Heartbeat logging
		if heartBeatCnt%30 == 1 {
			o.clientsMutex.RLock()
			totalConnections := len(o.tradesClientDetails)
			o.clientsMutex.RUnlock()
			xlog.Infof("Trades Websocket Heartbeat: %d markets, %d total connections",
				len(marketConnMap), totalConnections)
		}
	}
}

func (o *OrderBookWebsocket) getCachedTradeData(marketId uint) *CachedTradeData {
	if o.tradesCache == nil {
		return nil
	}

	o.tradesCacheMutex.RLock()
	defer o.tradesCacheMutex.RUnlock()

	return o.tradesCache[marketId]
}

func (o *OrderBookWebsocket) broadcastToMarketClients(conns []xsocket.XConn, tradeResponses []TradeResponse) {
	// Pre-serialize the message once for all clients
	messageBytes, err := json.Marshal(tradeResponses)
	if err != nil {
		xlog.Errorf("Failed to marshal trade responses: %v", err)
		return
	}

	for _, xConn := range conns {
		// Check if client is still registered before sending
		o.clientsMutex.RLock()
		_, exists := o.tradesClientDetails[xConn]
		o.clientsMutex.RUnlock()

		if !exists {
			// Client was already removed, skip
			continue
		}

		if err := xConn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			// Only log as warning if it's not a close-related error
			if !isCloseError(err) {
				xlog.Warnf("Failed to send trade data to client: %v", err)
			} else {
				xlog.Infof("Client disconnected: %v", err)
			}

			// Remove the disconnected client from the trades client details
			o.clientsMutex.Lock()
			delete(o.tradesClientDetails, xConn)
			o.clientsMutex.Unlock()
			// Unregister the client from the websocket
			o.websocket.UnregisterClient(xConn)
		}
	}
}

// isCloseError checks if the error is related to connection closure
func isCloseError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "websocket: close sent") ||
		strings.Contains(errStr, "websocket: close received") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset by peer")
}

func (o *OrderBookWebsocket) cleanupStaleConnections() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		o.clientsMutex.Lock()

		// Check trades clients
		for xConn := range o.tradesClientDetails {
			if err := xConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				xlog.Infof("Removing stale trades client: %v", err)
				delete(o.tradesClientDetails, xConn)
				o.websocket.UnregisterClient(xConn)
			}
		}

		// Check orderbook clients
		for xConn := range o.clientDetails {
			if err := xConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				xlog.Infof("Removing stale orderbook client: %v", err)
				delete(o.clientDetails, xConn)
				o.websocket.UnregisterClient(xConn)
			}
		}

		o.clientsMutex.Unlock()
	}
}

// Just to get total connection - monitoring
func (o *OrderBookWebsocket) getTotalConnections(marketConnMap map[uint][]xsocket.XConn) int {
	total := 0
	for _, conns := range marketConnMap {
		total += len(conns)
	}
	return total
}

func (o *OrderBookWebsocket) UnregisterOrderbookClient(xConn xsocket.XConn) {
	o.websocket.UnregisterClient(xConn)
	o.clientsMutex.Lock()
	delete(o.clientDetails, xConn)
	o.clientsMutex.Unlock()
}

func (o *OrderBookWebsocket) UnregisterMarketPricesClient(xConn xsocket.XConn) {
	o.websocket.UnregisterClient(xConn)
	o.clientsMutex.Lock()
	delete(o.marketPricesClientStatus, xConn)
	o.clientsMutex.Unlock()
}
func (o *OrderBookWebsocket) IsValidMarketId(marketId uint) bool {
	for _, id := range o.marketIds {
		if id == marketId {
			return true
		}
	}
	return false
}

func (o *OrderBookWebsocket) CacheOrderbookForTrader() {
	//ToDo - finalise the time interval between two orderbook caches
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	heartBeatCnt := 0

	doWork := func() {
		cacheMutex.Lock()
		depth := uint32(10)
		for _, id := range o.marketIds {
			traderOrderBookCache[id] = client.GlobalEngineClient.GetOrderbookLevels("TRADER", id, depth)
			solverOrderBookCache[id] = client.GlobalEngineClient.GetOrderbookLevels("SOLVER", id, depth)
		}
		cacheMutex.Unlock()

		// Every one minute log a heartbeat
		if heartBeatCnt += 1; heartBeatCnt%60 == 1 {
			xlog.Infof("Orderbook Websocket Heartbeat: Orderbook caching successfully")
		}
	}

	doWork()
	for range ticker.C {
		doWork()
	}
}

func combineOrderbooks(trader, solver *client.ReadableOrderbookLevels) *client.ReadableOrderbookLevels {
	if trader != nil && solver != nil && trader.MarketId != solver.MarketId {
		xlog.Errorf("Mismatched marketIds: trader.MarketId=%d, solver.MarketId=%d", trader.MarketId, solver.MarketId)
		return nil
	}

	if trader == nil && solver == nil {
		return nil
	}

	var combinedAsks []client.ReadableOrderbookLevel
	if trader != nil && trader.Asks != nil {
		combinedAsks = append(combinedAsks, *trader.Asks...)
	}
	if solver != nil && solver.Asks != nil {
		combinedAsks = append(combinedAsks, *solver.Asks...)
	}
	sort.Slice(combinedAsks, func(i, j int) bool {
		priceI, err1 := strconv.ParseFloat(combinedAsks[i].Price, 64)
		priceJ, err2 := strconv.ParseFloat(combinedAsks[j].Price, 64)
		if err1 != nil || err2 != nil {
			return false
		}
		return priceI < priceJ
	})

	var combinedBids []client.ReadableOrderbookLevel
	if trader != nil && trader.Bids != nil {
		combinedBids = append(combinedBids, *trader.Bids...)
	}
	if solver != nil && solver.Bids != nil {
		combinedBids = append(combinedBids, *solver.Bids...)
	}
	sort.Slice(combinedBids, func(i, j int) bool {
		priceI, err1 := strconv.ParseFloat(combinedBids[i].Price, 64)
		priceJ, err2 := strconv.ParseFloat(combinedBids[j].Price, 64)
		if err1 != nil || err2 != nil {
			return false
		}
		return priceI > priceJ
	})

	timestamp := uint64(0)
	if trader != nil && trader.Timestamp > timestamp {
		timestamp = trader.Timestamp
	}
	if solver != nil && solver.Timestamp > timestamp {
		timestamp = solver.Timestamp
	}

	marketId := uint(0)
	if trader != nil {
		marketId = trader.MarketId
	} else if solver != nil {
		marketId = solver.MarketId
	}

	return &client.ReadableOrderbookLevels{
		Asks:      &combinedAsks,
		Bids:      &combinedBids,
		Timestamp: timestamp,
		MarketId:  marketId,
	}
}

func (o *OrderBookWebsocket) BroadcastOrderbook() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	heartBeatCnt := 0

	doWork := func() {
		cacheMutex.RLock()
		o.clientsMutex.RLock()

		for xConn, clientDetail := range o.clientDetails {
			traderOB := traderOrderBookCache[clientDetail.marketId]
			solverOB := solverOrderBookCache[clientDetail.marketId]

			combinedOB := combineOrderbooks(traderOB, solverOB)
			if combinedOB != nil {
				o.websocket.BroadcastMessageToClient(xConn, combinedOB)
			} else {
				xlog.Warnf("Orderbook Websocket - No orderbook data found for market ID: %d", clientDetail.marketId)
			}
		}

		o.clientsMutex.RUnlock()
		cacheMutex.RUnlock()

		// Every one minute log a heartbeat
		if heartBeatCnt += 1; heartBeatCnt%60 == 1 {
			xlog.Infof("Orderbook Websocket Heartbeat: Orderbook broadcasted successfully")
		}
	}

	// Starts at t = 0
	doWork()
	for range ticker.C {
		doWork()
	}
}

func (o *OrderBookWebsocket) BroadcastMarketPrices() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	heartBeatCnt := 0
	var orderbook *client.ReadableOrderbookLevels

	doWork := func() {
		o.clientsMutex.RLock()
		cacheMutex.RLock()
		for xConn, client := range o.marketPricesClientStatus {
			if client {
				for _, id := range o.marketIds {
					//We are only fetching trader prices in the orderbook since this will be broadcasted to users.
					orderbook = solverOrderBookCache[id]

					if orderbook != nil {
						o.websocket.BroadcastMessageToClient(xConn, orderbook)
					} else {
						xlog.Warnf("Orderbook Websocket - No orderbook data found for market ID: %d\n. Disconnecting client", id)
					}
				}
			}
		}

		cacheMutex.RUnlock()
		o.clientsMutex.RUnlock()
		// Every one minute log a heartbeat
		if heartBeatCnt += 1; heartBeatCnt%60 == 1 {
			xlog.Infof("Orderbook Websocket Heartbeat: Orderbook broadcasted successfully")
		}
	}

	// Starts at t = 0
	doWork()
	for range ticker.C {
		doWork()
	}
}

func (o *OrderBookWebsocket) BackgroundTradeDataFetcher() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	// Initial fetch on startup
	o.updateTradeCache()

	for range ticker.C {
		o.updateTradeCache()
	}
}

func (o *OrderBookWebsocket) updateTradeCache() {
	fdb := &db.FillDB{}
	marketIds := o.marketIds

	allMarketFills := fdb.GetRecentFillsByMarketIds(marketIds, 30)

	o.tradesCacheMutex.Lock()
	defer o.tradesCacheMutex.Unlock()

	for marketId, fills := range allMarketFills {
		if fills == nil || len(*fills) == 0 {
			continue
		}

		tradeResponses := make([]TradeResponse, len(*fills))
		for i, fill := range *fills {
			tradeResponses[i] = TradeResponse{
				Price:     fill.Pricex18.String(),
				Size:      fill.Amountx18.String(),
				Side:      string(fill.Side),
				Timestamp: fill.CreatedAt,
			}
		}

		o.tradesCache[marketId] = &CachedTradeData{
			Trades:    tradeResponses,
			UpdatedAt: time.Now(),
		}
	}

	xlog.Infof("Background Trade Data Fetcher: Cache updated for %d markets", len(allMarketFills))
}
