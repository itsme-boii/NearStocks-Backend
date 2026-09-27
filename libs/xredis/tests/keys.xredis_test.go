package tests

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderbookKey(t *testing.T) {
	marketId := uint(1)
	party := ctypes.PARTY_SOLVER
	side := ctypes.ORDER_SIDE_BUY
	expected := "LOGX_ORDERBOOK_LEVEL:1:SOLVER:BUY"
	assert.Equal(t, expected, xredis.GetOrderbookKey(marketId, party, side))
}

func TestOrderbookQtyKey(t *testing.T) {
	marketId := uint(1)
	party := ctypes.PARTY_SOLVER
	side := ctypes.ORDER_SIDE_BUY
	expected := "LOGX_ORDERBOOK_LEVEL_QTY:1:SOLVER:BUY"
	assert.Equal(t, expected, xredis.GetOrderbookQtyKey(marketId, party, side))
}

func TestOrderKey(t *testing.T) {
	marketId := uint(1)
	party := ctypes.PARTY_SOLVER
	side := ctypes.ORDER_SIDE_BUY
	priceQuantum := uint64(100)
	expected := "LOGX_LEVEL_ORDERS:1:SOLVER:BUY:100"
	assert.Equal(t, expected, xredis.GetOrdersAtLvlKey(marketId, party, side, priceQuantum))
}

func TestOrderDetailsKey(t *testing.T) {
	orderId := uint(1)
	expected := "LOGX_ORDER_DETAILS:1"
	assert.Equal(t, expected, xredis.GetOrderDetailsKey(orderId))
}

func TestBalanceKey(t *testing.T) {
	subaccountID := "0x0000000000015631e1e23aeEFae435106017BBcee2b91676E43B00000000000"
	expected := "v0_balance_0x0000000000015631e1e23aeEFae435106017BBcee2b91676E43B00000000000"
	assert.Equal(t, expected, xredis.GetBalanceKey(subaccountID))
}

func TestBalanceField(t *testing.T) {
	productId := uint32(1)
	expected := "1"
	assert.Equal(t, expected, xredis.GetBalanceField(productId))
}

func TestFundingRateKey(t *testing.T) {
	symbol := "ETH"
	expected := "ETH_funding_rate"
	assert.Equal(t, expected, xredis.GetFundingRateKey(symbol))
}

func TestCumulativeFundingRateKey(t *testing.T) {
	symbol := "ETH"
	expected := "ETH-USD_cumulative_funding_rate"
	assert.Equal(t, expected, xredis.GetCumulativeFundingRateKey(symbol))
}

func TestBalanceLockerKey(t *testing.T) {
	subaccountID := "0x0000000000015631e1e23aeEFae435106017BBcee2b91676E43B00000000000"
	expected := "LOCK_BALANCE_0x0000000000015631e1e23aeEFae435106017BBcee2b91676E43B00000000000"
	assert.Equal(t, expected, xredis.GetBalanceLockerKey(subaccountID))
}

func TestBalanceLockerField(t *testing.T) {
	entity := ctypes.LOCKER_ENTITY_ORDER
	entityId := "1"
	expected := "ORDER_1"
	assert.Equal(t, expected, xredis.GetBalanceLockerField(entity, entityId))
}

func TestBalanceLockerValue(t *testing.T) {
	productId := uint32(1)
	amountStr := "100"
	expected := "1_100"
	assert.Equal(t, expected, xredis.GetBalanceLockerValue(productId, amountStr))
}

func TestOIKey(t *testing.T) {
	expected := "NETWORK_OI"
	assert.Equal(t, expected, xredis.GetOIKey())
}
