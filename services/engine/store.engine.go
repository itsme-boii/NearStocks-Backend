package engine

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

	"github.com/redis/go-redis/v9"
)

// ---- LOGX_LEVEL_ORDERS:MARKET_ID:PARTY:SIDE:PRICE_LVL ---- SORTED SET-------//
// ---- LOGX_ORDER_DETAILS:ORDER_ID ---- HASH SET ------//
// ---- LOGX_ORDERBOOK_LEVEL:MARKET_ID:PARTY:SIDE ---- SORTED SET-------//
// ---- LOGX_ORDERBOOK_LEVEL_QTY:MARKET_ID:PARTY:SIDE ---- HASH SET -------//

// Add / Increase orderbook level quantity
func (ob *Orderbook) addOrderbookLevel(order *db.OrderTable) error {
	obQtyKey := xredis.GetOrderbookQtyKey(ob.market.ID, order.Party, order.Side)
	priceQuantum := cutils.X18ToQuantum(order.Pricex18.Val, ob.market.PriceToQtmConversionExpo)
	remainingAmountQuantum := cutils.X18ToQuantum(order.RemainingAmount().Val, ob.market.AmtToQtmConversionExpo)

	// xlog.Infof("Add orderbook level (qtm): %v | key: %v | amountQtm: %v\n", priceQuantum, obQtyKey, remainingAmountQuantum)

	errIncr := ob.redisClient.HIncrBy(context.Background(), obQtyKey, fmt.Sprintf("%v", priceQuantum), int64(remainingAmountQuantum)).Err()
	if errIncr != nil {
		xlog.Infof("Unable to increase orderbook level quantity: %v\n", errIncr)
		return errIncr
	}

	obLevelKey := xredis.GetOrderbookKey(ob.market.ID, order.Party, order.Side)
	cutils.LogByParty(order.Party, "Add orderbook level: %v | key: %v | amount (qtm): %v\n", priceQuantum, obLevelKey, remainingAmountQuantum)

	errAdd := ob.redisClient.ZAdd(context.Background(), obLevelKey, redis.Z{
		Score:  float64(priceQuantum),
		Member: priceQuantum,
	}).Err()

	if errAdd != nil {
		xlog.Infof("Unable to add orderbook level: %v\n", errAdd)
		return errAdd
	}
	return nil
}

// Decrease orderbook level quantity
// Remove orderbook level if quantity is zero
func (ob *Orderbook) removeOrderbookLevel(order *db.OrderTable) error {
	obQtyKey := xredis.GetOrderbookQtyKey(ob.market.ID, order.Party, order.Side)
	priceQuantum := cutils.X18ToQuantum(order.Pricex18.Val, ob.market.PriceToQtmConversionExpo)
	remainingAmountQuantum := cutils.X18ToQuantum(order.RemainingAmount().Val, ob.market.AmtToQtmConversionExpo)

	cutils.LogByParty(order.Party, "Remove orderbook level (qtm): %v | key: %v | amountQtm: %v\n", priceQuantum, obQtyKey, remainingAmountQuantum)

	//Single context to pass to all redis commands for consistency
	ctx := context.Background()

	quantity, err := ob.redisClient.HGet(ctx, obQtyKey, fmt.Sprintf("%v", priceQuantum)).Uint64()
	if err == redis.Nil {
		xlog.Infof("Orderbook level quantity not found: %v\n", err)
		return nil
	}
	if err != nil {
		xlog.Infof("Unable to get orderbook level quantity: %v\n", err)
		return err
	}

	if quantity < remainingAmountQuantum {
		xlog.Infof("Order quantity %v is greater than orderbook level quantity: %v\n", remainingAmountQuantum, quantity)
		quantity = remainingAmountQuantum
		//TODO: check why quantitiy < remainingAmountQuantum. remainingAmountQuantum comes from DB, this means we didn't update reduced amount in db last time when matched or cancelled.
	}

	errDecr := ob.redisClient.HIncrBy(ctx, obQtyKey, fmt.Sprintf("%v", priceQuantum), -int64(remainingAmountQuantum)).Err()
	if errDecr != nil {
		xlog.Infof("Unable to decrease orderbook level quantity: %v\n", errDecr)
		return errDecr
	}
	quantity -= remainingAmountQuantum

	// Remove orderbook level if remaining quantity is zero
	if quantity == 0 {
		obKey := xredis.GetOrderbookKey(ob.market.ID, order.Party, order.Side)

		errRem := ob.redisClient.ZRem(ctx, obKey, fmt.Sprintf("%v", priceQuantum)).Err()
		if errRem != nil {
			xlog.Infof("Unable to remove orderbook level: %v\n", errRem)
			return errRem
		}

		// Remove orderbook level quantity
		errDel := ob.redisClient.HDel(ctx, obQtyKey, fmt.Sprintf("%v", priceQuantum)).Err()
		if errDel != nil {
			xlog.Infof("Unable to delete orderbook level quantity: %v\n", errDel)
			return errDel
		}

		cutils.LogByParty(order.Party, "Removed orderbook level (qtm): %v\n", priceQuantum)
	} else {
		cutils.LogByParty(order.Party, "Skipping removeOrderbookLevel... Remaining quantity: %v\n", quantity)
	}
	return nil
}

