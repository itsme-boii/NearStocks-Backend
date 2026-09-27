package controller

import (
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/libs/xsocket"
	"net/http"
	"time"

	"math/big"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	INTERVAL_1M  = "1m"
	INTERVAL_5M  = "5m"
	INTERVAL_15M = "15m"
	INTERVAL_1H  = "1h"
	INTERVAL_4H  = "4h"
	INTERVAL_1D  = "1d"
)

type MarketPricesWebsocket struct {
	websocket    *xsocket.Websocket
	clientStatus map[xsocket.XConn]bool
	marketIds    []uint32
	redisClient  *redis.Client
	currentOHLC  map[string]map[uint32]*OHLC
}

type PriceUpdate struct {
	Token       uint32 `json:"token"`
	PriceUpdate struct {
		Price struct {
			Type string `json:"type"`
			Hex  string `json:"hex"`
		} `json:"price"`
		Expo        int   `json:"expo"`
		PublishTime int64 `json:"publishTime"`
	} `json:"priceUpdate"`
}

type OHLC struct {
	StartTime int64  `json:"t"`
	EndTime   int64  `json:"T"`
	Open      string `json:"o"`
	Close     string `json:"c"`
	High      string `json:"h"`
	Low       string `json:"l"`
}

// Helper function moved to package level for reuse
func getIntervalDuration(interval string) time.Duration {
	switch interval {
	case INTERVAL_1M:
		return time.Minute
	case INTERVAL_5M:
		return 5 * time.Minute
	case INTERVAL_15M:
		return 15 * time.Minute
	case INTERVAL_1H:
		return time.Hour
	case INTERVAL_4H:
		return 4 * time.Hour
	case INTERVAL_1D:
		return 24 * time.Hour
	default:
		return time.Minute
	}
}

