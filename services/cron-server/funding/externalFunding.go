package funding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

var externalFundingRedisClient *redis.Client

// InitialiseExternalFundingRates initializes the Redis client.
func InitialiseExternalFundingRates() {
	externalFundingRedisClient = xredis.GetRedisClient()
	xlog.Infof("External Funding Rate Cron - Successfully Initialized")
}

// StartExternalFundingRateCron starts a cron job that runs every minute.
func StartExternalFundingRateCron() {
	xlog.Infof("Starting External Funding Rate Cron...")
	c := cron.New(cron.WithSeconds())

	// Run the cron job every 30 sec
	c.AddFunc("*/30 * * * * *", func() {
		xlog.Infof("External Funding Rate Cron running: %v", time.Now())

		// Fetch active perp markets
		activeMarkets := (&db.MarketDB{}).GetAllActivePerpMarkets()
		if len(activeMarkets) == 0 {
			xlog.Warnf("No active perpetual markets found.")
			return
		}

		symbols := make([]string, len(activeMarkets))
		for i, market := range activeMarkets {
			symbols[i] = market.Symbol
		}

		// Fetch funding rates from APIs
		fundingRates, err := fetchAllFundingRates(symbols)
		if err != nil {
			xlog.Errorf("Error fetching funding rates: %v", err)
			return
		}

		// Serialize updated data and store in Redis
		redisKey := xredis.GetExternalFundingRates()
		serializedData, err := json.Marshal(fundingRates)
		if err != nil {
			xlog.Errorf("Error serializing updated funding data: %v", err)
			return
		}

		err = externalFundingRedisClient.Set(context.Background(), redisKey, serializedData, time.Minute*5).Err()
		if err != nil {
			xlog.Errorf("Error storing updated funding data in Redis: %v", err)
		}
	})

	// Start the cron scheduler
	c.Start()

	// Keep the cron scheduler running in the background
	select {}
}

// fetchAllFundingRates fetches funding rates from Binance, Bybit, Bitget, and Hyperliquid.
func fetchAllFundingRates(symbols []string) (map[string]map[string]string, error) {
	combinedRates := make(map[string]map[string]string)

	// Fetch funding rates from Hyperliquid
	hyperliquidRates, errHyperliquid := fetchAllHyperliquidFundingRates()
	if errHyperliquid != nil {
		xlog.Errorf("Error fetching funding rates from Hyperliquid: %v", errHyperliquid)
	}

	// Fetch funding rates from Bitget
	bitgetRates, errBitget := fetchAllBitgetFundingRates()
	if errBitget != nil {
		xlog.Errorf("Error fetching funding rates from Bitget: %v", errBitget)
	}

	// Process funding rates for each symbol
	for _, symbol := range symbols {
		// Normalize symbol by removing "-USD"
		normalizedSymbol := strings.TrimSuffix(strings.TrimSuffix(symbol, "-USD"), "-USDC")
		bitgetSymbol := normalizedSymbol + "USDT" // Bitget uses USDT directly

		// Initialize rates map for the symbol
		rates := make(map[string]string)

		// Add rates from Hyperliquid for Binance, Bybit, and Hyperliquid
		if hlRates, ok := hyperliquidRates[normalizedSymbol]; ok {
			rates["BINANCE"] = safeGetRate(hlRates, "BinPerp")
			rates["BYBIT"] = safeGetRate(hlRates, "BybitPerp")
			rates["HYPERLIQUID"] = safeGetRate(hlRates, "HlPerp")
		} else {
			// Default to "0" if no rates available from Hyperliquid
			rates["BINANCE"] = "0"
			rates["BYBIT"] = "0"
			rates["HYPERLIQUID"] = "0"
		}

		// Add Bitget rates independently
		rates["BITGET"] = safeGetRate(bitgetRates, bitgetSymbol)

		combinedRates[symbol] = rates
	}

	return combinedRates, nil
}

// safeGetRate safely retrieves a funding rate from the given map, defaulting to "0" if not found.
func safeGetRate(rates map[string]string, symbol string) string {
	if rate, exists := rates[symbol]; exists {
		return rate
	}
	return "0"
}

