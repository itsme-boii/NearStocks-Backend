package conditional

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/engine/utils"
	"math/big"
	"strconv"
	"sync"

	"github.com/redis/go-redis/v9"
)

type TriggerOrderbook struct {
	market      *db.MarketTable
	redisClient *redis.Client
	mutex       *sync.Mutex
}

func NewTriggerOrderbook(_market *db.MarketTable) *TriggerOrderbook {
	return &TriggerOrderbook{
		market:      _market,
		redisClient: xredis.GetRedisClient(),
		mutex:       &sync.Mutex{},
	}
}

// Lock
func (tob *TriggerOrderbook) Lock() {
	tob.mutex.Lock()
}

// Unlock
func (tob *TriggerOrderbook) Unlock() {
	tob.mutex.Unlock()
}

// Add / Increase orderbook level quantity
func (tob *TriggerOrderbook) addOrderbookLevel(order *db.OrderTable) error {
	obQtyKey := xredis.GetTriggerOrderbookQtyKey(tob.market.ID, order.Party, order.GetTriggerDirection())
	triggerPriceQuantum := cutils.X18ToQuantum(order.TriggerPricex18.Val, tob.market.PriceToQtmConversionExpo)
	remainingAmountQuantum := cutils.X18ToQuantum(order.RemainingAmount().Val, tob.market.AmtToQtmConversionExpo)

	xlog.Infof("Add trigger orderbook level (qtm): %v | key: %v | amountQtm: %v\n", triggerPriceQuantum, obQtyKey, remainingAmountQuantum)

	errIncr := tob.redisClient.HIncrBy(context.Background(), obQtyKey, xredis.GetTriggerOrderbookQtyField(triggerPriceQuantum), int64(remainingAmountQuantum)).Err()
	if errIncr != nil {
		xlog.Infof("Unable to increase trigger orderbook level quantity: %v\n", errIncr)
		return errIncr
	}

	obLevelKey := xredis.GetTriggerOrderbookKey(tob.market.ID, order.Party, order.GetTriggerDirection())
	xlog.Infof("Add trigger orderbook level: %v | key: %v | amount (qtm): %v\n", triggerPriceQuantum, obLevelKey, remainingAmountQuantum)

	errAdd := tob.redisClient.ZAdd(context.Background(), obLevelKey, redis.Z{
		Score:  float64(triggerPriceQuantum),
		Member: triggerPriceQuantum,
	}).Err()

	if errAdd != nil {
		xlog.Infof("Unable to add trigger orderbook level: %v\n", errAdd)
		return errAdd
	}
	return nil
}

// Decrease orderbook level quantity
// Remove orderbook level if quantity is zero
func (tob *TriggerOrderbook) removeOrderbookLevel(order *db.OrderTable) error {
	obQtyKey := xredis.GetTriggerOrderbookQtyKey(tob.market.ID, order.Party, order.GetTriggerDirection())
	triggerPriceQuantum := cutils.X18ToQuantum(order.TriggerPricex18.Val, tob.market.PriceToQtmConversionExpo)
	remainingAmountQuantum := cutils.X18ToQuantum(order.RemainingAmount().Val, tob.market.AmtToQtmConversionExpo)
	obQtyField := xredis.GetTriggerOrderbookQtyField(triggerPriceQuantum)

	xlog.Infof("Remove trigger orderbook level (qtm): %v | key: %v | amountQtm: %v\n", triggerPriceQuantum, obQtyKey, remainingAmountQuantum)

	//Single context to pass to all redis commands for consistency
	ctx := context.Background()

	quantity, err := tob.redisClient.HGet(ctx, obQtyKey, obQtyField).Uint64()
	if err == redis.Nil {
		xlog.Infof("Trigger Orderbook level quantity not found: %v\n", err)
		return nil
	}
	if err != nil {
		xlog.Infof("Unable to get trigger orderbook level quantity: %v\n", err)
		return err
	}

	if quantity < remainingAmountQuantum {
		xlog.Infof("ALERT ALERT: Trigger Order quantity %v is greater than orderbook level quantity: %v\n", remainingAmountQuantum, quantity)
		quantity = remainingAmountQuantum
	}

	errDecr := tob.redisClient.HIncrBy(ctx, obQtyKey, obQtyField, -int64(remainingAmountQuantum)).Err()
	if errDecr != nil {
		xlog.Infof("Unable to decrease orderbook level quantity: %v\n", errDecr)
		return errDecr
	}
	quantity -= remainingAmountQuantum

	// Remove orderbook level if remaining quantity is zero
	if quantity == 0 {
		obKey := xredis.GetTriggerOrderbookKey(tob.market.ID, order.Party, order.GetTriggerDirection())
		errRem := tob.redisClient.ZRem(ctx, obKey, obQtyField).Err()
		if errRem != nil {
			xlog.Infof("Unable to remove orderbook level: %v\n", errRem)
			return errRem
		}

		// Remove orderbook level quantity
		errDel := tob.redisClient.HDel(ctx, obQtyKey, obQtyField).Err()
		if errDel != nil {
			xlog.Infof("Unable to delete orderbook level quantity: %v\n", errDel)
			return errDel
		}

		xlog.Infof("Removed trigger orderbook level (qtm): %v\n", triggerPriceQuantum)
	} else {
		xlog.Infof("Skipping trigger removeOrderbookLevel... Remaining quantity: %v\n", quantity)
	}
	return nil
}

