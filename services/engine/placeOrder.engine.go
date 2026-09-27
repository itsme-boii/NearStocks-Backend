package engine

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	perputils "github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	balanceTypes "github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/services/engine/types"
	"math/big"

	"github.com/redis/go-redis/v9"
)

type OrderMatchData struct {
	matchPricex18  *big.Int
	matchAmountx18 *big.Int
}

type PlaceOrderState struct {
	takerOrderPerpBalance *struct {
		NetAmount *big.Int
	}
	currentMakerOrderPerpBalance *struct {
		NetAmount *big.Int
	}
}

func NewPlaceOrderState() *PlaceOrderState {
	return &PlaceOrderState{
		takerOrderPerpBalance: &struct {
			NetAmount *big.Int
		}{NetAmount: new(big.Int).SetInt64(0)},
		currentMakerOrderPerpBalance: &struct {
			NetAmount *big.Int
		}{NetAmount: new(big.Int).SetInt64(0)},
	}
}

func (pos *PlaceOrderState) reduceTakerOrderPerpBalance(absAmount *big.Int, order db.OrderTable) {
	// For non reduce only orders this calculation is useless
	if !order.IsReduce {
		return
	}

	// If order side is buy that means we are reducing the short position
	if order.Side == ctypes.ORDER_SIDE_BUY {
		pos.takerOrderPerpBalance.NetAmount = new(big.Int).Add(pos.takerOrderPerpBalance.NetAmount, absAmount)
	} else {
		pos.takerOrderPerpBalance.NetAmount = new(big.Int).Sub(pos.takerOrderPerpBalance.NetAmount, absAmount)
	}
}

func (pos *PlaceOrderState) reduceMakerOrderPerpBalance(absAmount *big.Int, order db.OrderTable) {
	// For non reduce only orders this calculation is useless
	if !order.IsReduce {
		return
	}

	// If order side is buy that means we are reducing the short position
	if order.Side == ctypes.ORDER_SIDE_BUY {
		pos.currentMakerOrderPerpBalance.NetAmount = new(big.Int).Add(pos.currentMakerOrderPerpBalance.NetAmount, absAmount)
	} else {
		pos.currentMakerOrderPerpBalance.NetAmount = new(big.Int).Sub(pos.currentMakerOrderPerpBalance.NetAmount, absAmount)
	}
}

func (pos *PlaceOrderState) getTakerPerpBalance() *big.Int {
	return pos.takerOrderPerpBalance.NetAmount
}

func (pos *PlaceOrderState) getMakerPerpBalance() *big.Int {
	return pos.currentMakerOrderPerpBalance.NetAmount
}

func (ob *Orderbook) getCurrentNetPerpAmount(subaccountId string, orderId uint) (*big.Int, error) {
	// Get perp balance
	// Get the perp balance of the subaccount
	// If the net amount is negative then order should be long order only otherwise we should cancel the order
	// Similarly, if the net amount is positive then order should be short order only otherwise we should cancel the order
	subaccountHex, _ := cutils.SubaccountIdToHex(subaccountId)
	perpBalance, err := ob.balanceClient.GetPerpSinglePosition(subaccountHex, ob.market.ID)

	if err != nil {
		xlog.Errorf("%v - Error while fetching perp balance from balance service. Cancelling. This means either we are sending wrong product id or there is some issue in the balance service", orderId)
		return nil, fmt.Errorf("something went wrong in balance service")
	}

	return ctypes.NewBigIntFromString(perpBalance.Amount).Val, nil
}

// TODO: Make all add operations atomic
// 1. Get all order price quantums for the opposite side
// 2. For each price quantum, get all orders
// 3. Match orders
// 4. Add remaining orders to the orderbook
// 5. Remove matched orders from the orderbook
// 6. Update order statuses in redis
// For market orders price will be 0
// Naive Algo: Fetch 100 price levels and match orders until slippage is within tolerance

