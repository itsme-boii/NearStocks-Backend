package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/freya/config"
	"math/big"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type OkxClient struct {
	conn            *websocket.Conn
	lastMessageTime time.Time  // Track the last received message time to keep socket connection alive
	mutex           sync.Mutex // Protect access to lastMessageTime
	PositionsMap    map[string]map[string]interface{}

	// OKX connection params
	apiKey               string
	secretKey            string
	passPhrase           string
	websocketUrl         string
	privateSocketUrlPath string
}

func NewOkxClient() *OkxClient {
	return &OkxClient{
		websocketUrl:         os.Getenv("WEBSOCKET_URL"),
		apiKey:               os.Getenv("API_KEY"),
		secretKey:            os.Getenv("SECRET_KEY"),
		passPhrase:           os.Getenv("PASS_PHRASE"),
		privateSocketUrlPath: "/ws/v5/private",
	}
}

// InitializeOkxClient connects and subscribes to channels
func (client *OkxClient) InitializeOkxClient() error {
	client.websocketUrl = os.Getenv("WEBSOCKET_URL")
	client.apiKey = os.Getenv("API_KEY")
	client.secretKey = os.Getenv("SECRET_KEY")
	client.passPhrase = os.Getenv("PASS_PHRASE")
	client.privateSocketUrlPath = "/ws/v5/private"

	client.PositionsMap = make(map[string]map[string]interface{})

	xlog.Infof("okx client - logging in to OKX websocket...")

	err := client.ConnectToWebSocket(client.websocketUrl, client.apiKey, client.passPhrase, client.secretKey, client.privateSocketUrlPath)
	if err != nil {
		return fmt.Errorf("okx client - unable to log into OKX due to error: %v, aborting", err)
	}

	// Subscribe to orders channel
	client.SubscribeToChannel("orders", map[string]interface{}{
		"channel":  "orders",
		"instType": "SWAP",
	})

	// Subscribe to positions channel
	client.SubscribeToChannel("positions", map[string]interface{}{
		"channel":  "positions",
		"instType": "SWAP",
		// we subscribe for position data every 1 second
		"extraParams": "{\"updateInterval\": \"1000\"}",
	})

	// If we do not receive message from socket for 30 seconds, OKX will terminate our websocket connection
	// we have to ping the socket connection if we do not get a message in a span of 30 seconds
	// more details - https://www.okx.com/docs-v5/en/?shell#overview-websocket-connect
	go client.socketPingPong()

	return nil
}

// Generate signature using HMAC SHA256 and Base64 encoding for login
func (client *OkxClient) generateSignature(timestamp, method, requestPath, secretKey string) string {
	message := timestamp + method + requestPath
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (client *OkxClient) socketPingPong() {
	// Check every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		client.mutex.Lock()
		timeSinceLastMessage := time.Since(client.lastMessageTime)
		client.mutex.Unlock()

		if timeSinceLastMessage > 25*time.Second {
			xlog.Infof("Okx Client - No message received in 25 seconds, sending ping...")
			client.sendPing()
		}
	}
}

// Send a ping message as a simple "ping" string
func (client *OkxClient) sendPing() {
	err := client.conn.WriteMessage(websocket.TextMessage, []byte("ping"))
	if err != nil {
		xlog.Errorf("Okx Client - Failed to send ping: %v", err)
	} else {
		xlog.Infof("Okx Client - Sent ping message")
	}
}

// Update the last message time in a thread-safe manner
func (client *OkxClient) updateLastMessageTime() {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	client.lastMessageTime = time.Now()
}

// ConnectToWebSocket connects to the WebSocket and logs in
func (client *OkxClient) ConnectToWebSocket(wsURL, apiKey, passphrase, secretKey, path string) error {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sign := client.generateSignature(timestamp, "GET", "/users/self/verify", secretKey)

	u := url.URL{Scheme: "wss", Host: wsURL, Path: path}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	// Prepare login request payload
	loginPayload := map[string]interface{}{
		"op": "login",
		"args": []map[string]string{
			{
				"apiKey":     apiKey,
				"passphrase": passphrase,
				"timestamp":  timestamp,
				"sign":       sign,
			},
		},
	}

	// Send login request
	err = conn.WriteJSON(loginPayload)
	if err != nil {
		return err
	}

	// Wait for login response
	_, message, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("okx Client - failed to read WebSocket message: %v", err)
	}

	// Parse login response
	var response map[string]interface{}
	if err := json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("okx Client - failed to parse login response: %v", err)
	}

	// Check if login was successful
	event, ok := response["event"].(string)
	if !ok || event != "login" {
		return fmt.Errorf("okx Client - invalid login response format")
	}

	code, ok := response["code"].(string)
	if !ok || code != "0" {
		return fmt.Errorf("login failed: %s", response["msg"])
	}

	xlog.Infof("Okx Client - Login successful. Connection ID: %v", response["connId"])
	client.conn = conn
	// Start listening for messages from the new connection
	go client.listenForMessages()

	return nil
}

