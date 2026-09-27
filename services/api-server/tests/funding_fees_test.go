package tests

import (
	"context"
	"encoding/json"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/client"
	"github/eugenix-io/logx-inf-backend/services/balance-server/services"
	balanceTypes "github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"

	"github/eugenix-io/logx-inf-backend/testutils"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// Verifies that funding fees are properly tracked and stored in engine response for regular order matches
func TestFundingFeesInRegularOrders(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	traderAliceSubaccountId := "2_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"
	solverSubaccountId := contractUtils.AMM_SUBACCOUNT_ID_1

	// Define expected funding fees for testing
	expectedMakerFundingFees := cutils.FloatStrToX18("10.5") // Mock value: 10.5 USDC
	expectedTakerFundingFees := cutils.FloatStrToX18("-5.2") // Mock value: -5.2 USDC (user receives)

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()
		defer patches.Reset()

		// Mock the balance service to return specific funding fees
		patches.ApplyFunc((*services.BalanceService).AtomicUpdateBalanceForOrderMatch, func(
			_ *services.BalanceService,
			_ balanceTypes.UpdateSubaccountForMatchRequest,
		) (*services.ErrorUpdatingBalance, *big.Int, *big.Int, *big.Int, *big.Int) {
			return &services.ErrorUpdatingBalance{MakerError: false, TakerError: false},
				cutils.GetBig0(), // makerRealizedPnl
				cutils.GetBig0(), // takerRealizedPnl
				expectedMakerFundingFees, // makerFundingFees
				expectedTakerFundingFees // takerFundingFees
		})

		time.Sleep(900 * time.Millisecond)

		closeEngineSrv := testutils.InitEngineServer(t)
		defer closeEngineSrv()

		time.Sleep(100 * time.Millisecond)

		client.InitEngineClient()

		//###############################################################################################
		//-------------------- 1. Place SOLVER limit sell order which won't be matched ------------------
		//###############################################################################################
		resp := client.GlobalEngineClient.CancelBulkAndPlaceOrder(&[]db.OrderTable{},
			&db.OrderTable{
				BaseTable: db.BaseTable{
					ID: 1,
				},
				MarketId:       contractUtils.ETH_MARKET,
				SubaccountId:   solverSubaccountId,
				Party:          ctypes.PARTY_SOLVER,
				Side:           ctypes.ORDER_SIDE_SELL,
				BrokerId:       2,
				Type:           ctypes.ORDER_TYPE_LIMIT,
				TotalFilledx18: ctypes.NewBigIntFromString("0"),
				Pricex18:       cutils.FloatStrToBigIntX18("100"),
				Amountx18:      cutils.FloatStrToBigIntX18("10"),
				ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
				Status:         ctypes.ORDER_STATUS_OPEN,
				Timestamp:      cutils.TimestampMilliNow(),
				IsReduce:       false,
				Signature:      "",
				SessionKey:     "",
			},
		)

		assert.NotNil(t, resp)
		assert.NoErrorf(t, resp.Error, "Failed to place order: %v", resp.Error)

		//###############################################################################################
		//-------------------- 2. Place Trader market buy order which will be matched -------------------
		//###############################################################################################
		patchBalanceUnlock := gomonkey.ApplyFuncReturn((*services.BalanceLockerService).AtomicUpdateSubaccountForMatch, nil)

		resp2 := client.GlobalEngineClient.PlaceOrder(
			&db.OrderTable{
				BaseTable: db.BaseTable{
					ID: 2,
				},
				MarketId:       contractUtils.ETH_MARKET,
				SubaccountId:   traderAliceSubaccountId,
				Party:          ctypes.PARTY_TRADER,
				Side:           ctypes.ORDER_SIDE_BUY,
				BrokerId:       2,
				Type:           ctypes.ORDER_TYPE_MARKET,
				TotalFilledx18: ctypes.NewBigIntFromString("0"),
				Pricex18:       cutils.FloatStrToBigIntX18("0"),
				Amountx18:      cutils.FloatStrToBigIntX18("1"),
				ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
				Status:         ctypes.ORDER_STATUS_OPEN,
				Timestamp:      cutils.TimestampMilliNow(),
				IsReduce:       false,
				Signature:      "",
				SessionKey:     "",
			},
		)

		assert.NotNilf(t, resp2, "Engine Response is nil")
		assert.NoErrorf(t, resp2.Error, "Failed to place order: %v", resp2.Error)
		assert.Equalf(t, len(*resp2.MatchedMakerOrders), 1, "Expected 1 matched maker order, got %d", len(*resp2.MatchedMakerOrders))

		matchedOrder := (*resp2.MatchedMakerOrders)[0]

		// Verify funding fees are present in the engine response and match mocked values
		assert.NotNilf(t, matchedOrder.MakerFundingFees, "MakerFundingFees should not be nil in engine response")
		assert.NotNilf(t, matchedOrder.TakerFundingFees, "TakerFundingFees should not be nil in engine response")
		assert.Equalf(t, matchedOrder.MakerFundingFees.Cmp(expectedMakerFundingFees), 0,
			"MakerFundingFees should be %s (mocked value), got %s",
			expectedMakerFundingFees.String(), matchedOrder.MakerFundingFees.String())
		assert.Equalf(t, matchedOrder.TakerFundingFees.Cmp(expectedTakerFundingFees), 0,
			"TakerFundingFees should be %s (mocked value), got %s",
			expectedTakerFundingFees.String(), matchedOrder.TakerFundingFees.String())

		xlog.Infof("✅ Funding fees verified in engine response - Maker: %s, Taker: %s",
			matchedOrder.MakerFundingFees.String(), matchedOrder.TakerFundingFees.String())

		respJson2, _ := json.Marshal(resp2)
		xlog.Infof("Funding Fees Test - Order placed: %+v", string(respJson2))

		patchBalanceUnlock.Reset()
	})
}

