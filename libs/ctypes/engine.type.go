package ctypes

import "math/big"

// TODO: Move to types

// State represents the structure from the contract
type MarketPositionState struct {
	LongOpenInterest  *big.Int
	ShortOpenInterest *big.Int
}
