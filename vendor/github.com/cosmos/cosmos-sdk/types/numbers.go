package types

import (
	"errors"
	"math/big"
)

const (
	ErrZeroDividend = "Dividend is zero "
	ErrIntOverflow  = "Int Overflow "
)

func Mul64(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	if (c < 0) == ((a < 0) != (b < 0)) {
		if c/b == a {
			return c, true
		}
	}
	return c, false
}

func MulQuoDec(a, b, c Dec) (Dec, error) {
	if c.IsZero() {
		return Dec{}, errors.New(ErrZeroDividend)
	}
	r, ok := Mul64(a.RawInt(), b.RawInt())
	if !ok {
		var bi big.Int
		bi.Quo(bi.Mul(big.NewInt(a.RawInt()), big.NewInt(b.RawInt())), big.NewInt(c.RawInt()))
		if !bi.IsInt64() {
			return Dec{}, errors.New(ErrIntOverflow)
		}
		return NewDec(bi.Int64()), nil
	}
	return NewDec(r / c.RawInt()), nil
}

func MulBigInt(a, b *big.Int) *big.Int {
	if a == nil || b == nil {
		panic("arguments can not be nil")
	}
	var bi big.Int
	bi.Mul(a, b)
	return &bi
}

func QuoBigInt(a, b *big.Int) *big.Int {
	if a == nil || b == nil {
		panic("arguments can not be nil")
	}
	var bi big.Int
	bi.Quo(a, b)
	return &bi
}
