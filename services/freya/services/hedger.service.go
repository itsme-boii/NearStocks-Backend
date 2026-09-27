// Service to fetch OI Imbalance, Poll for any new entries to fills database
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type HedgerService struct {
	fillDb                db.FillDB
	notifyChan            chan string // Channel for handling notifications
	okxClient             *OkxClient
	redisClient           *redis.Client
	internalSubaccountIDs map[string]bool
}

// Initialise the ExchangeService, including notification handling
func (hs *HedgerService) InitializeHedgerService() {
	// Initialize the notification channel
	hs.notifyChan = make(chan string)

	hs.fillDb.MatchOrderTriggerFunction()
	hs.fillDb.MatchOrderTrigger()

	var okxClient *OkxClient = NewOkxClient()
	// Initialize the OKX client
	err := okxClient.InitializeOkxClient()
	if err != nil {
		log.Fatalf("Hedger Service - failed to initialize OKX client: %v", err)
	}

	hs.okxClient = okxClient
	hs.redisClient = xredis.GetRedisClient()

	ammSubaccountID := os.Getenv("AMM_SUBACCOUNT_ID")
	hs.internalSubaccountIDs = make(map[string]bool)
	for _, id := range strings.Split(os.Getenv("INTERNAL_SUBACCOUNT_IDS"), ",") {
		hs.internalSubaccountIDs[id] = true
	}
	// Add ammSubaccountID to the map as well
	hs.internalSubaccountIDs[ammSubaccountID] = true

	// Sync positions
	go hs.SyncPositions()

	// Start listening for match order trigger notifications
	go hs.ListenForMatchOrderTriggerNotifications()
}

func (hs *HedgerService) ListenForMatchOrderTriggerNotifications() {
	// Create a new listener for PostgreSQL notifications
	listener := pq.NewListener(os.Getenv("DSN"), 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			xlog.Errorf("Hedger Service - Listener event error: %v", err)
		}

		switch ev {
		case pq.ListenerEventConnected:
			xlog.Infof("Hedger Service - Listener connected to the database")
		case pq.ListenerEventConnectionAttemptFailed:
			xlog.Errorf("Hedger Service - Listener connection attempt failed")
		case pq.ListenerEventDisconnected:
			xlog.Errorf("Hedger Service - Listener disconnected")
		}
	})

	err := listener.Listen("match_order_channel")
	if err != nil {
		xlog.Errorf("Hedger Service - Exchange Service - Error starting listener: %v", err)
		return
	}

	xlog.Infof("Hedger Service - Listening for notifications on 'match_order_channel'...")

	// Goroutine to receive notifications and send them to global notifyChan
	go func() {
		for {
			select {
			case notification := <-listener.Notify:
				if notification != nil {
					hs.notifyChan <- notification.Extra // Send payload to notifyChan
				} else {
					xlog.Errorf("Hedger Service - Received a nil notification") // Log nil notifications for debugging
				}
			case <-time.After(30 * time.Second):
				xlog.Infof("Hedger Service - No new notifications, pinging listener...")
				err := listener.Ping()
				if err != nil {
					xlog.Errorf("Hedger Service - Ping failed: %v", err) // Handle ping failure
				}
			}
		}
	}()
}

