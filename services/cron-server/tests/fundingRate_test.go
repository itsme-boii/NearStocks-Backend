package tests

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/cron-server/funding"
	"github/eugenix-io/logx-inf-backend/testutils"
)

func TestCalculateFundingRate(t *testing.T) {
	testCases := []struct {
		productId    uint
		openInterest string
		totalOI      string
	}{
		{productId: 1, openInterest: "6894112000000000000", totalOI: "400000"},      // ETH
		{productId: 3, openInterest: "-2313393501000000000", totalOI: "600000"},     // BTC
		{productId: 5, openInterest: "-30876027000000000000", totalOI: "200000"},    // LINK
		{productId: 91, openInterest: "133421493872938000000000", totalOI: "20000"}, // LOGX
	}

	testutils.WithSetupMockRedis(t, func() {
		redisClient := xredis.GetRedisClient()

		// Set up funding rate factor in Redis
		fundingRateFactorKey := xredis.GetAMMFundingRateFactorKey()
		err := redisClient.Set(context.Background(), fundingRateFactorKey, "100", 0).Err()
		if err != nil {
			t.Fatalf("Error setting funding rate factor: %v", err)
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("Product ID %d", tc.productId), func(t *testing.T) {
				fundingRate, err := funding.CalculateFundingRate(tc.productId, tc.openInterest, tc.totalOI)
				if err != nil {
					t.Fatalf("Error calculating funding rate for product %d: %v", tc.productId, err)
				}

				// Get funding caps using the function with fallback logic
				fundingCaps, err := funding.GetFundingCaps(redisClient, tc.productId)
				if err != nil {
					t.Fatalf("Error fetching funding caps for product %d: %v", tc.productId, err)
				}

				apy := calculateAPY(fundingRate)
				maxCap := float64(fundingCaps.MaxFundingCap)
				minCap := float64(fundingCaps.MinFundingCap)

				xlog.Infof("Calculate Funding Rate Test - Product ID: %d, Funding rate APY: %.2f%%, Min: %.2f%%, Max: %.2f%%",
					tc.productId, apy, minCap, maxCap)

				// Handle negative funding rates by using absolute value for comparison
				absApy := apy
				if apy < 0 {
					absApy = -apy
				}

				// Add some tolerance for the randomized bounds (the function adds 1-6 to max and subtracts 1-6 from min)
				tolerance := 10.0 // Account for randomization in the calculation
				expectedMin := minCap - tolerance
				expectedMax := maxCap + tolerance

				// Assert that absolute APY is within expected bounds
				if absApy < expectedMin || absApy > expectedMax {
					t.Fatalf("Funding rate APY %.2f%% (abs: %.2f%%) is out of expected bounds [%.2f%%, %.2f%%] for product ID %d",
						apy, absApy, expectedMin, expectedMax, tc.productId)
				}
			})
		}
	})
}

// calculateAPY converts the funding rate to an annualized percentage yield.
func calculateAPY(fundingRate int64) float64 {
	fundingRateInt := new(big.Int).SetInt64(fundingRate)

	// Multiplier for annualization
	multiplier := big.NewInt(365 * 24 * 60 * 60 * 100)

	// Multiply the funding rate by the multiplier
	fundingRateInt.Mul(fundingRateInt, multiplier)

	// Convert to big.Float and divide by 1e18
	fundingRateFloat := new(big.Float).SetInt(fundingRateInt)
	divisor := new(big.Float).SetFloat64(1e18)
	result := new(big.Float).Quo(fundingRateFloat, divisor)

	// Convert to float64
	resultFloat64, _ := result.Float64()
	return resultFloat64
}

