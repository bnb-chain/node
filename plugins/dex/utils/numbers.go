package utils

import "math/big"

// CalBigNotionalInt64() calculate the multiply value of notional based on price and qty
// both price and qty are in int64 with 1e8 as decimals
// TODO: here the floor divide is used. there may cause small residual.
func CalBigNotionalInt64(price, qty int64) int64 {
	var bi big.Int
	return bi.Div(bi.Mul(big.NewInt(qty), big.NewInt(price)), big.NewInt(1e8)).Int64()
}

func CalBigNotional(price, qty int64) *big.Int {
	var bi big.Int
	return bi.Div(bi.Mul(big.NewInt(qty), big.NewInt(price)), big.NewInt(1e8))
}

// IsExceedMaxNotional return the result that is the product of price and quantity exceeded max notional
func IsExceedMaxNotional(price, qty int64) bool {
	var bi big.Int
	return !bi.Div(bi.Mul(big.NewInt(qty), big.NewInt(price)), big.NewInt(1e8)).IsInt64()
}

