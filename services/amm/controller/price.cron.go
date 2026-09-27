package controller

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	priceCache = make(map[uint]float64)
	cacheMutex = sync.RWMutex{}
	appState   = appstate.NewAppState()
)

func fetchAllPrices() (map[uint]float64, error) {
	fetchExpiration := 1000 * time.Millisecond
	oraclePrices, err := appState.GetAllOraclePrices(fetchExpiration)
	if err != nil {
		return nil, fmt.Errorf("price cron - failed to get oracle prices from appstate: %v", err)
	}

	pricesX18 := mapPricesForAllMarkets(oraclePrices)

	// converting uint32 to uint for backward compatibility
	prices := make(map[uint]float64)
	for k, v := range pricesX18 {
		prices[uint(k)] = v
	}

	return prices, nil
}

func mapPricesForAllMarkets(oraclePrices map[string]ctypes.OraclePrice) map[uint32]float64 {
	prices := make(map[uint32]float64)

	marketList := contractUtils.ALL_PERPS_ON_CONTRACT 
	overrideMarketIds := cutils.GetTrimmedSplitEnv("OVERRIDE_MARKET_IDS", ",")
	if len(overrideMarketIds) > 0 {
		marketListFiltered := cutils.FilterSlice(marketList, func(marketId uint32) bool {
			return cutils.SliceExists(overrideMarketIds, fmt.Sprintf("%d", marketId))
		})
		marketList = marketListFiltered
	}

	for _, marketId := range marketList {
		symbol, exists := marketutils.GetBaseSymbolForProduct(marketId)
		if !exists {
			xlog.Errorf("Price Cron - symbol not found for marketId: %d", marketId)
			continue
		}

		if priceData, exists := oraclePrices[symbol]; exists {
			priceStr := cutils.X18ToFloatStr(priceData.Pricex18)
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				xlog.Errorf("Price Cron - failed to convert price string to float64 for symbol %s: %v", symbol, err)
				continue
			}
			if price <= 0 {
				// Excluded markets for XTRUMP and HARRIS from price check
				if marketId != contractUtils.HARRIS_MARKET && marketId != contractUtils.TRUMP_MARKET {
					xlog.Errorf("Price Cron - invalid price %f for symbol %s (marketId: %d)", price, symbol, marketId)
				}
				continue
			}

			prices[marketId] = price
		} else {
			xlog.Errorf("Price Cron - no oracle price found for symbol %s (marketId: %d)", symbol, marketId)
		}
	}

	return prices
}

func getPriceFromCache(marketId uint) (float64, bool) {
	cacheMutex.RLock()
	price, exists := priceCache[marketId]
	cacheMutex.RUnlock()
	return price, exists
}

func StartPriceCron() {
	c := cron.New()
	xlog.Infof("Starting price cron job")
	ceid, err := c.AddFunc("@every 1s", func() {
		prices, err := fetchAllPrices()
		if err != nil {
			xlog.Errorf("Price Cron - Error fetching all prices: %v\n", err)
			xclient.GlobalDiscordClient.SendWebhookMessage("Price Cron - Error fetching all prices: " + err.Error())
			return
		}

		cacheMutex.Lock()
		for marketID, price := range prices {
			priceCache[marketID] = price
		}
		cacheMutex.Unlock()
		// log every 2 minutes
		if atomic.AddInt64(&logCounter, 1)%120 == 0 {
			xlog.Infof("Price Cron - Successfully updated prices for perp markets")
		}
	})
	if err != nil {
		xlog.Errorf("Price Cron - Error adding cron job: %v\n", err)
	}

	xlog.Infof("Price Cron - Cron job added with ID: %v\n", ceid)

	c.Start()
}