// Handling for Reduce only orders -
// 1. Fetch the perp balance for the subaccount which has reduce only order
// 2. Now, when matching the orders, remaining order amount will be min(remaining order amount, perp balance opposite side)
// 3. The remaining order should be cancelled
func (ob *Orderbook) placeOrder(order *db.OrderTable) (remTakerOrder *types.TakerOrderRespData, matchedMakerOrders []types.MakerOrderRespData, cancelledOrders []types.CancelTakerOrderRespData, err error) {
	cutils.LogByParty(order.Party, "%v - Placing order\n", order.ID)

	placeOrderState := NewPlaceOrderState()

	var oppPriceQuantums *[]uint64
	oppParty := order.Party.Opposite()
	oppSide := order.Side.Opposite()

	// Copy of the order to update the total filled amount
	remainingTakerOrder := *order
	var remainingMakerOrder db.OrderTable
	// Update Remaining taker order for response data
	remTakerOrder = &types.TakerOrderRespData{
		ID:             order.ID,
		TotalFilledx18: remainingTakerOrder.TotalFilledx18.Val, // This will be zero but still adding for clarity
	}

	// Get opposite price quantums
	cutils.LogByParty(order.Party, "%v - fetching price quantums", order.ID)
	// A market order with a real (nonzero) price is the client's slippage-bounded worst price
	// (order.controller.go): match it exactly like a limit order — bounded to price levels at
	// least as good as that price — instead of walking the book with no price bound at all.
	hasBoundedPrice := order.IsLimitOrder() || (order.IsMarketOrder() && order.Pricex18.Val.Sign() > 0)
	if hasBoundedPrice {
		oppPriceQuantums, err = ob.GetMatchingPriceQuantums(oppParty, oppSide, order.Pricex18.Val)
		if err != nil {
			if err == redis.Nil {
				xlog.Infof("%v - No matching price quantums for order", order.ID)
			} else if err != nil {
				xlog.Errorf("%v - failed to fetch matching price quantums from redis", order.ID)
			}
			cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "System fault: Unable to fetch matching price quantums from orderbook"})
			return remTakerOrder, matchedMakerOrders, cancelledOrders, err
		}
	} else if order.IsIOCOrder() {
		// TODO: Optimise this by incremently fetching the price levels in this order 10, 20, 40, 80 ....
		oppPriceQuantums, err = ob.GetPriceQuantumsByRank(oppParty, oppSide, 0, 100)
		if err != nil {
			if err == redis.Nil {
				xlog.Infof("%v - No price quantums found. Cancelling", order.ID)
			} else {
				xlog.Infof("%v - Error while fetching price quantums. Cancelling", order.ID)
			}
			cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "System fault: Unable to fetch price quantums from orderbook"})
			return remTakerOrder, matchedMakerOrders, cancelledOrders, err
		}
	} else {
		cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "System fault: Incorrect order type"})
		return remTakerOrder, matchedMakerOrders, cancelledOrders, err
	}

	// When the order is reduce only then we'll have to get the perp balance of the subaccount for calculation
	if order.IsReduce {
		placeOrderState.takerOrderPerpBalance.NetAmount, err = ob.getCurrentNetPerpAmount(order.SubaccountId, order.ID)
		if err != nil {
			cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "System fault: Unable to fetch perp balance from balance service"})
			return remTakerOrder, matchedMakerOrders, cancelledOrders, err
		}

		// If the net amount is negative then order should be long order only otherwise we should cancel the order
		// Similarly, if the net amount is positive then order should be short order only otherwise we should cancel the order
		if !order.CanReduce(placeOrderState.getTakerPerpBalance()) {
			cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "Cannot reduce the position with this order as it will lead to opposite position"})
			return remTakerOrder, matchedMakerOrders, cancelledOrders, nil
		}
	}

	// Check for matching orders for each price quantum until the order is filled
	cutils.LogByParty(order.Party, "%v - matching with %d price levels", order.ID, len(*oppPriceQuantums))
	for _, oppPriceQuantum := range *oppPriceQuantums {
		makerOrders := ob.GetOrdersForPriceQuantum(oppParty, oppSide, oppPriceQuantum)
		xlog.Debugf("%v - Number of maker orders for price quantum: %v | %v\n", order.ID, oppPriceQuantum, len(makerOrders))

		for _, makerOrder := range makerOrders {
			// If maker order is expired, cancel the order
			if makerOrder.IsExpired() {
				errRemoveMaker := ob.RemoveOrder(makerOrder)
				// NOTE: This is critical error. Should not happen
				if errRemoveMaker != nil {
					xlog.Warnf("%v - Unable to remove maker order: %v | error: %v", order.ID, makerOrder.ID, errRemoveMaker)
				}
				xlog.Infof("%v - Maker order %v expired. ExpiryTs: %v. Cancelling...\n", order.ID, makerOrder.ID, makerOrder.ExpiryTs)
				cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: makerOrder.ID, CancelReason: "Order expired"})
				continue
			}

			// When the order is reduce only then we'll have to get the perp balance of the subaccount for calculation
			if makerOrder.IsReduce {
				placeOrderState.currentMakerOrderPerpBalance.NetAmount, err = ob.getCurrentNetPerpAmount(makerOrder.SubaccountId, order.ID)
				if err != nil {
					return remTakerOrder, matchedMakerOrders, cancelledOrders, err
				}

				if !makerOrder.CanReduce(placeOrderState.getMakerPerpBalance()) {
					errRemoveMaker := ob.RemoveOrder(makerOrder)
					// NOTE: This is critical error. Should not happen
					if errRemoveMaker != nil {
						xlog.Warnf("%v - Unable to remove maker order: %v | error: %v", order.ID, makerOrder.ID, errRemoveMaker)
						// Add Cancel taker order
						cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: makerOrder.ID, CancelReason: "System fault: Maker order not removed from orderbook"})

						// TODO: Add custom error
						return remTakerOrder, matchedMakerOrders, cancelledOrders, nil
					}
					cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: makerOrder.ID, CancelReason: "Cannot reduce the position with this order as it will lead to opposite position"})
					continue
				}
			}

			xlog.Debugf("%v - Amount quantum (x18) left: maker: %v | taker: %v\n", remainingTakerOrder.ID, makerOrder.RemainingAmount().Val, remainingTakerOrder.RemainingAmount().Val)

			// Get match amount
			takerMatchableAmount := remainingTakerOrder.GetMatchableAmount(placeOrderState.getTakerPerpBalance())
			makerMatchableAmount := makerOrder.GetMatchableAmount(placeOrderState.getMakerPerpBalance())

			xlog.Debugf("%v - Matchable Amount quantum (x18) left: maker: %v | taker: %v\n", remainingTakerOrder.ID, makerMatchableAmount, takerMatchableAmount)

			// Matchable amounts will be positive for both maker and taker orders
			// Set match parameters
			orderMatchData := &OrderMatchData{
				matchPricex18:  makerOrder.Pricex18.Val,
				matchAmountx18: cutils.MinBigInt(takerMatchableAmount, makerMatchableAmount),
			}

			xlog.Infof("%v - Matched order with maker: %v, at price: %v and amount: %v, \n", remainingTakerOrder.ID, makerOrder.ID, orderMatchData.matchPricex18, orderMatchData.matchAmountx18)

			// Get txn counter
			transactionCounter, err := transaction.IncrementCounter(1)
			if err != nil {
				xlog.Errorf("%v - Error while incrementing transaction counter: %v", order.ID, err)
				// Cancelling taker order
				cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "System fault: Unable to increment transaction counter"})
				return remTakerOrder, matchedMakerOrders, cancelledOrders, err
			}

			takerSuccess, makerSuccess, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees, err := ob.handleUpdateLockBalanceForMatch(makerOrder, &remainingTakerOrder, orderMatchData, transactionCounter)

			// NOTE: Error only comes when balance service is down or request is invalid. In both cases we should not proceed
			if err != nil {
				xlog.Warnf("%v - Error while updating subaccount's locked balances (taker: %s, maker: %s) for match on marketId: %d, with error: %v ", order.ID, remainingTakerOrder.SubaccountId, makerOrder.SubaccountId, ob.market.ID, err)
				// Cancelling taker order
				cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "System fault: Unable to update subaccount's locked balances"})
				return remTakerOrder, matchedMakerOrders, cancelledOrders, err
			}

			// Cancel remaining maker order if balance is not sufficient
			if !makerSuccess {
				xlog.Debugf("%v - Maker order %v - Insufficient balance\n", order.ID, makerOrder.ID)
				errRemoveMaker := ob.RemoveOrder(makerOrder)
				// NOTE: This is critical error. Should not happen
				if errRemoveMaker != nil {
					xlog.Warnf("%v - Unable to remove maker order: %v | error: %v", order.ID, makerOrder.ID, errRemoveMaker)
					// Add Cancel taker order
					cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remTakerOrder.ID, CancelReason: "System fault: Maker order not removed from orderbook"})

					// TODO: Add custom error
					return remTakerOrder, matchedMakerOrders, cancelledOrders, nil
				}
				cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: makerOrder.ID, CancelReason: "Insufficient balance"})
				continue
			}
			// Cancel remaining taker order if balance is not sufficient
			if !takerSuccess {
				xlog.Infof("Taker order %v - Insufficient balance\n", remainingTakerOrder.ID)
				cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "Insufficient balance"})

				// TODO: Add custom error
				return remTakerOrder, matchedMakerOrders, cancelledOrders, nil
			}

			if makerSuccess && takerSuccess {
				xlog.Infof("Subaccounts updated successfully for match: maker=%v taker=%v price=%v amount=%v",
					makerOrder.ID, remainingTakerOrder.ID, orderMatchData.matchPricex18, orderMatchData.matchAmountx18)
				// Update remaining taker order
				// Remaining TotalFilled += match amount
				remainingTakerOrder.TotalFilledx18.Val = new(big.Int).Add(remainingTakerOrder.TotalFilledx18.Val, orderMatchData.matchAmountx18)
				placeOrderState.reduceTakerOrderPerpBalance(orderMatchData.matchAmountx18, remainingTakerOrder)
				remTakerOrder.TotalFilledx18 = remainingTakerOrder.TotalFilledx18.Copy().Val

				// Maker Order Resp Data
				makerOrderRespData := types.MakerOrderRespData{
					ID:               makerOrder.ID,
					TotalFilledx18:   new(big.Int).Add(makerOrder.TotalFilledx18.Val, orderMatchData.matchAmountx18),
					Pricex18:         orderMatchData.matchPricex18,
					MakerRealizedPnl: makerRealizedPnl, // Include Maker Realized PnL
					TakerRealizedPnl: takerRealizedPnl,
					MakerFundingFees: makerFundingFees,
					TakerFundingFees: takerFundingFees,
					TxnCounter:       transactionCounter,
					MatchedAmountx18: orderMatchData.matchAmountx18,
				}

				// Add matched orders to the response | Total filled will be previous total filled + match amount
				matchedMakerOrders = append(matchedMakerOrders, makerOrderRespData)

				// Remove maker order from orderbook
				errRemoveMaker := ob.RemoveOrder(makerOrder)

				// NOTE: This is critical error. Should not happen
				if errRemoveMaker != nil {
					xlog.Warnf("%v - Unable to remove maker order: %v | error: %v", order.ID, makerOrder.ID, errRemoveMaker)
					// TODO: ADD ALARM: Orderbook not updated correctly
					// Currently there is nothing to worry as subaccounts are updated successfully
					continue
				}

				remainingMakerOrder = *makerOrder
				// Update the remaining maker order
				remainingMakerOrder.TotalFilledx18.Val = new(big.Int).Add(remainingMakerOrder.TotalFilledx18.Val, orderMatchData.matchAmountx18)
				placeOrderState.reduceMakerOrderPerpBalance(orderMatchData.matchAmountx18, remainingMakerOrder)
				// There is a possibility that remaining maker order is not fully filled: IsReduce case. We should cancel the remaining maker order
				if remainingMakerOrder.GetMatchableAmount(placeOrderState.getMakerPerpBalance()).Sign() == 0 {
					// Check if remaining maker order is fully filled. If not, then cancel it because there is no matchable amount left
					if remainingMakerOrder.RemainingAmount().Sign() != 0 {
						xlog.Debugf("Maker order is a reduce only order and not fully filled because position cannot be reduced further. Cancelling remaining maker order: %v", remainingMakerOrder.ID)
						cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingMakerOrder.ID, CancelReason: "Maker order is a reduce only order and not fully filled because position cannot be reduced further"})
					}
					// Else maker order is completely filled
				} else {
					// Add remaining maker order to the orderbook
					xlog.Debugf("Maker order not fully filled. Adding remaining maker order to orderbook. Amount req (x18): %v | Total filled (x18): %v", remainingMakerOrder.Amountx18.Val, remainingMakerOrder.TotalFilledx18.Val)
					remainingMakerOrder.Status = ctypes.ORDER_STATUS_PARTIAL
					errRemoveMaker := ob.AddOrder(&remainingMakerOrder)

					// NOTE: This is critical error. Should not happen
					if errRemoveMaker != nil {
						xlog.Errorf("engine - failed to add remaining maker order: id=%v market=%d error=%v",
							remainingMakerOrder.ID, ob.market.ID, errRemoveMaker)
						// Add to cancelled orders
						cancelledOrders = append(cancelledOrders, types.CancelTakerOrderRespData{ID: remainingMakerOrder.ID, CancelReason: "Unable to add remaining maker order to orderbook"})
					}

				}

				// If remaining taker order has no matchable amount left , break the loop
				if remainingTakerOrder.GetMatchableAmount(placeOrderState.getTakerPerpBalance()).Sign() == 0 {
					break
				}
			}
		}

		if remainingTakerOrder.GetMatchableAmount(placeOrderState.getTakerPerpBalance()).Sign() == 0 {
			break
		}
	}

	// Add remaining taker order to the orderbook
	errAddTaker := ob.handleRemainingTakerOrder(&remainingTakerOrder, placeOrderState.getTakerPerpBalance(), &cancelledOrders)

	return remTakerOrder, matchedMakerOrders, cancelledOrders, errAddTaker
}

