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
	engineTypes "github/eugenix-io/logx-inf-backend/services/engine/types"
	"math/big"

	"github/eugenix-io/logx-inf-backend/testutils"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// 1. Place SOLVER limit order which won't be matched
// 2. Place TRADER market order which will be matched
func TestMessageControllerE2E(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	traderAliceSubaccountId := "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_2"
	solverSubaccountId := contractUtils.AMM_SUBACCOUNT_ID_1

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()

		defer patches.Reset()
		// Wait 100 ms to run the server
		time.Sleep(900 * time.Millisecond)

		closeEngineSrv := testutils.InitEngineServer(t)
		defer closeEngineSrv()

		// Wait 100 ms to run the server
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
				BrokerId:       1,
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
		assert.Emptyf(t, resp.MatchedMakerOrders, "Expected no matched maker orders, got %+v", resp.MatchedMakerOrders)
		assert.Emptyf(t, resp.CancelledOrders, "Expected no matched taker orders, got %+v", resp.CancelledOrders)
		assert.Emptyf(t, resp.SkippedCancelOrdersWithReason, "Expected no skipped cancel orders, got %+v", resp.SkippedCancelOrdersWithReason)
		assert.Equalf(t, resp.TakerOrder.ID, uint(1), "Expected order ID to be 1, got %d", resp.TakerOrder.ID)
		assert.Equalf(t, resp.TakerOrder.TotalFilledx18, cutils.GetBig0(), "Expected total filled to be 0, got %s", resp.TakerOrder.TotalFilledx18.String())

		respJson, _ := json.Marshal(resp)

		xlog.Infof("Step 1: Solver Order placed: %+v\n\n", string(respJson))

		//###############################################################################################
		//-------------------- 2. Place Trader market buy order which will be matched -------------------
		//###############################################################################################

		resp2 := client.GlobalEngineClient.PlaceOrder(
			&db.OrderTable{
				BaseTable: db.BaseTable{
					ID: 2,
				},
				MarketId:       contractUtils.ETH_MARKET,
				SubaccountId:   traderAliceSubaccountId,
				Party:          ctypes.PARTY_TRADER,
				Side:           ctypes.ORDER_SIDE_BUY,
				BrokerId:       1,
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

		assert.NotNilf(t, resp2, "Engine Response is nil for step 2")
		assert.NoErrorf(t, resp2.Error, "Failed to place order: %v", resp2.Error)
		assert.Equalf(t, len(*resp2.MatchedMakerOrders), 1, "Expected 1 matched maker order, got %d", len(*resp2.MatchedMakerOrders))
		assert.Emptyf(t, resp2.CancelledOrders, "Expected no matched taker orders, got %+v", resp2.CancelledOrders)

		expectedMakerOrder := engineTypes.MakerOrderRespData{
			ID:               1,
			TotalFilledx18:   cutils.FloatStrToX18("1"),
			Pricex18:         cutils.FloatStrToX18("100"),
			MakerRealizedPnl: cutils.GetBig0(),
			TakerRealizedPnl: cutils.GetBig0(),
			MakerFundingFees: cutils.GetBig0(),
			TakerFundingFees: cutils.GetBig0(),
			TxnCounter:       1,
			MatchedAmountx18: cutils.FloatStrToX18("1"),
		}
		expectedTakerOrder := engineTypes.TakerOrderRespData{
			ID:             2,
			TotalFilledx18: cutils.FloatStrToX18("1"),
		}

		assert.EqualValuesf(t, (*resp2.MatchedMakerOrders)[0], expectedMakerOrder, "Expected matched order to be %+v, got %+v", expectedMakerOrder, (*resp2.MatchedMakerOrders)[0])

		assert.EqualValuesf(t, resp2.TakerOrder, &expectedTakerOrder, "Expected taker order to be %+v, got %+v", expectedTakerOrder, resp2.TakerOrder)

		respJson2, _ := json.Marshal(resp2)
		xlog.Infof("Step 2: Trader Order placed: %+v", string(respJson2))

		//###############################################################################################
		//-------------------- 3. Place Trader limit buy order which will be matched -------------------
		//###############################################################################################
		// For this we'll be apply patch to mock successful balance unlock

		patchBalanceUnlock := gomonkey.ApplyFuncReturn((*services.BalanceLockerService).AtomicUpdateSubaccountForMatch, nil)

		resp3 := client.GlobalEngineClient.PlaceOrder(
			&db.OrderTable{
				BaseTable: db.BaseTable{
					ID: 3,
				},
				MarketId:       contractUtils.ETH_MARKET,
				SubaccountId:   traderAliceSubaccountId,
				Party:          ctypes.PARTY_TRADER,
				Side:           ctypes.ORDER_SIDE_BUY,
				BrokerId:       1,
				Type:           ctypes.ORDER_TYPE_LIMIT,
				TotalFilledx18: ctypes.NewBigIntFromString("0"),
				Pricex18:       cutils.FloatStrToBigIntX18("101"), // Price is higher than the solver order which means match price will be of solver
				Amountx18:      cutils.FloatStrToBigIntX18("2"),
				ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
				Status:         ctypes.ORDER_STATUS_OPEN,
				Timestamp:      cutils.TimestampMilliNow(),
				IsReduce:       false,
				Signature:      "",
				SessionKey:     "",
			},
		)

		assert.NotNilf(t, resp3, "Engine Response is nil for step 3")
		assert.NoErrorf(t, resp3.Error, "Failed to place order: %v", resp3.Error)
		assert.Equalf(t, len(*resp3.MatchedMakerOrders), 1, "Expected 1 matched maker order, got %d", len(*resp3.MatchedMakerOrders))
		assert.Emptyf(t, resp3.CancelledOrders, "Expected no matched taker orders, got %+v", resp3.CancelledOrders)

		expectedMakerOrder3 := engineTypes.MakerOrderRespData{
			ID:               1,
			TotalFilledx18:   cutils.FloatStrToX18("3"), // 1 + 2
			Pricex18:         cutils.FloatStrToX18("100"),
			MakerRealizedPnl: cutils.GetBig0(),
			TakerRealizedPnl: cutils.GetBig0(),
			MakerFundingFees: cutils.GetBig0(),
			TakerFundingFees: cutils.GetBig0(),
			TxnCounter:       2,
			MatchedAmountx18: cutils.FloatStrToX18("2"),
		}
		expectedTakerOrder3 := engineTypes.TakerOrderRespData{
			ID:             3,
			TotalFilledx18: cutils.FloatStrToX18("2"),
		}

		assert.EqualValuesf(t, (*resp3.MatchedMakerOrders)[0], expectedMakerOrder3, "Expected matched order to be %+v, got %+v", expectedMakerOrder3, (*resp3.MatchedMakerOrders)[0])

		assert.EqualValuesf(t, resp3.TakerOrder, &expectedTakerOrder3, "Expected taker order to be %+v, got %+v", expectedTakerOrder3, resp3.TakerOrder)

		respJson3, _ := json.Marshal(resp3)
		xlog.Infof("Step 3: Trader Limit Order placed: %+v", string(respJson3))

		patchBalanceUnlock.Reset()

		//###############################################################################################
		//------------- 3. Place Trader limit buy order which won't be matched because of lock ----------
		//###############################################################################################

		resp4 := client.GlobalEngineClient.PlaceOrder(
			&db.OrderTable{
				BaseTable: db.BaseTable{
					ID: 4,
				},
				MarketId:       contractUtils.ETH_MARKET,
				SubaccountId:   traderAliceSubaccountId,
				Party:          ctypes.PARTY_TRADER,
				Side:           ctypes.ORDER_SIDE_BUY,
				BrokerId:       1,
				Type:           ctypes.ORDER_TYPE_LIMIT,
				TotalFilledx18: ctypes.NewBigIntFromString("0"),
				Pricex18:       cutils.FloatStrToBigIntX18("101"), // Price is higher than the solver order which means match price will be of solver
				Amountx18:      cutils.FloatStrToBigIntX18("3"),
				ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
				Status:         ctypes.ORDER_STATUS_OPEN,
				Timestamp:      cutils.TimestampMilliNow(),
				IsReduce:       false,
				Signature:      "",
				SessionKey:     "",
			},
		)

		assert.NotNilf(t, resp4, "Engine Response is nil for step 4")
		assert.NoErrorf(t, resp4.Error, "Failed to place order: %v", resp4.Error)
		assert.Emptyf(t, resp4.MatchedMakerOrders, "Expected no matched maker orders, got %+v", resp4.MatchedMakerOrders)
		assert.Equalf(t, len(*resp4.CancelledOrders), 1, "Expected 1 cancelled order, got %d", len(*resp4.CancelledOrders))

		expectedCancelledOrder := engineTypes.CancelTakerOrderRespData{
			ID:           4,
			CancelReason: "Insufficient balance",
		}

		expectedTakerOrder4 := engineTypes.TakerOrderRespData{
			ID:             4,
			TotalFilledx18: cutils.FloatStrToX18("0"),
		}

		assert.EqualValuesf(t, (*resp4.CancelledOrders)[0], expectedCancelledOrder, "Expected cancelled order to be %+v, got %+v", expectedCancelledOrder, (*resp4.CancelledOrders)[0])
		assert.EqualValuesf(t, resp4.TakerOrder, &expectedTakerOrder4, "Expected taker order to be %+v, got %+v", expectedTakerOrder4, resp4.TakerOrder)

		respJson4, _ := json.Marshal(resp4)
		xlog.Infof("Step 4: Trader Limit Order placed: %+v", string(respJson4))

		// This is a test for vulnerable case for order id 40092388 and 38169670
		// There was a reduce only order which was filled more than the position size
		// id    	  |          created_at           |          updated_at           | side |  type  |  status   | market_id |    requested_amount    |              total_filled              |                 price                  |             trigger_price              | is_reduce
		// ----------+-------------------------------+-------------------------------+------+--------+-----------+-----------+------------------------+----------------------------------------+----------------------------------------+----------------------------------------+-----------
		// Solver and Trader

		//  38169611 | 2024-11-02 01:28:25.175596+00 | 2024-11-02 01:28:28.385597+00 | SELL | LIMIT  | CANCELLED |         1  |    33.0000000000000000 | 				 0.18339700000000000000 |                  2514.1313880000000000 | 0.000000000000000000000000000000000000 | f
		//  38169670 | 2024-11-02 01:28:26.436962+00 | 2024-11-02 01:28:26.5453+00   | BUY  | MARKET | FILLED    |         1  | 0.18339700000000000000 |                 0.18339700000000000000 | 0.000000000000000000000000000000000000 | 0.000000000000000000000000000000000000 | f

		//  40092325 | 2024-11-03 02:25:15.629559+00 | 2024-11-03 02:25:18.755107+00 | BUY  | LIMIT  | CANCELLED |         1  |    33.0000000000000000 | 				 0.21765000000000000000 |                  2459.9939760000000000 | 0.000000000000000000000000000000000000 | f
		//  40092388 | 2024-11-03 02:25:17.074189+00 | 2024-11-03 02:25:17.129316+00 | SELL | MARKET | FILLED    |         1  | 0.21765000000000000000 |                 0.21765000000000000000 | 0.000000000000000000000000000000000000 | 0.000000000000000000000000000000000000 | t
		// Conclusion: There was no code change required for this test and existing code is working as expected. Root cause for failure on production was outage of api server
		func() {
			xredis.GetRedisClient().FlushAll(context.Background())

			//###############################################################################################
			//-------------------- 1. Place SOLVER limit buy order which won't be matched ------------------
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
					BrokerId:       1,
					Type:           ctypes.ORDER_TYPE_LIMIT,
					TotalFilledx18: ctypes.NewBigIntFromString("0"),
					Pricex18:       cutils.FloatStrToBigIntX18("2514.1313880000000000"),
					Amountx18:      cutils.FloatStrToBigIntX18("33"),
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
			assert.Emptyf(t, resp.MatchedMakerOrders, "Expected no matched maker orders, got %+v", resp.MatchedMakerOrders)
			assert.Emptyf(t, resp.CancelledOrders, "Expected no cancelled order, got %+v", resp.CancelledOrders)
			assert.Emptyf(t, resp.SkippedCancelOrdersWithReason, "Expected no skipped cancel orders, got %+v", resp.SkippedCancelOrdersWithReason)
			assert.Equalf(t, resp.TakerOrder.ID, uint(1), "Expected order ID to be 1, got %d", resp.TakerOrder.ID)
			assert.Equalf(t, resp.TakerOrder.TotalFilledx18, cutils.GetBig0(), "Expected total filled to be 0, got %s", resp.TakerOrder.TotalFilledx18.String())

			respJson, _ := json.Marshal(resp)

			xlog.Infof("Step 1: Solver Order placed: %+v\n\n", string(respJson))

			//###############################################################################################
			//-------------------- 2. Place Trader market sell order which will be matched ------------------
			//###############################################################################################
			// Update balance first then place order

			traderSubaccountHex, _ := cutils.SubaccountIdToHex(traderAliceSubaccountId)
			traderBalances := subaccountTypes.SubaccountBalances{
				SubaccountId: traderSubaccountHex,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.Mulx18(big.NewInt(100)),
						Lockedx18:  cutils.GetBig0(),
					},
				},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
			}

			_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
				return subaccount.ReplaceBalanceInRedis(pipe, &traderBalances)
			})

			assert.NoError(t, err)

			resp2 := client.GlobalEngineClient.PlaceOrder(
				&db.OrderTable{
					BaseTable: db.BaseTable{
						ID: 2,
					},
					MarketId:       contractUtils.ETH_MARKET,
					SubaccountId:   traderAliceSubaccountId,
					Party:          ctypes.PARTY_TRADER,
					Side:           ctypes.ORDER_SIDE_BUY,
					BrokerId:       1,
					Type:           ctypes.ORDER_TYPE_MARKET,
					TotalFilledx18: ctypes.NewBigIntFromString("0"),
					Pricex18:       cutils.FloatStrToBigIntX18("0"),
					Amountx18:      cutils.FloatStrToBigIntX18("0.18339700000000000000"),
					ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
					Status:         ctypes.ORDER_STATUS_OPEN,
					Timestamp:      cutils.TimestampMilliNow(),
					IsReduce:       false,
					Signature:      "",
					SessionKey:     "",
				},
			)

			assert.NotNilf(t, resp2, "Engine Response is nil for step 2")
			assert.NoErrorf(t, resp2.Error, "Failed to place order: %v", resp2.Error)
			assert.Equalf(t, len(*resp2.MatchedMakerOrders), 1, "Expected 1 matched maker order, got %d", len(*resp2.MatchedMakerOrders))
			assert.Emptyf(t, resp2.CancelledOrders, "Expected no matched taker orders, got %+v", resp2.CancelledOrders)

			expectedMakerOrder := engineTypes.MakerOrderRespData{
				ID:               1,
				TotalFilledx18:   cutils.FloatStrToX18("0.18339700000000000000"),
				Pricex18:         cutils.FloatStrToX18("2514.1313880000000000"),
				MakerRealizedPnl: cutils.GetBig0(),
				TakerRealizedPnl: cutils.GetBig0(),
				MakerFundingFees: cutils.GetBig0(),
				TakerFundingFees: cutils.GetBig0(),
				TxnCounter:       1,
				MatchedAmountx18: cutils.FloatStrToX18("0.18339700000000000000"),
			}
			expectedTakerOrder := engineTypes.TakerOrderRespData{
				ID:             2,
				TotalFilledx18: cutils.FloatStrToX18("0.18339700000000000000"),
			}

			assert.EqualValuesf(t, (*resp2.MatchedMakerOrders)[0], expectedMakerOrder, "Expected matched order to be %+v, got %+v", expectedMakerOrder, (*resp2.MatchedMakerOrders)[0])

			assert.EqualValuesf(t, resp2.TakerOrder, &expectedTakerOrder, "Expected taker order to be %+v, got %+v", expectedTakerOrder, resp2.TakerOrder)

			respJson2, _ := json.Marshal(resp2)
			xlog.Infof("Step 2: Trader Order placed: %+v", string(respJson2))

			//###############################################################################################
			//-------------------- 3. Place Solver sell limit order sell order ------------------------------
			//###############################################################################################
			resp3 := client.GlobalEngineClient.CancelBulkAndPlaceOrder(&[]db.OrderTable{},
				&db.OrderTable{
					BaseTable: db.BaseTable{
						ID: 3,
					},
					MarketId:       contractUtils.ETH_MARKET,
					SubaccountId:   solverSubaccountId,
					Party:          ctypes.PARTY_SOLVER,
					Side:           ctypes.ORDER_SIDE_BUY,
					BrokerId:       1,
					Type:           ctypes.ORDER_TYPE_LIMIT,
					TotalFilledx18: ctypes.NewBigIntFromString("0"),
					Pricex18:       cutils.FloatStrToBigIntX18("2459.9939760000000000"),
					Amountx18:      cutils.FloatStrToBigIntX18("33"),
					ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
					Status:         ctypes.ORDER_STATUS_OPEN,
					Timestamp:      cutils.TimestampMilliNow(),
					IsReduce:       false,
					Signature:      "",
					SessionKey:     "",
				},
			)

			assert.NotNil(t, resp3)
			assert.NoErrorf(t, resp3.Error, "Failed to place order: %v", resp3.Error)
			assert.Emptyf(t, resp3.MatchedMakerOrders, "Expected no matched maker orders, got %+v", resp3.MatchedMakerOrders)
			assert.Emptyf(t, resp3.CancelledOrders, "Expected no matched taker orders, got %+v", resp3.CancelledOrders)
			assert.Emptyf(t, resp3.SkippedCancelOrdersWithReason, "Expected no skipped cancel orders, got %+v", resp3.SkippedCancelOrdersWithReason)
			assert.Equalf(t, resp3.TakerOrder.ID, uint(3), "Expected order ID to be 3, got %d", resp3.TakerOrder.ID)
			assert.Equalf(t, resp3.TakerOrder.TotalFilledx18, cutils.GetBig0(), "Expected total filled to be 0, got %s", resp3.TakerOrder.TotalFilledx18.String())

			respJson3, _ := json.Marshal(resp3)

			xlog.Infof("Step 3: Solver sell order placed: %+v\n\n", string(respJson3))

			//###############################################################################################
			//-------------------- 4. Place Trader reduce only market buy order which will be matched -------
			//###############################################################################################
			resp4 := client.GlobalEngineClient.PlaceOrder(
				&db.OrderTable{
					BaseTable: db.BaseTable{
						ID: 4,
					},
					MarketId:       contractUtils.ETH_MARKET,
					SubaccountId:   traderAliceSubaccountId,
					Party:          ctypes.PARTY_TRADER,
					Side:           ctypes.ORDER_SIDE_SELL,
					BrokerId:       1,
					Type:           ctypes.ORDER_TYPE_MARKET,
					TotalFilledx18: ctypes.NewBigIntFromString("0"),
					Pricex18:       cutils.FloatStrToBigIntX18("0"),
					Amountx18:      cutils.FloatStrToBigIntX18("0.21765000000000000000"),
					ExpiryTs:       cutils.TimestampMilliNow() + cutils.DAY_MILLI,
					Status:         ctypes.ORDER_STATUS_OPEN,
					Timestamp:      cutils.TimestampMilliNow(),
					IsReduce:       true,
					Signature:      "",
					SessionKey:     "",
				},
			)

			assert.NotNilf(t, resp4, "Engine Response is nil for step 4")
			assert.NoErrorf(t, resp4.Error, "Failed to place order: %v", resp4.Error)
			assert.Equalf(t, len(*resp4.MatchedMakerOrders), 1, "Expected 1 matched maker order, got %d", len(*resp4.MatchedMakerOrders))
			assert.Equalf(t, len(*resp4.CancelledOrders), 1, "Expected taker order to be cancelled, got %+v", *resp4.CancelledOrders)

			expectedMakerOrder4 := engineTypes.MakerOrderRespData{
				ID:               3,
				TotalFilledx18:   cutils.FloatStrToX18("0.18339700000000000000"),
				Pricex18:         cutils.FloatStrToX18("2459.9939760000000000"),
				MakerRealizedPnl: cutils.FloatStrToX18("9.928638948564"),
				TakerRealizedPnl: cutils.FloatStrToX18("-9.928638948564"),
				MakerFundingFees: cutils.GetBig0(),
				TakerFundingFees: cutils.GetBig0(),
				TxnCounter:       2,
				MatchedAmountx18: cutils.FloatStrToX18("0.18339700000000000000"),
			}
			expectedTakerOrder4 := engineTypes.TakerOrderRespData{
				ID:             4,
				TotalFilledx18: cutils.FloatStrToX18("0.18339700000000000000"),
			}

			assert.EqualValuesf(t, (*resp4.MatchedMakerOrders)[0], expectedMakerOrder4, "Expected matched order to be %+v, got %+v", expectedMakerOrder4, (*resp4.MatchedMakerOrders)[0])

			assert.EqualValuesf(t, resp4.TakerOrder, &expectedTakerOrder4, "Expected taker order to be %+v, got %+v", expectedTakerOrder4, resp4.TakerOrder)

			respJson4, _ := json.Marshal(resp4)
			xlog.Infof("Step 4: Trader Reduce only Buy Order placed: %+v", string(respJson4))
		}()

	})
}

