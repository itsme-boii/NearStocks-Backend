package ctypes

import "math/big"

type PositionSummary struct {
	SubaccountID string
	MarketID     uint
	TotalAmount  *big.Int
	OpenPrice    string
}