// Subscribe to a channel
func (client *OkxClient) SubscribeToChannel(channel string, args map[string]interface{}) error {
	subscribePayload := map[string]interface{}{
		"op":   "subscribe",
		"args": []map[string]interface{}{args},
	}

	err := client.conn.WriteJSON(subscribePayload)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel %s: %v", channel, err)
	}

	xlog.Infof("Okx Client - Subscribed to channel: %s", channel)
	return nil
}

// Listen for incoming WebSocket messages
func (client *OkxClient) listenForMessages() {
	for {
		messageType, message, err := client.conn.ReadMessage()
		if err != nil {
			xlog.Errorf("Okx Client - Error reading message: %v", err)

			if websocket.IsCloseError(err, 4004) {
				xlog.Infof("Okx Client - Connection closed with error code 4004, attempting to reconnect...")

				// Reconnect if 4004 error is encountered
				client.ConnectToWebSocket(client.websocketUrl, client.apiKey, client.passPhrase, client.secretKey, client.privateSocketUrlPath)
			}

			break
		}

		// Check the message type (TextMessage = string, BinaryMessage = byte data)
		switch messageType {
		case websocket.TextMessage:
			// Handle string messages (e.g., "pong" or other string responses)
			msgStr := string(message)

			// If it's a pong response, update last message time
			if msgStr == "pong" {
				client.updateLastMessageTime()
				xlog.Infof("Okx Client - Pong")
			} else {
				// Process other string messages as needed
				client.processOkxMessage(msgStr)
			}

		default:
			xlog.Errorf("Okx Client - Unknown message type: %d", messageType)
		}
	}
}

// Send custom message to WebSocket (for example, unsubscribe or custom actions)
func (client *OkxClient) SendMessage(payload interface{}) error {
	err := client.conn.WriteJSON(payload)
	if err != nil {
		return fmt.Errorf("okx Client - failed to send message: %v", err)
	}

	xlog.Infof("Okx Client - Sent message: %v", payload)
	return nil
}

func (client *OkxClient) PlaceMarketOrder(symbol, amountx18 string, isBuy bool) error {
	randomId := generateRandomNumber()
	var side string
	if isBuy {
		side = "buy"
	} else {
		side = "sell"
	}

	// convert amountx18 string into lot size
	sizeInLots, err := convertToLotSize(amountx18, symbol)
	if err != nil {
		return fmt.Errorf("okx Client - error converting amountx18 :%v into size in lots : %v", amountx18, err)
	}

	payload := map[string]interface{}{
		"id": randomId,
		"op": "order",
		"args": []map[string]string{
			{
				"instId":  symbol + "-USDT-SWAP",
				"tdMode":  "cross",
				"side":    side,
				"ordType": "market",
				// ToDo - pass size in lots, not the parameter
				"sz": sizeInLots,
			},
		},
	}

	err = client.SendMessage(payload)
	if err != nil {
		return fmt.Errorf("okx client - error sending place market order payload for symbol %v, amountx18 %v and isBuy%v", symbol, amountx18, isBuy)
	}
	return nil
}

// ToDo - Need to improve this function to handle a variety of responses better
func (client *OkxClient) processOkxMessage(msgStr string) {
	// Parse the JSON message
	var msg map[string]interface{}
	err := json.Unmarshal([]byte(msgStr), &msg)
	if err != nil {
		xlog.Errorf("Okx Client - Failed to unmarshal message: %v", err)
		return
	}

	// Check if the message is order-related or position-related
	if op, ok := msg["op"].(string); ok && op == "order" {
		// Print out order-related information
		xlog.Infof("Okx Client - Order Info: %s", msgStr)
	} else if arg, ok := msg["arg"].(map[string]interface{}); ok && arg["channel"] == "positions" {
		// Process positions data
		xlog.Infof("Okx Client - updating position data")
		data, ok := msg["data"].([]interface{})
		if ok {
			client.processPositionData(data)
		}
	}
}