// Testing removal of order that doesn't exist in redis
func TestMessageControllerE2ECloseOrder(t *testing.T) {
	t.Cleanup(testutils.ResetEnv)

	// traderAliceSubaccountId := "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_2"
	solverSubaccountId := contractUtils.AMM_SUBACCOUNT_ID_1

	testutils.WithSetupMockRedis(t, func() {
		closeBalanceSrv, patches := testutils.InitBalanceServer(t)
		defer closeBalanceSrv()

		defer patches.Reset()
		// Wait 100 ms to run the server
		time.Sleep(900 * time.Millisecond)

		closeEngineSrv := testutils.InitEngineServer(t)
		defer closeEngineSrv()

		// Wait 100 ms to run the server
		time.Sleep(100 * time.Millisecond)

		client.InitEngineClient()

		//###############################################################################################
		//-------------------- 1. Place SOLVER limit sell order which won't be matched ------------------
		//###############################################################################################
		resp := client.GlobalEngineClient.CancelBulkAndPlaceOrder(
			&[]db.OrderTable{
				{
					BaseTable: db.BaseTable{
						ID: 100,
					},
					MarketId:     contractUtils.ETH_MARKET,
					SubaccountId: solverSubaccountId,
					Party:        ctypes.PARTY_SOLVER,
					Side:         ctypes.ORDER_SIDE_SELL,
					Pricex18:     cutils.FloatStrToBigIntX18("100"),
				},
			},
			&db.OrderTable{
				BaseTable: db.BaseTable{
					ID: 1,
				},
				MarketId:       contractUtils.ETH_MARKET,
				SubaccountId:   solverSubaccountId,
				Party:          ctypes.PARTY_SOLVER,
				Side:           ctypes.ORDER_SIDE_SELL,
				BrokerId:       1,
				Type:           ctypes.ORDER_TYPE_LIMIT,
				TotalFilledx18: ctypes.NewBigIntFromString("0"),
				Pricex18:       cutils.FloatStrToBigIntX18("2514.1313880000000000"),
				Amountx18:      cutils.FloatStrToBigIntX18("33"),
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
		assert.Emptyf(t, resp.MatchedMakerOrders, "Expected no matched maker orders, got %+v", resp.MatchedMakerOrders)
		assert.Emptyf(t, resp.CancelledOrders, "Expected no cancelled order, got %+v", resp.CancelledOrders)
		assert.Equalf(t, len(*resp.SkippedCancelOrdersWithReason), 1, "Expected no skipped cancel orders, got %+v", resp.SkippedCancelOrdersWithReason)
		assert.Equalf(t, resp.TakerOrder.ID, uint(1), "Expected order ID to be 1, got %d", resp.TakerOrder.ID)
		assert.Equalf(t, resp.TakerOrder.TotalFilledx18, cutils.GetBig0(), "Expected total filled to be 0, got %s", resp.TakerOrder.TotalFilledx18.String())

		respJson, _ := json.Marshal(resp)

		xlog.Infof("Step 1: Solver Order placed: %+v\n\n", string(respJson))
	})

	t.Log("Test completed successfully")
}