func orderWithSign(amountx18 *big.Int, side ctypes.OrderSide) *big.Int {
	if side == ctypes.ORDER_SIDE_BUY {
		return amountx18
	}
	return new(big.Int).Neg(amountx18)
}

// Balance Service accepts hex subaccount id
// Interacts with balance service to update subaccount's locked balances
// For IOC taker orders, we do not need to update the lock balance. There is nothing like IOC maker order
// For liquidation order we do not need to balance from here. We'll update the balances in api service
func (ob *Orderbook) handleUpdateLockBalanceForMatch(makerOrder *db.OrderTable, remainingTakerOrder *db.OrderTable, orderMatchData *OrderMatchData, txnCounter uint) (takerSuccess bool, makerSuccess bool, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees *big.Int, err error) {
	makerSubaccountHex, _ := cutils.SubaccountIdToHex(makerOrder.SubaccountId)
	takerSubaccountHex, _ := cutils.SubaccountIdToHex(remainingTakerOrder.SubaccountId)

	updateSubaccountPayload := balanceTypes.UpdateSubaccountForMatchRequest{
		TxnCounter: &txnCounter,
		MarketId:   ob.market.ID,
		// NOTE NOTE NOTE: For liquidation orders, we do not need to update anything here. We'll update the balances in api service.
		// Why: Because we need to send the app state to balance service which is not possible right now from here.
		// Also, in every possible case we want to match so, no point in getting taker success, maker success and error
		IsLiquidation: remainingTakerOrder.IsLiquidationOrder(),
	}

	makerMatchAmountx18 := orderWithSign(orderMatchData.matchAmountx18, makerOrder.Side)
	takerMatchAmountx18 := orderWithSign(orderMatchData.matchAmountx18, remainingTakerOrder.Side)
	// NOTE: We take taker amount for the calculation of maker vQuote because it's negative of maker's amount and we want negative value
	vQuoteMakerx18 := cutils.Divx18(new(big.Int).Mul(takerMatchAmountx18, orderMatchData.matchPricex18))
	vQuoteTakerx18 := cutils.Divx18(new(big.Int).Mul(makerMatchAmountx18, orderMatchData.matchPricex18))

	updateSubaccountPayload.Maker = balanceTypes.BalancePerpPayload{
		SubaccountId: makerSubaccountHex,
		OrderId:      makerOrder.ID,
		// We are taking makerOrder's price not because that is equal to the match price but because value of quote getting unlocked should be according to maker's price
		UnlockQuotex18:   perputils.GetBigLockQuotex18(orderMatchData.matchAmountx18, makerOrder.Pricex18.Val, ob.market),
		Amountx18:        makerMatchAmountx18,
		VQuoteBalancex18: vQuoteMakerx18,
		SkipUnlock:       !makerOrder.RequireInitialMarginLock(),
	}

	updateSubaccountPayload.Taker = balanceTypes.BalancePerpPayload{
		SubaccountId: takerSubaccountHex,
		OrderId:      remainingTakerOrder.ID,
		// We are taking takerOrder's price because value of quote getting unlocked should be according to taker's initial price
		UnlockQuotex18:   perputils.GetBigLockQuotex18(orderMatchData.matchAmountx18, remainingTakerOrder.Pricex18.Val, ob.market),
		Amountx18:        takerMatchAmountx18,
		VQuoteBalancex18: vQuoteTakerx18,
		SkipUnlock:       !remainingTakerOrder.RequireInitialMarginLock(),
	}

	// Call the balance service to update the subaccounts and get the realized PnLs and funding fees
	takerSuccess, makerSuccess, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees, err = ob.balanceClient.UpdateSubaccountsForMatch(updateSubaccountPayload)
	return takerSuccess, makerSuccess, makerRealizedPnl, takerRealizedPnl, makerFundingFees, takerFundingFees, err
}