// Get all INCR direction trigger orders below the priceQuantum and all DECR direction trigger orders above the priceQuantum
func (ob *TriggerOrderbook) addOrder(order *db.OrderTable) error {
	triggerPriceQuantum := cutils.X18ToQuantum(order.TriggerPricex18.Val, ob.market.PriceToQtmConversionExpo)
	remainingAmountQuantum := cutils.X18ToQuantum(order.RemainingAmount().Val, ob.market.AmtToQtmConversionExpo)
	orderKey := xredis.GetTriggerOrdersAtLvlKey(ob.market.ID, order.Party, order.GetTriggerDirection(), triggerPriceQuantum)
	xlog.Infof("Add order to level (qtm): %v | key: %v | amountQtm: %v\n", triggerPriceQuantum, orderKey, remainingAmountQuantum)

	errAdd := ob.redisClient.ZAdd(context.Background(), orderKey, redis.Z{
		Score:  float64(order.Timestamp),
		Member: order.ID,
	}).Err()
	if errAdd != nil {
		xlog.Infof("Unable to add order: %v\n", errAdd)
		return errAdd
	}

	obDetailsKey := xredis.GetOrderDetailsKey(order.ID)
	xlog.Infof("Add order details: %v | key: %v | amount: %v\n", triggerPriceQuantum, obDetailsKey, remainingAmountQuantum)

	errHSet := ob.redisClient.HSet(context.Background(), obDetailsKey, order).Err()
	if errHSet != nil {
		xlog.Infof("Unable to add order details: %v\n", errHSet)
		return errHSet
	}
	return nil
}

// TODO: Move this to a common place
func (ob *TriggerOrderbook) getOrderDetails(orderId uint) (*db.OrderTable, error) {
	key := xredis.GetOrderDetailsKey(orderId)

	orderDetails, err := ob.redisClient.HGetAll(context.Background(), key).Result()
	if len(orderDetails) == 0 {
		xlog.Errorf("No order details found for orderId: %v\n", orderId)
		return nil, fmt.Errorf("no order details found for orderId: %v", orderId)
	} else if err != nil {
		xlog.Errorf("Unable to get order details: %v\n", err)
		return nil, err
	}

	var order db.OrderTable
	order.ID = orderId
	err = db.StringMapToOrderTable(orderDetails, &order)
	return &order, err
}

