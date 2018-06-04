package types

import (
	"fmt"
	"math/big"
)

type Token struct {
	Name     string   `json:"Name"`
	Symbol   string   `json:"Symbol"`
	Supply   *big.Int `json:"Supply"`
	Decimals *big.Int `json:"Decimals"`
}

func (token Token) String() string {
	return fmt.Sprintf("{Name: %v, Symbol: %v, Supply: %v, Decimals: %v}", token.Name, token.Symbol, token.Supply, token.Decimals)
}