func (hs *HedgerService) HedgePositions() {
	// Process notifications from the global notifyChan
	for payload := range hs.notifyChan {
		var fill db.FillTable
		// Unmarshal as usual for all fields except market_id
		err := json.Unmarshal([]byte(payload), &fill)
		if err != nil {
			xlog.Errorf("Hedger Service - Failed to unmarshal payload: %v", err)
			continue
		}

		// Handle market_id separately by parsing the raw payload into a map
		var fillData map[string]interface{}
		err = json.Unmarshal([]byte(payload), &fillData)
		if err != nil {
			xlog.Errorf("Hedger Service - Failed to unmarshal payload into map for market_id: %v", err)
			continue
		}

		// Remove AMM subaccount and internal subaccounts here
		fillSubaccountID, ok := fillData["subaccount_id"].(string) // Ensure fillData["subaccount_id"] is a string
		if ok {
			if hs.internalSubaccountIDs[fillSubaccountID] {
				xlog.Infof("Hedger Service - Subaccount ID %s is an internal subaccount", fillSubaccountID)
				continue
			} else {
				xlog.Infof("Hedger Service - Subaccount ID %s is not an internal subaccount", fillSubaccountID)
			}
		} else {
			xlog.Errorf("Hedger Service - Subaccount ID is missing or invalid in fill data")
		}

		// Extract market_id manually from the map
		if marketID, ok := fillData["market_id"].(float64); ok {
			fill.MarketId = uint(marketID) // Convert float64 to uint
		}

		// Continue processing as usual
		amountx18 := fill.Amountx18.String()
		var isBuy bool
		if fill.Side == "BUY" {
			isBuy = true
		} else {
			isBuy = false
		}

		// Assuming that the symbol is mapped based on the market ID
		symbol := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(fill.MarketId)]

		xlog.Infof("Hedger Service - placing market order for symbol %v, amountX18 %v and isBuy %v", symbol, amountx18, isBuy)
		// Call the OKX client to place a market order
		err = hs.okxClient.PlaceMarketOrder(symbol, amountx18, isBuy)
		if err != nil {
			xlog.Errorf("Hedger Service - Error placing market order: %v", err)
		} else {
			xlog.Infof("Hedger Service - Successfully placed market order: symbol=%s, amountx18=%s, isBuy=%v", symbol, amountx18, isBuy)
		}
	}
}

func (hs *HedgerService) SyncPositions() {
	// Fetch the OI and direction of all markets
	marketOi, err := hs.FetchMarketOi()
	if err != nil {
		xlog.Errorf("Hedger Service - error fetching market OI, stopping hedger sync: %v", err)
		return
	}

	fmt.Printf("market OI %v", marketOi)
	for productID, symbol := range contractUtils.PRODUCT_ID_SYMBOL_TO_MAP {
		if productID%2 != 0 {
			productIDStr := strconv.Itoa(int(productID))
			if positionData, exists := hs.okxClient.PositionsMap[symbol]; exists {
				okxPositionx18 := positionData["amountx18"].(*big.Int)
				okxIsLong := positionData["is_long"].(bool)
				xlog.Infof("Hedger Service - positions exist for market ID %v on OKX with amount %v and is_long %v", productID, okxPositionx18, okxIsLong)

				// Ensure the marketOi has the necessary data
				if ammData, ok := marketOi[productIDStr]; ok {
					// Check the existence of "oi" and "is_long" in the map
					ammPosition, okPos := ammData["oi"].(string)
					ammIsLong, okLong := ammData["is_long"].(bool)

					if !okPos {
						xlog.Errorf("Hedger Service - 'oi' not found or invalid type for product ID %v", productID)
						continue
					}

					if !okLong {
						xlog.Errorf("Hedger Service - 'is_long' not found or invalid type for product ID %v", productID)
						continue
					}

					// Create a new big.Int to hold the converted value
					ammPositionx18 := new(big.Int)
					_, success := ammPositionx18.SetString(ammPosition, 10)
					if !success {
						xlog.Errorf("Failed to convert %s to big.Int", ammPosition)
						continue
					}

					// Case 1: Both positions are in the same direction (long/long or short/short)
					if okxIsLong == ammIsLong {
						deltaPosition := new(big.Int).Sub(ammPositionx18, okxPositionx18) // Calculate delta

						if deltaPosition.Sign() > 0 {
							// We need to open more positions to match AMM (same direction)
							sizex18 := deltaPosition.String()
							xlog.Infof("Hedger Service - Increasing position for %v: amount %v", symbol, sizex18)
							hs.okxClient.PlaceMarketOrder(symbol, sizex18, ammIsLong)
						} else if deltaPosition.Sign() < 0 {
							// We need to reduce positions to match AMM (same direction)
							sizex18 := new(big.Int).Abs(deltaPosition).String()
							xlog.Infof("Hedger Service - Reducing position for %v: amount %v", symbol, sizex18)
							hs.okxClient.PlaceMarketOrder(symbol, sizex18, !ammIsLong) // Opposite to reduce
						} else {
							xlog.Infof("Hedger Service - Positions already match for %v", symbol)
						}

						// Case 2: Positions are in opposite directions
					} else {
						// Calculate the total adjustment needed in one step
						sizex18 := new(big.Int).Add(ammPositionx18, okxPositionx18)
						hs.okxClient.PlaceMarketOrder(symbol, sizex18.String(), ammIsLong)

						xlog.Infof("Hedger Service - opening a position for %v: amount %v", symbol, sizex18)
					}
				} else {
					xlog.Errorf("Hedger Service - Missing market OI data for symbol: %v", symbol)
				}
			} else {
				xlog.Infof("Hedger Service - No existing positions for market ID %v", productID)

				// Ensure the marketOi has the necessary data
				if ammData, ok := marketOi[productIDStr]; ok {
					// Check the existence of "oi" and "is_long" in the map
					ammPosition, okPos := ammData["oi"].(string)
					ammIsLong, okLong := ammData["is_long"].(bool)

					if !okPos {
						xlog.Errorf("Hedger Service - 'oi' not found or invalid type for product ID %v", productID)
						continue
					}

					if !okLong {
						xlog.Errorf("Hedger Service - 'is_long' not found or invalid type for product ID %v", productID)
						continue
					}

					// Open a new position based on the AMM data
					hs.okxClient.PlaceMarketOrder(symbol, ammPosition, ammIsLong)
				} else {
					xlog.Errorf("Hedger Service - Missing market OI data for symbol: %v", symbol)
				}
			}
		}
	}
}

