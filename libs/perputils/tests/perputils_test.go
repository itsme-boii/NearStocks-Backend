package tests

import (
	"context"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/testutils"
	"math/big"
	"strconv"
	"testing"
)

func setup() {
	testutils.SetupContractEnv()
	contractUtils.Init()
	contract.Init()
}

func TestUpdateLongShortPositions(t *testing.T) {
	setup()

	// Define test cases in tabular form
	testCases := []struct {
		initialTotalLong   *big.Int
		initialTotalShort  *big.Int
		lastAmount         *big.Int
		finalAmount        *big.Int
		expectedTotalLong  *big.Int
		expectedTotalShort *big.Int
	}{
		// Case 1: new long position
		{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(100), big.NewInt(100), big.NewInt(0)},

		// Case 2: new long position
		{big.NewInt(100), big.NewInt(0), big.NewInt(100), big.NewInt(50), big.NewInt(150), big.NewInt(0)},

		// Case 3: new short position
		{big.NewInt(150), big.NewInt(0), big.NewInt(150), big.NewInt(-50), big.NewInt(100), big.NewInt(0)},

		// Case 4: new short position
		{big.NewInt(100), big.NewInt(0), big.NewInt(100), big.NewInt(-100), big.NewInt(0), big.NewInt(0)},

		// Case 5: new short position
		{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(-50), big.NewInt(0), big.NewInt(50)},

		// Case 6: new short position
		{big.NewInt(0), big.NewInt(50), big.NewInt(-50), big.NewInt(-100), big.NewInt(0), big.NewInt(150)},

		// Case 7: new long position differnt productId
		{big.NewInt(0), big.NewInt(150), big.NewInt(0), big.NewInt(100), big.NewInt(100), big.NewInt(150)},
	}

	// Iterate over test cases
	for i, tc := range testCases {
		finalLong, finalShort := perputils.NewPerpUtils().UpdateLongShortPositions(tc.initialTotalLong, tc.initialTotalShort, tc.lastAmount, tc.finalAmount)

		if finalLong.Cmp(tc.expectedTotalLong) != 0 || finalShort.Cmp(tc.expectedTotalShort) != 0 {
			t.Errorf("Test case %d failed: Expected finalTotalLong: %s, ifinalTotalShort: %s, got finalTotalLong: %s, finalTotalShort: %s",
				i+1, tc.expectedTotalLong.String(), tc.expectedTotalShort.String(), finalLong.String(), finalShort.String())
		}
	}
}

