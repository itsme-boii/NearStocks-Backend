package tests

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/client"
	apiServices "github/eugenix-io/logx-inf-backend/services/api-server/services"
	balanceServices "github/eugenix-io/logx-inf-backend/services/balance-server/services"
	balanceTypes "github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"testing"
	"time"

	"github/eugenix-io/logx-inf-backend/testutils"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// Verifies that funding fees are stored in database fills
func TestFundingFeesStoredInFills(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	traderAliceSubaccountId := "2_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"
	solverSubaccountId := contractUtils.AMM_SUBACCOUNT_ID_1

	// Define expected funding fees
	expectedMakerFundingFees := cutils.FloatStrToX18("12.5") // Mock value: 12.5 USDC
	expectedTakerFundingFees := cutils.FloatStrToX18("-7.3") // Mock value: -7.3 USDC (user receives)

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()
		defer patches.Reset()

		// Mock the balance service to return specific funding fees
		patches.ApplyFunc((*balanceServices.BalanceService).AtomicUpdateBalanceForOrderMatch, func(
			_ *balanceServices.BalanceService,
			_ balanceTypes.UpdateSubaccountForMatchRequest,
		) (*balanceServices.ErrorUpdatingBalance, *big.Int, *big.Int, *big.Int, *big.Int) {
			return &balanceServices.ErrorUpdatingBalance{MakerError: false, TakerError: false},
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

		// Setup database
		testutils.SetupDBEnv(t)
		db.Init()

		// Initialize contracts (required for MatchOrders call)
		contractUtils.Init()
		contract.Init()

		// Create market
		market := &db.MarketTable{
			BaseTable: db.BaseTable{
				ID: 1,
			},
			Symbol: "ETH-USD",
			Type:   ctypes.MarketType("PERPETUAL"),
		}
		(&db.MarketDB{}).Create(market)

		// Mock balance operations to avoid external dependencies
		patchBalanceUnlock := gomonkey.ApplyFuncReturn((*balanceServices.BalanceLockerService).AtomicUpdateSubaccountForMatch, nil)

		// Mock LockBalance to allow order placement
		patches.ApplyFunc((*xclient.BalanceClient).LockBalance, func(
			_ *xclient.BalanceClient,
			_ balanceTypes.LockBalanceRequest,
		) (bool, error) {
			return true, nil
		})

		// Mock contract MatchOrders call to return success without actually calling the contract
		patches.ApplyFunc((*contract.EndpointContract).MatchOrders, func(
			_ *contract.EndpointContract,
			_ contract.MatchOrderRequest,
			_ uint,
		) error {
			return nil // Return success without calling the contract
		})

		orderService := apiServices.NewOrderService()

		//###############################################################################################
		//-------------------- 1. Place SOLVER limit sell order through API server ------------------
		//###############################################################################################
		makerOrder := &db.OrderTable{
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
		}

		// Place maker order through API server - it will be added to orderbook but won't match yet
		_, _, status, err := orderService.PlaceOrder(makerOrder, market)
		assert.NoError(t, err)
		assert.Equal(t, 200, status)

		// Get the maker order from database (ID may have changed)
		updatedMakerOrder := (&db.OrderDB{}).GetById(makerOrder.ID)
		assert.NotNil(t, updatedMakerOrder, "Maker order should exist in database")

		//###############################################################################################
		//-------------------- 2. Place Trader market buy order through API server -------------------
		//###############################################################################################

		takerOrder := &db.OrderTable{
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
		}

		// Place order through API server - this will create fills via matchOnchainAndCreateFills
		_, _, status, err = orderService.PlaceOrder(takerOrder, market)
		assert.NoError(t, err)
		assert.Equal(t, 200, status)

		// Get the taker order from database
		updatedTakerOrder := (&db.OrderDB{}).GetById(takerOrder.ID)
		assert.NotNil(t, updatedTakerOrder, "Taker order should exist in database")

		// Verify fills were created in database with funding fees
		makerFills := (&db.FillDB{}).GetAllByOrderId(fmt.Sprintf("%d", updatedMakerOrder.ID))
		takerFills := (&db.FillDB{}).GetAllByOrderId(fmt.Sprintf("%d", updatedTakerOrder.ID))

		assert.Greaterf(t, len(*makerFills), 0, "Expected at least one maker fill in database")
		assert.Greaterf(t, len(*takerFills), 0, "Expected at least one taker fill in database")

		if len(*makerFills) > 0 {
			makerFill := (*makerFills)[0]
			assert.NotNilf(t, makerFill.FundingFeesx18.Val, "Maker fill FundingFeesx18 should not be nil")
			assert.Equalf(t, makerFill.FundingFeesx18.Val.Cmp(expectedMakerFundingFees), 0,
				"Maker fill FundingFeesx18 should be %s (mocked value), got %s",
				expectedMakerFundingFees.String(), makerFill.FundingFeesx18.Val.String())
			xlog.Infof("✅ API Server E2E - Maker fill FundingFeesx18: %s (expected: %s)",
				makerFill.FundingFeesx18.Val.String(), expectedMakerFundingFees.String())
		}

		if len(*takerFills) > 0 {
			takerFill := (*takerFills)[0]
			assert.NotNilf(t, takerFill.FundingFeesx18.Val, "Taker fill FundingFeesx18 should not be nil")
			assert.Equalf(t, takerFill.FundingFeesx18.Val.Cmp(expectedTakerFundingFees), 0,
				"Taker fill FundingFeesx18 should be %s (mocked value), got %s",
				expectedTakerFundingFees.String(), takerFill.FundingFeesx18.Val.String())
			xlog.Infof("✅ API Server E2E - Taker fill FundingFeesx18: %s (expected: %s)",
				takerFill.FundingFeesx18.Val.String(), expectedTakerFundingFees.String())
		}

		patchBalanceUnlock.Reset()
	})
}

