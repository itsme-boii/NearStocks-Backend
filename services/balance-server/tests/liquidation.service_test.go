package tests

import (
	"context"
	"encoding/json"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/services"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/testutils"
	"math/big"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var ONE = uint(1)

func TestSubaccountBalancesMarshalUnmarshalForRedis(t *testing.T) {
	balances := subaccountTypes.SubaccountBalances{
		SubaccountId: "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000001",
		SpotBalances: map[uint32]subaccountTypes.SpotBalance{
			4: {
				ProductId:  4,
				Balancex18: big.NewInt(100),
				Lockedx18:  big.NewInt(50),
			},
		},
		PerpBalances: map[uint32]subaccountTypes.PerpBalance{
			5: {
				Amountx18:             big.NewInt(100),
				VQuoteBalancex18:      big.NewInt(50),
				LastCumFundingRatex18: big.NewInt(10),
				ProductId:             5,
			},
		},
	}

	redisJson, err := balances.MarshalForRedis()
	if err != nil {
		t.Errorf("Error in MarshalForRedis: %v", err)
	}

	// Unmarshal
	var newBalances subaccountTypes.SubaccountBalances
	err = newBalances.UnmarshalForRedis(redisJson, balances.SubaccountId)
	if err != nil {
		t.Errorf("Error in UnmarshalForRedis: %v", err)
	}

	jsonBalances, err := json.Marshal(balances)
	if err != nil {
		t.Errorf("Error in Marshal: %v", err)
	}

	jsonNewBalances, err := json.Marshal(newBalances)

	if err != nil {
		t.Errorf("Error in Marshal: %v", err)
	}

	if string(jsonBalances) != string(jsonNewBalances) {
		t.Errorf("Expected: %v, got %v", string(jsonBalances), string(jsonNewBalances))
	}
}

func TestGetBalanceFromRedisAndReplace(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		subaccountId := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000002"
		lqs := services.NewLiquidationService()
		balances := lqs.MustGetBalanceFromRedis(subaccountId)

		spotBalanceFor4 := balances.MustGetSpotBalance(4)
		spotBalanceFor4.Balancex18 = new(big.Int).Add(spotBalanceFor4.Balancex18, big.NewInt(100))
		balances.SpotBalances[4] = spotBalanceFor4

		_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			err := subaccount.ReplaceBalanceInRedis(pipe, balances)
			if err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			t.Errorf("Error in ReplaceBalanceInRedis: %v", err)
		}

		// Fetch balance again and check if the balance is updated
		newBalances := lqs.MustGetBalanceFromRedis(subaccountId)
		diff := new(big.Int).Sub(newBalances.SpotBalances[4].Balancex18, balances.SpotBalances[4].Balancex18)
		if diff.Sign() != 0 {
			t.Errorf("Expected: 0, got %v", diff)
		}
	})
}