func (client *OkxClient) processPositionData(data []interface{}) {
	for _, entry := range data {
		position := entry.(map[string]interface{})

		// Fetch the required fields using the correct keys
		instId, _ := position["instId"].(string)
		sizeStr, _ := position["pos"].(string)    // Use "pos" for position size
		lastPxStr, _ := position["last"].(string) // Use "last" for last price

		// Extract symbol from instId using the helper function
		symbol := dirtyExtractSymbolFromInstId(instId)

		// Fetch the lot size conversion factor from SYMBOL_TO_LOT_SIZE
		lotSize, ok := config.SYMBOL_TO_LOT_SIZE[symbol]
		if !ok {
			xlog.Errorf("Unknown symbol in lot size map: %s", symbol)
			continue
		}

		// Convert position size (pos) to big.Float to handle large numbers
		sizeFloat := new(big.Float)
		_, ok = sizeFloat.SetString(sizeStr)
		if !ok {
			xlog.Errorf("Failed to parse position size for %s", instId)
			continue
		}
		// Determine if the position is long (true if pos > 0)
		isLong := sizeFloat.Sign() > 0

		// Multiply the position size by 10^18
		scaleFactor := new(big.Float).SetFloat64(1e18)
		amountScaled := new(big.Float).Mul(sizeFloat.Abs(sizeFloat), scaleFactor)

		// Divide by lot size to get the final amount in lots
		lotSizeFloat := new(big.Float).SetFloat64(lotSize)
		amountx18Float := new(big.Float).Quo(amountScaled, lotSizeFloat)

		// Convert the result back to big.Int (truncating any decimal points)
		amountx18 := new(big.Int)
		amountx18Float.Int(amountx18)

		// Store in the global PositionsMap (keyed by symbol, not instId)
		client.PositionsMap[symbol] = map[string]interface{}{
			"amountx18":  amountx18, // Store as big.Int
			"is_long":    isLong,
			"last_price": lastPxStr,
		}
	}

	// xlog.Infof("Updated position data: %v", client.PositionsMap)
}

func dirtyExtractSymbolFromInstId(instId string) string {
	// Split the instId (e.g., "BTC-USDT-SWAP") and return the first part (symbol)
	parts := strings.Split(instId, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return instId // Fallback to instId if no symbol could be extracted
}

func convertToLotSize(amountx18 string, symbol string) (string, error) {
	// Convert the amountx18 string to a *big.Int (since it's a large number)
	amountBigInt := new(big.Int)
	_, ok := amountBigInt.SetString(amountx18, 10) // base 10
	if !ok {
		return "", fmt.Errorf("invalid amountx18: %s", amountx18)
	}

	// Get the lot size for the productId
	lotSize, exists := config.SYMBOL_TO_LOT_SIZE[symbol]
	if !exists {
		return "", fmt.Errorf("symbol %s not found", symbol)
	}

	// Convert lotSize to a *big.Float to handle the multiplication
	lotSizeFloat := big.NewFloat(lotSize)

	// Convert amountBigInt (amountx18) to *big.Float
	amountBigFloat := new(big.Float).SetInt(amountBigInt)

	// Perform the multiplication: amountx18 * lotSize
	result := new(big.Float).Mul(amountBigFloat, lotSizeFloat)

	// Divide by 10^18 (since amountx18 represents a number scaled by 10^18)
	tenPower18 := new(big.Float).SetFloat64(1e18)
	result.Quo(result, tenPower18)

	// Round the result down to the nearest integer (truncate decimal places)
	intResult, _ := result.Int(nil) // This will round down and remove decimals

	// Convert the integer result to a string
	return intResult.String(), nil
}

// Function to generate an 8 digit random number
func generateRandomNumber() string {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Generate a random number between 10000000 and 99999999 (inclusive)
	randomNumber := rand.Intn(90000000) + 10000000

	// Convert the number to a string
	return fmt.Sprintf("%08d", randomNumber)
}
