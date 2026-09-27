package ctypes

import "math/big"

type LockerEntity string

const (
	LOCKER_ENTITY_ORDER    LockerEntity = "ORDER"
	LOCKER_ENTITY_FUNDING  LockerEntity = "FUNDING"
	LOCKER_ENTITY_WITHDRAW LockerEntity = "WITHDRAW"
)

type OraclePrice struct {
	Pricex18 *big.Int
	Symbol   string
}

type SpotBalance struct {
	ProductId       uint32 `json:"productId"`
	TokenBalance    string `json:"tokenBalance"`
	WithdrawBalance string `json:"withdrawBalance"`
}