// Test to check for delta (Balance) in case of Client and Contract > minimum difference to send discord log
func TestSendDifferenceInSpotBalanceMessage(t *testing.T) {
	testCases := []struct {
		contractBalanceStr string
		clientAmountStr    string
		productID          uint
	}{
		{ //diff is -
			contractBalanceStr: "1000000000000000000",
			clientAmountStr:    "1000000000000000005",
			productID:          1,
		},
		{ // 0 case
			contractBalanceStr: "20000000000000000000",
			clientAmountStr:    "20000000000000000000",
			productID:          21,
		},
		{ //diff is +
			contractBalanceStr: "25000000000000000022",
			clientAmountStr:    "25000000000000000011",
			productID:          19,
		},
		{ //diff is +
			contractBalanceStr: "6000000000000000020",
			clientAmountStr:    "6000000000000000009",
			productID:          13,
		},
		{ //diff is +
			contractBalanceStr: "0000000000000000120",
			clientAmountStr:    "-0000000000000000135",
			productID:          36,
		},
		{ //diff is + but smaller than 10
			contractBalanceStr: "0000000000000000005",
			clientAmountStr:    "-0000000000000000004",
			productID:          58,
		},
	}
	minmDelta := cutils.GetBigxCust(1)
	for _, item := range testCases {
		t.Run(fmt.Sprintf("ProductID_%d", item.productID), func(t *testing.T) {
			contractAmt := new(big.Int)
			clientAmt := new(big.Int)

			contractAmt.SetString(item.contractBalanceStr, 10)
			clientAmt.SetString(item.clientAmountStr, 10)

			diff := new(big.Int).Sub(contractAmt, clientAmt)

			if clientAmt.Cmp(contractAmt) != 0 && diff.Abs(diff).Cmp(minmDelta) == 1 {
				t.Logf("Significant balance delta detected for productID %d: Contract=%v, Client=%v", item.productID, contractAmt, clientAmt)
			}
		})
	}
}

