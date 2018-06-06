package types

import (
	"math/big"
)

type Number struct {
	Value string `json:"value"`
}

func NewNumber(n *big.Int) Number {
	return Number{n.String()}
}

func (num *Number) ToBigInt() *big.Int {
	res, _ := new(big.Int).SetString(num.Value, 10)
	return res
}

func(num *Number) String() string {
	return num.Value
}
