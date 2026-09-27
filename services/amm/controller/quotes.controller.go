package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/services/amm/config"
	"github/eugenix-io/logx-inf-backend/services/amm/utils"
	"github/eugenix-io/logx-inf-backend/xclient"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/libs/xsocket"
	"github/eugenix-io/logx-inf-backend/services/api-server/types"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

/*
Environment Variables:
- AMM_USE_BULK_QUOTES: Set to "1" to use bulk quote processing, otherwise uses individual quote processing.
  Note: API server dynamically detects message type, so no synchronization required between services.
- AMM_QUOTE_INTERVAL: Interval between quote cycles in seconds (default: 30)
- AMM_QUOTE_COUNT: Number of quotes per side (default: 10)
- AMM_QUOTE_SIZE_SCALE_FACTOR: Scale factor for quote sizes (default: 2.0)
- AMM_LOG_SAMPLE_RATE: Log sampling rate (default: 100)
- RFQ_EXPIRY_MILLIS: Quote expiry time in milliseconds (default: 60000)
*/

var (
	logCounter    int64 = 0
	logSampleRate int64 = 100 // Log 1 out of every 100 by default
)

var (
	expectedResponses int64
	receivedResponses int64
	cycleStartTime    int64
	totalCycles       int64
	totalDuration     int64
	avgDuration       int64
)

func logAverageTiming() {
	cycles := atomic.LoadInt64(&totalCycles)
	if cycles > 0 {
		total := atomic.LoadInt64(&totalDuration)
		avg := atomic.LoadInt64(&avgDuration)
		xlog.Infof("Rfq Client - Average timing stats | total_cycles=%d | avg_duration=%v | total_time=%v",
			cycles, time.Duration(avg), time.Duration(total))
	}
}

// Initialize log sampling configuration
func initLogSampling() {
	if rateStr := os.Getenv("AMM_LOG_SAMPLE_RATE"); rateStr != "" {
		if rate, err := strconv.ParseInt(rateStr, 10, 64); err == nil && rate > 0 {
			logSampleRate = rate
		}
	}

	xlog.Infof("AMM log sampling configured: general rate=1/%d", logSampleRate)
}

func logSampledInfo(format string, args ...interface{}) {
	count := atomic.AddInt64(&logCounter, 1)
	if count%logSampleRate == 0 {
		xlog.Infof(format, args...)
	}
}

func shouldSkipMarket(marketId uint) (bool, string) {
	// Skip all LogX market except JPY/USD (77) and EUR/USD (85)
	if (marketId >= 1 && marketId <= 101 && marketId != 77 && marketId != 85) || marketId == 183 {
		return true, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(marketId)]
	}
	return false, ""
}

var redisClient *redis.Client = xredis.GetRedisClient()

type SocketOrderHeader struct {
	BrokerId      uint   `header:"Broker-Id" binding:"required"`
	Key           string `header:"Logx-Key" binding:"required"`
	Secret        string `header:"Logx-Secret" binding:"required"`
	SignerAddress string `header:"Logx-Signer-Address" binding:"required"`
}

type SocketOrderBody struct {
	Signature     string       `json:"signature" binding:"required"`
	MarketId      uint         `json:"marketId" binding:"required"`
	IsBuy         bool         `json:"isBuy" binding:"required"`
	OrderType     string       `json:"orderType" binding:"required,oneof=LIMIT MARKET"`
	Amount        string       `json:"amount" binding:"required"`
	Price         string       `json:"price" binding:"required"`
	ExpiryTs      uint64       `json:"expiryTs" binding:"required"`
	Party         ctypes.Party `json:"party" binding:"oneof=TRADER SOLVER"`
	ChainId       uint         `json:"chainId" binding:"required"`
	IsReduce      bool         `json:"isReduce" binding:"required"`
	ProcessCancel bool         `json:"processCancel"`
}

type CreateOrderRequest struct {
	*SocketOrderHeader
	*SocketOrderBody
}

type BulkOrderRequest struct {
	*types.CreateBulkOrderHeader
	*types.CreateBulkOrderBody
}

var (
	baseAssetToIdMap                   map[string]uint
	marketIDs                          []uint
	marketIdToPriceToQtmConversionExpo map[uint]int
	marketIdToAmtToQtmConversionExpo   map[uint]int
	marketSpreads                      map[uint]map[string]float64
)
var scaleFactor = 2.0