// Fetch all Hyperliquid funding rates
func fetchAllHyperliquidFundingRates() (map[string]map[string]string, error) {
	baseURL := "https://api-ui.hyperliquid.xyz/info" // Actual Hyperliquid API URL

	// Define the request payload
	payload := map[string]string{
		"type": "predictedFundings",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Hyperliquid request payload: %w", err)
	}

	// Create a POST request
	resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Hyperliquid funding rates: %w", err)
	}
	defer resp.Body.Close()

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code from Hyperliquid: %d", resp.StatusCode)
	}

	// Parse the response body
	var data [][2]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Hyperliquid response: %w", err)
	}

	// Prepare rates map
	rates := make(map[string]map[string]string) // Symbol -> Exchange -> FundingRate
	for _, entry := range data {
		// Validate symbol entry
		symbol, ok := entry[0].(string)
		if !ok {
			xlog.Warnf("Unexpected symbol format: %+v", entry[0])
			continue
		}

		exchangeData, ok := entry[1].([]interface{})
		if !ok {
			continue
		}

		rates[symbol] = make(map[string]string)
		for _, exchangeEntry := range exchangeData {
			exchangeInfo, ok := exchangeEntry.([]interface{})
			if !ok || len(exchangeInfo) != 2 {
				continue
			}

			exchangeName, ok := exchangeInfo[0].(string)
			if !ok {
				continue
			}

			// Extract funding data
			if fundingData, ok := exchangeInfo[1].(map[string]interface{}); ok {
				if fundingRate, ok := fundingData["fundingRate"].(string); ok {
					if fundingRateFloat, err := strconv.ParseFloat(fundingRate, 64); err == nil {
						// Apply division by 8 for all except HlPerp
						if exchangeName == "HlPerp" {
							rates[symbol][exchangeName] = fmt.Sprintf("%.8f", fundingRateFloat*100)
						} else {
							rates[symbol][exchangeName] = fmt.Sprintf("%.8f", (fundingRateFloat*100)/8)
						}
					} else {
						rates[symbol][exchangeName] = "0"
					}
				} else {
					rates[symbol][exchangeName] = "0" // Default to 0 if funding rate is missing
				}
			}
		}
	}

	return rates, nil
}

// // Fetch all Binance funding rates
// func fetchAllBinanceFundingRates() (map[string]string, error) {
// 	baseURL := "https://fapi.binance.com/fapi/v1/premiumIndex"
// 	resp, err := http.Get(baseURL)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch Binance funding rates: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("non-200 status code from Binance: %d", resp.StatusCode)
// 	}

// 	var data []struct {
// 		Symbol          string `json:"symbol"`
// 		LastFundingRate string `json:"lastFundingRate"`
// 	}
// 	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
// 		return nil, fmt.Errorf("failed to decode Binance response: %w", err)
// 	}

// 	rates := make(map[string]string)
// 	for _, entry := range data {
// 		rate, err := strconv.ParseFloat(entry.LastFundingRate, 64)
// 		if err != nil {
// 			xlog.Warnf("Failed to parse funding rate for Binance symbol %s: %v", entry.Symbol, err)
// 			rates[entry.Symbol] = "0"
// 		} else {
// 			// Multiply rate by 100 and divide by 8
// 			rates[entry.Symbol] = fmt.Sprintf("%.8f", (rate*100)/8)
// 		}
// 	}

// 	return rates, nil
// }

// // Fetch all Bybit funding rates
// func fetchAllBybitFundingRates() (map[string]string, error) {
// 	baseURL := "https://api.bybit.com/v5/market/tickers?category=linear"
// 	resp, err := http.Get(baseURL)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch Bybit funding rates: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("non-200 status code from Bybit: %d", resp.StatusCode)
// 	}

// 	var data struct {
// 		Result struct {
// 			List []struct {
// 				Symbol      string `json:"symbol"`
// 				FundingRate string `json:"fundingRate"`
// 			} `json:"list"`
// 		} `json:"result"`
// 	}
// 	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
// 		return nil, fmt.Errorf("failed to decode Bybit response: %w", err)
// 	}

// 	rates := make(map[string]string)
// 	for _, entry := range data.Result.List {
// 		if entry.FundingRate == "" {
// 			rates[entry.Symbol] = "0"
// 			continue
// 		}

// 		rate, err := strconv.ParseFloat(entry.FundingRate, 64)
// 		if err != nil {
// 			xlog.Warnf("Failed to parse funding rate for Bybit symbol %s: %v", entry.Symbol, err)
// 			rates[entry.Symbol] = "0"
// 		} else {
// 			// Multiply rate by 100 and divide by 8  for hourly funding rate
// 			rates[entry.Symbol] = fmt.Sprintf("%.8f", (rate*100)/8)
// 		}
// 	}

// 	return rates, nil
// }

// Fetch all Bitget funding rates
func fetchAllBitgetFundingRates() (map[string]string, error) {
	baseURL := "https://api.bitget.com/api/v2/mix/market/tickers?productType=USDT-FUTURES" // Change to USDT professional futures
	resp, err := http.Get(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Bitget funding rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code from Bitget: %d", resp.StatusCode)
	}

	var result struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Symbol      string `json:"symbol"`
			FundingRate string `json:"fundingRate"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Bitget response: %w", err)
	}

	if result.Code != "00000" {
		return nil, fmt.Errorf("API error from Bitget: %s", result.Msg)
	}

	rates := make(map[string]string)
	for _, entry := range result.Data {
		rate, err := strconv.ParseFloat(entry.FundingRate, 64)
		if err != nil {
			xlog.Warnf("Failed to parse funding rate for Bitget symbol %s: %v", entry.Symbol, err)
			rates[entry.Symbol] = "0"
		} else {
			// Multiply rate by 100 and divide by 8 for hourly funding rate
			rates[entry.Symbol] = fmt.Sprintf("%.8f", (rate*100)/8)
		}
	}

	return rates, nil
}
