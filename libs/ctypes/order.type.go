// NOTE: DO NOT USE THIS FILE. IT WAS ADDED IN BEGINNING FOR TESTING PURPOSES ONLY.
package ctypes

type OrderType string
type OrderSide string
type OrderStatus string
type Party string
type Liquidity string
type FillType string
type TriggerCondition string
type TriggerDirection string

type CreateOrderBody struct {
	MarketId         *uint            `json:"marketId" binding:"required"`
	IsBuy            *bool            `json:"isBuy" binding:"required"`
	OrderType        OrderType        `json:"orderType" binding:"required,oneof=LIMIT MARKET"`
	AmountStr        string           `json:"amount" binding:"required"`
	PriceStr         string           `json:"price" binding:"required"`
	ExpiryTs         *uint64          `json:"expiryTs" binding:"required"`
	Party            Party            `json:"party" binding:"oneof=TRADER SOLVER"` // DEFAULT is TRADER
	IsReduce         *bool            `json:"isReduce" binding:"required"`
	TriggerPriceStr  string           `json:"triggerPrice" default:"0"`
	TriggerCondition TriggerCondition `json:"triggerCondition"`
}

const (
	// SIDE_BUY is used to represent a BUY order.
	ORDER_SIDE_BUY OrderSide = "BUY"
	// SIDE_SELL is used to represent a SELL order.
	ORDER_SIDE_SELL OrderSide = "SELL"

	//----------------------------------------------------------------
	ORDER_TYPE_LIMIT       OrderType = "LIMIT"
	ORDER_TYPE_MARKET      OrderType = "MARKET"
	ORDER_TYPE_LIQUIDATION OrderType = "LIQUIDATION"

	//----------------------------------------------------------------
	ORDER_STATUS_OPEN      OrderStatus = "OPEN"
	ORDER_STATUS_PARTIAL   OrderStatus = "PARTIAL"
	ORDER_STATUS_FILLED    OrderStatus = "FILLED"
	ORDER_STATUS_CANCELLED OrderStatus = "CANCELLED"

	PARTY_TRADER Party = "TRADER"
	PARTY_SOLVER Party = "SOLVER"

	LIQUIDITY_TAKER Liquidity = "TAKER"
	LIQUIDITY_MAKER Liquidity = "MAKER"

	FILL_LIMIT       FillType = "LIMIT"
	FILL_MARKET      FillType = "MARKET"
	FILL_LIQUIDATION FillType = "LIQUIDATION"

	TAKE_PROFIT TriggerCondition = "TP"
	STOP_LOSS   TriggerCondition = "SL"

	TRIGGER_DIRECTION_INCR TriggerDirection = "INCR"
	TRIGGER_DIRECTION_DECR TriggerDirection = "DECR"
)

// Validate order fields type implements RedisField
var _ RedisField = (*Party)(nil)
var _ RedisField = (*OrderSide)(nil)
var _ RedisField = (*OrderStatus)(nil)
var _ RedisField = (*OrderType)(nil)
var _ RedisField = (*TriggerCondition)(nil)

func (os *OrderSide) Opposite() OrderSide {
	if *os == ORDER_SIDE_BUY {
		return ORDER_SIDE_SELL
	} else if *os == ORDER_SIDE_SELL {
		return ORDER_SIDE_BUY
	}
	panic("Invalid OrderSide")
}

func NewOrderSide(isBuy bool) OrderSide {
	if isBuy {
		return ORDER_SIDE_BUY
	} else {
		return ORDER_SIDE_SELL
	}
}

func (p *Party) Opposite() Party {
	if *p == PARTY_TRADER {
		return PARTY_SOLVER
	} else if *p == PARTY_SOLVER {
		return PARTY_TRADER
	}
	panic("Invalid Party")
}

//------------START OF Marshal and Unmarshal functions for the types------------

func (p *Party) UnmarshalBinary(data []byte) error {
	// Convert string to bytes
	*p = Party(string(data))
	return nil
}

func (p Party) MarshalBinary() (data []byte, err error) {
	// Convert bytes to string
	return []byte(p), nil
}

func (os *OrderSide) UnmarshalBinary(data []byte) error {
	// Convert string to bytes
	*os = OrderSide(string(data))
	return nil
}

func (os OrderSide) MarshalBinary() (data []byte, err error) {
	// Convert bytes to string
	return []byte(os), nil
}

func (os *OrderStatus) UnmarshalBinary(data []byte) error {
	// Convert string to bytes
	*os = OrderStatus(string(data))
	return nil
}

func (os OrderStatus) MarshalBinary() (data []byte, err error) {
	// Convert bytes to string
	return []byte(os), nil
}

func (ot *OrderType) UnmarshalBinary(data []byte) error {
	// Convert string to bytes
	*ot = OrderType(string(data))
	return nil
}

func (ot OrderType) MarshalBinary() (data []byte, err error) {
	// Convert bytes to string
	return []byte(ot), nil
}

func (tc *TriggerCondition) UnmarshalBinary(data []byte) error {
	// Convert string to bytes
	*tc = TriggerCondition(string(data))
	return nil
}

func (tc TriggerCondition) MarshalBinary() (data []byte, err error) {
	// Convert bytes to string
	return []byte(tc), nil
}

//------------END OF Marshal and Unmarshal functions for the types------------