func TestSettlePnlForLiquidatee(t *testing.T) {
	// Apply the mock for GetBrokerFeeFactor at the beginning of the test
	patches := mockGetBrokerFeeFactor()
	defer patches.Reset()

	testutils.WithSetupMockRedis(t, func() {
		liquidatee := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000002"
		liquidator := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000001"
		lqs := services.NewLiquidationService()

		liquidateeBalances := &subaccountTypes.SubaccountBalances{
			SubaccountId: liquidatee,
			// Spot balance will have 100 USDC (ETH)
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
					ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
					Balancex18: cutils.Mulx18(big.NewInt(100)),
					Lockedx18:  big.NewInt(0),
				},
			},
			// Perp balance will have 1 ETH-USDC bought at 100
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{
				contractUtils.ETH_MARKET: {
					ProductId:             contractUtils.ETH_MARKET,
					Amountx18:             cutils.Mulx18(big.NewInt(1)),
					VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-100)),
					LastCumFundingRatex18: cutils.GetBig0(),
				},
			},
		}

		dummyPerpOraclePrices := make(map[uint32]*big.Int)
		for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
			dummyPerpOraclePrices[productId] = big.NewInt(0)
		}

		// Price went to $90
		dummyPerpOraclePrices[contractUtils.ETH_MARKET] = cutils.Mulx18(big.NewInt(90))

		dummySpotPrices := make(map[uint32]*big.Int)
		for _, productId := range contractUtils.ALL_SPOTS_IN_ORDER {
			dummySpotPrices[productId] = big.NewInt(0)
		}
		// Spot price of ETH-USDC is $1
		dummySpotPrices[contractUtils.QUOTE_TOKEN_PRODUCT_ID] = cutils.GetBigx18()

		liquidationRequest := &types.FinaliseLiquidationRequest{
			TxnCounter:             &ONE,
			LiquidateeOrderId:      1,
			LiquidatorSubaccountId: liquidator,
			LiquidateeSubaccountId: liquidatee,
			ProductId:              1,
			AmountX18:              cutils.Mulx18(big.NewInt(-1)), // For liquidatee it will be 1 ETH-USDC sold
			MatchPriceX18:          cutils.Mulx18(big.NewInt(90)), // Match price is $90
			PerpOraclePricesX18:    dummyPerpOraclePrices,
			SpotOraclePricesX18:    dummySpotPrices,
		}
		cumulativeFundingRate := cutils.GetBig0()
		// Settle PnL
		// PnL = 90 - 100 - trading and liquidation fee = -10 - (0.0005 + 0.001) * 90 = -10 - 0.135 = -10.135
		// Spot balance will be 100 - 10.135 = 89.865
		liquidateeBalances, realisedPnlLiquidateex18, _ := lqs.SettlePnlForLiquidatee(liquidationRequest, liquidateeBalances, cumulativeFundingRate)
		assert.Equal(t, 0, realisedPnlLiquidateex18.Cmp(cutils.Mulx18(big.NewInt(-10))), "Expected realised pnl to be -10135, but got %v", realisedPnlLiquidateex18)
		assert.Equal(t, 0, liquidateeBalances.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18.Cmp(cutils.MulxCust(big.NewInt(88605), 15)), "Expected spot balance to be 88.605, but got %v", liquidateeBalances.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18)
		assert.Equal(t, 0, liquidateeBalances.MustGetPerpBalance(contractUtils.ETH_MARKET).Amountx18.Sign(), "Expected perp balance to be 0")
		assert.Equal(t, 0, liquidateeBalances.MustGetPerpBalance(contractUtils.ETH_MARKET).VQuoteBalancex18.Sign(), "Expected perp balance to be 0")
		perputils := perputils.NewPerpUtils()
		_, err := perputils.GetTotalLongPosition(1)
		assert.NoError(t, err, "Error in getting total long position")
		_, err = perputils.GetTotalShortPosition(1)
		assert.NoError(t, err, "Error in getting total short position")
		// assert.Equal(t, cutils.Mulx18(big.NewInt(1000000000000000000)), totalLongPosition, "Expected Long OI to be 0") // it will be 10 if inital total will 100 for long
		// assert.Equal(t, cutils.Mulx18(big.NewInt(0)), totalShortPosition, "Expected SHORT OI to be 0")
	})
}

func TestUpdatePerpBalance(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		// Case: 1 - Remove entire position
		// Perp balance will have 1 ETH-USDC bought at 100
		ethPerpBalance := subaccountTypes.PerpBalance{
			ProductId:             contractUtils.ETH_MARKET,
			Amountx18:             cutils.Mulx18(big.NewInt(1)),
			VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-100)),
			LastCumFundingRatex18: cutils.GetBig0(),
		}

		// Update perp balance
		pnl, _ := ethPerpBalance.UpdateBalance(cutils.Mulx18(big.NewInt(-1)), cutils.Mulx18(big.NewInt(110)), cutils.GetBig0())

		assert.Equal(t, cutils.Mulx18(big.NewInt(10)), pnl, "Pnl should be +10")
		assert.Equal(t, 0, ethPerpBalance.Amountx18.Sign(), "Amount should be 0")
		assert.Equal(t, 0, ethPerpBalance.VQuoteBalancex18.Sign(), "VQuoteBalance should be 0")

		// Case: 2 - Increase position in same direction
		ethPerpBalance = subaccountTypes.PerpBalance{
			ProductId:             contractUtils.ETH_MARKET,
			Amountx18:             cutils.Mulx18(big.NewInt(1)),
			VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-100)),
			LastCumFundingRatex18: cutils.GetBig0(),
		}

		pnl, _ = ethPerpBalance.UpdateBalance(cutils.Mulx18(big.NewInt(2)), cutils.Mulx18(big.NewInt(-220)), cutils.GetBig0())
		assert.Equal(t, 0, pnl.Sign(), "Pnl should be 0 as there is no realisation of pnl")
		assert.Equal(t, 0, ethPerpBalance.Amountx18.Cmp(cutils.Mulx18(big.NewInt(3))), "Amount should be 3")
		assert.Equal(t, 0, ethPerpBalance.VQuoteBalancex18.Cmp(cutils.Mulx18(big.NewInt(-100-220))), "VQuoteBalance should be -320")

		// Case: 3 - Remove entire position and increase position in opposite direction
		ethPerpBalance = subaccountTypes.PerpBalance{
			ProductId:             contractUtils.ETH_MARKET,
			Amountx18:             cutils.Mulx18(big.NewInt(1)),
			VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-100)),
			LastCumFundingRatex18: cutils.GetBig0(),
		}

		pnl, _ = ethPerpBalance.UpdateBalance(cutils.Mulx18(big.NewInt(-2)), cutils.Mulx18(big.NewInt(220)), cutils.GetBig0())

		assert.Equal(t, cutils.Mulx18(big.NewInt(10)), pnl, "Pnl should be +10")
		assert.Equal(t, 0, ethPerpBalance.Amountx18.Cmp(cutils.Mulx18(big.NewInt(-1))), "Amount should be -1")
		assert.Equal(t, 0, ethPerpBalance.VQuoteBalancex18.Cmp(cutils.Mulx18(big.NewInt(110))), "VQuoteBalance should be +110")
	})
}