// InitializeGlobalVariables initializes the global mappings and other required variables
func InitializeGlobalVariables() error {
	// Initialize log sampling
	initLogSampling()

	// Initialize Redis config for sizes and slippages
	initializeRedisConfig()

	marketDB := &db.MarketDB{}
	marketSpreads = make(map[uint]map[string]float64)
	// Retrieve all active markets
	activeMarkets := marketDB.GetAllActiveMarkets()
	if activeMarkets == nil {
		xlog.Errorf("Failed to retrieve active markets")
		return fmt.Errorf("failed to retrieve active markets")
	}

	overrideMarketIds := cutils.GetTrimmedSplitEnv("OVERRIDE_MARKET_IDS", ",")
	if len(overrideMarketIds) > 0 {
		activeMarketsFiltered := cutils.FilterSlice(*activeMarkets, func(m db.MarketTable) bool {
			return cutils.SliceExists(overrideMarketIds, fmt.Sprintf("%d", m.ID))
		})
		activeMarkets = &activeMarketsFiltered
	}

	xlog.Infof("Current Active Markets : %v\n", len(*activeMarkets))
	// Retrieve ID to quantum maps
	var err error
	marketIdToAmtToQtmConversionExpo, marketIdToPriceToQtmConversionExpo, err = marketDB.GetIDToQtmConversionExpoMaps()
	if err != nil {
		return fmt.Errorf("failed to retrieve quantum maps: %v", err)
	}
	scaleFactorStr := os.Getenv("AMM_QUOTE_SIZE_SCALE_FACTOR")
	if scaleFactorStr != "" {
		parsedScale, parseErr := strconv.ParseFloat(scaleFactorStr, 64)
		if parseErr != nil {
			xlog.Warnf("Error parsing scale factor from ENV: %v. Using default: %v", parseErr, scaleFactor)
		} else {
			scaleFactor = parsedScale
			xlog.Infof("Using scale factor from ENV: %v", scaleFactor)
		}
	}
	// Initialize baseAssetToIdMap
	baseAssetToIdMap = make(map[string]uint)
	for _, market := range *activeMarkets {
		baseAssetToIdMap[market.BaseAsset] = market.ID
		marketIDs = append(marketIDs, market.ID)

		// Ensure the map for the specific marketId is initialized
		marketSpreads[market.ID] = map[string]float64{
			"bidSpread": 0.0,
			"askSpread": 0.0,
		}
	}

	return nil
}

func initializeRedisConfig() {
	useRedis := os.Getenv("AMM_USE_REDIS_CONFIG") == "1"
	config.SetSlippageRedisConfigEnabled(useRedis)

	if useRedis {
		xlog.Infof("AMM Config - Using Redis-based config")
	} else {
		xlog.Infof("AMM Config - Using file-based config (Redis disabled)")
	}
}

// Generate websocket connection headers
// Since errors are blocking we stop the program if any of the required environment variables are missing
func createSocketOrderHeader() *SocketOrderHeader {
	// Fetch and convert BrokerId
	brokerIdStr := os.Getenv("BROKER_ID")
	if brokerIdStr == "" {
		xlog.Fatalf("Rfq Client - BROKER_ID env var is not set")
	}

	brokerId, err := strconv.Atoi(brokerIdStr)
	if err != nil {
		xlog.Fatalf("Rfq Client - BROKER_ID env var is not set as an integer")
	}

	// Fetch other environment variables
	key := os.Getenv("LOGX_KEY")
	secret := os.Getenv("LOGX_SECRET")
	signerPrivateKey := os.Getenv("SOLVER_SIGNER_PRIVATE_KEY")

	// Check if any of the required environment variables are missing
	if key == "" || secret == "" || signerPrivateKey == "" {
		xlog.Fatalf("Rfq Client - LOGX_KEY, LOGX_SECRET or SOLVER_SIGNER_PRIVATE_KEY env vars are not set")
	}

	// Generate the signer address from the private key
	signerAddress := contractUtils.GetAddressFromPrivateKey(signerPrivateKey)

	// Create and return the SocketOrderHeader
	return &SocketOrderHeader{
		BrokerId:      uint(brokerId),
		Key:           key,
		Secret:        secret,
		SignerAddress: signerAddress,
	}
}