// NOTE: Currently this will return all orders for the given priceQuantum
// 1. Get first order by rank for the given priceQuantum
// 2. Get order details for the order
func (ob *TriggerOrderbook) GetOrdersForPriceQuantum(party ctypes.Party, direction ctypes.TriggerDirection, triggerPriceQuantum uint64) []*db.OrderTable {
	orderKey := xredis.GetTriggerOrdersAtLvlKey(ob.market.ID, party, direction, triggerPriceQuantum)

	values, err := ob.redisClient.ZRange(context.Background(), orderKey, 0, -1).Result()
	if err != nil {
		xlog.Infof("Unable to get trigger orders for party: %v | direction: %v | error: %v\n", party, direction, err)
		return nil
	}

	// Ideally this should never happen
	if len(values) == 0 {
		return []*db.OrderTable{}
	}

	xlog.Infof("Trigger Orders for trigger price quantum: %v | party: %v | direction: %v | values: %v\n", triggerPriceQuantum, party, direction, values)

	// Initialize with empty slice, NOT pre-allocated length
	orders := make([]*db.OrderTable, 0, len(values))
	for _, orderIdStr := range values {
		orderId, err := strconv.ParseUint(orderIdStr, 10, 64)
		if err != nil {
			xlog.Errorf("Unable to convert order id: %v to uint64: %v | Delete order id at trigger price quantum: %v for market id: %v ", orderIdStr, err, triggerPriceQuantum, ob.market.ID)
			continue
		}
		order, err := ob.getOrderDetails(uint(orderId))
		if err != nil {
			xlog.Errorf("Unable to get order details for id - %v: %v |  Delete order id at trigger price quantum: %v for market id: %v ", orderId, err, triggerPriceQuantum, ob.market.ID)
			continue
		}
		orders = append(orders, order)
	}

	return orders
}

// Remove order from orderbook level
// Remove order details
func (ob *TriggerOrderbook) removeOrder(order *db.OrderTable) error {
	if order == nil {
		xlog.Errorf("Cannot remove nil order from orderbook")
		return fmt.Errorf("cannot remove nil order from orderbook")
	}

	triggerPriceQuantum := cutils.X18ToQuantum(order.TriggerPricex18.Val, ob.market.PriceToQtmConversionExpo)
	orderKey := xredis.GetTriggerOrdersAtLvlKey(ob.market.ID, order.Party, order.GetTriggerDirection(), triggerPriceQuantum)
	xlog.Infof("Remove trigger order: %v | key: %v\n", order.ID, orderKey)

	numOrdersRemoved, errRem := ob.redisClient.ZRem(context.Background(), orderKey, order.ID).Result()
	if numOrdersRemoved == 0 {
		xlog.Errorf("Trigger Order not found in orderbook level: %v | market: %v | err: %v", triggerPriceQuantum, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(ob.market.ID)], errRem)
		return fmt.Errorf("order id - %v for market - %v not found in orderbook level ($) - %v", order.ID, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(ob.market.ID)], cutils.X18ToFloatStr(order.Pricex18.Val))
	}
	if errRem != nil {
		xlog.Infof("failed to remove %d orders from redis: %v\n", numOrdersRemoved, errRem)
		return errRem
	}
	xlog.Infof("Trigger Order Id - %v, Successfully removed %v orders from redis order book", order.ID, numOrdersRemoved)

	orderDetailKey := xredis.GetOrderDetailsKey(order.ID)

	xlog.Infof("Remove trigger order details: %v | key: %v\n", order.ID, orderDetailKey)
	numberOfOrderDetailsRemoved, errDel := ob.redisClient.Del(context.Background(), orderDetailKey).Result()
	if numberOfOrderDetailsRemoved == 0 {
		return fmt.Errorf("order details not found for order id - %v and market - %v", order.ID, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(ob.market.ID)])
	}

	if errDel != nil {
		xlog.Infof("Unable to delete trigger order details: %v\n", errDel)
		return errDel
	}
	return nil
}

// TODO: Make this operation atomic
// Add order with all operations
func (ob *TriggerOrderbook) AddOrder(order *db.OrderTable) error {
	// Order Id cannot be zero
	if order.ID == 0 {
		orderJson, _ := json.Marshal(order)
		xlog.Errorf("Order Id cannot be zero for order: %v", string(orderJson))
		return fmt.Errorf("order Id cannot be zero for order")
	}

	// Add order price level and add order details
	errAddOrder := ob.addOrder(order)
	if errAddOrder != nil {
		return errAddOrder
	}

	// Update orderbook
	errAddOrderbookLevel := ob.addOrderbookLevel(order)
	if errAddOrderbookLevel != nil {
		return errAddOrderbookLevel
	}

	return nil
}

