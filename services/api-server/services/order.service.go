package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/client"
	"github/eugenix-io/logx-inf-backend/services/api-server/types"
	balanceTypes "github/eugenix-io/logx-inf-backend/services/balance-server/types"
	engineTypes "github/eugenix-io/logx-inf-backend/services/engine/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type OrderService interface {
	GetCancellableOrdersFromDb(subaccountId string, marketId uint) *[]db.OrderTable
	CancelAllAndPlaceOrder(subaccountId string, newOrder *db.OrderTable, market *db.MarketTable) (takerOrder *db.OrderTable, skippedOrdersWithReason *[]engineTypes.SkipCancelOrderRespData, status int, err error)
	PlaceOrder(newOrder *db.OrderTable, market *db.MarketTable) (_takerOrder *db.OrderTable, FeeBonus ctypes.BigInt, status int, err error)
	PlaceConditionalOrder(conditionalOrder *db.OrderTable, market *db.MarketTable) (_takerOrder *db.OrderTable, status int, err error)
	PlaceLiquidationOrder(liquidationOrder *db.OrderTable, market *db.MarketTable, spotPricesx18, perpPricesx18 map[uint32]*big.Int) (_takerOrder *db.OrderTable, status int, err error)
	CancelAllOrders(subaccountId string, marketId uint) (engineResponse *engineTypes.EngineCancelOrdersBulkResponse, status int, err error)
	CancelOrderById(subaccountId string, orderId uint) (status int, err error)
	CancelAllAndPlaceBulkOrder(subaccountId string, newOrders *[]db.OrderTable, market *db.MarketTable, isBuy bool) (takerOrders *[]db.OrderTable, rejectedOrders []types.RejectedOrder, status int, err error)
}

type OrderServiceImpl struct {
	appState appstate.AppState
}

type MakerTakerFillData struct {
	makerFillData *db.FillTable
	takerFillData *db.FillTable
}

// validate OrderServiceImpl implements OrderService
var _ OrderService = &OrderServiceImpl{}

func NewOrderService() OrderService {
	return &OrderServiceImpl{
		appState: appstate.NewAppState(),
	}
}

func (o *OrderServiceImpl) GetCancellableOrdersFromDb(subaccountId string, marketId uint) *[]db.OrderTable {
	orders := (&db.OrderDB{}).GetAllCancellableOrdersForMarket(subaccountId, marketId)
	return orders
}

func (o *OrderServiceImpl) GetCancellableOrdersWithSideFromDb(subaccountId string, marketId uint, side ctypes.OrderSide) *[]db.OrderTable {
	orders := (&db.OrderDB{}).GetAllCancellableOrdersWithSideForMarket(subaccountId, marketId, side)
	return orders
}

func (o *OrderServiceImpl) updateAllOrderStatus(orderIds []uint, status ctypes.OrderStatus) error {
	if len(orderIds) == 0 {
		return nil
	}

	ordersPlaceholders := make([]db.OrderTable, len(orderIds))
	for i, orderId := range orderIds {
		ordersPlaceholders[i] = db.OrderTable{BaseTable: db.BaseTable{ID: orderId}}
	}

	result := (&db.OrderDB{}).UpdateMultiple(&ordersPlaceholders, map[string]interface{}{"status": status})
	if result == nil {
		errorMsg := fmt.Sprintf("Failed to update status to %s for orders: %v", status, orderIds)
		xlog.Errorf(errorMsg)
		return fmt.Errorf(errorMsg)
	}

	xlog.Debugf(
		"Order Service - Successfully updated status to %s for orders: %v",
		status,
		orderIds,
	)
	return nil
}

func (o *OrderServiceImpl) CancelOrderById(subaccountId string, orderId uint) (status int, err error) {
	order := (&db.OrderDB{}).GetBySubaccount_Id(subaccountId, orderId)
	if order == nil {
		return http.StatusNotFound, fmt.Errorf("order not found")
	} else if order.Status == ctypes.ORDER_STATUS_CANCELLED || order.Status == ctypes.ORDER_STATUS_FILLED {
		return http.StatusBadRequest, fmt.Errorf("order already cancelled or filled")
	}
	var engineResponse *engineTypes.EngineCancelOrdersBulkResponse

	engineResponse, status, err = o.cancelOrders(&[]db.OrderTable{*order})

	if err != nil {
		return http.StatusInternalServerError, err
	}
	// Ideally this will happen only when order is not present in orderbook
	if engineResponse.SkippedOrdersWithReason != nil && len(*engineResponse.SkippedOrdersWithReason) == 1 {
		return http.StatusBadRequest, fmt.Errorf((*engineResponse.SkippedOrdersWithReason)[0].Reason)
	}
	return status, err
}

func (o *OrderServiceImpl) CancelAllOrders(subaccountId string, marketId uint) (engineResponse *engineTypes.EngineCancelOrdersBulkResponse, status int, err error) {
	cancellableOrders := o.GetCancellableOrdersFromDb(subaccountId, marketId)
	return o.cancelOrders(cancellableOrders)
}

func (o *OrderServiceImpl) cancelOrders(cancellableOrders *[]db.OrderTable) (engineResponse *engineTypes.EngineCancelOrdersBulkResponse, status int, err error) {
	// No orders to cancel
	if len(*cancellableOrders) == 0 {
		xlog.Warnf("No orders received to cancelled")
		return &engineTypes.EngineCancelOrdersBulkResponse{
			SkippedOrdersWithReason: &[]engineTypes.SkipCancelOrderRespData{},
			CancelledOrders:         &[]uint{},
			Error:                   nil,
		}, http.StatusOK, nil
	}

	conditonalCancellableOrders := cutils.FilterSlice(*cancellableOrders, func(ord db.OrderTable) bool { return ord.IsConditional() })
	nonConditionalCancellableOrders := cutils.FilterSlice(*cancellableOrders, func(ord db.OrderTable) bool { return !ord.IsConditional() })

	if len(nonConditionalCancellableOrders) > 0 {
		// Make call to engine to cancel orders
		engineResponse1 := client.GlobalEngineClient.CancelOrdersBulk(&nonConditionalCancellableOrders)
		engineResponse = engineResponse1
	}

	if len(conditonalCancellableOrders) > 0 {
		// Make separate call to engine to cancel conditional orders
		engineResponse2 := client.GlobalEngineClient.CancelConditionalOrdersBulk(&conditonalCancellableOrders)
		if engineResponse == nil {
			engineResponse = engineResponse2
		} else {
			// Merge the responses
			if engineResponse.SkippedOrdersWithReason != nil && engineResponse2.SkippedOrdersWithReason != nil {
				*engineResponse.SkippedOrdersWithReason = append(*engineResponse.SkippedOrdersWithReason, *engineResponse2.SkippedOrdersWithReason...)
			} else if engineResponse2.SkippedOrdersWithReason != nil {
				engineResponse.SkippedOrdersWithReason = engineResponse2.SkippedOrdersWithReason
			}

			if engineResponse.CancelledOrders != nil && engineResponse2.CancelledOrders != nil {
				*engineResponse.CancelledOrders = append(*engineResponse.CancelledOrders, *engineResponse2.CancelledOrders...)
			} else if engineResponse2.CancelledOrders != nil {
				engineResponse.CancelledOrders = engineResponse2.CancelledOrders
			}
		}
	}

	// TODO: Check specific error first to initiate partial cancels if needed
	// This would never happen
	if engineResponse.Error != nil {
		xlog.Errorf("Error cancelling orders for subaccount: %s\n", (*cancellableOrders)[0].SubaccountId)
		return nil, http.StatusInternalServerError, engineResponse.Error
	}

	// Update order status in db
	if engineResponse.CancelledOrders != nil {
		err = o.updateAllOrderStatus(*engineResponse.CancelledOrders, ctypes.ORDER_STATUS_CANCELLED)
		if err != nil {
			xlog.Errorf("Failed to update order status in DB: %v", err)
		}

		// Unlock balance for cancelled orders
		for _, orderId := range *engineResponse.CancelledOrders {
			err := o.unlockOrderLockedBalance(orderId)
			if err != nil {
				xlog.Warnf("Unable to unlock balance for order: %v", orderId)
			}
		}
	}

	// Return skipped orders and cancelled orders along with status and error
	return engineResponse, http.StatusOK, nil
}

