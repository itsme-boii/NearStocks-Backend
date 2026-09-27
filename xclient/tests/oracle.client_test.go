package tests

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/testutils"
	"github/eugenix-io/logx-inf-backend/xclient"
	"os"
	"testing"
)

func TestGetAllPrices(t *testing.T) {
	t.Skip("Skipping oracle client test in normal test runs")
	// Create new client
	os.Setenv("ORACLE_SERVER_URL", "https://oracle.hundred.exchange")
	testutils.SetMainnetEnv()
	contractUtils.Init()

	oc := xclient.NewOracleClient()
	allPrices, err := oc.GetAllPrices()
	if err != nil {
		t.Fatalf("Failed to get all prices: %v", err)
	}

	for _, marketId := range contractUtils.ALL_PERPS_ON_CONTRACT {
		symbol, exists := marketutils.GetBaseSymbolForProduct(marketId)
		if !exists {
			t.Errorf("Symbol not found for market ID: %d", marketId)
			continue
		}

		price, exists := allPrices[symbol]
		if !exists {
			t.Errorf("Price not found for symbol: %s (market ID: %d)", symbol, marketId)
			continue
		}

		if price.PriceX == nil {
			t.Errorf("Price value is nil for symbol: %s (market ID: %d)", symbol, marketId)
			continue
		}

		t.Logf("Validated price for %s: %v ", symbol, price.PriceX)
	}

	ret, err := oc.GetAllCollateralTokenPrices()
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	t.Logf("Response: %v", ret)

	if len(ret) != 2 {
		t.Errorf("Expected 2 prices, got %v", len(ret))
	}
}