func TestGetTotalLongPosition(t *testing.T) {
	setup()
	// Use WithSetupMockRedis to initialize a Redis instance
	testutils.WithSetupMockRedis(t, func() {
		// Retrieve Redis client from the xredis package (or however you get the Redis client in your application)
		redisClient := xredis.GetRedisClient()

		// Initialize PerpUtils with the real Redis client

		productID := uint32(1)
		expectedValue := "1000"
		perputils := perputils.NewPerpUtils()

		longKey := xredis.GetTotalLongPositionKey(strconv.FormatUint(uint64(productID), 10))

		// Set initial value in Redis
		err := redisClient.Set(context.Background(), longKey, expectedValue, 0).Err()
		if err != nil {
			t.Fatalf("Failed to set up Redis key: %v", err)
		}

		// Execute the function
		result, err := perputils.GetTotalLongPosition(productID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Expected result
		expected := new(big.Int)
		expected.SetString(expectedValue, 10)

		// Validate result
		if result.Cmp(expected) != 0 {
			t.Errorf("Expected %s, got %s", expected.String(), result.String())
		}
	})
}
func TestGetTotalShortPosition(t *testing.T) {
	setup()
	// Use WithSetupMockRedis to initialize a Redis instance
	testutils.WithSetupMockRedis(t, func() {
		// Retrieve Redis client from the xredis package
		redisClient := xredis.GetRedisClient()

		// Initialize PerpUtils with the real Redis client
		perpUtils := perputils.NewPerpUtils()

		productID := uint32(2)
		expectedValue := "500"
		shortKey := xredis.GetTotalShortPositionKey(strconv.FormatUint(uint64(productID), 10))

		// Set initial value in Redis
		err := redisClient.Set(context.Background(), shortKey, expectedValue, 0).Err()
		if err != nil {
			t.Fatalf("Failed to set up Redis key: %v", err)
		}

		// Execute the function
		result, err := perpUtils.GetTotalShortPosition(productID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Expected result
		expected := new(big.Int)
		expected.SetString(expectedValue, 10)

		// Validate result
		if result.Cmp(expected) != 0 {
			t.Errorf("Expected %s, got %s", expected.String(), result.String())
		}
	})
}

func TestAddLongShortOIAtReddis(t *testing.T) {
	setup()
	// Use WithSetupMockRedis to initialize a Redis instance
	testutils.WithSetupMockRedis(t, func() {
		// Retrieve Redis client from the xredis package
		redisClient := xredis.GetRedisClient()

		// Initialize PerpUtils with the real Redis client
		perpUtils := perputils.NewPerpUtils()

		productID := uint32(3)
		currentVquote := big.NewInt(1000)
		deltaVquote := big.NewInt(500)
		longKey := xredis.GetTotalLongPositionKey(strconv.FormatUint(uint64(productID), 10))
		shortKey := xredis.GetTotalShortPositionKey(strconv.FormatUint(uint64(productID), 10))

		// Set initial values in Redis
		err := redisClient.Set(context.Background(), longKey, "1000", 0).Err()
		if err != nil {
			t.Fatalf("Failed to set up Redis key: %v", err)
		}

		err = redisClient.Set(context.Background(), shortKey, "500", 0).Err()
		if err != nil {
			t.Fatalf("Failed to set up Redis key: %v", err)
		}

		// Execute the function
		err = perpUtils.AddLongShortOIAtReddis(productID, currentVquote, deltaVquote)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Validate the long position was updated
		longPosition, err := redisClient.Get(context.Background(), longKey).Result()
		if err != nil {
			t.Fatalf("Error fetching long position: %v", err)
		}

		expectedLong := new(big.Int).Add(big.NewInt(1000), big.NewInt(500)).String()
		if longPosition != expectedLong {
			t.Errorf("Expected long position to be %s, got %s", expectedLong, longPosition)
		}
	})
}
func TestGetMarketOICap(t *testing.T) {
	setup()
	// Use WithSetupMockRedis to initialize a Redis instance
	testutils.WithSetupMockRedis(t, func() {
		// Retrieve Redis client from the xredis package
		redisClient := xredis.GetRedisClient()

		// Initialize PerpUtils with the real Redis client
		perpUtils := perputils.NewPerpUtils()

		marketID := uint32(1)
		expectedValue := "2500000"
		longMarketOICapKey := xredis.GetMarketLongOICap(strconv.FormatUint(uint64(marketID), 10))
		shortMarketOICapKey := xredis.GetMarketShortOICap(strconv.FormatUint(uint64(marketID), 10))

		// Set initial value in Redis
		err := redisClient.Set(context.Background(), longMarketOICapKey, expectedValue, 0).Err()
		if err != nil {
			t.Fatalf("Failed to set up Redis key: %v", err)
		}

		err = redisClient.Set(context.Background(), shortMarketOICapKey, expectedValue, 0).Err()
		if err != nil {
			t.Fatalf("Failed to set up Redis key: %v", err)
		}

		// Execute the function
		longResult, shortResult, err := perpUtils.GetMarketOICap(marketID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Expected result
		expected := new(big.Int)
		expected.SetString(expectedValue, 10)

		// Validate result
		if longResult.Cmp(expected) != 0 {
			t.Errorf("Expected %s, got %s", expected.String(), longResult.String())
		}

		if shortResult.Cmp(expected) != 0 {
			t.Errorf("Expected %s, got %s", expected.String(), shortResult.String())
		}
	})
}

func TestGetSubaccountOICap(t *testing.T) {
	setup()
	// Use WithSetupMockRedis to initialize a Redis instance
	testutils.WithSetupMockRedis(t, func() {
		// Retrieve Redis client from the xredis package
		redisClient := xredis.GetRedisClient()

		// Initialize PerpUtils with the real Redis client
		perpUtils := perputils.NewPerpUtils()

		expectedValue := "500000"
		subaccountOICapKey := xredis.GetSubaccountOICap()

		// Set initial value in Redis
		err := redisClient.Set(context.Background(), subaccountOICapKey, expectedValue, 0).Err()
		if err != nil {
			t.Fatalf("Failed to set up Redis key: %v", err)
		}

		// Execute the function
		result, err := perpUtils.GetSubaccountOICap()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Expected result
		expected := new(big.Int)
		expected.SetString(expectedValue, 10)

		// Validate result
		if result.Cmp(expected) != 0 {
			t.Errorf("Expected %s, got %s", expected.String(), result.String())
		}
	})
}

func TestGetGlobalNegativeOICap(t *testing.T) {
	setup()
	// Use WithSetupMockRedis to initialize a Redis instance
	testutils.WithSetupMockRedis(t, func() {
		// Retrieve Redis client from the xredis package
		redisClient := xredis.GetRedisClient()

		// Initialize PerpUtils with the real Redis client
		perpUtils := perputils.NewPerpUtils()

		expectedValue := "1000000"
		globalNegativeOICapKey := xredis.GetGlobalNegativeOICap()

		// Set initial value in Redis
		err := redisClient.Set(context.Background(), globalNegativeOICapKey, expectedValue, 0).Err()
		if err != nil {
			t.Fatalf("Failed to set up Redis key: %v", err)
		}

		// Execute the function
		result, err := perpUtils.GetGlobalNegativeOICap()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Expected result
		expected := new(big.Int)
		expected.SetString(expectedValue, 10)

		// Validate result
		if result.Cmp(expected) != 0 {
			t.Errorf("Expected %s, got %s", expected.String(), result.String())
		}
	})
}