// TODO: Make this operation atomic
// Add order with all operations
func (ob *TriggerOrderbook) RemoveOrder(order *db.OrderTable) error {
	// Check if order is nil
	if order == nil {
		xlog.Errorf("Cannot remove nil order from orderbook")
		return fmt.Errorf("cannot remove nil order from orderbook")
	}

	// Remove order and details from orderbook price level
	errRemOrder := ob.removeOrder(order)
	if errRemOrder != nil {
		return errRemOrder
	}

	// Update orderbook
	errRemOrderbookLevel := ob.removeOrderbookLevel(order)
	if errRemOrderbookLevel != nil {
		xlog.Debugf("Error while removing orderbook level: %v\n", errRemOrderbookLevel)
		return errRemOrderbookLevel
	}

	return nil
}

// Returns all orders in sorted order of priceQuantum | INCR direction orders will be sorted in ascending order of priceQuantum and DECR direction orders will be sorted in descending order of priceQuantum
func (ob *TriggerOrderbook) GetEligiblePriceQuantums(party ctypes.Party, direction ctypes.TriggerDirection, oraclePricex18 *big.Int) (*[]uint64, error) {
	triggerPriceQuantum := cutils.X18ToQuantum(oraclePricex18, ob.market.PriceToQtmConversionExpo)
	var priceQuantumStrs []string
	var err error

	// Get all price levels greater than or equal to the requested price | descending order -> DECR direction
	cutils.LogByParty(party, "GetEligibleTriggerPriceQuantums for party: %v | direction: DECR | triggerPriceQtm: %v\n", party, triggerPriceQuantum)
	obKey := xredis.GetTriggerOrderbookKey(ob.market.ID, party, direction)
	obField := xredis.GetTriggerOrderbookQtyField(triggerPriceQuantum)

	if direction == ctypes.TRIGGER_DIRECTION_DECR {
		priceQuantumStrs, err = ob.redisClient.ZRevRangeByScore(context.Background(), obKey, &redis.ZRangeBy{
			Min: obField,
			Max: "+inf",
		}).Result()
	} else {
		// Get all price levels lower than or equal to the requested price | increasing order -> Incr direction
		cutils.LogByParty(party, "GetEligibleTriggerPriceQuantums for party: %v | direction: %v | triggerPriceQtm: %v\n", party, direction, triggerPriceQuantum)
		priceQuantumStrs, err = ob.redisClient.ZRangeByScore(context.Background(), obKey, &redis.ZRangeBy{
			Min: "(0",
			Max: obField,
		}).Result()
	}

	if err != nil {
		xlog.Infof("Unable to get trigger orderbook price quantums for key:%v : %v\n", obKey, err)
		return nil, err
	}

	if len(priceQuantumStrs) == 0 {
		return &[]uint64{}, nil
	}

	return utils.ConvertStringSliceToUint64Slice(priceQuantumStrs), nil
}

// Returns all orders in sorted order of priceQuantum | INCR direction orders will be sorted in ascending order of priceQuantum and DECR direction orders will be sorted in descending order of priceQuantum
func (ob *TriggerOrderbook) GetEligibleOrders(party ctypes.Party, direction ctypes.TriggerDirection, oraclePricex18 *big.Int) ([]*db.OrderTable, error) {
	priceQuantums, err := ob.GetEligiblePriceQuantums(party, direction, oraclePricex18)
	if err != nil {
		return nil, err
	}

	var orders []*db.OrderTable
	for _, priceQuantum := range *priceQuantums {
		ordersForPriceQuantum := ob.GetOrdersForPriceQuantum(party, direction, priceQuantum)
		orders = append(orders, ordersForPriceQuantum...)
	}

	return orders, nil
}
