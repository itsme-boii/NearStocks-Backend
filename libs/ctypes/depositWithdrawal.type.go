package ctypes

import (
	"math/big"
)

type NetDepositResult struct {
	SubaccountId string
	NetSum       *big.Int
}