func TestSettleLiqPnlUsingSpots(t *testing.T) {
	// Create a mock liquidation request
	liquidatee := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000002"
	liquidator := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000001"
	liquidationReq := &types.FinaliseLiquidationRequest{
		LiquidateeOrderId:      1,
		LiquidatorSubaccountId: liquidatee,
		LiquidateeSubaccountId: liquidator,
		ProductId:              1,
		AmountX18:              cutils.Mulx18(big.NewInt(-1)),
		MatchPriceX18:          cutils.Mulx18(big.NewInt(100)),
		PerpOraclePricesX18:    nil,
		SpotOraclePricesX18: map[uint32]*big.Int{
			contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)),
		},
	}

	// Create a mock subaccount balance
	subaccountBalances := &subaccountTypes.SubaccountBalances{
		SubaccountId: "liquidatee",
		SpotBalances: map[uint32]subaccountTypes.SpotBalance{
			contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
				ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
				Balancex18: cutils.Mulx18(big.NewInt(11)),
				Lockedx18:  big.NewInt(0),
			},
		},
		PerpBalances: nil,
	}

	// Case: -$10 pnl
	totalPnlx36 := new(big.Int).Mul(big.NewInt(-10), cutils.GetBigx36())

	// Call the SettleLiqPnlUsingSpots function
	result := services.SettleLiqPnlUsingSpots(liquidationReq.SpotOraclePricesX18, totalPnlx36, subaccountBalances)

	// Assert the expected result
	assert.EqualValues(t, 0, result.Cmp(cutils.Mulx18(big.NewInt(1))), "Expected quote balance to be 1")
	assert.EqualValues(t, 0, subaccountBalances.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18.Cmp(cutils.Mulx18(big.NewInt(1))), "Expected spot balance to be 1")
}

// func TestSettleLiqPnlUsingPerps(t *testing.T) {
// 	// Create a mock liquidation request
// 	liquidatee := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000002"
// 	liquidator := "0x0000000000015631e1e23aeefae435106017bbcee2b91676e43b000000000002"

// 	liquidationReq := &subaccountTypes.FinaliseLiquidationRequest{
// 		LiquidateeOrderId:      1,
// 		LiquidatorSubaccountId: liquidator,
// 		LiquidateeSubaccountId: liquidatee,
// 		ProductId:              1,
// 		AmountX18:              cutils.Mulx18(big.NewInt(-1)),
// 		MatchPriceX18:          cutils.Mulx18(big.NewInt(100)),
// 		PerpOraclePricesX18: map[uint32]*big.Int{
// 			contractUtils.ETH_MARKET: cutils.Mulx18(big.NewInt(200)), // $200
// 		},
// 		SpotOraclePricesX18: nil,
// 	}

// 	// Create a mock subaccount balance
// 	subaccountBalances := &subaccountTypes.SubaccountBalances{
// 		SubaccountId: liquidatee,
// 		SpotBalances: nil,
// 		// Bought 1 ETH-USDC at $100
// 		PerpBalances: map[uint32]subaccountTypes.PerpBalance{
// 			contractUtils.ETH_MARKET: {
// 				ProductId:             contractUtils.ETH_MARKET,
// 				Amountx18:             cutils.Mulx18(big.NewInt(1)),
// 				VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-100)),
// 				LastCumFundingRatex18: cutils.GetBig0(),
// 			},
// 		},
// 	}

