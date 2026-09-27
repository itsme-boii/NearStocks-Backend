package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/engine"
	"github/eugenix-io/logx-inf-backend/services/engine/types"
	"github/eugenix-io/logx-inf-backend/testutils"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// Helper function to create test orders with realistic data
func createTestOrder(id uint, marketId uint, side ctypes.OrderSide, orderType ctypes.OrderType, amount, price string, subaccountId string) *db.OrderTable {
	return &db.OrderTable{
		BaseTable:      db.BaseTable{ID: id},
		SubaccountId:   subaccountId,
		BrokerId:       1,
		MarketId:       marketId,
		Side:           side,
		Type:           orderType,
		Amountx18:      cutils.FloatStrToBigIntX18(amount),
		Pricex18:       cutils.FloatStrToBigIntX18(price),
		TotalFilledx18: ctypes.NewBigIntFromString("0"),
		Status:         ctypes.ORDER_STATUS_OPEN,
		Party:          ctypes.PARTY_TRADER,
		ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
		Timestamp:      cutils.TimestampMilliNow(),
		IsReduce:       false,
		Signature:      "",
		SessionKey:     "",
	}
}

// Helper function to create solver orders
func createSolverOrder(id uint, marketId uint, side ctypes.OrderSide, orderType ctypes.OrderType, amount, price string) *db.OrderTable {
	order := createTestOrder(id, marketId, side, orderType, amount, price, contractUtils.AMM_SUBACCOUNT_ID_1)
	order.Party = ctypes.PARTY_SOLVER
	return order
}

// Helper function to setup trader balance
func setupTraderBalance(t *testing.T, subaccountId string, usdcBalance string) {
	traderSubaccountHex, _ := cutils.SubaccountIdToHex(subaccountId)
	traderBalances := subaccountTypes.SubaccountBalances{
		SubaccountId: traderSubaccountHex,
		SpotBalances: map[uint32]subaccountTypes.SpotBalance{
			contractUtils.ARB_USDC: {
				ProductId:  contractUtils.ARB_USDC,
				Balancex18: cutils.FloatStrToBigIntX18(usdcBalance).Val,
				Lockedx18:  cutils.GetBig0(),
			},
		},
		PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
	}

	_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		return subaccount.ReplaceBalanceInRedis(pipe, &traderBalances)
	})
	assert.NoError(t, err)
}

// Helper function to create a test market
func createTestMarket() *db.MarketTable {
	return &db.MarketTable{
		BaseTable:                db.BaseTable{ID: contractUtils.ETH_MARKET},
		Symbol:                   "ETH-USD",
		Type:                     ctypes.PERPETUAL,
		MinAmountx18:             ctypes.NewBigInt(cutils.FloatStrToBigIntX18("0.001").Val),  // 0.001 ETH
		MaxPositionValuex18:      ctypes.NewBigInt(cutils.FloatStrToBigIntX18("100000").Val), // 100k
		IsActive:                 true,
		PriceToQtmConversionExpo: 2,
		AmtToQtmConversionExpo:   18,
	}
}