func (hs *HedgerService) FetchMarketOi() (map[string]map[string]interface{}, error) {
	// Initialize a map to store the final response
	response := make(map[string]map[string]interface{})

	// Fetch the OI and direction of all markets
	for productID := range contractUtils.PRODUCT_ID_SYMBOL_TO_MAP {
		// Convert productID to an integer for odd check
		if productID%2 != 0 {
			// Convert productID to string
			productIDStr := strconv.FormatUint(uint64(productID), 10)

			// Get the Redis keys for long and short positions
			longKey := xredis.GetTotalLongPositionKey(productIDStr)
			shortKey := xredis.GetTotalShortPositionKey(productIDStr)

			// Fetch long and short values from Redis
			longValueStr, err := hs.redisClient.Get(context.Background(), longKey).Result()
			if err != nil {
				xlog.Errorf("Error fetching long position for productID %s: %v", productIDStr, err)
				return nil, fmt.Errorf("error fetching long position for productID %s: %w", productIDStr, err)
			}

			shortValueStr, err := hs.redisClient.Get(context.Background(), shortKey).Result()
			if err != nil {
				xlog.Errorf("Error fetching short position for productID %s: %v", productIDStr, err)
				return nil, fmt.Errorf("error fetching short position for productID %s: %w", productIDStr, err)
			}

			// Parse the long and short values into big.Int
			longPos := new(big.Int)
			shortPos := new(big.Int)

			_, ok := longPos.SetString(longValueStr, 10) // Base 10
			if !ok {
				xlog.Errorf("Error parsing long position for productID %s: invalid format", productIDStr)
				return nil, fmt.Errorf("error parsing long position for productID %s: invalid format", productIDStr)
			}

			_, ok = shortPos.SetString(shortValueStr, 10) // Base 10
			if !ok {
				xlog.Errorf("Error parsing short position for productID %s: invalid format", productIDStr)
				return nil, fmt.Errorf("error parsing short position for productID %s: invalid format", productIDStr)
			}

			// Calculate io (Imbalance) using big.Int's Abs
			oi := new(big.Int).Abs(new(big.Int).Sub(longPos, shortPos))

			// Determine is_long based on comparison of longPos and shortPos
			isLong := longPos.Cmp(shortPos) > 0

			// Add the results to the response map
			response[productIDStr] = map[string]interface{}{
				"long":    longPos.String(),  // Convert back to string for consistency in JSON response
				"short":   shortPos.String(), // Convert back to string for consistency in JSON response
				"oi":      oi.String(),       // Imbalance in string form
				"is_long": isLong,
			}
		}
	}

	// Return the response and no error if everything is successful
	return response, nil
}