type OrderbookLevel struct {
	PriceQuantum    uint64
	QuantityQuantum uint64
}

func (ob *Orderbook) getOrderBookLevelSize(party ctypes.Party, side ctypes.OrderSide) (uint32, error) {
	obKey := xredis.GetOrderbookKey(ob.market.ID, party, side)

	size, err := ob.redisClient.ZCard(context.Background(), obKey).Result()
	if err == redis.Nil {
		xlog.Infof("No order size for marketId: %d for side: %s from party:%s ", ob.market.ID, side, party)
		return 0, err
	} else if err != nil {
		xlog.Infof("Unable to get orderbook level size: %v\n", err)
		return 0, err
	}
	return uint32(size), nil
}

// NOTE: Not allowing end as -1 for now
func (ob *Orderbook) getOrderBookLevels(party ctypes.Party, side ctypes.OrderSide, offset uint32, limit uint32) ([]*OrderbookLevel, error) {
	var orderbookLevels []*OrderbookLevel = []*OrderbookLevel{}
	var err error
	var priceQuantums []string

	obKey := xredis.GetOrderbookKey(ob.market.ID, party, side)

	if side == ctypes.ORDER_SIDE_BUY {
		priceQuantums, err = ob.redisClient.ZRevRange(context.Background(), obKey, int64(offset), int64(offset+limit-1)).Result()
	} else if side == ctypes.ORDER_SIDE_SELL {
		priceQuantums, err = ob.redisClient.ZRange(context.Background(), obKey, int64(offset), int64(offset+limit-1)).Result()
	}
	if err == redis.Nil {
		xlog.Infof("No order level for marketId: %d for side: %s from party:%s ", ob.market.ID, side, party)
		return nil, err
	} else if err != nil {
		xlog.Infof("Unable to get orderbook levels: %v\n", err)
		return nil, err
	}
	if len(priceQuantums) == 0 {
		return orderbookLevels, nil
	}

	obQtyKey := xredis.GetOrderbookQtyKey(ob.market.ID, party, side)

	// Get orderbook level quantities
	orderbookLevels = make([]*OrderbookLevel, len(priceQuantums))
	for idx, priceQuantumStr := range priceQuantums {
		priceQuantum, err := strconv.ParseUint(priceQuantumStr, 10, 64)
		if err != nil {
			xlog.Infof("Unable to convert price quantum to uint64: %v\n", err)
			return nil, err
		}

		quantity, err := ob.redisClient.HGet(context.Background(), obQtyKey, priceQuantumStr).Uint64()
		if err == redis.Nil {
			xlog.Warnf("Orderbook level quantity not found for priceQuantum: %v | marketID: %v | side: %v. Returning 0 values", priceQuantum, ob.market.ID, side)
			orderbookLevels[idx] = &OrderbookLevel{
				PriceQuantum:    priceQuantum,
				QuantityQuantum: 0,
			}
		} else if err != nil {
			xlog.Infof("Unable to get orderbook level quantity for marketID: %v, priceQuantum: %v | Error: %v\n", ob.market.ID, priceQuantumStr, err)
			return nil, err
		} else {
			orderbookLevels[idx] = &OrderbookLevel{
				PriceQuantum:    priceQuantum,
				QuantityQuantum: quantity,
			}
		}
	}
	return orderbookLevels, nil
}