func convertToHTTPHeader(header *SocketOrderHeader) http.Header {
	httpHeader := http.Header{}
	httpHeader.Add("Broker-Id", fmt.Sprintf("%d", header.BrokerId))
	httpHeader.Add("Logx-Key", header.Key)
	httpHeader.Add("Logx-Secret", header.Secret)
	httpHeader.Add("Logx-Signer-Address", header.SignerAddress)
	return httpHeader
}

// Setup WebSocket connection
func setupWebSocketConnection(serverURL string, headers http.Header) (xsocket.XConn, error) {
	_conn, _, err := websocket.DefaultDialer.Dial(serverURL, headers)
	if err != nil {
		return nil, fmt.Errorf("rfq client - failed to connect to WebSocket server: %v", err)
	}
	xConn := xsocket.NewXConn(_conn)
	// Keep this as a regular log since it's not high frequency
	xlog.Infof("Rfq Client - Connected to WebSocket server")
	return xConn, nil
}

func listenToWebSocket(xConn xsocket.XConn) {
	defer xConn.Close()

	avgTicker := time.NewTicker(60 * time.Second)
	defer avgTicker.Stop()

	for {
		select {
		case <-avgTicker.C:
			logAverageTiming()
		default:
			messageType, message, err := xConn.ReadMessage()
			if err != nil {
				xlog.Warnf("Rfq Client - Error reading websocket message. Closing connection. %v", err)
				return
			}

			if messageType == websocket.TextMessage {
				received := atomic.AddInt64(&receivedResponses, 1)
				expected := atomic.LoadInt64(&expectedResponses)

				if expected > 0 && received >= expected {
					startTime := atomic.LoadInt64(&cycleStartTime)
					duration := time.Duration(time.Now().UnixNano() - startTime)
					cycles := atomic.AddInt64(&totalCycles, 1)
					newTotal := atomic.AddInt64(&totalDuration, int64(duration))
					avg := newTotal / cycles
					atomic.StoreInt64(&avgDuration, avg)

					xlog.Infof("Rfq Client - bulk quote timing | cycle=%d | duration=%v | expected=%d | received=%d | avg_duration=%v",
						cycles, duration, expected, received, time.Duration(avg))

					atomic.StoreInt64(&expectedResponses, 0)
					atomic.StoreInt64(&receivedResponses, 0)
				}
			}
			// Use sampled logging for frequent messages
			logSampledInfo("Rfq Client - Received message: %s\n", message)

			if messageType == websocket.CloseMessage {
				xlog.Infof("Rfq Client - Closing socket connection.")
				return
			}
		}
	}
}

func getPriceAndSizeWithCustomSpread(marketId uint, isBid bool, spread float64, price float64, size float64) (string, string, error) {
	// Get the precision values from the global mappings
	priceQuantumExpo, priceQuantumExpoExists := marketIdToPriceToQtmConversionExpo[marketId]
	sizeQuantumExpo, sizeQuantumExpoExists := marketIdToAmtToQtmConversionExpo[marketId]

	if !priceQuantumExpoExists || !sizeQuantumExpoExists {
		return "0", "0", fmt.Errorf("rfq client - quantum values not found for market id: %d", marketId)
	}
	if isBid {
		price *= (1 - spread)
	} else {
		price *= (1 + spread)

	}

	priceStr := utils.FormatNumberWithPrecision(price, priceQuantumExpo)
	sizeStr := utils.FormatNumberWithPrecision(size, sizeQuantumExpo)

	return priceStr, sizeStr, nil
}

var quoteExpiryTsMs int64