func TestCancelBulkAndPlaceBulkOrder(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	traderAliceSubaccountId := "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_2"

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()
		defer patches.Reset()

		// Wait for balance server to start
		time.Sleep(900 * time.Millisecond)

		// Setup engine environment (including BALANCE_SERVER_URL)
		testutils.SetupEngineEnv()

		// Setup trader balance
		setupTraderBalance(t, traderAliceSubaccountId, "100000")

		//###############################################################################################
		//------------ 1. Create orderbook and place initial orders  -------------------------
		//###############################################################################################

		// Create the orderbook  (no HTTP layer)
		testMarket := createTestMarket()
		orderbook := engine.NewOrderbook(testMarket)

		// Place first solver order  on orderbook
		order1 := createSolverOrder(100, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_BUY, ctypes.ORDER_TYPE_LIMIT, "1.0", "3500")
		resp1 := orderbook.PlaceOrder(&types.PlaceOrderMsg{Order: order1})
		assert.NotNil(t, resp1)
		assert.Nil(t, resp1.Error)
		assert.Equal(t, uint(100), resp1.TakerOrder.ID)

		// Place second solver order  on orderbook
		order2 := createSolverOrder(101, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_SELL, ctypes.ORDER_TYPE_LIMIT, "1.5", "3600")
		resp2 := orderbook.PlaceOrder(&types.PlaceOrderMsg{Order: order2})
		assert.NotNil(t, resp2)
		assert.Nil(t, resp2.Error)
		assert.Equal(t, uint(101), resp2.TakerOrder.ID)

		//###############################################################################################
		//------------ 2. Test CancelBulkAndPlaceBulkOrder function directly -------------------------
		//###############################################################################################

		// Create orders to cancel (the ones we just placed)
		cancelOrders := []db.OrderTable{
			*createSolverOrder(100, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_BUY, ctypes.ORDER_TYPE_LIMIT, "1.0", "3500"),
			*createSolverOrder(101, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_SELL, ctypes.ORDER_TYPE_LIMIT, "1.5", "3600"),
		}

		// Create new orders to place
		placeOrders := []db.OrderTable{
			*createSolverOrder(200, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_BUY, ctypes.ORDER_TYPE_LIMIT, "2.0", "3450"),
			*createSolverOrder(201, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_SELL, ctypes.ORDER_TYPE_LIMIT, "1.8", "3650"),
			*createTestOrder(202, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_BUY, ctypes.ORDER_TYPE_LIMIT, "0.5", "3480", traderAliceSubaccountId),
		}

		// Create the message
		msg := &types.CancelBulkAndPlaceBulkOrderMsg{
			CancelOrders: &cancelOrders,
			PlaceOrders:  &placeOrders,
		}

		// Execute CancelBulkAndPlaceBulkOrder  on the orderbook
		bulkResp := orderbook.CancelBulkAndPlaceBulkOrder(msg)

		// Verify response
		assert.NotNil(t, bulkResp)
		assert.Nil(t, bulkResp.Error)

		// Check cancelled orders
		assert.NotNil(t, bulkResp.CancelledOrders)
		assert.Equal(t, 2, len(*bulkResp.CancelledOrders), "Should have cancelled 2 orders")

		// Verify cancel reasons
		for _, cancelledOrder := range *bulkResp.CancelledOrders {
			assert.Equal(t, "User request", cancelledOrder.CancelReason)
		}

		// Check placed orders
		assert.NotNil(t, bulkResp.PlaceOrderResponses)
		assert.Equal(t, 3, len(*bulkResp.PlaceOrderResponses), "Should have placed 3 orders")

		// Verify placed order IDs
		expectedOrderIDs := []uint{200, 201, 202}
		for i, placeResp := range *bulkResp.PlaceOrderResponses {
			assert.Equal(t, expectedOrderIDs[i], placeResp.TakerOrder.ID)
			assert.Equal(t, cutils.GetBig0(), placeResp.TakerOrder.TotalFilledx18, "New orders should not be filled initially")
		}

		// Check skipped orders (should be empty for successful case)
		assert.NotNil(t, bulkResp.SkippedCancelOrdersWithReason)

		respJson, _ := json.Marshal(bulkResp)
		xlog.Infof("CancelBulkAndPlaceBulkOrder call response: %s", string(respJson))
	})
}

func TestCancelBulkAndPlaceBulkOrder_EmptyInputs(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()
		defer patches.Reset()

		time.Sleep(900 * time.Millisecond)
		testutils.SetupEngineEnv()

		// Create orderbook
		testMarket := createTestMarket()
		orderbook := engine.NewOrderbook(testMarket)

		//###############################################################################################
		//------------ Test 1: Both arrays empty -------------------------
		//###############################################################################################

		emptyCancel := []db.OrderTable{}
		emptyPlace := []db.OrderTable{}

		msg1 := &types.CancelBulkAndPlaceBulkOrderMsg{
			CancelOrders: &emptyCancel,
			PlaceOrders:  &emptyPlace,
		}

		resp1 := orderbook.CancelBulkAndPlaceBulkOrder(msg1)

		// Verify response structure is correct even with empty inputs
		assert.NotNil(t, resp1)
		assert.Nil(t, resp1.Error)
		assert.NotNil(t, resp1.CancelledOrders)
		assert.NotNil(t, resp1.PlaceOrderResponses)
		assert.NotNil(t, resp1.SkippedCancelOrdersWithReason)
		assert.Equal(t, 0, len(*resp1.CancelledOrders))
		assert.Equal(t, 0, len(*resp1.PlaceOrderResponses))

		xlog.Infof("Empty inputs test passed")

		//###############################################################################################
		//------------ Test 2: Only cancel orders (empty place) -------------------------
		//###############################################################################################

		// First place an order to cancel
		order := createSolverOrder(300, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_BUY, ctypes.ORDER_TYPE_LIMIT, "1.0", "3500")
		orderbook.PlaceOrder(&types.PlaceOrderMsg{Order: order})

		cancelOnly := []db.OrderTable{*order}
		emptyPlace2 := []db.OrderTable{}

		msg2 := &types.CancelBulkAndPlaceBulkOrderMsg{
			CancelOrders: &cancelOnly,
			PlaceOrders:  &emptyPlace2,
		}

		resp2 := orderbook.CancelBulkAndPlaceBulkOrder(msg2)

		assert.NotNil(t, resp2)
		assert.Nil(t, resp2.Error)
		assert.Equal(t, 1, len(*resp2.CancelledOrders), "Should cancel 1 order")
		assert.Equal(t, 0, len(*resp2.PlaceOrderResponses), "Should place 0 orders")

		//###############################################################################################
		//------------ Test 3: Only place orders (empty cancel) -------------------------
		//###############################################################################################

		emptyCancel3 := []db.OrderTable{}
		placeOnly := []db.OrderTable{
			*createSolverOrder(400, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_BUY, ctypes.ORDER_TYPE_LIMIT, "1.0", "3400"),
		}

		msg3 := &types.CancelBulkAndPlaceBulkOrderMsg{
			CancelOrders: &emptyCancel3,
			PlaceOrders:  &placeOnly,
		}

		resp3 := orderbook.CancelBulkAndPlaceBulkOrder(msg3)

		assert.NotNil(t, resp3)
		assert.Nil(t, resp3.Error)
		assert.Equal(t, 0, len(*resp3.CancelledOrders), "Should cancel 0 orders")
		assert.Equal(t, 1, len(*resp3.PlaceOrderResponses), "Should place 1 order")

		xlog.Infof("EmptyInputs test completed successfully")
	})
}