// TODO: Optimise on the data we are storing in HSET
func (ob *Orderbook) addOrder(order *db.OrderTable) error {
	// Add order to orderbook level
	priceQuantum := cutils.X18ToQuantum(order.Pricex18.Val, ob.market.PriceToQtmConversionExpo)
	remainingAmountQuantum := cutils.X18ToQuantum(order.RemainingAmount().Val, ob.market.AmtToQtmConversionExpo)
	orderKey := xredis.GetOrdersAtLvlKey(ob.market.ID, order.Party, order.Side, priceQuantum)
	cutils.LogByParty(order.Party, "Add order to level (qtm): %v | key: %v | amountQtm: %v\n", priceQuantum, orderKey, remainingAmountQuantum)

	errAdd := ob.redisClient.ZAdd(context.Background(), orderKey, redis.Z{
		Score:  float64(order.Timestamp),
		Member: order.ID,
	}).Err()
	if errAdd != nil {
		xlog.Infof("Unable to add order: %v\n", errAdd)
		return errAdd
	}

	obDetailsKey := xredis.GetOrderDetailsKey(order.ID)
	cutils.LogByParty(order.Party, "Add order details: %v | key: %v | amount: %v\n", priceQuantum, obDetailsKey, remainingAmountQuantum)

	errHSet := ob.redisClient.HSet(context.Background(), obDetailsKey, order).Err()
	if errHSet != nil {
		xlog.Infof("Unable to add order details: %v\n", errHSet)
		return errHSet
	}
	return nil
}

func (ob *Orderbook) getOrderDetails(orderId uint) (*db.OrderTable, error) {
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
func (ob *Orderbook) GetOrdersForPriceQuantum(party ctypes.Party, side ctypes.OrderSide, priceQuantum uint64) []*db.OrderTable {
	orderKey := xredis.GetOrdersAtLvlKey(ob.market.ID, party, side, priceQuantum)

	values, err := ob.redisClient.ZRange(context.Background(), orderKey, 0, -1).Result()
	if err != nil {
		xlog.Infof("Unable to get orders for party: %v | side: %v | error: %v\n", party, side, err)
		return nil
	}

	// Ideally this should never happen
	if len(values) == 0 {
		return []*db.OrderTable{}
	}

	cutils.LogByParty(party, "Orders for price quantum: %v | party: %v | side: %v | values: %v\n", priceQuantum, party, side, values)

	orders := make([]*db.OrderTable, 0)
	for _, orderIdStr := range values {
		orderId, err := strconv.ParseUint(orderIdStr, 10, 64)
		if err != nil {
			xlog.Errorf("Unable to convert order id: %v to uint64: %v | Delete order id at price quantum: %v for market id: %v ", orderId, err, priceQuantum, ob.market.ID)
			continue
		}
		order, err := ob.getOrderDetails(uint(orderId))
		if err != nil {
			xlog.Errorf("Unable to get order details for id - %v: %v |  Delete order id at price quantum: %v for market id: %v ", orderId, err, priceQuantum, ob.market.ID)
			continue
		}
		orders = append(orders, order)
	}

	return orders
}

// Remove order from orderbook level
// Remove order details
func (ob *Orderbook) removeOrder(order *db.OrderTable) error {
	priceQuantum := cutils.X18ToQuantum(order.Pricex18.Val, ob.market.PriceToQtmConversionExpo)
	orderKey := xredis.GetOrdersAtLvlKey(ob.market.ID, order.Party, order.Side, priceQuantum)
	cutils.LogByParty(order.Party, "Remove order: %v | key: %v\n", order.ID, orderKey)

	numOrdersRemoved, errRem := ob.redisClient.ZRem(context.Background(), orderKey, order.ID).Result()
	if numOrdersRemoved == 0 {
		xlog.Errorf("Order not found in orderbook level: %v | market: %v | err: %v", priceQuantum, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(ob.market.ID)], errRem)
		return fmt.Errorf("order id - %v for market - %v not found in orderbook level ($) - %v", order.ID, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(ob.market.ID)], cutils.X18ToFloatStr(order.Pricex18.Val))
	}

	if errRem != nil {
		xlog.Infof("failed to remove %d orders from redis: %v\n", numOrdersRemoved, errRem)
		return errRem
	}
	cutils.LogByParty(order.Party, "Order Id - %v, Successfully removed %v orders from redis order book", order.ID, numOrdersRemoved)

	orderDetailKey := xredis.GetOrderDetailsKey(order.ID)
	cutils.LogByParty(order.Party, "Remove order details: %v | key: %v\n", order.ID, orderDetailKey)

	numberOfOrderDetailsRemoved, errDel := ob.redisClient.Del(context.Background(), orderDetailKey).Result()
	if numberOfOrderDetailsRemoved == 0 {
		return fmt.Errorf("order details not found for order id - %v and market - %v", order.ID, contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[uint32(ob.market.ID)])
	}

	if errDel != nil {
		xlog.Infof("Unable to delete order details: %v\n", errDel)
		return errDel
	}
	return nil
}