func generatePayloadWithCustomPriceAndSize(marketId uint, isBuy bool, priceStr, sizeStr string) (SocketOrderBody, error) {
	var socketOrderBody SocketOrderBody
	priceX18 := cutils.FloatStrToX18(priceStr)
	sizeX18 := cutils.FloatStrToX18(sizeStr)
	if !isBuy {
		sizeX18.Neg(sizeX18)
	}

	subAccountIdStr := os.Getenv("SUB_ACCOUNT_ID")
	if subAccountIdStr == "" {
		xlog.Fatalf("Rfq Client - SUB_ACCOUNT_ID env var is not set")
	}

	expiryTsMs := time.Now().UnixMilli() + quoteExpiryTsMs
	marketIdBigInt := new(big.Int).SetInt64(int64(marketId))

	signatureGenerationPayload := contractUtils.BuildOrderSigPayload{
		SubAccountId: subAccountIdStr,
		PriceX18:     priceX18,
		Amount:       sizeX18,
		Expiration:   uint64(expiryTsMs),
		IsReduce:     false,
		ChainId:      contractUtils.SessionKeyChainId(),
		ProductId:    marketIdBigInt,
	}

	// fmt.Printf("Signature Generation Payload %v\n", signatureGenerationPayload)

	signingPrivateKey := os.Getenv("SOLVER_SIGNER_PRIVATE_KEY")
	if signingPrivateKey == "" {
		xlog.Fatalf("Rfq CLient - SOLVER_SIGNER_PRIVATE_KEY env var is not set")
	}

	signedPayload, err := contractUtils.BuildOrderSignaturePayload(signingPrivateKey, signatureGenerationPayload)
	if err != nil {
		return socketOrderBody, err
	}

	socketOrderBody = SocketOrderBody{
		Signature:     signedPayload.Signature,
		MarketId:      marketId,
		IsBuy:         isBuy,
		OrderType:     "LIMIT",
		Amount:        sizeStr,
		Price:         priceStr,
		ExpiryTs:      uint64(expiryTsMs),
		ChainId:       uint(contractUtils.SessionKeyChainId()),
		IsReduce:      false,
		Party:         ctypes.PARTY_SOLVER,
		ProcessCancel: true,
	}

	return socketOrderBody, nil
}

func generateBulkOrderPayload(marketId uint, isBuy bool, spreads []float64, sizes []float64, oraclePrice float64) (*BulkOrderRequest, error) {
	if len(spreads) != len(sizes) {
		return nil, fmt.Errorf("spreads and sizes arrays must have the same length")
	}

	header := createSocketOrderHeader()
	bulkHeader := &types.CreateBulkOrderHeader{
		BrokerId:      header.BrokerId,
		SignerAddress: header.SignerAddress,
	}

	orders := make([]types.SingleOrderBody, len(spreads))

	for i, spread := range spreads {
		priceStr, sizeStr, err := getPriceAndSizeWithCustomSpread(marketId, isBuy, spread, oraclePrice, sizes[i])
		if err != nil {
			return nil, fmt.Errorf("failed to compute price/size w/ spread for index %d: %w", i, err)
		}

		// Generate signature for this order
		priceX18 := cutils.FloatStrToX18(priceStr)
		sizeX18 := cutils.FloatStrToX18(sizeStr)
		if !isBuy {
			sizeX18.Neg(sizeX18)
		}

		subAccountIdStr := os.Getenv("SUB_ACCOUNT_ID")
		if subAccountIdStr == "" {
			return nil, fmt.Errorf("SUB_ACCOUNT_ID env var is not set")
		}

		expiryTsMs := time.Now().UnixMilli() + quoteExpiryTsMs
		marketIdBigInt := new(big.Int).SetInt64(int64(marketId))

		signatureGenerationPayload := contractUtils.BuildOrderSigPayload{
			SubAccountId: subAccountIdStr,
			PriceX18:     priceX18,
			Amount:       sizeX18,
			Expiration:   uint64(expiryTsMs),
			IsReduce:     false,
			ChainId:      contractUtils.SessionKeyChainId(),
			ProductId:    marketIdBigInt,
		}

		signingPrivateKey := os.Getenv("SOLVER_SIGNER_PRIVATE_KEY")
		if signingPrivateKey == "" {
			return nil, fmt.Errorf("SOLVER_SIGNER_PRIVATE_KEY env var is not set")
		}

		signedPayload, err := contractUtils.BuildOrderSignaturePayload(signingPrivateKey, signatureGenerationPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to build signature for order %d: %w", i, err)
		}

		// Create order details
		orderDetails := &ctypes.CreateOrderBody{
			MarketId:  &marketId,
			IsBuy:     &isBuy,
			OrderType: ctypes.ORDER_TYPE_LIMIT,
			AmountStr: sizeStr,
			PriceStr:  priceStr,
			ExpiryTs:  &[]uint64{uint64(expiryTsMs)}[0],
			Party:     ctypes.PARTY_SOLVER,
			IsReduce:  &[]bool{false}[0],
		}

		orders[i] = types.SingleOrderBody{
			Signature: signedPayload.Signature,
			Details:   orderDetails,
		}
	}

	// Create bulk order body
	bulkBody := &types.CreateBulkOrderBody{
		Orders:   orders,
		MarketId: &marketId,
		IsBuy:    &isBuy,
	}

	return &BulkOrderRequest{
		CreateBulkOrderHeader: bulkHeader,
		CreateBulkOrderBody:   bulkBody,
	}, nil
}