func RegisterMarketPricesWebsocket(rg *gin.RouterGroup, wsUpgrader *websocket.Upgrader) *MarketPricesWebsocket {
	marketPricesWs := &MarketPricesWebsocket{
		websocket:    xsocket.NewWebsocket(wsUpgrader, nil),
		clientStatus: make(map[xsocket.XConn]bool),
		redisClient:  xredis.GetRedisClient(),
		marketIds: func() []uint32 {
			// WARNING !!! ToDo : we need to fetch only pre-market market IDs here.
			uints, err := (&db.PreMarketsDB{}).GetUniqueProductIDs()
			if err != nil {
				fmt.Printf("Error fetching unique product IDs: %v\n", err)
				return nil
			}
			uint32s := make([]uint32, len(uints))
			for i, v := range uints {
				uint32s[i] = uint32(v)
			}
			return uint32s
		}(),
		currentOHLC: make(map[string]map[uint32]*OHLC),
	}

	// Initialize maps for each interval
	intervals := []string{INTERVAL_1M, INTERVAL_5M, INTERVAL_15M, INTERVAL_1H, INTERVAL_4H, INTERVAL_1D}
	for _, interval := range intervals {
		marketPricesWs.currentOHLC[interval] = make(map[uint32]*OHLC)
	}

	// Initialize OHLC data from database for each market and interval
	currentTime := time.Now()
	for _, id := range marketPricesWs.marketIds {
		// Get current price for initialization
		price, err := FetchCurrentPrice(id)
		if err != nil {
			fmt.Printf("Error fetching price for market %d: %v\n", id, err)
			continue
		}
		priceFloat := new(big.Float).SetInt(price)
		// Divide by 10^18 to get the price in decimal form
		priceFloat.Quo(priceFloat, new(big.Float).SetInt64(1e18))

		priceStr := priceFloat.Text('f', 5)

		for _, interval := range intervals {
			latestCandle, err := (&db.PreMarketCandleDB{}).GetLatestCandleForInterval(id, interval)
			if err != nil {
				fmt.Printf("Error fetching latest candle for market %d, interval %s: %v\n", id, interval, err)
				continue
			}

			duration := getIntervalDuration(interval)

			if latestCandle == nil {
				// No previous candle exists, start fresh
				startTime := currentTime.Truncate(duration)
				endTime := startTime.Add(duration)

				marketPricesWs.currentOHLC[interval][id] = &OHLC{
					StartTime: startTime.UnixMilli(),
					EndTime:   endTime.UnixMilli(),
					Open:      priceStr,
					Close:     priceStr,
					High:      priceStr,
					Low:       priceStr,
				}
				continue
			}

			lastEndTime := time.UnixMilli(latestCandle.EndTime)
			timeSinceLastCandle := currentTime.Sub(lastEndTime)

			if timeSinceLastCandle >= duration {
				// Backfill missing intervals
				nextStartTime := lastEndTime
				for nextStartTime.Before(currentTime.Truncate(duration)) {
					endTime := nextStartTime.Add(duration)

					// Create a candle for the missing interval
					candle := &db.PreMarketCandleTable{
						ProductID:  id,
						StartTime:  nextStartTime.UnixMilli(),
						EndTime:    endTime.UnixMilli(),
						Interval:   interval,
						OpenPrice:  priceStr,
						HighPrice:  priceStr,
						LowPrice:   priceStr,
						ClosePrice: priceStr,
					}

					// Check if a candle with the same product_id, interval, and time range already exists
					exists, err := (&db.PreMarketCandleDB{}).CandleExists(id, interval, nextStartTime.UnixMilli(), endTime.UnixMilli())
					if err != nil {
						fmt.Printf("Error checking for existing candle: %v\n", err)
					}

					if !exists {
						_, err := (&db.PreMarketCandleDB{}).AddPreMarketCandle(*candle)
						if err != nil {
							fmt.Printf("Error storing backfilled candle data for interval %s: %v\n", interval, err)
						}
					}

					nextStartTime = endTime
				}

				// Set up the current interval
				startTime := currentTime.Truncate(duration)
				endTime := startTime.Add(duration)

				marketPricesWs.currentOHLC[interval][id] = &OHLC{
					StartTime: startTime.UnixMilli(),
					EndTime:   endTime.UnixMilli(),
					Open:      priceStr,
					Close:     priceStr,
					High:      priceStr,
					Low:       priceStr,
				}
			} else {
				// Continue with the current interval
				marketPricesWs.currentOHLC[interval][id] = &OHLC{
					StartTime: latestCandle.EndTime,
					EndTime:   time.UnixMilli(latestCandle.EndTime).Add(duration).UnixMilli(),
					Open:      latestCandle.OpenPrice,
					Close:     priceStr,
					High:      latestCandle.HighPrice,
					Low:       latestCandle.LowPrice,
				}
			}
		}
	}

	// Start broadcasting prices
	go marketPricesWs.BroadcastPrices()

	rg.GET("/pre-market/prices", marketPricesWs.RegisterClient)

	// Return the pointer
	return marketPricesWs
}

func (m *MarketPricesWebsocket) RegisterClient(ctx *gin.Context) {
	conn, err := m.websocket.UpgradeWS(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Could not upgrade connection to WebSocket"})
		return
	}

	xConn := xsocket.NewXConn(conn)
	defer xConn.Close()

	m.websocket.RegisterClient(xConn)
	m.clientStatus[xConn] = true

	// Handle connection lifecycle
	for {
		messageType, _, err := xConn.ReadMessage()
		if err != nil || messageType == websocket.CloseMessage {
			m.UnregisterClient(xConn)
			break
		}
	}
}

func (m *MarketPricesWebsocket) UnregisterClient(xConn xsocket.XConn) {
	m.websocket.UnregisterClient(xConn)
	delete(m.clientStatus, xConn)
}