// Verifies that funding fees are stored in database fills for liquidation orders
func TestFundingFeesStoredInFillsForLiquidationOrders(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	traderAliceSubaccountId := "2_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"
	solverSubaccountId := contractUtils.AMM_SUBACCOUNT_ID_1

	// expected funding fees for liquidation
	expectedMakerFundingFees := cutils.FloatStrToX18("15.3") // liquidator
	expectedTakerFundingFees := cutils.FloatStrToX18("-8.7") // liquidatee receives

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()
		defer patches.Reset()

		// Mock the balance service liquidation method to return specific funding fees
		patches.ApplyFunc((*xclient.BalanceClient).UpdateSubaccountsForLiquidation, func(
			_ *xclient.BalanceClient,
			_ balanceTypes.FinaliseLiquidationRequest,
		) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
			return cutils.GetBig0(), // makerRealizedPnlx18
				cutils.GetBig0(), // takerRealizedPnlx18
				expectedMakerFundingFees, // makerFundingFeesx18 (liquidator)
				expectedTakerFundingFees, // takerFundingFeesx18 (liquidatee)
				nil
		})

		time.Sleep(900 * time.Millisecond)

		closeEngineSrv := testutils.InitEngineServer(t)
		defer closeEngineSrv()

		time.Sleep(100 * time.Millisecond)

		client.InitEngineClient()

		// Setup database
		testutils.SetupDBEnv(t)
		db.Init()

		// Initialize contracts (required for LiquidateSubaccount call)
		contractUtils.Init()
		contract.Init()

		// Create market
		market := &db.MarketTable{
			BaseTable: db.BaseTable{
				ID: 1,
			},
			Symbol: "ETH-USD",
			Type:   ctypes.MarketType("PERPETUAL"),
		}
		(&db.MarketDB{}).Create(market)

		// Mock balance service operations
		patchBalanceUnlock := gomonkey.ApplyFuncReturn((*balanceServices.BalanceLockerService).AtomicUpdateSubaccountForMatch, nil)

		patches.ApplyFunc((*xclient.BalanceClient).LockBalance, func(
			_ *xclient.BalanceClient,
			_ balanceTypes.LockBalanceRequest,
		) (bool, error) {
			return true, nil
		})

		// Mock contract call to return success without actually calling the contract
		patches.ApplyFunc((*contract.EndpointContract).LiquidateSubaccount, func(
			_ *contract.EndpointContract,
			_ contract.LiquidationRequest,
			_ uint,
		) error {
			return nil
		})

		orderService := apiServices.NewOrderService()

		//###############################################################################################
		//-------------------- 1. Place SOLVER limit buy order (liquidator) through API server --------
		//###############################################################################################
		makerOrder := &db.OrderTable{
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
		}

		// Place maker order through API server - it will be added to orderbook but won't match yet
		_, _, status, err := orderService.PlaceOrder(makerOrder, market)
		assert.NoError(t, err)
		assert.Equal(t, 200, status)

		// Get the maker order from database
		updatedMakerOrder := (&db.OrderDB{}).GetById(makerOrder.ID)
		assert.NotNil(t, updatedMakerOrder, "Maker order should exist in database")

		//###############################################################################################
		//-------------------- 2. Set up trader balance with position to be liquidated ----------------
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

		_, err = xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			return subaccount.ReplaceBalanceInRedis(pipe, &traderBalances)
		})
		assert.NoError(t, err)

		//###############################################################################################
		//-------------------- 3. Place liquidation order through API server -------------------------
		//###############################################################################################
		liquidationOrder := &db.OrderTable{
			BaseTable: db.BaseTable{
				ID: 2,
			},
			MarketId:       contractUtils.ETH_MARKET,
			SubaccountId:   traderAliceSubaccountId,
			Party:          ctypes.PARTY_TRADER,
			Side:           ctypes.ORDER_SIDE_SELL, // to close long position
			BrokerId:       2,
			Type:           ctypes.ORDER_TYPE_LIQUIDATION,
			TotalFilledx18: ctypes.NewBigIntFromString("0"),
			Pricex18:       cutils.FloatStrToBigIntX18("0"),
			Amountx18:      cutils.FloatStrToBigIntX18("0.2"),
			ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
			Status:         ctypes.ORDER_STATUS_OPEN,
			Timestamp:      cutils.TimestampMilliNow(),
			IsReduce:       true,
			Signature:      "",
			SessionKey:     "",
		}

		spotPricesx18 := map[uint32]*big.Int{
			contractUtils.ARB_USDC: cutils.Mulx18(big.NewInt(1)), // USDC = 1
		}
		perpPricesx18 := map[uint32]*big.Int{
			contractUtils.ETH_MARKET: cutils.FloatStrToBigIntX18("2500").Val, // ETH = 2500
		}

		// Place liquidation order through API server - this will create fills via matchOnchainAndCreateFills
		_, status, err = orderService.PlaceLiquidationOrder(liquidationOrder, market, spotPricesx18, perpPricesx18)
		assert.NoError(t, err)
		assert.Equal(t, 200, status)

		// Get the liquidation order from database
		updatedLiquidationOrder := (&db.OrderDB{}).GetById(liquidationOrder.ID)
		assert.NotNil(t, updatedLiquidationOrder, "Liquidation order should exist in database")

		// Verify fills were created in database with funding fees
		makerFills := (&db.FillDB{}).GetAllByOrderId(fmt.Sprintf("%d", updatedMakerOrder.ID))
		liquidationFills := (&db.FillDB{}).GetAllByOrderId(fmt.Sprintf("%d", updatedLiquidationOrder.ID))

		assert.Greaterf(t, len(*makerFills), 0, "Expected at least one maker fill in database")
		assert.Greaterf(t, len(*liquidationFills), 0, "Expected at least one liquidation fill in database")

		if len(*makerFills) > 0 {
			makerFill := (*makerFills)[0]
			assert.NotNilf(t, makerFill.FundingFeesx18.Val, "Maker fill FundingFeesx18 should not be nil")
			assert.Equalf(t, makerFill.FundingFeesx18.Val.Cmp(expectedMakerFundingFees), 0,
				"Maker fill FundingFeesx18 should be %s (mocked value), got %s",
				expectedMakerFundingFees.String(), makerFill.FundingFeesx18.Val.String())
			xlog.Infof("✅ API Server E2E Liquidation - Maker fill FundingFeesx18: %s (expected: %s)",
				makerFill.FundingFeesx18.Val.String(), expectedMakerFundingFees.String())
		}

		if len(*liquidationFills) > 0 {
			liquidationFill := (*liquidationFills)[0]
			assert.NotNilf(t, liquidationFill.FundingFeesx18.Val, "Liquidation fill FundingFeesx18 should not be nil")
			assert.Equalf(t, liquidationFill.FundingFeesx18.Val.Cmp(expectedTakerFundingFees), 0,
				"Liquidation fill FundingFeesx18 should be %s (mocked value), got %s",
				expectedTakerFundingFees.String(), liquidationFill.FundingFeesx18.Val.String())
			xlog.Infof("✅ API Server E2E Liquidation - Liquidatee fill FundingFeesx18: %s (expected: %s)",
				liquidationFill.FundingFeesx18.Val.String(), expectedTakerFundingFees.String())
		}

		patchBalanceUnlock.Reset()
	})
}