func getSmoothExponentialSizes(baseSize float64, numQuotes int) []float64 {
	if numQuotes <= 0 {
		return nil
	}
	sizes := make([]float64, numQuotes)
	exps := make([]float64, numQuotes)
	var sumExp float64

	ratio := 1.1 + rand.Float64()*0.4
	for i := 0; i < numQuotes; i++ {
		wiggle := 0.9 + rand.Float64()*0.4
		val := math.Pow(ratio, float64(i)) * wiggle
		exps[i] = val
		sumExp += val
	}

	scale := baseSize / sumExp
	for i := 0; i < numQuotes; i++ {
		sizes[i] = exps[i] * scale
	}
	return sizes
}

func sendQuotes(xConn xsocket.XConn, serverURL string, httpHeader http.Header) {
	// Get from env otherwise default to 1 minute
	var err error
	quoteExpiryTsMs = 1 * cutils.MINUTE_MILLI
	if os.Getenv("RFQ_EXPIRY_MILLIS") != "" && os.Getenv("RFQ_EXPIRY_MILLIS") != "0" {
		quoteExpiryTsMs, err = strconv.ParseInt(os.Getenv("RFQ_EXPIRY_MILLIS"), 10, 64)
		if err != nil {
			xlog.Warnf("Rfq Client - Error parsing RFQ_EXPIRY env var: %v. Defaulting to %d", err, quoteExpiryTsMs)
		} else {
			xlog.Warnf("Rfq Client - RFQ_EXPIRY env var set to %d", quoteExpiryTsMs)
		}
	} else {
		xlog.Warnf("Rfq Client - RFQ_EXPIRY env var is not set. Defaulting to %d", quoteExpiryTsMs)
	}

	intervalStr := os.Getenv("AMM_QUOTE_INTERVAL")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		// Default to 30 seconds if the env variable is not set or invalid
		interval = 30
	}

	// Number of quotes per side (buy/sell)
	numQuotes := 10
	if n, err := strconv.Atoi(os.Getenv("AMM_QUOTE_COUNT")); err == nil {
		numQuotes = n
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	doWork := func() {
		defer cutils.LogTime(time.Now(), "SendQuotes")

		marketSizes := config.GetSizesBatch(redisClient, marketIDs)
		marketSlippages := config.GetSlippagesBatch(redisClient, marketIDs)

		// Track statistics for summary logging
		successCount := 0
		totalQuotes := 0
		errorCount := 0

		for _, marketId := range marketIDs {
			if shouldSkip, symbol := shouldSkipMarket(marketId); shouldSkip {
				xlog.Debugf("Rfq Client - Skipping market ID %d (%s)", marketId, symbol)
				continue
			}

			price, exists := getPriceFromCache(marketId)
			if !exists {
				xlog.Errorf("Rfq Client - cannot fetch the oracle price for market id: %d", marketId)
				errorCount++
				continue
			}

			size := marketSizes[marketId]
			baseSpread := marketSlippages[marketId]

			marketSpreads[marketId]["askSpread"] = baseSpread
			marketSpreads[marketId]["bidSpread"] = baseSpread

			bidSizes := getSmoothExponentialSizes(size, numQuotes)
			askSizes := getSmoothExponentialSizes(size, numQuotes)
			if len(bidSizes) < 1 || len(askSizes) < 1 {
				xlog.Warnf("Rfq Client - Invalid sizes array for marketId %d", marketId)
				errorCount++
				continue
			}

			// --------------------
			// 1) CANCEL BID (index=0)
			// --------------------
			cancelSpread := calcLinearSpread(baseSpread, 0, numQuotes)
			if err := sendSingleQuote(
				xConn, serverURL, httpHeader,
				marketId,
				true,         // isBuy => BID
				cancelSpread, // i=0
				price,
				bidSizes[0],
				true, // ProcessCancel = true
			); err != nil {
				xlog.Warnf("Rfq Client - Error canceling BID (mkt=%d): %v", marketId, err)
				errorCount++
				continue
			}
			totalQuotes++

			// --------------------
			// 2) CANCEL ASK (index=0)
			// --------------------
			if err := sendSingleQuote(
				xConn, serverURL, httpHeader,
				marketId,
				false,        // isBuy => ASK
				cancelSpread, // i=0 again
				price,
				askSizes[0],
				true, // ProcessCancel = true
			); err != nil {
				xlog.Warnf("Rfq Client - Error canceling ASK (mkt=%d): %v", marketId, err)
				errorCount++
				continue
			}
			totalQuotes++

			// --------------------
			// 3) Place remaining BIDs (index=1..numQuotes-1)
			// --------------------
			for i := 1; i < numQuotes; i++ {
				spread := calcLinearSpread(baseSpread, i, numQuotes)
				if err := sendSingleQuote(
					xConn, serverURL, httpHeader,
					marketId,
					true, // BID
					spread,
					price,
					bidSizes[i],
					false, // ProcessCancel = false
				); err != nil {
					xlog.Warnf("Rfq Client - Error placing BID idx=%d (mkt=%d): %v", i, marketId, err)
					errorCount++
					break
				}
				totalQuotes++
			}

			// --------------------
			// 4) Place remaining ASKs (index=1..numQuotes-1)
			// --------------------
			for i := 1; i < numQuotes; i++ {
				spread := calcLinearSpread(baseSpread, i, numQuotes)
				if err := sendSingleQuote(
					xConn, serverURL, httpHeader,
					marketId,
					false, // ASK
					spread,
					price,
					askSizes[i],
					false, // ProcessCancel = false
				); err != nil {
					xlog.Warnf("Rfq Client - Error placing ASK idx=%d (mkt=%d): %v", i, marketId, err)
					errorCount++
					break
				}
				totalQuotes++
			}

		}

		logSampledInfo("Rfq Client - Quote cycle completed: %d/%d markets successful, %d total quotes, %d errors",
			successCount, len(marketIDs), totalQuotes, errorCount)
	}

	doWork()

	for range ticker.C {
		doWork()

		err := writeSpreadsToRedis()
		if err != nil {
			xlog.Errorf("Error writing spreads to Redis: %v", err)
		}
	}

}