// 	// Case: -$10 pnl
// 	totalPnlx36 := new(big.Int).Mul(big.NewInt(-100), cutils.GetBigx36())

// 	// Create an instance of the LiquidationServiceImpl
// 	liquidationService := services.NewLiquidationService()

// 	// Call the SettleLiqPnlUsingPerps function
// 	result := liquidationService.SettleLiqPnlUsingPerps(liquidationReq, totalPnlx36, subaccountBalances)

// 	// Assert the expected result
// 	assert.EqualValues(t, 0, result.Sign(), "Expected pnl to be 0")
// 	assert.EqualValues(t, 0, subaccountBalances.MustGetPerpBalance(contractUtils.ETH_MARKET).Amountx18.Cmp(cutils.Mulx18(big.NewInt(1))), "Expected perp balance to be 1. No change should happen")
// 	assert.EqualValues(t, 0, subaccountBalances.MustGetPerpBalance(contractUtils.ETH_MARKET).VQuoteBalancex18.Cmp(cutils.Mulx18(big.NewInt(-200))), "VQuote should decrease to -200")
// }

func TestFinaliseLiquidation(t *testing.T) {
	// Apply the mock for GetBrokerFeeFactor at the beginning of the test
	patches := mockGetBrokerFeeFactor()
	defer patches.Reset()

	testutils.SetMainnetEnv()
	testutils.SetupBalanceMainnetEnv()
	defer testutils.ResetEnv()

	type TestCase struct {
		liquidateeBalances           *subaccountTypes.SubaccountBalances
		liquidatorBalances           *subaccountTypes.SubaccountBalances
		liquidationReq               *types.FinaliseLiquidationRequest
		cumulativeFundingRateDataMap map[uint32]*xredis.CumulativeFundingRateData
		expectedLiquidateeBalances   *subaccountTypes.SubaccountBalances
		expectedLiquidatorBalances   *subaccountTypes.SubaccountBalances
		expectedLiquidateePnl        *big.Int
		expectedLiquidatorPnl        *big.Int
		name                         string
	}

	var err error
	liquidatee := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000002"
	liquidator := "0x0000000000015631e1e23aeefae435106017bbcee2b91676e43b000000000002"

	testutils.WithSetupMockRedis(t, func() {
		testCases := []TestCase{
			{
				name: `Case 1: Liquidation with negative pnl
				Net loss is Sell Quote - Buy Quote - trading fee - liquidation fee
				Net loss is $100 - $300 - 100*0.0005 -100*0.001 = -$200 - $0.05 - $0.1 = -$200.15
				This will be settled using the spot balance of 100 USDC and Unrealised pnl of $101 of ETH-USDC`,
				cumulativeFundingRateDataMap: map[uint32]*xredis.CumulativeFundingRateData{
					contractUtils.TRUMP_MARKET: {
						CumulativeFundingRate:      "0",
						CumulativeFundingTimestamp: "0",
					}},
				liquidationReq: &types.FinaliseLiquidationRequest{
					TxnCounter:             &ONE,
					LiquidateeOrderId:      1,
					LiquidateeSubaccountId: liquidatee,
					LiquidatorSubaccountId: liquidator,
					ProductId:              contractUtils.TRUMP_MARKET,
					AmountX18:              cutils.Mulx18(big.NewInt(-1)),
					MatchPriceX18:          cutils.Mulx18(big.NewInt(100)), // Selling at $100
					PerpOraclePricesX18: map[uint32]*big.Int{
						contractUtils.ETH_MARKET:   cutils.Mulx18(big.NewInt(200)), // $200
						contractUtils.TRUMP_MARKET: cutils.Mulx18(big.NewInt(100)), // $100
					},
					SpotOraclePricesX18: map[uint32]*big.Int{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)), // $1
					},
				},
				liquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(100)), // 100 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(1)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-99)), // Bought at $99
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.TRUMP_MARKET: {
							ProductId:             contractUtils.TRUMP_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(1)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-300)), // Bought at $300
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				liquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedLiquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-10155), 16),
							Lockedx18:  big.NewInt(0),
						},
					},
					// Unchanged
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(1)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-99)),
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.TRUMP_MARKET: {
							ProductId:             contractUtils.TRUMP_MARKET,
							Amountx18:             cutils.GetBig0(),
							VQuoteBalancex18:      cutils.GetBig0(),
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.TRUMP_MARKET: {
							ProductId:             contractUtils.TRUMP_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(1)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-100)), // Bought at $300
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidateePnl: cutils.Mulx18(big.NewInt(-200)),
				expectedLiquidatorPnl: cutils.Mulx18(big.NewInt(0)),
			},
			{
				name: `Case 2: Liquidation with negative pnl`,
				cumulativeFundingRateDataMap: map[uint32]*xredis.CumulativeFundingRateData{
					contractUtils.BTC_MARKET: {
						CumulativeFundingRate:      "0",
						CumulativeFundingTimestamp: "0",
					},
				},
				liquidationReq: &types.FinaliseLiquidationRequest{
					TxnCounter:             &ONE,
					LiquidateeOrderId:      1,
					LiquidateeSubaccountId: liquidatee,
					LiquidatorSubaccountId: liquidator,
					ProductId:              contractUtils.BTC_MARKET,
					AmountX18:              cutils.Mulx18(big.NewInt(-5)),
					MatchPriceX18:          cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					PerpOraclePricesX18: map[uint32]*big.Int{
						contractUtils.BTC_MARKET: cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					},
					SpotOraclePricesX18: map[uint32]*big.Int{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)), // $1
					},
				},
				liquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(10)), // 10 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-1000)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				liquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedLiquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-13376), 15), // 0.512 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.GetBig0(),
							VQuoteBalancex18:      cutils.GetBig0(),
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-992)), // Bought at $300
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidateePnl: cutils.Mulx18(big.NewInt(-8)), // $(200 - $198.4) * 5
				expectedLiquidatorPnl: cutils.Mulx18(big.NewInt(0)),
			},
			{
				name: `Case 3: Liquidation with negative pnl 2`,
				cumulativeFundingRateDataMap: map[uint32]*xredis.CumulativeFundingRateData{
					contractUtils.BTC_MARKET: {
						CumulativeFundingRate:      "0",
						CumulativeFundingTimestamp: "0",
					},
				},
				liquidationReq: &types.FinaliseLiquidationRequest{
					TxnCounter:             &ONE,
					LiquidateeOrderId:      1,
					LiquidateeSubaccountId: liquidatee,
					LiquidatorSubaccountId: liquidator,
					ProductId:              contractUtils.BTC_MARKET,
					AmountX18:              cutils.Mulx18(big.NewInt(-5)),
					MatchPriceX18:          cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					PerpOraclePricesX18: map[uint32]*big.Int{
						contractUtils.ETH_MARKET: cutils.Mulx18(big.NewInt(102)),        // $102
						contractUtils.BTC_MARKET: cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					},
					SpotOraclePricesX18: map[uint32]*big.Int{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)), // $1
					},
				},
				liquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-500)), // Avg price = $100
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-1000)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				liquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedLiquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-23376), 15),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-500)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.GetBig0(),
							VQuoteBalancex18:      cutils.GetBig0(),
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-992)), // Bought at $300
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidateePnl: cutils.Mulx18(big.NewInt(-8)),
				expectedLiquidatorPnl: cutils.Mulx18(big.NewInt(0)),
			},
			{
				name: `Case 4: Liquidation with negative pnl 3`,
				cumulativeFundingRateDataMap: map[uint32]*xredis.CumulativeFundingRateData{
					contractUtils.BTC_MARKET: {
						CumulativeFundingRate:      "0",
						CumulativeFundingTimestamp: "0",
					},
				},
				liquidationReq: &types.FinaliseLiquidationRequest{
					TxnCounter:             &ONE,
					LiquidateeOrderId:      1,
					LiquidateeSubaccountId: liquidatee,
					LiquidatorSubaccountId: liquidator,
					ProductId:              contractUtils.BTC_MARKET,
					AmountX18:              cutils.Mulx18(big.NewInt(-5)),
					MatchPriceX18:          cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					PerpOraclePricesX18: map[uint32]*big.Int{
						contractUtils.ETH_MARKET: cutils.Mulx18(big.NewInt(101)),        // $102
						contractUtils.BTC_MARKET: cutils.MulxCust(big.NewInt(1984), 17), // $198.4
						contractUtils.SOL_MARKET: cutils.Mulx18(big.NewInt(101)),
					},
					SpotOraclePricesX18: map[uint32]*big.Int{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)), // $1
					},
				},
				liquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-500)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-1000)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.SOL_MARKET: {
							ProductId:             contractUtils.SOL_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-500)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				liquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedLiquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-23376), 15),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-500)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.SOL_MARKET: {
							ProductId:             contractUtils.SOL_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-500)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.GetBig0(),
							VQuoteBalancex18:      cutils.GetBig0(),
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-992)), // Bought at $300
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidateePnl: cutils.Mulx18(big.NewInt(-8)),
				expectedLiquidatorPnl: cutils.Mulx18(big.NewInt(0)),
			},
			{
				name: `Case 5: Liquidation with negative pnl 4`,
				cumulativeFundingRateDataMap: map[uint32]*xredis.CumulativeFundingRateData{
					contractUtils.BTC_MARKET: {
						CumulativeFundingRate:      "0",
						CumulativeFundingTimestamp: "0",
					},
				},
				liquidationReq: &types.FinaliseLiquidationRequest{
					TxnCounter:             &ONE,
					LiquidateeOrderId:      1,
					LiquidateeSubaccountId: liquidatee,
					LiquidatorSubaccountId: liquidator,
					ProductId:              contractUtils.BTC_MARKET,
					AmountX18:              cutils.Mulx18(big.NewInt(-5)),
					MatchPriceX18:          cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					PerpOraclePricesX18: map[uint32]*big.Int{
						contractUtils.ETH_MARKET: cutils.Mulx18(big.NewInt(101)),        // $102
						contractUtils.BTC_MARKET: cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					},
					SpotOraclePricesX18: map[uint32]*big.Int{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)), // $1
						contractUtils.OSTRICH_USDC_UPDATED:            cutils.Mulx18(big.NewInt(2)), // $2
					},
				},
				liquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.Mulx18(big.NewInt(3)), // 3 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-500)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-1000)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				liquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedLiquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-17376), 15),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-500)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-992)), // Bought at $300
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidateePnl: cutils.Mulx18(big.NewInt(-8)),
				expectedLiquidatorPnl: cutils.Mulx18(big.NewInt(0)),
			},
			{
				name: `Case 6: Liquidation with negative pnl and initial quote is also negative`,
				cumulativeFundingRateDataMap: map[uint32]*xredis.CumulativeFundingRateData{
					contractUtils.BTC_MARKET: {
						CumulativeFundingRate:      "0",
						CumulativeFundingTimestamp: "0",
					},
				},
				liquidationReq: &types.FinaliseLiquidationRequest{
					TxnCounter:             &ONE,
					LiquidateeOrderId:      1,
					LiquidateeSubaccountId: liquidatee,
					LiquidatorSubaccountId: liquidator,
					ProductId:              contractUtils.BTC_MARKET,
					AmountX18:              cutils.Mulx18(big.NewInt(-5)),
					MatchPriceX18:          cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					PerpOraclePricesX18: map[uint32]*big.Int{
						contractUtils.BTC_MARKET: cutils.MulxCust(big.NewInt(1984), 17), // $198.4
					},
					SpotOraclePricesX18: map[uint32]*big.Int{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)), // $1
						contractUtils.OSTRICH_USDC_UPDATED:               cutils.Mulx18(big.NewInt(1)), // $1
					},
				},
				liquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(-1)), // -1 USDC
							Lockedx18:  big.NewInt(0),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.Mulx18(big.NewInt(25)), // 25 USDT
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-1000)), // Avg price = $200
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				liquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedLiquidateeBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidatee,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.MulxCust(big.NewInt(624), 15), // 11 USDT
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.GetBig0(),
							VQuoteBalancex18:      cutils.GetBig0(),
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidatorBalances: &subaccountTypes.SubaccountBalances{
					SubaccountId: liquidator,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(500)), // 500 USDC
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.BTC_MARKET: {
							ProductId:             contractUtils.BTC_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(5)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-992)), // Bought at $300
							LastCumFundingRatex18: cutils.GetBig0(),
						},
					},
				},
				expectedLiquidateePnl: cutils.Mulx18(big.NewInt(-8)), // $(200 - $198.4) * 5
				expectedLiquidatorPnl: cutils.Mulx18(big.NewInt(0)),
			},
		}

		// prices come from each request; funding is read back from what the case stored in Redis
		liquidationService := &services.LiquidationServiceImpl{RedisClient: xredis.GetRedisClient(), SubaccountBal: subaccount.NewSubaccountBalanceImpl(), AppState: redisFundingAppState{}}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Update the balances
				liquidateeBalances := tc.liquidateeBalances
				liquidatorBalances := tc.liquidatorBalances
				liquidationReq := tc.liquidationReq

				_, err = liquidationService.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
					err := subaccount.ReplaceBalanceInRedis(pipe, liquidateeBalances)
					if err != nil {
						return err
					}
					err = subaccount.ReplaceBalanceInRedis(pipe, liquidatorBalances)
					if err != nil {
						return err
					}

					for productId, data := range tc.cumulativeFundingRateDataMap {
						err = pipe.Set(context.Background(), xredis.GetCumulativeFundingRateKey(contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[productId]), *data, 0).Err()
						if err != nil {
							return err
						}
					}

					return err
				})
				assert.NoErrorf(t, err, "Error in setting up balances: %v", err)

				resp, err := liquidationService.FinaliseLiquidation(liquidationReq)
				assert.NoErrorf(t, err, "Error in finalising liquidation: %v", err)

				balances := liquidationService.MustGetSubaccountBalancesFromIds([]string{liquidatee, liquidator})
				liquidateeBalances = &balances[0]
				liquidatorBalances = &balances[1]

				assert.Equal(t, tc.expectedLiquidateeBalances.Equals(liquidateeBalances), true, "Expected liquidatee balances to be %+v\n\n, but got %+v", tc.expectedLiquidateeBalances, liquidateeBalances)
				assert.Equal(t, tc.expectedLiquidatorBalances.Equals(liquidatorBalances), true, "Expected liquidator balances to be %+v\n\n, but got %+v", tc.expectedLiquidatorBalances, liquidatorBalances)
				assert.Equal(t, tc.expectedLiquidateePnl, resp.RealisedLiquidateePnlx18, "Expected liquidatee pnl to be %v, but got %v", tc.expectedLiquidateePnl, resp.RealisedLiquidateePnlx18)
				assert.Equal(t, tc.expectedLiquidatorPnl, resp.RealisedLiquidatorPnlx18, "Expected liquidator pnl to be %v, but got %v", tc.expectedLiquidatorPnl, resp.RealisedLiquidatorPnlx18)

				// Clear redis
				_, err = liquidationService.GetRedisClient().FlushAll(context.Background()).Result()
				assert.NoErrorf(t, err, "Error in flushing redis: %v", err)
			})
		}
	})
}

