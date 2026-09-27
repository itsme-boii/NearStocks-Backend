package testutils

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/engine"
	"github/eugenix-io/logx-inf-backend/services/engine/controller"
	"log"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		xlog.Fatalf("Error parsing int: %s", err)
	}
	return i
}

// TODO - Move this to a common test util
func addEntriesForMarket() {
	var markets = []map[string]string{
		{
			"symbol":                         "ETH-USD",
			"type":                           "PERPETUAL",
			"id":                             "1",
			"amt_to_qtm_conversion_expo":     "9",
			"price_to_qtm_conversion_expo":   "6",
			"max_position_valuex18":          "1000000000000000000000000000",
			"min_amountx18":                  "100000000000000",
			"base_asset":                     "ETH",
			"quote_asset":                    "USDC",
			"maker_fee_fractionx18":          "500000000000000",
			"taker_fee_fractionx18":          "500000000000000",
			"initial_margin_fractionx18":     "50000000000000000",
			"maintenance_margin_fractionx18": "20000000000000000",
		},
	}

	for _, market := range markets {
		m := (&db.MarketDB{}).Create(&db.MarketTable{
			Symbol: market["symbol"],
			Type:   ctypes.MarketType(market["type"]),
			BaseTable: db.BaseTable{
				ID: uint(mustParseInt(market["id"])),
			},
			AmtToQtmConversionExpo:       int32(mustParseInt(market["amt_to_qtm_conversion_expo"])),
			PriceToQtmConversionExpo:     int32(mustParseInt(market["price_to_qtm_conversion_expo"])),
			MaxPositionValuex18:          ctypes.NewBigIntFromString(market["max_position_valuex18"]),
			MinAmountx18:                 ctypes.NewBigIntFromString(market["min_amountx18"]),
			BaseAsset:                    market["base_asset"],
			QuoteAsset:                   market["quote_asset"],
			MakerFeeFractionx18:          ctypes.NewBigIntFromString(market["maker_fee_fractionx18"]),
			TakerFeeFractionx18:          ctypes.NewBigIntFromString(market["taker_fee_fractionx18"]),
			InitialMarginFractionx18:     ctypes.NewBigIntFromString(market["initial_margin_fractionx18"]),
			MaintenanceMarginFractionx18: ctypes.NewBigIntFromString(market["maintenance_margin_fractionx18"]),
			IsActive:                     true,
		})

		xlog.Debugf("Added Market ID: %+v", m.ID)
	}
}

func InitEngineServer(t *testing.T) func() {
	// Setup matching engine
	// Mock db for orderbook

	SetMainnetEnv()
	SetupEngineEnv()

	SetupDBEnv(t)
	db.Init()

	// Add entries for market
	addEntriesForMarket()

	// Init engine
	engine.Init()

	// Init gin router
	r := gin.Default()

	// Register routes
	rg := r.Group("/api/v1")
	controller.RegisterMessageController(rg)

	// Start Matching server
	port := os.Getenv("PORT")

	// Define the server and set its properties
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	closeServer := func() {
		xlog.Infof("Closing Balance Server")
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Fatalf("Server close failed: %+v", err)
		} else {
			xlog.Infof("Balance Server closed")
		}
	}

	return closeServer
}