func calcLinearSpread(baseSpread float64, i, numQuotes int) float64 {
	if numQuotes == 0 {
		return 0.0
	}
	return baseSpread * float64(i+1) / float64(numQuotes)
}

func sendSingleQuote(
	xConn xsocket.XConn,
	serverURL string,
	httpHeader http.Header,
	marketId uint,
	isBuy bool,
	spread float64,
	oraclePrice float64,
	size float64,
	cancelFirst bool,
) error {

	priceStr, sizeStr, err := getPriceAndSizeWithCustomSpread(
		marketId, isBuy, spread, oraclePrice, size,
	)
	if err != nil {
		return fmt.Errorf("failed to compute price/size w/ spread: %w", err)
	}

	payload, err := generatePayloadWithCustomPriceAndSize(marketId, isBuy, priceStr, sizeStr)
	if err != nil {
		return fmt.Errorf("failed to build payload: %w", err)
	}
	payload.ProcessCancel = cancelFirst

	if err := xConn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
		xlog.Warnf("Rfq Client - Lost connection, trying to reconnect...")
		newConn, rErr := setupWebSocketConnection(serverURL, httpHeader)
		if rErr != nil {
			return fmt.Errorf("error reconnecting: %w", rErr)
		}
		xConn = newConn
	}

	if err := xConn.WriteJSON(payload); err != nil {
		xclient.GlobalDiscordClient.SendWebhookMessage(
			fmt.Sprintf("AMM - Error sending quote | mkt=%v | isBuy=%v | err=%v", marketId, isBuy, err),
		)
		return fmt.Errorf("error writing JSON: %w", err)
	}

	return nil
}