// Test to check for delta (FundingX18, Quotebalance, Amount) in case of Client and Contract > minimum difference to send discord log
func TestSendDifferenceInPerpBalanceMessage(t *testing.T) {
	testCases := []struct {
		contractAmountStr  string
		clientAmountStr    string
		contractFundingStr string
		clientFundingStr   string
		contractQuoteStr   string
		clientQuoteStr     string
		productID          uint
	}{

		{ // 0 case
			contractAmountStr:  "20000000000000000000",
			clientAmountStr:    "20000000000000000000",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          1,
		},
		{ // 0 case
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "20000000000000000040",
			clientQuoteStr:     "20000000000000000040",
			productID:          2,
		},
		{ //0 case
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "20000000000000000050",
			clientFundingStr:   "20000000000000000050",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          3,
		},
		{ //All 0 case
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          4,
		},
		{ //diff is +
			contractAmountStr:  "25000000000000000022",
			clientAmountStr:    "25000000000000000011", // diff = 11 (above threshold)
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          5,
		},
		{ //diff is +
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "500000000145",
			clientFundingStr:   "500000000135",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          6,
		},
		{ //diff is +
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "6000000000000000020",
			clientQuoteStr:     "6000000000000000009",
			productID:          7,
		},
		{ //diff is + for all
			contractAmountStr:  "25000000000000000032",
			clientAmountStr:    "25000000000000000020",
			contractFundingStr: "25000000000000000035",
			clientFundingStr:   "25000000000000000020",
			contractQuoteStr:   "6000000000000000032",
			clientQuoteStr:     "6000000000000000020",
			productID:          8,
		},
		{ //diff is + but less than minDiff
			contractAmountStr:  "25000000000000000022",
			clientAmountStr:    "25000000000000000020",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          9,
		},
		{ //diff is + but less than minDiff
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "500000000145",
			clientFundingStr:   "500000000140",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          10,
		},
		{ //diff is + but less than minDiff
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "6000000000000000020",
			clientQuoteStr:     "6000000000000000012",
			productID:          30,
		},
		{ //diff is + but less than minDiff for all
			contractAmountStr:  "12",
			clientAmountStr:    "8",
			contractFundingStr: "15",
			clientFundingStr:   "9",
			contractQuoteStr:   "6000000000000000020",
			clientQuoteStr:     "6000000000000000012",
			productID:          12,
		},
		{ //diff is -
			contractAmountStr:  "15",
			clientAmountStr:    "28",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          13,
		},
		{ //diff is -
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "19",
			clientFundingStr:   "32",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          14,
		},
		{ //diff is -
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "6000000000000000020",
			clientQuoteStr:     "6000000000000000032",
			productID:          15,
		},
		{ //diff is - for all
			contractAmountStr:  "15",
			clientAmountStr:    "28",
			contractFundingStr: "14",
			clientFundingStr:   "25",
			contractQuoteStr:   "20",
			clientQuoteStr:     "35",
			productID:          16,
		},
		{ //diff is -but less than minDiff
			contractAmountStr:  "15",
			clientAmountStr:    "18",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          17,
		},
		{ //diff is -but less than minDiff
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "19",
			clientFundingStr:   "22",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          18,
		},
		{ //diff is - but less than minDiff
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "6000000000000000020",
			clientQuoteStr:     "6000000000000000022",
			productID:          19,
		},
		{ //diff is - for all
			contractAmountStr:  "15",
			clientAmountStr:    "18",
			contractFundingStr: "14",
			clientFundingStr:   "15",
			contractQuoteStr:   "20",
			clientQuoteStr:     "25",
			productID:          20,
		},
		{ //diff is +
			contractAmountStr:  "15",
			clientAmountStr:    "-18",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          21,
		},
		{ //diff is +
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "19",
			clientFundingStr:   "-22",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          22,
		},
		{ //diff is +
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "20",
			clientQuoteStr:     "-22",
			productID:          23,
		},
		{ //diff is +
			contractAmountStr:  "15",
			clientAmountStr:    "-18",
			contractFundingStr: "14",
			clientFundingStr:   "-15",
			contractQuoteStr:   "20",
			clientQuoteStr:     "-25",
			productID:          24,
		},
		{ //diff is +
			contractAmountStr:  "5",
			clientAmountStr:    "-3",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          25,
		},
		{ //diff is +
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "9",
			clientFundingStr:   "-1",
			contractQuoteStr:   "0",
			clientQuoteStr:     "0",
			productID:          26,
		},
		{ //diff is +
			contractAmountStr:  "0",
			clientAmountStr:    "0",
			contractFundingStr: "0",
			clientFundingStr:   "0",
			contractQuoteStr:   "5",
			clientQuoteStr:     "-3",
			productID:          27,
		},
		{ //diff is +
			contractAmountStr:  "2",
			clientAmountStr:    "-4",
			contractFundingStr: "2",
			clientFundingStr:   "-6",
			contractQuoteStr:   "2",
			clientQuoteStr:     "-7",
			productID:          28,
		},
	}
	minimumDelta := cutils.GetBigxCust(1)
	for _, item := range testCases {
		t.Run(fmt.Sprintf("ProductID_%d", item.productID), func(t *testing.T) {
			contractAmt := new(big.Int)
			clientAmt := new(big.Int)
			contractFunding := new(big.Int)
			clientFunding := new(big.Int)
			contractQuote := new(big.Int)
			clientQuote := new(big.Int)

			contractAmt.SetString(item.contractAmountStr, 10)
			clientAmt.SetString(item.clientAmountStr, 10)
			contractFunding.SetString(item.contractFundingStr, 10)
			clientFunding.SetString(item.clientFundingStr, 10)
			contractQuote.SetString(item.contractQuoteStr, 10)
			clientQuote.SetString(item.clientQuoteStr, 10)

			diffAmount := new(big.Int).Sub(contractAmt, clientAmt)
			diffFundingX18 := new(big.Int).Sub(contractFunding, clientFunding)
			diffQuoteBalance := new(big.Int).Sub(contractQuote, clientQuote)

			if new(big.Int).Abs(diffFundingX18).Cmp(minimumDelta) == 1 ||
				new(big.Int).Abs(diffQuoteBalance).Cmp(minimumDelta) == 1 ||
				new(big.Int).Abs(diffAmount).Cmp(minimumDelta) == 1 {
				t.Logf("Significant balance delta detected for productID %d: Contract=%v, Client=%v", item.productID, contractAmt, clientAmt)
			}
		})
	}
}