// TODO: Make this operation atomic
// Add order with all operations
func (ob *Orderbook) AddOrder(order *db.OrderTable) error {
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
func (ob *Orderbook) RemoveOrder(order *db.OrderTable) error {
	// Remove order and details from orderbook price level
	errRemOrder := ob.removeOrder(order)
	if errRemOrder != nil {
		return errRemOrder
	}

	// Update orderbook
	errRemOrderbookLevel := ob.removeOrderbookLevel(order)
	if errRemOrderbookLevel != nil {
		return errRemOrderbookLevel
	}

	return nil
}

func (ob *Orderbook) UpdateRedisOrder(oldOrder *db.OrderTable, newOrder *db.OrderTable) error {
	errRemOrder := ob.RemoveOrder(oldOrder)
	if errRemOrder != nil {
		return errRemOrder
	}

	errAddOrder := ob.AddOrder(newOrder)
	if errAddOrder != nil {
		return errAddOrder
	}

	return nil
}

func (ob *Orderbook) GetPriceQuantumsByRank(party ctypes.Party, side ctypes.OrderSide, offset uint32, limit uint32) (*[]uint64, error) {
	cutils.LogByParty(party, "GetPriceQuantumsByRank for party: %v | side: %v | offset: %v | limit: %v\n", party, side, offset, limit)
	var priceQuantumStrs []string
	var err error

	obKey := xredis.GetOrderbookKey(ob.market.ID, party, side)

	if side == ctypes.ORDER_SIDE_SELL {
		priceQuantumStrs, err = ob.redisClient.ZRange(context.Background(), obKey, int64(offset), int64(offset+limit-1)).Result()
	} else if side == ctypes.ORDER_SIDE_BUY {
		priceQuantumStrs, err = ob.redisClient.ZRevRange(context.Background(), obKey, int64(offset), int64(offset+limit-1)).Result()
	}

	if err != nil {
		xlog.Infof("Unable to get orderbook price quantums: %v\n", err)
		return nil, err
	}

	if len(priceQuantumStrs) == 0 {
		return &[]uint64{}, nil
	}

	return utils.ConvertStringSliceToUint64Slice(priceQuantumStrs), nil
}

func (ob *Orderbook) GetMatchingPriceQuantums(party ctypes.Party, side ctypes.OrderSide, pricexRequestx18 *big.Int) (*[]uint64, error) {
	priceQuantum := cutils.X18ToQuantum(pricexRequestx18, ob.market.PriceToQtmConversionExpo)
	xlog.Debugf("GetMatchingPriceQuantums for party: %v | side: %v | priceQtm: %v\n", party, side, priceQuantum)
	var priceQuantumStrs []string
	var err error

	obKey := xredis.GetOrderbookKey(ob.market.ID, party, side)

	if side == ctypes.ORDER_SIDE_BUY {
		// Get all price levels greater than or equal to the requested price | descending order
		priceQuantumStrs, err = ob.redisClient.ZRevRangeByScore(context.Background(), obKey, &redis.ZRangeBy{
			Min: fmt.Sprintf("%v", priceQuantum),
			Max: "+inf",
		}).Result()
	} else if side == ctypes.ORDER_SIDE_SELL {
		// Get all price levels from 0 (not included) to requested price | ascending order
		priceQuantumStrs, err = ob.redisClient.ZRangeByScore(context.Background(), obKey, &redis.ZRangeBy{
			Min: "(0",
			Max: fmt.Sprintf("%v", priceQuantum),
		}).Result()
	}

	if err != nil {
		xlog.Infof("Unable to get orderbook price quantums: %v\n", err)
		return nil, err
	}

	if len(priceQuantumStrs) == 0 {
		return &[]uint64{}, nil
	}

	return utils.ConvertStringSliceToUint64Slice(priceQuantumStrs), nil
}

type FullOrderbook struct {
	Party     ctypes.Party      `json:"-"`
	Timestamp uint64            `json:"timestamp"`
	MarketId  uint              `json:"marketId"`
	Asks      []*OrderbookLevel `json:"asks"`
	Bids      []*OrderbookLevel `json:"bids"`
}

// TODO: Remove crossing data if needed
// Currently only fetching first 10 levels
func (ob *Orderbook) GetFullOrderBookLevels(party ctypes.Party, depth uint32) (*FullOrderbook, error) {
	asks, err := ob.getOrderBookLevels(party, ctypes.ORDER_SIDE_SELL, 0, depth)
	if err != nil {
		return nil, err
	}

	bids, err := ob.getOrderBookLevels(party, ctypes.ORDER_SIDE_BUY, 0, depth)
	if err != nil {
		return nil, err
	}

	return &FullOrderbook{
		MarketId:  ob.market.ID,
		Timestamp: cutils.TimestampMilliNow(),
		Party:     party,
		Asks:      asks,
		Bids:      bids,
	}, nil
}

// Get first 10 price levels with quantities for the given side
// Calculate the price * quantity for each level
// Calculate the total notional for the first 10 levels
// If the total notional value is less than the impact notional, get the next 20 (2x) levels
// Repeat the above steps until the total notional value is greater than the impact notional
func (ob *Orderbook) getImpactPricex18(party ctypes.Party, side ctypes.OrderSide, impactNotionalx18 *big.Int) (*big.Int, error) {

	levelSize, err := ob.getOrderBookLevelSize(party, side)
	if err != nil {
		return nil, err
	} else if levelSize == 0 {
		return cutils.GetBig0(), fmt.Errorf("Orderbook level size is zero")
	}

	// Fetch 50 levels at a time. Then fetch exponentially increasing levels until impact notional is reached
	fetchLimit := uint32(50)
	totalNotionalx36 := cutils.GetBig0()
	totalQtyx18 := cutils.GetBig0()
	impactNotionalx36 := cutils.Mulx18(impactNotionalx18)

	for offset := uint32(0); offset < levelSize; offset += fetchLimit {
		levels, err := ob.getOrderBookLevels(party, side, offset, fetchLimit)
		if err != nil {
			return nil, err
		}

		for _, level := range levels {
			curPricex18 := cutils.QuantumToX18(level.PriceQuantum, ob.market.PriceToQtmConversionExpo)
			curQtyx18 := cutils.QuantumToX18(level.QuantityQuantum, ob.market.AmtToQtmConversionExpo)
			curNotionalx36 := new(big.Int).Mul(curPricex18, curQtyx18)
			// Total Notional + Current Notion >= Impact notional
			if (new(big.Int).Add(totalNotionalx36, curNotionalx36)).Cmp(impactNotionalx36) >= 0 {
				// Notional left = Impact Notional - Total Notional
				notionalLeftx36 := new(big.Int).Sub(impactNotionalx36, totalNotionalx36)
				// Qty required = Notional left / Current Price
				qtyReqx18 := new(big.Int).Div(notionalLeftx36, curPricex18)
				// Total notional becomes equal to impactNotional
				totalNotionalx36 = new(big.Int).Set(impactNotionalx36)
				// Increase the total quantity
				totalQtyx18 = new(big.Int).Add(totalQtyx18, qtyReqx18)
				break
			} else {
				// Add current notional and quantity to total
				totalNotionalx36 = new(big.Int).Add(totalNotionalx36, curNotionalx36)
				totalQtyx18 = new(big.Int).Add(totalQtyx18, curQtyx18)
			}
		}
		fetchLimit *= 2
	}
	// Impact price = Total Notional / Total Qty
	if totalQtyx18.Cmp(cutils.GetBig0()) == 0 {
		xlog.Errorf("Total quantity is zero. Level Size: %+v | side: %v", levelSize, side)
	}

	impactPricex18 := new(big.Int).Div(totalNotionalx36, totalQtyx18)
	return impactPricex18, nil
}

type ImpactPrices struct {
	Bid float64 `json:"bid"`
	Ask float64 `json:"ask"`
}

// Get ask and bid impact prices
// Errors are only logged and not returned to the caller
func (ob *Orderbook) GetImpactPrices(party ctypes.Party, impactNotionalStr string) *ImpactPrices {
	impactNotionalx18 := cutils.FloatStrToX18(impactNotionalStr)
	bidImpactPricex18, errBid := ob.getImpactPricex18(party, ctypes.ORDER_SIDE_BUY, impactNotionalx18)
	askImpactPricex18, errAsk := ob.getImpactPricex18(party, ctypes.ORDER_SIDE_SELL, impactNotionalx18)
	if errBid != nil {
		xlog.Warnf(errBid.Error())
	}
	if errAsk != nil {
		xlog.Warnf(errAsk.Error())
	}

	bigImpactPriceStr := cutils.X18ToFloatStr(bidImpactPricex18)
	askImpactPriceStr := cutils.X18ToFloatStr(askImpactPricex18)

	bidImpactPrice, _ := strconv.ParseFloat(bigImpactPriceStr, 64)
	askImpactPrice, _ := strconv.ParseFloat(askImpactPriceStr, 64)

	return &ImpactPrices{
		Bid: bidImpactPrice,
		Ask: askImpactPrice,
	}
}