func sendBulkQuotes(xConn xsocket.XConn, serverURL string, httpHeader http.Header) {
	// Get from env otherwise default to 1 minute
	var err error
	quoteExpiryTsMs = 1 * cutils.MINUTE_MILLI
	if os.Getenv("RFQ_EXPIRY_MILLIS") != "" && os.Getenv("RFQ_EXPIRY_MILLIS") != "0" {
		quoteExpiryTsMs, err = strconv.ParseInt(os.Getenv("RFQ_EXPIRY_MILLIS"), 10, 64)
		if err != nil {
			xlog.Warnf("Rfq Client - Error parsing RFQ_EXPIRY env var: %v. Defaulting to %d", err, quoteExpiryTsMs)
		} else {
			xlog.Warnf("Rfq Client - RFQ_EXPIRY env var set to %d", quoteExpiryTsMs)
		}
	} else {
		xlog.Warnf("Rfq Client - RFQ_EXPIRY env var is not set. Defaulting to %d", quoteExpiryTsMs)
	}

	intervalStr := os.Getenv("AMM_QUOTE_INTERVAL")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		// Default to 30 seconds if the env variable is not set or invalid
		interval = 30
	}

	// Number of quotes per side (buy/sell)
	numQuotes := 10
	if n, err := strconv.Atoi(os.Getenv("AMM_QUOTE_COUNT")); err == nil {
		numQuotes = n
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	doWork := func() {
		atomic.StoreInt64(&cycleStartTime, time.Now().UnixNano())
		atomic.StoreInt64(&expectedResponses, 0)
		atomic.StoreInt64(&receivedResponses, 0)

		marketSizes := config.GetSizesBatch(redisClient, marketIDs)
		marketSlippages := config.GetSlippagesBatch(redisClient, marketIDs)

		// Track statistics for summary logging
		successCount := 0
		totalQuotes := 0
		errorCount := 0

		for _, marketId := range marketIDs {
			if shouldSkip, symbol := shouldSkipMarket(marketId); shouldSkip {
				xlog.Debugf("Rfq Client - Skipping market ID %d (%s)", marketId, symbol)
				continue
			}

			price, exists := getPriceFromCache(marketId)
			if !exists {
				xlog.Errorf("Rfq Client - cannot fetch the oracle price for market id: %d", marketId)
				errorCount++
				continue
			}
			size := marketSizes[marketId]
			if err != nil {
				xlog.Errorf("Rfq Client - Error getting oracle price and size for marketId %d: %v", marketId, err)
				errorCount++
				continue
			}
			baseSpread := marketSlippages[marketId]
			marketSpreads[marketId]["askSpread"] = baseSpread
			marketSpreads[marketId]["bidSpread"] = baseSpread

			bidSizes := getSmoothExponentialSizes(size, numQuotes)
			askSizes := getSmoothExponentialSizes(size, numQuotes)
			if len(bidSizes) < 1 || len(askSizes) < 1 {
				xlog.Warnf("Rfq Client - Invalid sizes array for marketId %d", marketId)
				errorCount++
				continue
			}

			// Generate spreads for all quotes
			bidSpreads := make([]float64, numQuotes)
			askSpreads := make([]float64, numQuotes)
			for i := 0; i < numQuotes; i++ {
				bidSpreads[i] = calcLinearSpread(baseSpread, i, numQuotes)
				askSpreads[i] = calcLinearSpread(baseSpread, i, numQuotes)
			}

			// Send bulk BID orders
			isBuy := true
			bidPayload, err := generateBulkOrderPayload(marketId, isBuy, bidSpreads, bidSizes, price)
			if err != nil {
				xlog.Errorf("Rfq Client - Error generating bulk BID payload for marketId %d: %v", marketId, err)
				errorCount++
				continue
			}

			if err := sendBulkOrder(xConn, serverURL, httpHeader, bidPayload); err != nil {
				xlog.Errorf("Rfq Client - Error sending bulk BID orders for marketId %d: %v", marketId, err)
				errorCount++
				continue
			}
			totalQuotes += numQuotes
			successCount++

			atomic.AddInt64(&expectedResponses, 1)
			// Send bulk ASK orders
			isBuy = false
			askPayload, err := generateBulkOrderPayload(marketId, isBuy, askSpreads, askSizes, price)
			if err != nil {
				xlog.Errorf("Rfq Client - Error generating bulk ASK payload for marketId %d: %v", marketId, err)
				errorCount++
				continue
			}

			if err := sendBulkOrder(xConn, serverURL, httpHeader, askPayload); err != nil {
				xlog.Errorf("Rfq Client - Error sending bulk ASK orders for marketId %d: %v", marketId, err)
				errorCount++
				continue
			}
			totalQuotes += numQuotes
			successCount++

			atomic.AddInt64(&expectedResponses, 1)
		}

		logSampledInfo("Rfq Client - Bulk quote cycle completed: %d/%d markets successful, %d total quotes, %d errors, %d expected responses",
			successCount, len(marketIDs), totalQuotes, errorCount, atomic.LoadInt64(&expectedResponses))
	}

	doWork()

	for range ticker.C {
		doWork()

		err := writeSpreadsToRedis()
		if err != nil {
			xlog.Errorf("Error writing spreads to Redis: %v", err)
		}
	}
}

