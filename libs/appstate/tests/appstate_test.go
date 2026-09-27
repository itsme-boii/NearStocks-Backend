package tests

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/testutils/mocks"
	"github/eugenix-io/logx-inf-backend/xclient"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetAllOraclePrices(t *testing.T) {
	// Get without expiry
	xclient.GlobalOracleClient = mocks.NewMockOracleClient()

	appStateNew := appstate.NewAppState()

	oraclePrices, err := appStateNew.GetAllOraclePrices()
	assert.NoErrorf(t, err, "Expected no error, got %v", err)
	assert.NotNil(t, oraclePrices, "Expected not nil, got nil")

	// Get with expiry
	fetchExpiration := time.Second * 1
	oraclePrices, err = appStateNew.GetAllOraclePrices(fetchExpiration)
	assert.NoErrorf(t, err, "Expected no error, got %v", err)
	assert.NotNil(t, oraclePrices, "Expected not nil, got nil")

	// Check if the function is called only once
	cnt := xclient.GlobalOracleClient.GetFunctionCallCount("GetAllPrices")
	assert.Equalf(t, 1, cnt, "Expected 1, got %v", cnt)

	fetchExpiration = time.Microsecond * 1
	oraclePrices, err = appStateNew.GetAllOraclePrices(fetchExpiration)
	assert.NoErrorf(t, err, "Expected no error, got %v", err)
	assert.NotNil(t, oraclePrices, "Expected not nil, got nil")

	// Check if the function is called twice
	cnt = xclient.GlobalOracleClient.GetFunctionCallCount("GetAllPrices")
	assert.Equalf(t, 2, cnt, "Expected 2, got %v", cnt)
}

func TestGetAllOraclePricesParallel1(t *testing.T) {
	// Get without expiry
	xclient.GlobalOracleClient = mocks.NewMockOracleClient()
	appStateNew := appstate.NewAppState()

	// Test without caching
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			oraclePrices, err := appStateNew.GetAllOraclePrices(0)
			assert.NoErrorf(t, err, "Expected no error, got %v", err)
			assert.NotNil(t, oraclePrices, "Expected not nil, got nil")
		}()
	}

	wg.Wait()
	fnCount := xclient.GlobalOracleClient.GetFunctionCallCount("GetAllPrices")
	assert.Equalf(t, 100, fnCount, "Expected 100, got %v", fnCount)
}

func TestGetAllOraclePricesParallel2(t *testing.T) {
	// Test with caching
	// Reset state
	appStateNew := appstate.NewAppState()
	xclient.GlobalOracleClient = mocks.NewMockOracleClient()
	appStateNew.GetAllOraclePrices()

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			fetchExpiration := time.Second * 2
			oraclePrices, err := appStateNew.GetAllOraclePrices(fetchExpiration)
			assert.NoErrorf(t, err, "Expected no error, got %v", err)
			assert.NotNil(t, oraclePrices, "Expected not nil, got nil")
		}()
	}

	wg.Wait()
	fnCount := xclient.GlobalOracleClient.GetFunctionCallCount("GetAllPrices")
	assert.Equalf(t, 1, fnCount, "Expected 1, got %v", fnCount)
}

func TestGetAllOraclePricesParallel3(t *testing.T) {
	// Test will all different expiry
	// Reset state
	appStateNew := appstate.NewAppState()
	xclient.GlobalOracleClient = mocks.NewMockOracleClient()
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchExpiration := time.Microsecond * time.Duration(i)
			oraclePrices, err := appStateNew.GetAllOraclePrices(fetchExpiration)
			assert.NoErrorf(t, err, "Expected no error, got %v", err)
			assert.NotNil(t, oraclePrices, "Expected not nil, got nil")
		}()
	}
	wg.Wait()
	fnCount := xclient.GlobalOracleClient.GetFunctionCallCount("GetAllPrices")
	fmt.Printf("fnCount: %v\n", fnCount)
	assert.Greater(t, fnCount, 1, "Expected more than 1, got %v", fnCount)
	assert.Less(t, fnCount, 100, "Expected less than 100, got %v", fnCount)
}