// NOTE: Contains Redis Locking to Avoid repeated cancellations
func (o *OrderServiceImpl) CancelAllAndPlaceOrder(subaccountId string, newOrder *db.OrderTable, market *db.MarketTable) (takerOrder *db.OrderTable, skippedOrdersWithReason *[]engineTypes.SkipCancelOrderRespData, status int, err error) {

	type Resp struct {
		TakerOrder                    *db.OrderTable
		SkippedCancelOrdersWithReason *[]engineTypes.SkipCancelOrderRespData
		Status                        int
	}

	resp, err := xredis.WithRedisLock(xredis.GetCancelAllAndPlaceOrderLockKey(uint32(market.ID), subaccountId), func() (*Resp, error) {
		// Get cancellable orders from db
		cancellableOrders := o.GetCancellableOrdersWithSideFromDb(subaccountId, market.ID, newOrder.Side)
		xlog.Infof("Cancellable orders: %v", cutils.MapSlice(*cancellableOrders, func(order db.OrderTable) uint {
			return order.ID
		}))

		// Store order in db
		newOrder = (&db.OrderDB{}).CreateOrder(*newOrder)
		if newOrder == nil || newOrder.ID == 0 {
			msg := "unable to create order. Error while placing order in db"
			xlog.Errorf(msg)
			return &Resp{
				TakerOrder:                    nil,
				SkippedCancelOrdersWithReason: nil,
				Status:                        http.StatusInternalServerError,
			}, errors.New(msg)
		}

		cutils.LogByParty(
			newOrder.Party,
			"%v - order controller - cancelAllAndPlaceOrder: subaccount=%s market=%d price=%s size=%s side=%s party=%s",
			newOrder.ID,
			newOrder.SubaccountId,
			newOrder.MarketId,
			newOrder.Pricex18.Val.String(),
			newOrder.Amountx18.Val.String(),
			newOrder.Side,
			newOrder.Party,
		)
		// Lock balance for new order
		err = o.lockBalanceForOrder(newOrder, market)
		if err != nil {
			// Update order status to cancelled
			(&db.OrderDB{}).Update(newOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
			return &Resp{
				TakerOrder:                    nil,
				SkippedCancelOrdersWithReason: nil,
				Status:                        http.StatusForbidden,
			}, err
		}

		// Defer unlock balance
		defer o.unlockOrderLockedBalance(newOrder.ID)

		// Make call to engine to cancel bulk orders and place new order
		engineResponse := client.GlobalEngineClient.CancelBulkAndPlaceOrder(cancellableOrders, newOrder)
		if engineResponse == nil {
			// Update order status to cancelled
			(&db.OrderDB{}).Update(newOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
			return &Resp{
				TakerOrder:                    nil,
				SkippedCancelOrdersWithReason: nil,
				Status:                        http.StatusInternalServerError,
			}, fmt.Errorf("error in placing order with global engine. Looks like engine server is down")
		} else if engineResponse.Error != nil {
			// Update order status to cancelled
			(&db.OrderDB{}).Update(newOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
			msg := fmt.Sprintf("%v - Error cancelling orders and placing new order | Error: %v\n", newOrder.ID, engineResponse.Error.Error())
			return &Resp{
				TakerOrder:                    nil,
				SkippedCancelOrdersWithReason: nil,
				Status:                        http.StatusInternalServerError,
			}, errors.New(msg)
		}

		engineRespJson, _ := json.Marshal(engineResponse)
		cutils.LogByParty(
			newOrder.Party,
			"engine response for order=%d party=%s: %s",
			newOrder.ID, newOrder.Party, string(engineRespJson),
		)

		if engineResponse.SkippedCancelOrdersWithReason != nil && len(*engineResponse.SkippedCancelOrdersWithReason) > 0 {
			var ordersToCancel []uint
			skippedOrderMap := make(map[uint]string)

			for _, skippedOrder := range *engineResponse.SkippedCancelOrdersWithReason {
				skippedOrderMap[skippedOrder.ID] = skippedOrder.Reason

				// If the order was skipped because it is not in the orderbook, we will cancel it in DB as well
				if strings.Contains(skippedOrder.Reason, "Order not found in orderbook") {
					ordersToCancel = append(ordersToCancel, skippedOrder.ID)
					xlog.Infof("Order %d skipped with reason '%s', marking as cancelled in DB",
						skippedOrder.ID, skippedOrder.Reason)
				}
			}

			// Mark skipped orders in DB that were not found in orderbook
			if len(ordersToCancel) > 0 {
				err := o.updateAllOrderStatus(ordersToCancel, ctypes.ORDER_STATUS_CANCELLED)
				if err != nil {
					xlog.Errorf("Failed to update order status in DB: %v", err)
				}
			}
		}

		// Match orders on smart contract and perform necessary operations
		placeOrderEngineResponse := &engineTypes.EnginePlaceOrderResponse{
			TakerOrder:         engineResponse.TakerOrder,
			MatchedMakerOrders: engineResponse.MatchedMakerOrders,
			CancelledOrders:    engineResponse.CancelledOrders,
		}

		takerOrder, _, status, err = o.matchOnchainAndCreateFills(placeOrderEngineResponse, market, nil, nil)
		o.cancelOrdersInDbAndUnlockBalance(placeOrderEngineResponse)

		// return taker order
		return &Resp{
			TakerOrder:                    takerOrder,
			SkippedCancelOrdersWithReason: engineResponse.SkippedCancelOrdersWithReason,
			Status:                        http.StatusOK,
		}, nil
	})

	// MOST LIKELY THIS WILL HAPPEN DUE TO REDIS LOCK TIMEOUT
	if err != nil && resp == nil {
		xlog.Errorf("Error in CancelAllAndPlaceOrder: %v", err)
		return nil, nil, http.StatusInternalServerError, err
	}

	return resp.TakerOrder, resp.SkippedCancelOrdersWithReason, resp.Status, err
}

// Returns rejected orders with reason and successful taker orders
func (o *OrderServiceImpl) CancelAllAndPlaceBulkOrder(subaccountId string, newOrders *[]db.OrderTable, market *db.MarketTable, isBuy bool) (takerOrders *[]db.OrderTable, rejectedOrders []types.RejectedOrder, status int, err error) {
	if newOrders == nil || len(*newOrders) == 0 {
		return nil, nil, http.StatusBadRequest, fmt.Errorf("no new orders provided")
	}

	type Resp struct {
		TakerOrders    *[]db.OrderTable
		RejectedOrders *[]types.RejectedOrder
		Status         int
	}

	resp, err := xredis.WithRedisLock(xredis.GetCancelAllAndPlaceOrderLockKey(uint32(market.ID), subaccountId), func() (*Resp, error) {
		// Get cancellable orders from db
		cancellableOrders := o.GetCancellableOrdersWithSideFromDb(subaccountId, market.ID, ctypes.NewOrderSide(isBuy))
		xlog.Infof("Cancellable orders: %v", cutils.MapSlice(*cancellableOrders, func(order db.OrderTable) uint {
			return order.ID
		}))

		rejectedOrders := make([]types.RejectedOrder, 0)
		validNewOrders := make([]db.OrderTable, 0)
		// Store order in db
		for _, order := range *newOrders {
			newOrder := (&db.OrderDB{}).CreateOrder(order)
			if newOrder == nil || newOrder.ID == 0 {
				msg := "unable to create order. Error while placing order in db"
				xlog.Errorf(msg)
				rejectedOrders = append(rejectedOrders, types.RejectedOrder{
					Order:  order,
					Reason: msg,
				})
				continue
			}

			cutils.LogByParty(
				newOrder.Party,
				"%v - order controller - cancelAllAndPlaceOrder: subaccount=%s market=%d price=%s size=%s side=%s party=%s",
				newOrder.ID,
				newOrder.SubaccountId,
				newOrder.MarketId,
				newOrder.Pricex18.Val.String(),
				newOrder.Amountx18.Val.String(),
				newOrder.Side,
				newOrder.Party,
			)
			// Lock balance for new order
			err = o.lockBalanceForOrder(newOrder, market)
			if err != nil {
				// Update order status to cancelled
				(&db.OrderDB{}).Update(newOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
				msg := fmt.Sprintf("Unable to lock balance for order: %v", newOrder.ID)
				xlog.Errorf(msg)
				rejectedOrders = append(rejectedOrders, types.RejectedOrder{
					Order:  order,
					Reason: msg,
				})
				continue
			}

			// Defer unlock balance
			defer o.unlockOrderLockedBalance(newOrder.ID)
			validNewOrders = append(validNewOrders, *newOrder)
		}

		validOrdersMap := cutils.UniqueSliceToMap(validNewOrders, func(order db.OrderTable) uint {
			return order.ID
		})

		// Make call to engine to cancel bulk orders and place new order
		engineResponse := client.GlobalEngineClient.CancelBulkAndPlaceBulkOrder(cancellableOrders, &validNewOrders)

		if engineResponse == nil || engineResponse.Error != nil {
			var msg string
			if engineResponse == nil {
				msg = "Error in placing order with global engine. Looks like engine server is down | Response is nil"
			} else {
				msg = fmt.Sprintf("Error in placing order with global engine. Looks like engine server is down | Error: %v", engineResponse.Error)
			}
			xlog.Errorf(msg)

			for _, order := range validNewOrders {
				// Update order status to cancelled
				(&db.OrderDB{}).Update(&order, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
				rejectedOrders = append(rejectedOrders, types.RejectedOrder{
					Order:  order,
					Reason: msg,
				})
			}
			// TODO: Fix this
			return &Resp{
				TakerOrders:    nil,
				RejectedOrders: &rejectedOrders,
				Status:         http.StatusInternalServerError,
			}, fmt.Errorf(msg)
		}

		engineRespJson, _ := json.Marshal(engineResponse)
		cutils.LogByParty(
			(*newOrders)[0].Party,
			"engine response for order=%d party=%s: %s",
			(*newOrders)[0].ID, (*newOrders)[0].Party, string(engineRespJson),
		)

		if engineResponse.SkippedCancelOrdersWithReason != nil && len(*engineResponse.SkippedCancelOrdersWithReason) > 0 {
			var ordersToCancel []uint
			skippedOrderMap := make(map[uint]string)

			for _, skippedOrder := range *engineResponse.SkippedCancelOrdersWithReason {
				skippedOrderMap[skippedOrder.ID] = skippedOrder.Reason

				// If the order was skipped because it is not in the orderbook, we will cancel it in DB as well
				if strings.Contains(skippedOrder.Reason, "Order not found in orderbook") {
					ordersToCancel = append(ordersToCancel, skippedOrder.ID)
					xlog.Infof("Order %d skipped with reason '%s', marking as cancelled in DB",
						skippedOrder.ID, skippedOrder.Reason)
				}
			}

			// Mark skipped orders in DB that were not found in orderbook
			if len(ordersToCancel) > 0 {
				err := o.updateAllOrderStatus(ordersToCancel, ctypes.ORDER_STATUS_CANCELLED)
				if err != nil {
					xlog.Errorf("Failed to update order status in DB: %v", err)
				}
			}
		}

		// Cancel orders which were placed in db to be cancelled
		_placeOrderEngineResponse := &engineTypes.EnginePlaceOrderResponse{
			TakerOrder:         nil,
			MatchedMakerOrders: nil,
			CancelledOrders:    engineResponse.CancelledOrders,
		}
		o.cancelOrdersInDbAndUnlockBalance(_placeOrderEngineResponse)

		takerOrders := make([]db.OrderTable, 0)
		// Update db for match order / fills and then update smart contract
		for _, orderResp := range *engineResponse.PlaceOrderResponses {
			_order := validOrdersMap[orderResp.TakerOrder.ID]
			if orderResp.Error != nil {
				msg := fmt.Sprintf("Error in placing order with global engine. Looks like engine server is down | Error: %v", orderResp.Error)
				xlog.Errorf(msg)
				// Update order status to cancelled
				(&db.OrderDB{}).Update(&_order, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
				rejectedOrders = append(rejectedOrders, types.RejectedOrder{
					Order:  _order,
					Reason: msg,
				})
				continue
			} else {
				// FIXME: Do not assume this will always be successful
				updatedTakerOrder, _, _, _ := o.matchOnchainAndCreateFills(&orderResp, market, nil, nil)
				takerOrders = append(takerOrders, *updatedTakerOrder)
				o.cancelOrdersInDbAndUnlockBalance(&orderResp)
			}
		}
		return &Resp{
			TakerOrders:    &takerOrders,
			RejectedOrders: &rejectedOrders,
			Status:         http.StatusOK,
		}, nil
	})

	if resp == nil {
		// MOST LIKELY THIS WILL HAPPEN DUE TO REDIS LOCK TIMEOUT
		xlog.Errorf("Error in CancelAllAndPlaceBulkOrder: %v", err)
		return nil, cutils.MapSlice(*newOrders, func(order db.OrderTable) types.RejectedOrder {
			return types.RejectedOrder{
				Order:  order,
				Reason: fmt.Sprintf("Error in CancelAllAndPlaceBulkOrder: %v", err),
			}
		}), http.StatusInternalServerError, err
	} else if err != nil {
		// If there is an error, we will return the rejected orders with reason
		xlog.Errorf("Error in CancelAllAndPlaceBulkOrder: %v", err)
		return nil, *resp.RejectedOrders, resp.Status, err
	}

	return resp.TakerOrders, *resp.RejectedOrders, resp.Status, nil
}

// Locks initial margin + taker fee for limit orders
// Skips locking for IOC orders
// Skips locking for reduce only orders because isReduce orders will always decrease initial / maintenance margin requirement for the position
func (o *OrderServiceImpl) lockBalanceForOrder(newOrder *db.OrderTable, market *db.MarketTable) error {
	if !newOrder.RequireInitialMarginLock() {
		return nil
	}

	orderSubaccountHex, _ := cutils.SubaccountIdToHex(newOrder.SubaccountId)
	if contractUtils.IsAMMAccount(orderSubaccountHex) {
		xlog.Debugf("%v - AMM Order (%v)... No need to lock balance", newOrder.Type, newOrder.ID)
		return nil
	}

	lockQuotex18 := perputils.GetBigLockQuotex18(newOrder.Amountx18.Val, newOrder.Pricex18.Val, market)
	currentSubaccountHex, _ := cutils.SubaccountIdToHex(newOrder.SubaccountId)

	// Balance Service accepts hex subaccount id
	lockSuccess, lockErr := xclient.GlobalBalanceClient.LockBalance(balanceTypes.LockBalanceRequest{
		SubaccountId: currentSubaccountHex,
		Entity:       ctypes.LOCKER_ENTITY_ORDER,
		EntityId:     fmt.Sprintf("%v", newOrder.ID),
		ProductId:    contractUtils.QUOTE_TOKEN_PRODUCT_ID,
		LockQuotex18: lockQuotex18,
	})
	if !lockSuccess {
		// For now we just assume user is not having enough balance and the connection with balance service is fine
		xlog.Errorf("%v - Error while locking balance: %v", newOrder.ID, lockErr)
		return fmt.Errorf("%v - Error while locking balance: %v", newOrder.ID, lockErr)
	}
	cutils.LogByParty(
		newOrder.Party,
		"Order Service - %v - OrderSize: %s lockQuote(X18): %s, Pricex18: %s",
		newOrder.ID,
		newOrder.Amountx18.Val.String(),
		lockQuotex18.String(),
		newOrder.Pricex18.Val.String(),
	)
	return nil
}

// Unlocks the remaining balance of the order if it is fully filled or cancelled
// We are not taking order rather taking the order id so, we can fetch the latest status of the order
// No need to unlock balance for liquidation orders
func (o *OrderServiceImpl) unlockOrderLockedBalance(orderId uint) error {
	order := (&db.OrderDB{}).GetById(orderId)
	if order == nil {
		xlog.Errorf("%v - Failed to unlock balance as order not found in db", orderId)
		return nil
		//todo: handle gracefully
		//fmt.Errorf("unlock balance for order unsuccessful as order not found in db. Order ID: %v", orderId)
	}

	orderSubaccountIdHex, _ := cutils.SubaccountIdToHex(order.SubaccountId)
	if contractUtils.IsAMMAccount(orderSubaccountIdHex) {
		xlog.Debugf("%v - AMM Order (%v)... No need to unlock balance", order.Type, orderId)
		return nil
	}

	if order.IsIOCOrder() {
		cutils.LogByParty(
			order.Party,
			"%v - IOC Order (%v)... No need to unlock balance",
			order.Type, orderId,
		)
		return nil
	}

	if order.IsReduce {
		cutils.LogByParty(
			order.Party,
			"%v - Reduce Only Order ... No need to unlock balance",
			orderId,
		)
		return nil
	}

	// Unlock the remaining balance if order is fully filled or cancelled
	if order.Status == ctypes.ORDER_STATUS_FILLED || order.Status == ctypes.ORDER_STATUS_CANCELLED {
		currentSubaccountHex, _ := cutils.SubaccountIdToHex(order.SubaccountId)
		unlockSuccess, unlockErr := xclient.GlobalBalanceClient.UnlockFullBalance(balanceTypes.UnlockBalanceRequest{
			SubaccountId: currentSubaccountHex,
			Entity:       ctypes.LOCKER_ENTITY_ORDER,
			EntityId:     fmt.Sprintf("%v", order.ID),
		})

		if !unlockSuccess {
			xlog.Errorf("%v - Couldn't unlock full balance for order. err: %v", orderId, unlockErr)
			return fmt.Errorf("%v - couldn't unlock full balance for order", orderId)
		}
	}
	cutils.LogByParty(
		order.Party,
		"Order Service - %v - Unlocked Full Balance.",
		order.ID,
	)
	return nil
}

// Returns the maker and taker realised pnl x18
func (o *OrderServiceImpl) matchLiquidationOrdersOnChain(takerOrder *db.OrderTable, makerOrder *db.OrderTable, matchAmountx18 *big.Int, spotOraclePriceX18, perpOraclePricesX18 map[uint32]*big.Int, transactionCounter uint) (*big.Int, *big.Int, *big.Int, *big.Int, error) {

	// Oracle price is required for liquidation order
	// TODO: Move this logic to a util function -> Product Id to spot mapping is as follows: PriceOfProduct = spotPricesx18[productId/2]
	matchLiquidationOrderRequest := contract.LiquidationRequest{
		ProductId:              uint32(takerOrder.MarketId),
		LiquidatorSubaccountId: makerOrder.SubaccountId,
		LiquidatorPriceX18:     makerOrder.Pricex18.Val,
		LiquidatorAmount:       makerOrder.Amountx18.Val,
		LiquidatorExpiryTs:     makerOrder.ExpiryTs,
		LiquidatorIsReduce:     makerOrder.IsReduce,
		LiquidatorSessionKey:   makerOrder.SessionKey,
		LiquidatorSide:         makerOrder.Side,

		LiquidateeSubaccountId: takerOrder.SubaccountId,
		LiquidateeAmount:       takerOrder.Amountx18.Val,
		PerpOraclePricesX18:    perpOraclePricesX18,
		SpotOraclePricesX18:    spotOraclePriceX18,
		LiquidateeSide:         takerOrder.Side,

		LiquidatorSignature: makerOrder.Signature,
		MatchedAmountx18:    matchAmountx18,
	}

	// Convert matchOrderRequest to json
	matchOrderRequestJson, _ := json.Marshal(matchLiquidationOrderRequest)
	xlog.Debugf("%v - Matched order request: %v", takerOrder.ID, string(matchOrderRequestJson))

	takerSubaccountIdHex, _ := cutils.SubaccountIdToHex(takerOrder.SubaccountId)
	makerSubaccountIdHex, _ := cutils.SubaccountIdToHex(makerOrder.SubaccountId)

	matchAmountWithTakerSignx18 := new(big.Int).Abs(matchAmountx18)
	if takerOrder.Side == ctypes.ORDER_SIDE_SELL {
		matchAmountWithTakerSignx18 = new(big.Int).Neg(matchAmountWithTakerSignx18)
	}

	makerRealisedPnlx18, takerRealisedPnlx18, makerFundingFeesx18, takerFundingFeesx18, err := updateBalancesForLiquidation(takerSubaccountIdHex, makerSubaccountIdHex, matchAmountWithTakerSignx18, uint32(takerOrder.MarketId), takerOrder.ID, makerOrder.Pricex18.Val, perpOraclePricesX18, spotOraclePriceX18, transactionCounter)
	if err != nil {
		xlog.Errorf("%v - Severe Error updating balances for liquidation with maker: %v | %v", takerOrder.ID, makerOrder.ID, err)
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), fmt.Errorf("%v - Severe Error updating balances for liquidation with maker: %v | %v", takerOrder.ID, makerOrder.ID, err)
	}

	// Send request on contract
	if err := contract.GlobalContracts.EndpointContract.LiquidateSubaccount(matchLiquidationOrderRequest, transactionCounter); err != nil {
		xlog.Errorf("Cannot match order on contract: %v", err)

		// NOTE: We should assume that the issue is with the contract call and not user's fault
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), fmt.Errorf("cannot match order on contract: %v", err)
	}

	return makerRealisedPnlx18, takerRealisedPnlx18, makerFundingFeesx18, takerFundingFeesx18, nil
}

// Matched amount is abs value of matched amount
func (o *OrderServiceImpl) matchOrdersOnChain(takerOrder *db.OrderTable, makerOrder *db.OrderTable, matchedAmountx18 *big.Int, transactionCounter uint) error {

	matchOrderRequest := contract.MatchOrderRequest{
		ProductId:         uint32(takerOrder.MarketId),
		TakerSubaccountId: takerOrder.SubaccountId,
		MakerSubaccountId: makerOrder.SubaccountId,
		TakerPriceX18:     takerOrder.Pricex18.Val,
		MakerPriceX18:     makerOrder.Pricex18.Val,
		TakerExpiryTs:     takerOrder.ExpiryTs,
		MakerExpiryTs:     makerOrder.ExpiryTs,
		TakerMaxAmountX18: takerOrder.Amountx18.Val,
		MakerMaxAmountX18: makerOrder.Amountx18.Val,
		MakerSignature:    makerOrder.Signature,
		TakerSignature:    takerOrder.Signature,
		TakerSessionKey:   takerOrder.SessionKey,
		MakerSessionKey:   makerOrder.SessionKey,
		TakerIsReduce:     takerOrder.IsReduce,
		MakerIsReduce:     makerOrder.IsReduce,
		TakerSide:         takerOrder.Side,
		MatchedAmountx18:  matchedAmountx18,
	}

	// Convert matchOrderRequest to json
	matchOrderRequestJson, _ := json.Marshal(matchOrderRequest)
	xlog.Debugf("Matched order request: %v", string(matchOrderRequestJson))

	// Send request on contract
	if err := contract.GlobalContracts.EndpointContract.MatchOrders(matchOrderRequest, transactionCounter); err != nil {
		xlog.Errorf("Cannot match order on contract: %v", err)

		// NOTE: We should assume that the issue is with the contract call and not user's fault
		return fmt.Errorf("cannot match order on contract: %v", err)
	}

	return nil
}

// Update balances for liquidation and return maker and taker realised pnl
func updateBalancesForLiquidation(liquidateeIdHex, liquidatorIdHex string, liquidateeAmountWithTakerSignx18 *big.Int, productId uint32, liquidateeOrderId uint, matchPricex18 *big.Int, perpOraclePricesX18, spotOraclePricesX18 map[uint32]*big.Int, txnCounter uint) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	makerRealisedPnlx18, takerRealisedPnlx18, makerFundingFeesx18, takerFundingFeesx18, err := xclient.GlobalBalanceClient.UpdateSubaccountsForLiquidation(balanceTypes.FinaliseLiquidationRequest{
		TxnCounter:             &txnCounter,
		LiquidateeOrderId:      liquidateeOrderId,
		LiquidateeSubaccountId: liquidateeIdHex,
		LiquidatorSubaccountId: liquidatorIdHex,
		ProductId:              productId,
		AmountX18:              liquidateeAmountWithTakerSignx18, // NOTE: that this is the amount of the liquidatee along with sign
		MatchPriceX18:          matchPricex18,
		PerpOraclePricesX18:    perpOraclePricesX18,
		SpotOraclePricesX18:    spotOraclePricesX18,
	})
	return makerRealisedPnlx18, takerRealisedPnlx18, makerFundingFeesx18, takerFundingFeesx18, err
}

func (o *OrderServiceImpl) createFillsForMatchedOrders(
	matchedAmountx18 *big.Int,
	takerOrder *db.OrderTable,
	makerOrder *db.OrderTable,
	market *db.MarketTable,
	makerRealizedPnl *big.Int,
	takerRealizedPnl *big.Int,
	makerFundingFees *big.Int,
	takerFundingFees *big.Int) ctypes.BigInt {

	bigIntfillAmtx18 := ctypes.NewBigInt(matchedAmountx18)
	bigIntfillPricex18 := makerOrder.Pricex18

	totalNotionalx18 := cutils.Divx18(new(big.Int).Mul(bigIntfillAmtx18.Val, bigIntfillPricex18.Val))

	var bigIntMakerFeex18 ctypes.BigInt = ctypes.BigInt{
		Val: big.NewInt(0),
	}
	var bigIntTakerFeex18 ctypes.BigInt = ctypes.BigInt{
		Val: big.NewInt(0),
	}

	var takerFeeBonus ctypes.BigInt = ctypes.BigInt{
		Val: big.NewInt(0),
	}
	var makerFeeBonus ctypes.BigInt = ctypes.BigInt{
		Val: big.NewInt(0),
	}

	ammSubaccountID := os.Getenv("AMM_SUBACCOUNT_ID")
	takerSubaccountHex, _ := cutils.SubaccountIdToHex(takerOrder.SubaccountId)
	makerSubaccountHex, _ := cutils.SubaccountIdToHex(makerOrder.SubaccountId)

	brokerId := 1 // default

	var nonAmmSubaccountId string
	if takerSubaccountHex != ammSubaccountID {
		nonAmmSubaccountId = takerOrder.SubaccountId
	} else {
		nonAmmSubaccountId = makerOrder.SubaccountId
	}

	if parts := strings.Split(nonAmmSubaccountId, "_"); len(parts) >= 1 {
		if id, err := strconv.Atoi(parts[0]); err == nil && id >= 0 {
			brokerId = id
		}
	}

	factor := cutils.GetBrokerFeeFactor(brokerId)
	makerFeeFraction := cutils.MulxCust(big.NewInt(factor), 13)

	if takerSubaccountHex != ammSubaccountID {
		bigIntTakerFeex18 = ctypes.NewBigInt(cutils.Divx18(new(big.Int).Mul(totalNotionalx18, cutils.MulxCust(big.NewInt(factor), 13))))
	}
	if makerSubaccountHex != ammSubaccountID {
		bigIntMakerFeex18 = ctypes.NewBigInt(cutils.Divx18(new(big.Int).Mul(totalNotionalx18, makerFeeFraction)))
	}

	if takerOrder.IsLiquidationOrder() {
		liquidationFeex18 := cutils.Divx18(new(big.Int).Mul(marketutils.GetLiquidationFractionx18(uint32(market.ID)), totalNotionalx18))
		bigIntTakerFeex18 = ctypes.BigInt{
			Val: new(big.Int).Add(bigIntTakerFeex18.Val, liquidationFeex18),
		}
		// Maker fee is 0 for liquidation orders
		bigIntMakerFeex18 = ctypes.BigInt{
			Val: big.NewInt(0),
		}
	}

	// Convert to ctypes.BigInt
	makerRealizedPnlx18 := ctypes.NewBigInt(makerRealizedPnl)
	takerRealizedPnlx18 := ctypes.NewBigInt(takerRealizedPnl)

	internalSubaccountIDs := strings.Split(os.Getenv("INTERNAL_SUBACCOUNT_IDS"), ",")

	if takerSubaccountHex != ammSubaccountID && !takerOrder.IsLiquidationOrder() {

		// Check if takerSubaccountHex is in the internalSubaccountIDs array
		if contains(internalSubaccountIDs, takerSubaccountHex) {
			// If the subaccount is in the internal list, set takerFeeBonus to 0
			takerFeeBonus = ctypes.NewBigInt(big.NewInt(0))
		} else {
			// Otherwise, proceed with the normal fee bonus logic
			if cutils.HasSubaccountGotExtraFeeBonus(takerSubaccountHex) {
				takerFeeBonus = o.applySkewToFee(bigIntTakerFeex18, "left")
			} else {
				takerFeeBonus = o.applySkewToFee(bigIntTakerFeex18, "right")
				cutils.SubaccountDoneFirstBonusTrade(takerSubaccountHex)
			}
		}
	}

	if makerSubaccountHex != ammSubaccountID && !makerOrder.IsLiquidationOrder() {

		// Check if makerSubaccountHex is in the internalSubaccountIDs array
		if contains(internalSubaccountIDs, makerSubaccountHex) {
			// If the subaccount is in the internal list, set makerFeeBonus to 0
			makerFeeBonus = ctypes.NewBigInt(big.NewInt(0))
		} else {
			// Otherwise, proceed with the normal fee bonus logic
			if cutils.HasSubaccountGotExtraFeeBonus(makerSubaccountHex) {
				makerFeeBonus = o.applySkewToFee(bigIntMakerFeex18, "left") // left is just for naming for first trade used left as convection
			} else {
				makerFeeBonus = o.applySkewToFee(bigIntMakerFeex18, "right") //same for right
				cutils.SubaccountDoneFirstBonusTrade(makerSubaccountHex)
			}
		}
	}

	makerTakerFillData := &MakerTakerFillData{
		// For Maker
		makerFillData: &db.FillTable{
			BrokerId:       makerOrder.BrokerId,
			OrderId:        makerOrder.ID,
			SubaccountId:   makerOrder.SubaccountId,
			MarketId:       makerOrder.MarketId,
			Liquidity:      ctypes.LIQUIDITY_MAKER,
			Type:           ctypes.FillType(makerOrder.Type),
			Side:           makerOrder.Side,
			Amountx18:      bigIntfillAmtx18,
			Pricex18:       bigIntfillPricex18,
			Feex18:         bigIntMakerFeex18,
			RealizedPnlx18: makerRealizedPnlx18,
			FundingFeesx18: ctypes.NewBigInt(makerFundingFees),
			IsReduce:       makerOrder.IsReduce,
		},
		// For Taker
		takerFillData: &db.FillTable{
			BrokerId:       takerOrder.BrokerId,
			OrderId:        takerOrder.ID,
			SubaccountId:   takerOrder.SubaccountId,
			MarketId:       takerOrder.MarketId,
			Liquidity:      ctypes.LIQUIDITY_TAKER,
			Type:           ctypes.FillType(takerOrder.Type),
			Side:           takerOrder.Side,
			Amountx18:      bigIntfillAmtx18,
			Pricex18:       bigIntfillPricex18,
			Feex18:         bigIntTakerFeex18,
			RealizedPnlx18: takerRealizedPnlx18,
			FundingFeesx18: ctypes.NewBigInt(takerFundingFees),
			IsReduce:       takerOrder.IsReduce,
		},
	}

	// TODO: Check if these can be done in parallel
	(&db.FillDB{}).Create(makerTakerFillData.makerFillData)
	(&db.FillDB{}).Create(makerTakerFillData.takerFillData)

	if makerFeeBonus.Val.String() != "0" {
		rewardMaker := &db.UserFeeRewardsTable{
			SubaccountId: makerOrder.SubaccountId,
			FeeBonusx18:  makerFeeBonus,
			Symbol:       contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.LOGX],
			FillTableId:  makerTakerFillData.makerFillData.ID, // Reference to the fill table entry
		}
		(&db.UserRewardsDB{}).Create(rewardMaker)
	}

	if takerFeeBonus.Val.String() != "0" {
		rewardTaker := &db.UserFeeRewardsTable{
			SubaccountId: takerOrder.SubaccountId,
			FeeBonusx18:  takerFeeBonus,
			Symbol:       contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.LOGX],

			FillTableId: makerTakerFillData.takerFillData.ID, // Reference to the fill table entry
		}
		(&db.UserRewardsDB{}).Create(rewardTaker)
	}

	// Return takerFeeBonus
	return takerFeeBonus
}
func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
func (o *OrderServiceImpl) FetchOraclePrice(marketId uint32) (*big.Int, error) {
	// Initialize the app stats

	// Fetch oracle prices from the app state

	fetchExpiration := time.Second * 1
	oraclePricesMap, err := o.appState.GetAllOraclePrices(fetchExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to get app state: %v", err)
	}

	// Use marketId to get the symbol from the product ID map
	symbol, ok := marketutils.GetBaseSymbolForProduct(marketId)
	if !ok {
		return nil, fmt.Errorf("invalid market ID: %d", marketId)
	}

	// Fetch the price from the oraclePricesMap based on the symbol
	price, exists := oraclePricesMap[symbol]
	if !exists {
		return nil, fmt.Errorf("price not found for symbol: %s", symbol)
	}

	// Return the price as *big.Int
	return price.Pricex18, nil
}

// Function to generate a skew between 20% to 40% based on skew direction ("left" or "right") and multiply it with the fee
func (o *OrderServiceImpl) applySkewToFee(fee ctypes.BigInt, skewDirection string) ctypes.BigInt {
	// Always return 0 regardless of input parameters
	return ctypes.NewBigInt(big.NewInt(0))
}

func (o *OrderServiceImpl) PlaceOrder(newOrder *db.OrderTable, market *db.MarketTable) (_takerOrder *db.OrderTable, FeeBonus ctypes.BigInt, status int, err error) {
	// Store order in db
	newOrder = (&db.OrderDB{}).CreateOrder(*newOrder)

	if newOrder == nil || newOrder.ID == 0 {
		xlog.Errorf("Unable to create order. Error while placing order in db")
		return nil, ctypes.NewBigInt(big.NewInt(0)), http.StatusInternalServerError, fmt.Errorf("error while placing order in db")
	}

	cutils.LogByParty(
		newOrder.Party,
		"Order Controller - created new order in database for subAccount %s with ID %d, market ID %d, Price %s, Size %s and Side %s",
		newOrder.SubaccountId,
		newOrder.ID,
		newOrder.MarketId,
		newOrder.Pricex18.Val.String(), newOrder.Amountx18.Val.String(), newOrder.Side)

	// Lock balance
	err = o.lockBalanceForOrder(newOrder, market)
	if err != nil {
		// Update order status to cancelled
		(&db.OrderDB{}).Update(newOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
		return nil, ctypes.NewBigInt(big.NewInt(0)), http.StatusForbidden, err
	}

	// Defer unlock balance
	defer o.unlockOrderLockedBalance(newOrder.ID)

	if newOrder.IsConditional() {
		engineResponse := client.GlobalEngineClient.PlaceConditionalOrder(newOrder)
		if engineResponse.Error != nil {
			xlog.Errorf("%v", engineResponse.Error.Error())
			(&db.OrderDB{}).Update(newOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
			// NOTE: For now we just assume matching engine is down or buggy
			return nil, ctypes.NewBigInt(big.NewInt(0)), http.StatusInternalServerError, fmt.Errorf("error in placing order with global engine: %v", engineResponse.Error.Error())
		}
		return newOrder, ctypes.NewBigInt(big.NewInt(0)), http.StatusOK, nil
	} else {
		// Make call to engine to place order
		engineResponse := client.GlobalEngineClient.PlaceOrder(newOrder)
		if engineResponse.Error != nil {
			xlog.Errorf("%v", engineResponse.Error.Error())
			// Update order status to cancelled
			(&db.OrderDB{}).Update(newOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
			// NOTE: For now we just assume matching engine is down or buggy
			return nil, ctypes.NewBigInt(big.NewInt(0)), http.StatusInternalServerError, fmt.Errorf("error in placing order with global engine: %v", engineResponse.Error.Error())
		}

		engineRespJson, _ := json.Marshal(engineResponse)
		cutils.LogByParty(
			newOrder.Party,
			"engine response for order=%d party=%s: %s",
			newOrder.ID, newOrder.Party, string(engineRespJson),
		)
		// Call matchOnchainAndCreateFills to match orders and get the FeeBonus
		updatedTakerOrder, FeeBonus, status, err := o.matchOnchainAndCreateFills(engineResponse, market, nil, nil)

		// Cancel orders and unlock balance after matching
		o.cancelOrdersInDbAndUnlockBalance(engineResponse)
		// Return the updated taker order, FeeBonus, status, and error
		return updatedTakerOrder, FeeBonus, status, err
	}
}

func (o *OrderServiceImpl) PlaceConditionalOrder(conditionalOrder *db.OrderTable, market *db.MarketTable) (_takerOrder *db.OrderTable, status int, err error) {
	xlog.Infof("Order Controller - placing new conditional order in database for subAccount %s with ID %d, market ID %d, Price %s, Size %s and Side %s", conditionalOrder.SubaccountId, conditionalOrder.ID, conditionalOrder.MarketId, conditionalOrder.Pricex18.Val.String(), conditionalOrder.Amountx18.Val.String(), conditionalOrder.Side)

	// Make call to engine to place order
	engineResponse := client.GlobalEngineClient.PlaceOrder(conditionalOrder)
	if engineResponse.Error != nil {
		xlog.Errorf("%v", engineResponse.Error.Error())
		(&db.OrderDB{}).Update(conditionalOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
		// NOTE: For now we just assume matching engine is down or buggy
		return nil, http.StatusInternalServerError, fmt.Errorf("error in placing order with global engine: %v", engineResponse.Error.Error())
	}

	engineRespJson, _ := json.Marshal(engineResponse)
	xlog.Infof("%v - Engine Response %+v\n", conditionalOrder.ID, string(engineRespJson))

	// Call matchOnchainAndCreateFills to match orders and get the FeeBonus
	updatedTakerOrder, _, status, err := o.matchOnchainAndCreateFills(engineResponse, market, nil, nil)

	// Cancel orders and unlock balance after matching
	o.cancelOrdersInDbAndUnlockBalance(engineResponse)

	// Return the updated taker order, FeeBonus, status, and error
	return updatedTakerOrder, status, err
}

func (o *OrderServiceImpl) PlaceLiquidationOrder(liquidationOrder *db.OrderTable, market *db.MarketTable, spotPricesx18, perpPricesx18 map[uint32]*big.Int) (_takerOrder *db.OrderTable, status int, err error) {
	// Store order in db
	newOrder := (&db.OrderDB{}).CreateOrder(*liquidationOrder)
	if newOrder == nil || newOrder.ID == 0 {
		xlog.Errorf("Unable to create liquidation order. Error while placing order in db")
		return nil, http.StatusInternalServerError, fmt.Errorf("error while placing order in db")
	}

	xlog.Infof("Order Controller - created new liquidation order in database for subAccount %s with ID %d, market ID %d, Price %s, Size %s and Side %s", newOrder.SubaccountId, newOrder.ID, newOrder.MarketId, newOrder.Pricex18.Val.String(), newOrder.Amountx18.Val.String(), newOrder.Side)
	// NOTE: No need to lock balance for liquidation orders

	// Make call to engine to place order
	engineResponse := client.GlobalEngineClient.PlaceOrder(newOrder)
	if engineResponse.Error != nil {
		xlog.Errorf("%v", engineResponse.Error.Error())
		(&db.OrderDB{}).Update(newOrder, map[string]interface{}{"status": ctypes.ORDER_STATUS_CANCELLED})
		// NOTE: For now we just assume matching engine is down or buggy
		return nil, http.StatusInternalServerError, fmt.Errorf("error in placing order with global engine: %v", engineResponse.Error.Error())
	}

	engineRespJson, _ := json.Marshal(engineResponse)
	xlog.Infof("%v - Engine Response %+v\n", newOrder.ID, string(engineRespJson))

	updatedTakerOrder, _, status, err := o.matchOnchainAndCreateFills(engineResponse, market, spotPricesx18, perpPricesx18)
	o.cancelOrdersInDbAndUnlockBalance(engineResponse)
	return updatedTakerOrder, status, err
}

func (o *OrderServiceImpl) matchOnchainAndCreateFills(
	engineResponse *engineTypes.EnginePlaceOrderResponse,
	market *db.MarketTable,
	spotPricesx18,
	perpPricesx18 map[uint32]*big.Int) (_takerOrder *db.OrderTable, FeeBonus ctypes.BigInt, status int, err error) {

	var takerOrder db.OrderTable
	FeeBonus = ctypes.NewBigInt(big.NewInt(0)) // Initialize FeeBonus to 0

	if engineResponse.TakerOrder != nil {
		// Check if MatchedMakerOrders is nil to prevent panic
		if engineResponse.MatchedMakerOrders == nil {
			xlog.Debugf("MatchedMakerOrders is nil for taker order: %v", engineResponse.TakerOrder.ID)
			// Handle case where there are no matched maker orders - fetch the order from DB
			_takerOrder := (&db.OrderDB{}).GetById(engineResponse.TakerOrder.ID)
			if _takerOrder == nil {
				xlog.Errorf("Taker order not found in db: %v", engineResponse.TakerOrder.ID)
				return nil, FeeBonus, http.StatusInternalServerError, fmt.Errorf("taker order not found in db")
			}
			// Update the filled amount
			_takerOrder.TotalFilledx18 = ctypes.NewBigInt(engineResponse.TakerOrder.TotalFilledx18)
			return _takerOrder, FeeBonus, http.StatusOK, nil
		}

		makerIdToOrderResp := cutils.UniqueSliceToMap(*engineResponse.MatchedMakerOrders, func(order engineTypes.MakerOrderRespData) uint { return order.ID })

		// Order ids to fetch order from db
		makerOrderIds := cutils.MapSlice(*engineResponse.MatchedMakerOrders, func(order engineTypes.MakerOrderRespData) uint { return order.ID })

		// Use debug level for frequent matching operations to reduce CloudWatch logs
		xlog.Debugf("%v - matching on chain and creating fills", engineResponse.TakerOrder.ID)
		xlog.Debugf("%v - number of maker orders to match: %v", engineResponse.TakerOrder.ID, len(makerOrderIds))

		_, err := xredis.WithRedisLock(xredis.GetOrderLockKey(engineResponse.TakerOrder.ID), func() (*xredis.NOOP, error) {
			_takerOrder := (&db.OrderDB{}).GetById(engineResponse.TakerOrder.ID)
			if _takerOrder == nil {
				xlog.Errorf("Taker order not found in db - highly unexpected: %v", engineResponse.TakerOrder.ID)
				return nil, fmt.Errorf("taker order not found in db")
			}

			takerOrder = *_takerOrder

			for _, orderId := range makerOrderIds {
				// Acquire lock to avoid race condition for db writes
				xredis.WithRedisLock(xredis.GetOrderLockKey(orderId), func() (*xredis.NOOP, error) {
					makerOrder := (&db.OrderDB{}).GetById(orderId)
					makerResponse := makerIdToOrderResp[orderId]
					// Matched amount = Maker orders new total filled - old total filled
					matchedAmountx18 := makerResponse.MatchedAmountx18
					makerRealisedPnlx18 := makerResponse.MakerRealizedPnl
					takerRealisedPnlx18 := makerResponse.TakerRealizedPnl
					makerFundingFeesx18 := makerResponse.MakerFundingFees
					takerFundingFeesx18 := makerResponse.TakerFundingFees

					if takerOrder.IsLiquidationOrder() {
						makerRealisedPnlx18, takerRealisedPnlx18, makerFundingFeesx18, takerFundingFeesx18, err = o.matchLiquidationOrdersOnChain(&takerOrder, makerOrder, matchedAmountx18, spotPricesx18, perpPricesx18, makerResponse.TxnCounter)
						// For liquidation orders, use funding fees from liquidation service
					} else {
						err = o.matchOrdersOnChain(&takerOrder, makerOrder, matchedAmountx18, makerResponse.TxnCounter)
					}

					if err != nil {
						xlog.Errorf("Error matching orders: %v", err)
					}

					// Create fills in db
					// FIXME - handle for liquidation realized pnl
					takerFeeBonus := o.createFillsForMatchedOrders(matchedAmountx18, &takerOrder, makerOrder, market, makerRealisedPnlx18, takerRealisedPnlx18, makerFundingFeesx18, takerFundingFeesx18)

					FeeBonus.Val = new(big.Int).Add(FeeBonus.Val, takerFeeBonus.Val)

					// Update order's total filled and status
					makerOrder.TotalFilledx18 = makerOrder.TotalFilledx18.Add(ctypes.NewBigInt(matchedAmountx18))
					takerOrder.TotalFilledx18 = takerOrder.TotalFilledx18.Add(ctypes.NewBigInt(matchedAmountx18))
					makerOrder.SyncStatusFromTotalFilled()

					// Update order in db
					res := (&db.OrderDB{}).Update(makerOrder, map[string]interface{}{"total_filledx18": makerOrder.TotalFilledx18, "status": makerOrder.Status})
					if res == nil {
						xlog.Errorf("Error updating maker order in db: %v", makerOrder.ID)
					}

					return &xredis.NOOP{}, nil
				})
			}

			// Update taker order in db
			takerOrder.SyncStatusFromTotalFilled()
			res := (&db.OrderDB{}).Update(&takerOrder, map[string]interface{}{"total_filledx18": takerOrder.TotalFilledx18, "status": takerOrder.Status})
			if res == nil {
				xlog.Errorf("Error updating taker order in db: %v", takerOrder.ID)
			}
			return nil, nil
		})

		if err != nil {
			xlog.Errorf("Error while matching orders: %v", err)
			return nil, ctypes.NewBigInt(big.NewInt(0)), http.StatusInternalServerError, err
		}
	} else {
		xlog.Warnf("_matchOrders: No taker order received. Ideally this should have not happened")
	}

	return &takerOrder, FeeBonus, http.StatusOK, nil
}

// It is expected that this function will never fail
func (o *OrderServiceImpl) cancelOrdersInDbAndUnlockBalance(engineResponse *engineTypes.EnginePlaceOrderResponse) {
	// Update status of cancelled orders
	// TODO: Check if this can be done in parallel
	if engineResponse.CancelledOrders == nil {
		xlog.Debugf("No cancelled orders received from engine")
		return
	}
	cancelledOrderIds := cutils.MapSlice(*engineResponse.CancelledOrders, func(order engineTypes.CancelTakerOrderRespData) uint { return order.ID })
	xlog.Debugf("Cancelling orders: %v", cancelledOrderIds)
	err := o.updateAllOrderStatus(cancelledOrderIds, ctypes.ORDER_STATUS_CANCELLED)
	if err != nil {
		xlog.Errorf("Failed to update order status in DB: %v", err)
	}
	// Unlock balance for cancelled orders
	for _, order := range *engineResponse.CancelledOrders {
		err := o.unlockOrderLockedBalance(order.ID)
		if err != nil {
			xlog.Warnf("Unable to unlock balance for order: %v", order.ID)
		}
	}
}
