package utils

import (
	"math/big"

	"github.com/binance-chain/node/common/utils"
)

// CalBigNotionalInt64() calculate the multiply value of notional based on price and qty
// both price and qty are in int64 with 1e8 as decimals
// TODO: here the floor divide is used. there may cause small residual.
func CalBigNotionalInt64(price, qty int64) int64 {
	res, ok := utils.Mul64(price, qty)
	if ok {
		// short cut
		return res / 1e8
	}

	var bi big.Int
	return bi.Div(bi.Mul(big.NewInt(qty), big.NewInt(price)), big.NewInt(1e8)).Int64()
}

func CalBigNotional(price, qty int64) *big.Int {
	var bi big.Int
	return bi.Div(bi.Mul(big.NewInt(qty), big.NewInt(price)), big.NewInt(1e8))
}

// IsExceedMaxNotional return the result that is the product of price and quantity exceeded max notional
func IsExceedMaxNotional(price, qty int64) bool {
	// The four short-cuts can cover most of the cases.
	if price <= 1e8 || qty <= 1e8 {
		return false
	}
	if _, ok := utils.Mul64(price, qty); ok {
		return false
	}
	if _, ok := utils.Mul64(price, qty/1e8); !ok {
		return true
	}
	if _, ok := utils.Mul64(price/1e8, qty); !ok {
		return true
	}

	var bi big.Int
	return !bi.Div(bi.Mul(big.NewInt(qty), big.NewInt(price)), big.NewInt(1e8)).IsInt64()
}

// min notional is 1, so we need to ensure price * qty / 1e8 >= 1
func IsUnderMinNotional(price, qty int64) bool {
	if p, ok := utils.Mul64(price, qty); !ok {
		return false
	} else {
		return p < 1e8
	}
}