func TestCancelBulkAndPlaceBulkOrder_PartialFailures(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	traderAliceSubaccountId := "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_2"

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()
		defer patches.Reset()

		time.Sleep(900 * time.Millisecond)
		testutils.SetupEngineEnv()
		setupTraderBalance(t, traderAliceSubaccountId, "100000")

		// Create orderbook
		testMarket := createTestMarket()
		orderbook := engine.NewOrderbook(testMarket)

		//###############################################################################################
		//------------ Setup: Place one order that exists -------------------------
		//###############################################################################################

		existingOrder := createSolverOrder(500, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_BUY, ctypes.ORDER_TYPE_LIMIT, "1.0", "3500")
		orderbook.PlaceOrder(&types.PlaceOrderMsg{Order: existingOrder})

		//###############################################################################################
		//------------ Test: Mix of existing and non-existing cancels + valid places -------------------------
		//###############################################################################################

		// Mix: one order exists (should cancel), one doesn't exist (should skip)
		cancelOrders := []db.OrderTable{
			*existingOrder, // exists - should cancel
			*createSolverOrder(999, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_SELL, ctypes.ORDER_TYPE_LIMIT, "1.5", "3600"), // doesn't exist - should skip
		}

		// Valid orders to place
		placeOrders := []db.OrderTable{
			*createSolverOrder(600, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_BUY, ctypes.ORDER_TYPE_LIMIT, "2.0", "3450"),
			*createTestOrder(601, contractUtils.ETH_MARKET, ctypes.ORDER_SIDE_SELL, ctypes.ORDER_TYPE_LIMIT, "1.0", "3650", traderAliceSubaccountId),
		}

		msg := &types.CancelBulkAndPlaceBulkOrderMsg{
			CancelOrders: &cancelOrders,
			PlaceOrders:  &placeOrders,
		}

		resp := orderbook.CancelBulkAndPlaceBulkOrder(msg)

		// Verify response
		assert.NotNil(t, resp)
		assert.Nil(t, resp.Error, "Function should not error even with partial failures")

		// Check cancelled orders - should have 1 successful cancel
		assert.NotNil(t, resp.CancelledOrders)
		assert.Equal(t, 1, len(*resp.CancelledOrders), "Should have 1 successfully cancelled order")
		assert.Equal(t, uint(500), (*resp.CancelledOrders)[0].ID, "Should cancel the existing order")

		// Check skipped orders - should have 1 skipped cancel
		assert.NotNil(t, resp.SkippedCancelOrdersWithReason)
		assert.Equal(t, 1, len(*resp.SkippedCancelOrdersWithReason), "Should have 1 skipped cancel order")
		assert.Equal(t, uint(999), (*resp.SkippedCancelOrdersWithReason)[0].ID, "Should skip the non-existing order")

		// Check placed orders - should place all valid orders
		assert.NotNil(t, resp.PlaceOrderResponses)
		assert.Equal(t, 2, len(*resp.PlaceOrderResponses), "Should place all valid orders")

		// Verify placed order IDs
		expectedPlaceIDs := []uint{600, 601}
		for i, placeResp := range *resp.PlaceOrderResponses {
			assert.Equal(t, expectedPlaceIDs[i], placeResp.TakerOrder.ID)
		}

		xlog.Infof("PartialFailures test completed successfully")
	})
}
