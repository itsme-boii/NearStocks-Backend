package conditional

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	apiserverTypes "github/eugenix-io/logx-inf-backend/services/api-server/types"
	"github/eugenix-io/logx-inf-backend/services/engine/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"net/http"
)

func (tob *TriggerOrderbook) PlaceConditionalOrder(msg *types.PlaceConditionalOrderMsg) *types.EnginePlaceConditionalOrderResponse {
	if msg.Order == nil || !msg.Order.IsValidTriggerCondition() {
		xlog.Errorf("Invalid conditional order. Order: %v", msg.Order)
		return &types.EnginePlaceConditionalOrderResponse{
			Error: fmt.Errorf("invalid conditional order"),
		}
	}

	// Place the order in the redis
	err := tob.AddOrder(msg.Order)
	if err != nil {
		xlog.Errorf("Error while placing conditional order. Failed with error: %v", err)
	}
	return &types.EnginePlaceConditionalOrderResponse{
		Error: err,
	}
}

func (tob *TriggerOrderbook) CancelConditionalOrdersBulk(msg *types.CancelConditonalOrdersBulkMsg) *types.EngineCancelConditionalOrdersBulkResponse {
	if msg.Orders == nil {
		xlog.Errorf("Invalid conditional orders. Orders: %v", msg.Orders)
		return &types.EngineCancelConditionalOrdersBulkResponse{
			Error: fmt.Errorf("invalid conditional orders"),
		}
	}

	var skippedOrderWithReason []types.SkipCancelOrderRespData
	var cancelledOrderIds []uint

	for _, order := range *msg.Orders {
		if order.ID == 0 {
			skippedOrderWithReason = append(skippedOrderWithReason, types.SkipCancelOrderRespData{
				ID:     order.ID,
				Reason: "Invalid order. Order is 0",
			})
		} else if !order.IsConditional() {
			skippedOrderWithReason = append(skippedOrderWithReason, types.SkipCancelOrderRespData{
				ID:     order.ID,
				Reason: "Invalid order. Order is not conditional",
			})
		} else {
			err := tob.RemoveOrder(&order)
			if err != nil {
				xlog.Errorf("Error while cancelling conditional order: %v. Failed with error: %v", order.ID, err)
				skippedOrderWithReason = append(skippedOrderWithReason, types.SkipCancelOrderRespData{
					ID:     order.ID,
					Reason: "Order not found in orderbook",
				})
			} else {
				cancelledOrderIds = append(cancelledOrderIds, order.ID)
			}
		}
	}
	return &types.EngineCancelConditionalOrdersBulkResponse{
		SkippedOrdersWithReason: &skippedOrderWithReason,
		CancelledOrders:         &cancelledOrderIds,
		Error:                   nil,
	}
}