func (m *MarketPricesWebsocket) updateOHLC(id uint32, priceStr string, currentTime time.Time) {
	// Update OHLC for each interval
	for interval := range m.currentOHLC {
		duration := getIntervalDuration(interval)

		if m.currentOHLC[interval][id] == nil {
			// Round down to the previous interval
			startTime := currentTime.Truncate(duration)
			endTime := startTime.Add(duration)

			m.currentOHLC[interval][id] = &OHLC{
				StartTime: startTime.UnixMilli(),
				EndTime:   endTime.UnixMilli(),
				Open:      priceStr,
				Close:     priceStr,
				High:      priceStr,
				Low:       priceStr,
			}
			continue
		}

		hloc := m.currentOHLC[interval][id]
		hloc.Close = priceStr

		// Update high/low
		currentPrice, _ := new(big.Float).SetString(priceStr)
		high, _ := new(big.Float).SetString(hloc.High)
		low, _ := new(big.Float).SetString(hloc.Low)

		if currentPrice.Cmp(high) > 0 {
			hloc.High = priceStr
		}
		if currentPrice.Cmp(low) < 0 {
			hloc.Low = priceStr
		}

		// Check if current time has reached or passed the end time
		endTime := time.UnixMilli(hloc.EndTime)
		if currentTime.Unix() >= endTime.Unix() {
			candle := &db.PreMarketCandleTable{
				ProductID:  id,
				StartTime:  hloc.StartTime,
				EndTime:    hloc.EndTime,
				Interval:   interval,
				OpenPrice:  hloc.Open,
				HighPrice:  hloc.High,
				LowPrice:   hloc.Low,
				ClosePrice: hloc.Close,
			}

			// Check if a candle with the same product_id, interval, and time range already exists
			exists, err := (&db.PreMarketCandleDB{}).CandleExists(id, interval, hloc.StartTime, hloc.EndTime)
			if err != nil {
				fmt.Printf("Error checking for existing candle: %v\n", err)
			}

			if !exists {
				_, err := (&db.PreMarketCandleDB{}).AddPreMarketCandle(*candle)
				if err != nil {
					fmt.Printf("Error storing candle data for interval %s: %v\n", interval, err)
				}
			}

			// Start new candle
			newStartTime := endTime
			newEndTime := newStartTime.Add(duration)

			m.currentOHLC[interval][id] = &OHLC{
				StartTime: newStartTime.UnixMilli(),
				EndTime:   newEndTime.UnixMilli(),
				Open:      priceStr,
				Close:     priceStr,
				High:      priceStr,
				Low:       priceStr,
			}
		}
	}
	// fmt.Printf("\n=== OHLC Data for Market ID %d ===\n", id)
	// for interval, marketData := range m.currentOHLC {
	// 	if hloc, exists := marketData[id]; exists {
	// 		fmt.Printf("Interval %s:\n  Start: %s\n  End: %s\n  Open: %s\n  High: %s\n  Low: %s\n  Close: %s\n\n",
	// 			interval,
	// 			time.UnixMilli(hloc.StartTime).Format(time.RFC3339),
	// 			time.UnixMilli(hloc.EndTime).Format(time.RFC3339),
	// 			hloc.Open,
	// 			hloc.High,
	// 			hloc.Low,
	// 			hloc.Close,
	// 		)
	// 	}
	// }
}

func (m *MarketPricesWebsocket) BroadcastPrices() {
	// ToDo : We are currently broadcasting every 5 seconds. We need to change to 1 second and make it configurable.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initialize map if not already initialized
	if m.currentOHLC == nil {
		m.currentOHLC = make(map[string]map[uint32]*OHLC)
	}

	doWork := func() {
		updates := make([]interface{}, 2)
		updates[0] = "priceUpdate"
		currentTime := time.Now()

		for _, id := range m.marketIds {
			price, err := FetchCurrentPrice(id)
			if err == nil {
				// Convert price to string for OHLC
				priceFloat := new(big.Float).SetInt(price)
				priceFloatDiv18 := new(big.Float).Quo(priceFloat, new(big.Float).SetInt64(1e18))
				priceStrDiv18 := priceFloatDiv18.Text('f', 5)

				// Update OHLC data
				m.updateOHLC(id, priceStrDiv18, currentTime)

				update := PriceUpdate{
					Token: id,
					PriceUpdate: struct {
						Price struct {
							Type string `json:"type"`
							Hex  string `json:"hex"`
						} `json:"price"`
						Expo        int   `json:"expo"`
						PublishTime int64 `json:"publishTime"`
					}{
						Price: struct {
							Type string `json:"type"`
							Hex  string `json:"hex"`
						}{
							Type: "BigNumber",
							Hex:  "0x" + price.Text(16),
						},
						Expo:        -18,
						PublishTime: time.Now().Unix(),
					},
				}
				updates[1] = update

				// Broadcast to all connected clients
				for xConn, active := range m.clientStatus {
					if active {
						m.websocket.BroadcastMessageToClient(xConn, updates)
					}
				}
			}
		}
	}

	doWork()
	for range ticker.C {
		doWork()
	}
}