func sendBulkOrder(
	xConn xsocket.XConn,
	serverURL string,
	httpHeader http.Header,
	payload *BulkOrderRequest,
) error {
	if err := xConn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
		xlog.Warnf("Rfq Client - Lost connection, trying to reconnect...")
		newConn, rErr := setupWebSocketConnection(serverURL, httpHeader)
		if rErr != nil {
			return fmt.Errorf("error reconnecting: %w", rErr)
		}
		xConn = newConn
	}

	if err := xConn.WriteJSON(payload); err != nil {
		xclient.GlobalDiscordClient.SendWebhookMessage(
			fmt.Sprintf("AMM - Error sending bulk order | mkt=%v | isBuy=%v | err=%v", *payload.MarketId, *payload.IsBuy, err),
		)
		return fmt.Errorf("error writing JSON: %w", err)
	}

	return nil
}

func writeSpreadsToRedis() error {
	redisKey := xredis.GetMarketSpreadKey()
	jsonData, err := json.Marshal(marketSpreads)
	if err != nil {
		xlog.Errorf("failed to serialize market spreads: %v", err)
		return fmt.Errorf("failed to serialize market spreads: %v", err)
	}

	err = redisClient.Set(context.Background(), redisKey, jsonData, 0).Err()
	if err != nil {
		xlog.Errorf("failed to write market spreads to Redis: %v", err)
		return fmt.Errorf("failed to write market spreads to Redis: %v", err)
	}

	return nil
}

// Main function
func RfqClient() {
	apiServerWs := os.Getenv("API_SERVER_WS")
	if apiServerWs == "" {
		log.Fatal("Rfq Client - API_SERVER_WS env var is not set")
	}

	serverURL := fmt.Sprintf("%v/ws/v1/quote", apiServerWs)
	headers := createSocketOrderHeader()

	httpHeader := convertToHTTPHeader(headers)

	var xConn xsocket.XConn
	var err error

	for {
		xConn, err = setupWebSocketConnection(serverURL, httpHeader)
		if err != nil {
			xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("AMM - Error connecting to WebSocket server: %v.. \nThis is a fatal error. Please quicky look into this", err))
			xlog.Errorf(err.Error())
		} else {
			// ENV to determine which quote sending method to use
			useBulkQuotes := os.Getenv("AMM_USE_BULK_QUOTES")
			if useBulkQuotes == "1" {
				xlog.Infof("Rfq Client - Using bulk quotes implementation")
				go sendBulkQuotes(xConn, serverURL, httpHeader)
			} else {
				xlog.Infof("Rfq Client - Using individual quotes implementation")
				go sendQuotes(xConn, serverURL, httpHeader)
			}

			listenToWebSocket(xConn)

			// Close the connection explicitly before retrying
			xConn.Close()

			// Connection closed, retrying...
			xlog.Warnf("Rfq Client - Connection closed. Reconnecting in 5 seconds...")
			go func() {
				xclient.GlobalDiscordClient.SendWebhookMessage("AMM - Connection closed. Reconnecting in 5 seconds...")
			}()
		}
		time.Sleep(5 * time.Second) // Fixed retry delay before attempting to reconnect
	}
}