// Fetch all INCR and DECR trigger orders from the redis
// Sort them in descending order of price delta from the oracle price
// If there is clash in price delta, then sort them in ascending order of timestamp
// Place the trigger orders in the sorted order
func (tob *TriggerOrderbook) ExecuteTriggerOrders(oraclePriceX18 *big.Int, orderLimit int64) {
	sortedTriggerOrders, err := tob.GetTriggerOrders(oraclePriceX18)
	if err != nil {
		xlog.Errorf("Error while executing Trigger orders. Failed with error: %v", err)
	}

	xlog.Debugf("Sorted Trigger Orders: %+v", sortedTriggerOrders)

	// Place the trigger orders in the sorted order
	for idx, order := range sortedTriggerOrders {
		if int64(idx) >= orderLimit {
			xlog.Infof("Order limit reached. Skipping the rest of the orders")
			break
		}

		if order == nil {
			xlog.Errorf("Encountered nil order in sortedTriggerOrders at index %d", idx)
			continue
		}

		//Remove the order from the redis
		//If the order is not found in the redis, then skip the order
		err = func() error {
			tob.Lock()
			defer tob.Unlock()
			if order == nil {
				xlog.Errorf("Attempted to remove a nil order from the orderbook")
				return fmt.Errorf("cannot remove nil order from orderbook")
			}
			return tob.RemoveOrder(order)
		}()

		if err != nil {
			xlog.Errorf("Error while executing Trigger order: %v. Order doesn't exist: %v", order.ID, err)
			continue
		}

		resp, status, err := xclient.GlobalApiServerClient.PlaceConditionalOrder(apiserverTypes.ConditionalOrderRequest{Id: order.ID})
		if err != nil {
			// NOTE NOTE NOTE: For now if there an error we will just place the order back to the redis
			xlog.Errorf("Error while placing conditional order: %v. Failed with error: %v....Placing order back to tob", order.ID, err)
			if cutils.SliceExists([]int{http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusForbidden}, status) {
				err = func() error {
					tob.Lock()
					defer tob.Unlock()
					return tob.AddOrder(order)
				}()
				if err != nil {
					xlog.Errorf("Error while placing order back to tob: %v. Failed with error: %v", order.ID, err)
				}
			} else {
				xlog.Errorf("Trigger order - Error while placing conditional order: %v. Failed with error: %v", order.ID, err)
			}
			xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Trigger order - Error while placing conditional order: %v. Status: %v | Failed with error: %v", order.ID, status, err))
			continue
		}
		// NOTE NOTE NOTE NOTE: Otherwise for now assume order was successfully placed
		amountFilledx18, ok := new(big.Int).SetString(resp.TotalFilledx18, 10)
		if !ok {
			xlog.Errorf("Error converting amount filled for trigger order ID: %v. This should never happen and means there is a bug in api response", order.ID)
			continue
		}
		amountRequestedx18, ok := new(big.Int).SetString(resp.Amountx18, 10)
		if !ok {
			xlog.Errorf("Error converting amount requested for trigger order ID: %v. This should never happen and means there is a bug in api response", order.ID)
			continue
		}
		if amountFilledx18.Cmp(amountRequestedx18) == 0 {
			xlog.Infof("Trigger order ID: %v was successfully executed", order.ID)
		} else if amountFilledx18.Sign() == 0 {
			xlog.Errorf("Trigger order ID: %v was cancelled", order.ID)
		} else {
			xlog.Warnf("Trigger order ID: %v was partially filled", order.ID)
		}
	}
}

// We assume that the trigger orders are only for TRADERS
func (tob *TriggerOrderbook) GetTriggerOrders(oraclePricex18 *big.Int) ([]*db.OrderTable, error) {
	incrTriggerOrders, err := tob.GetEligibleOrders(ctypes.PARTY_TRADER, ctypes.TRIGGER_DIRECTION_INCR, oraclePricex18)
	if err != nil {
		return nil, err
	}

	decrTriggerOrders, err := tob.GetEligibleOrders(ctypes.PARTY_TRADER, ctypes.TRIGGER_DIRECTION_DECR, oraclePricex18)
	if err != nil {
		return nil, err
	}

	sortedTriggerOrders := getSortedTriggerOrders(oraclePricex18, incrTriggerOrders, decrTriggerOrders)
	return sortedTriggerOrders, nil
}

// Incr trigger orders will have ascending order of price delta from the oracle price
// Decr trigger orders will have descending order of price delta from the oracle price
// Sort the orders in descending order of price delta from the oracle price
// If there is clash in price delta, then sort them in ascending order of timestamp
func getSortedTriggerOrders(oraclePricex18 *big.Int, incrTriggerOrders, decrTriggerOrders []*db.OrderTable) []*db.OrderTable {
	i := 0
	j := 0
	k := 0
	lenIncr := len(incrTriggerOrders)
	lenDecr := len(decrTriggerOrders)
	sortedTriggerOrders := make([]*db.OrderTable, lenIncr+lenDecr)

	for i < lenIncr || j < lenDecr {
		if (i == lenIncr) || ((j != lenDecr) && comparator(oraclePricex18, decrTriggerOrders[j], incrTriggerOrders[i])) {
			sortedTriggerOrders[k] = decrTriggerOrders[j]
			j++
		} else {
			sortedTriggerOrders[k] = incrTriggerOrders[i]
			i++
		}
		k++
	}
	return sortedTriggerOrders
}

// True means abs(oraclePricex18 - triggerPrice1) > abs(oraclePricex18 - triggerPrice2)
func comparator(oraclePricex18 *big.Int, order1, order2 *db.OrderTable) bool {
	absDelta1 := new(big.Int).Abs(new(big.Int).Sub(oraclePricex18, order1.TriggerPricex18.Val))
	absDelta2 := new(big.Int).Abs(new(big.Int).Sub(oraclePricex18, order2.TriggerPricex18.Val))
	cmp := absDelta1.Cmp(absDelta2)
	return cmp == 1 || (cmp == 0 && order1.Timestamp < order2.Timestamp)
}