// 1. If the remaining taker order's matchable amount is zero
// - But the remaining amount is not zero, then cancel the order because it's a reduce only order and can't be filled further
// - If the remaining amount is zero, then mark the order as filled
// 2. If the remaining taker order's matchable amount is not zero
// - If the order is limit order, add the remaining order to the orderbook
// - If the order is IOC, cancel the order
func (ob *Orderbook) handleRemainingTakerOrder(remainingTakerOrder *db.OrderTable, perpNetAmount *big.Int, cancelledOrders *[]types.CancelTakerOrderRespData) error {
	if remainingTakerOrder.GetMatchableAmount(perpNetAmount).Sign() == 0 {
		if remainingTakerOrder.RemainingAmount().Sign() != 0 {
			cutils.LogByParty(remainingTakerOrder.Party, "%v - Cancelling partially filled reduce only order\n", remainingTakerOrder.ID)
			*cancelledOrders = append(*cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "Reduce only order couldn't be filled futher because position cannot be reduced further"})
		} else {
			cutils.LogByParty(remainingTakerOrder.Party, "%v - fully filled.", remainingTakerOrder.ID)
		}
	} else {
		cutils.LogByParty(remainingTakerOrder.Party, "%v - Amount req (x18): %v | Total filled (x18): %v\n", remainingTakerOrder.ID, remainingTakerOrder.Amountx18.Val, remainingTakerOrder.TotalFilledx18.Val)
		if remainingTakerOrder.IsLimitOrder() {
			cutils.LogByParty(remainingTakerOrder.Party, "%v - Add remaining limit order to orderbook. Amount remaining: %v\n", remainingTakerOrder.ID, remainingTakerOrder.RemainingAmount())
			remainingTakerOrder.Status = ctypes.ORDER_STATUS_PARTIAL
			errNewTaker := ob.AddOrder(remainingTakerOrder)

			// NOTE: This is critical error. Should not happen
			if errNewTaker != nil {
				xlog.Errorf("%v - Unable to add remaining order to orderbook\n", remainingTakerOrder.ID)
				*cancelledOrders = append(*cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "Unable to add remaining taker order to orderbook"})
				return errNewTaker
			}
		} else {
			cutils.LogByParty(remainingTakerOrder.Party, "%v - Cancelling partially filled IOC %v order\n", remainingTakerOrder.ID, remainingTakerOrder.Type)
			*cancelledOrders = append(*cancelledOrders, types.CancelTakerOrderRespData{ID: remainingTakerOrder.ID, CancelReason: "IOC order couldn't be filled completely"})
		}
	}
	return nil
}