func TestSettleUsingInsurance(t *testing.T) {
	type TestCase struct {
		subaccountBalance         *subaccountTypes.SubaccountBalances
		insuranceBalance          *subaccountTypes.SubaccountBalances
		expectedSubaccountBalance *subaccountTypes.SubaccountBalances
		expectedInsuranceBalance  *subaccountTypes.SubaccountBalances
		spotPricesX18             map[uint32]*big.Int
		expectedErrorMsg          string
		name                      string
	}

	subaccountId := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000002"

	testutils.WithSetupMockRedis(t, func() {
		testCases := []TestCase{
			{
				name: `Case 1: Settle using insurance with when net spot balance is positive. No change in insurance balance`,
				spotPricesX18: map[uint32]*big.Int{
					contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)),
					contractUtils.OSTRICH_USDC_UPDATED:            cutils.MulxCust(big.NewInt(99), 16), // 0.99 dollars
				},
				subaccountBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountId,
					// Net spot balance = -99 + 103.95 = $4.95
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(-99)),
							Lockedx18:  big.NewInt(0),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.Mulx18(big.NewInt(105)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				insuranceBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: contractUtils.INSURANCE_SUBACCOUNT_ID,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(2)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				expectedSubaccountBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountId,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(0)),
							Lockedx18:  big.NewInt(0),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.Mulx18(big.NewInt(5)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				// Unchanged
				expectedInsuranceBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: contractUtils.INSURANCE_SUBACCOUNT_ID,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(2)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
			},
			{
				name: `Case 2: Settle using insurance with when net spot balance is negative. Insurance balance is used`,
				spotPricesX18: map[uint32]*big.Int{
					contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)),
					contractUtils.OSTRICH_USDC_UPDATED:            cutils.MulxCust(big.NewInt(99), 16), // 0.99 dollars
				},
				subaccountBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountId,
					// Net spot balance = -100 + 99 = -$1
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(-100)),
							Lockedx18:  big.NewInt(0),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				expectedSubaccountBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountId,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(0)),
							Lockedx18:  big.NewInt(0),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.Mulx18(big.NewInt(0)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				insuranceBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: contractUtils.INSURANCE_SUBACCOUNT_ID,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(2)),
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedInsuranceBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: contractUtils.INSURANCE_SUBACCOUNT_ID,
					// $1 settled using insurance
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(1)),
							Lockedx18:  big.NewInt(0),
						},
					},
				},
			},
			{
				name: `Case 3: Settle using insurance with when net spot balance is negative and only quote token balance is present. Insurance is sufficient`,
				spotPricesX18: map[uint32]*big.Int{
					contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)),
				},
				subaccountBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountId,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(-100)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				insuranceBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: contractUtils.INSURANCE_SUBACCOUNT_ID,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(104)),
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedSubaccountBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountId,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(0)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				expectedInsuranceBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: contractUtils.INSURANCE_SUBACCOUNT_ID,
					// $100 settled using insurance
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(4)),
							Lockedx18:  big.NewInt(0),
						},
					},
				},
			},
			{
				name: `Case 4: Settle using insurance with when net spot balance is negative and only quote token balance is present. Insurance is insufficient`,
				spotPricesX18: map[uint32]*big.Int{
					contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1)),
				},
				subaccountBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountId,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(-100)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				insuranceBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: contractUtils.INSURANCE_SUBACCOUNT_ID,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(3)),
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedSubaccountBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountId,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(-100)),
							Lockedx18:  big.NewInt(0),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				expectedInsuranceBalance: &subaccountTypes.SubaccountBalances{
					SubaccountId: contractUtils.INSURANCE_SUBACCOUNT_ID,
					// Insufficient insurance
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.Mulx18(big.NewInt(3)),
							Lockedx18:  big.NewInt(0),
						},
					},
				},
				expectedErrorMsg: "insufficient insurance balance",
			},
		}

		liquidationService := services.NewLiquidationService()

		for _, tc := range testCases {
			subaccountBalances := tc.subaccountBalance
			// Set initial balances in redis
			_, err := liquidationService.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
				err := subaccount.ReplaceBalanceInRedis(pipe, subaccountBalances)
				assert.NoErrorf(t, err, "Error in setting up balances: %v", err)

				err = subaccount.ReplaceBalanceInRedis(pipe, tc.insuranceBalance)
				assert.NoErrorf(t, err, "Error in setting up balances: %v", err)
				return nil
			})

			assert.NoErrorf(t, err, "Error in setting up balances: %v", err)

			err = liquidationService.SettleUsingInsurance(&types.SettleWithInsuranceRequest{
				SubaccountId:        subaccountId,
				SpotOraclePricesX18: tc.spotPricesX18,
			})

			if tc.expectedErrorMsg != "" {
				assert.Errorf(t, err, "Expected error message to be %s, but got %v", tc.expectedErrorMsg, err)
				return
			}

			assert.NoErrorf(t, err, "Error in settling using insurance: %v", err)

			balances := liquidationService.MustGetSubaccountBalancesFromIds([]string{subaccountId, contractUtils.INSURANCE_SUBACCOUNT_ID})
			subaccountBalances = &balances[0]
			insuranceBalances := &balances[1]

			assert.Equal(t, tc.expectedSubaccountBalance.Equals(subaccountBalances), true, "Expected subaccount balances to be %+v\n\n, but got %+v", tc.expectedSubaccountBalance, subaccountBalances)
			assert.Equalf(t, tc.expectedInsuranceBalance.Equals(insuranceBalances), true, "Expected insurance balances to be %+v\n\n, but got %+v", tc.expectedInsuranceBalance, tc.insuranceBalance)
		}
	})
}

// Use this function to set new balances in actual redis
func TestUpdatePerpBalanceLive(t *testing.T) {
	t.Skip("This test is for updating balances in actual redis")

	liquidationService := services.NewLiquidationService()
	subaccountBalances := &subaccountTypes.SubaccountBalances{
		SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000001",
		SpotBalances: map[uint32]subaccountTypes.SpotBalance{
			contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
				ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
				Balancex18: cutils.Mulx18(big.NewInt(10)),
				Lockedx18:  big.NewInt(0),
			},
		},
		PerpBalances: map[uint32]subaccountTypes.PerpBalance{
			contractUtils.ETH_MARKET: {
				ProductId:             contractUtils.ETH_MARKET,
				Amountx18:             cutils.Mulx18(big.NewInt(1)),
				VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-5000)),
				LastCumFundingRatex18: cutils.GetBig0(),
			},
		},
	}

	// Case: 1 - Remove entire position
	// Perp balance will have 1 ETH-USDC bought at 100
	_, err := liquidationService.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		return subaccount.ReplaceBalanceInRedis(pipe, subaccountBalances)
	})
	assert.NoErrorf(t, err, "Error in setting up balances: %v", err)
}
