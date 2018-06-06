package types

import (
	"fmt"
)

type Token struct {
	Name     string `json:"Name"`
	Symbol   string `json:"Symbol"`
	Supply   Number `json:"Supply"`
	Decimals Number `json:"Decimals"`
}

func (token Token) String() string {
	return fmt.Sprintf("{Name: %v, Symbol: %v, Supply: %v, Decimals: %v}", token.Name, token.Symbol, token.Supply, token.Decimals)
}