// Verifies that funding fees are properly tracked and stored in engine response for liquidation order matches
func TestFundingFeesInLiquidationOrders(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	traderAliceSubaccountId := "2_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"
	solverSubaccountId := contractUtils.AMM_SUBACCOUNT_ID_1

	// Define expected funding fees for liquidation
	expectedMakerFundingFees := cutils.FloatStrToX18("15.3") // Mock value: 15.3 USDC
	expectedTakerFundingFees := cutils.FloatStrToX18("-8.7") // Mock value: -8.7 USDC (liquidatee receives)

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()
		defer patches.Reset()

		// Mock the balance service match method to return specific funding fees for liquidation orders
		patches.ApplyFunc((*xclient.BalanceClient).UpdateSubaccountsForMatch, func(
			_ *xclient.BalanceClient,
			req balanceTypes.UpdateSubaccountForMatchRequest,
		) (bool, bool, *big.Int, *big.Int, *big.Int, *big.Int, error) {
			// Returns: takerSuccess, makerSuccess, makerRealizedPnlx18, takerRealizedPnlx18, makerFundingFeesx18, takerFundingFeesx18, err
			// For liquidation orders, return the mocked funding fees
			if req.IsLiquidation {
				return true, true, // takerSuccess, makerSuccess
					cutils.GetBig0(), // makerRealizedPnlx18
					cutils.GetBig0(), // takerRealizedPnlx18
					expectedMakerFundingFees, // makerFundingFeesx18 (liquidator)
					expectedTakerFundingFees, // takerFundingFeesx18 (liquidatee)
					nil
			}
			return true, true, cutils.GetBig0(), cutils.GetBig0(), cutils.GetBig0(), cutils.GetBig0(), nil
		})

		time.Sleep(900 * time.Millisecond)

		closeEngineSrv := testutils.InitEngineServer(t)
		defer closeEngineSrv()

		time.Sleep(100 * time.Millisecond)

		client.InitEngineClient()

		//###############################################################################################
		//-------------------- 1. Place SOLVER limit buy order (liquidator) ------------------
		//###############################################################################################
		resp := client.GlobalEngineClient.CancelBulkAndPlaceOrder(&[]db.OrderTable{},
			&db.OrderTable{
				BaseTable: db.BaseTable{
					ID: 1,
				},
				MarketId:       contractUtils.ETH_MARKET,
				SubaccountId:   solverSubaccountId,
				Party:          ctypes.PARTY_SOLVER,
				Side:           ctypes.ORDER_SIDE_BUY,
				BrokerId:       2,
				Type:           ctypes.ORDER_TYPE_LIMIT,
				TotalFilledx18: ctypes.NewBigIntFromString("0"),
				Pricex18:       cutils.FloatStrToBigIntX18("2500"),
				Amountx18:      cutils.FloatStrToBigIntX18("0.2"),
				ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
				Status:         ctypes.ORDER_STATUS_OPEN,
				Timestamp:      cutils.TimestampMilliNow(),
				IsReduce:       false,
				Signature:      "",
				SessionKey:     "",
			},
		)

		assert.NotNil(t, resp)
		assert.NoErrorf(t, resp.Error, "Failed to place order: %v", resp.Error)

		//###############################################################################################
		//-------------------- 2. Set up trader balance with position to be liquidated ---
		//###############################################################################################
		traderSubaccountHex, _ := cutils.SubaccountIdToHex(traderAliceSubaccountId)
		// Trader has a long position (positive Amountx18) that needs to be liquidated
		traderBalances := subaccountTypes.SubaccountBalances{
			SubaccountId: traderSubaccountHex,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.ARB_USDC: {
					ProductId:  contractUtils.ARB_USDC,
					Balancex18: cutils.Mulx18(big.NewInt(100)),
					Lockedx18:  cutils.GetBig0(),
				},
			},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{
				contractUtils.ETH_MARKET: {
					ProductId:             contractUtils.ETH_MARKET,
					Amountx18:             cutils.FloatStrToBigIntX18("0.2").Val,  // Long position of 0.2 ETH
					VQuoteBalancex18:      cutils.FloatStrToBigIntX18("-500").Val, // Negative vQuote (paid for position)
					LastCumFundingRatex18: cutils.GetBig0(),
				},
			},
		}

		_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			return subaccount.ReplaceBalanceInRedis(pipe, &traderBalances)
		})
		assert.NoError(t, err)

		//###############################################################################################
		//-------------------- 3. Place liquidation order through engine client ------------------
		//###############################################################################################
		// Note: Liquidation orders have Type: ORDER_TYPE_LIQUIDATION and IsReduce: true
		resp2 := client.GlobalEngineClient.PlaceOrder(
			&db.OrderTable{
				BaseTable: db.BaseTable{
					ID: 2,
				},
				MarketId:       contractUtils.ETH_MARKET,
				SubaccountId:   traderAliceSubaccountId,
				Party:          ctypes.PARTY_TRADER,
				Side:           ctypes.ORDER_SIDE_SELL, // Selling to close long position
				BrokerId:       2,
				Type:           ctypes.ORDER_TYPE_LIQUIDATION,
				TotalFilledx18: ctypes.NewBigIntFromString("0"),
				Pricex18:       cutils.FloatStrToBigIntX18("0"),
				Amountx18:      cutils.FloatStrToBigIntX18("0.2"),
				ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
				Status:         ctypes.ORDER_STATUS_OPEN,
				Timestamp:      cutils.TimestampMilliNow(),
				IsReduce:       true, // Liquidation orders are always reduce-only
				Signature:      "",
				SessionKey:     "",
			},
		)

		assert.NotNilf(t, resp2, "Engine Response is nil")
		assert.NoErrorf(t, resp2.Error, "Failed to place liquidation order: %v", resp2.Error)
		assert.Equalf(t, len(*resp2.MatchedMakerOrders), 1, "Expected 1 matched maker order, got %d", len(*resp2.MatchedMakerOrders))

		matchedOrder2 := (*resp2.MatchedMakerOrders)[0]

		// Verify funding fees are present in the engine response for liquidation order
		assert.NotNilf(t, matchedOrder2.MakerFundingFees, "MakerFundingFees should not be nil in engine response for liquidation order")
		assert.NotNilf(t, matchedOrder2.TakerFundingFees, "TakerFundingFees should not be nil in engine response for liquidation order")

		// Verify funding fees match expected values (from mocked balance service)
		assert.Equalf(t, matchedOrder2.MakerFundingFees.Cmp(expectedMakerFundingFees), 0,
			"MakerFundingFees should be %s (mocked value), got %s",
			expectedMakerFundingFees.String(), matchedOrder2.MakerFundingFees.String())
		assert.Equalf(t, matchedOrder2.TakerFundingFees.Cmp(expectedTakerFundingFees), 0,
			"TakerFundingFees should be %s (mocked value), got %s",
			expectedTakerFundingFees.String(), matchedOrder2.TakerFundingFees.String())

		xlog.Infof("Liquidation order - MakerFundingFees: %s, TakerFundingFees: %s",
			matchedOrder2.MakerFundingFees.String(), matchedOrder2.TakerFundingFees.String())

		respJson2, _ := json.Marshal(resp2)
		xlog.Infof("Funding Fees Test - Liquidation Order placed: %+v", string(respJson2))
	})
}
