package cutils

import "math/big"

func GetBig0() *big.Int {
	return big.NewInt(0)
}

func GetBigx36() *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
}

func GetBigx18() *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
}

func GetBigx6() *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
}

func GetBigx12() *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)
}

func GetBigxCust(exp uint32) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}
