package types

import (
	"errors"
	"fmt"

	"github.com/BiJie/BinanceChain/common/utils"
)

func ValidateSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("token symbol cannot be empty")
	}

	if !utils.IsAlphaNum(symbol) {
		return errors.New("token symbol should be alphanumeric")
	}

	return nil
}

// TODO: to make the size of block header fixed and predictable, we may need change to type of "Supply" and "Decimal"
// and we should decide the range of the two variables.
type Token struct {
	Name    string `json:"Name"`
	Symbol  string `json:"Symbol"`
	Supply  Number `json:"Supply"`
	Decimal Number `json:"Decimal"`
}

func (token *Token) Validate() error {
	ValidateSymbol(token.Symbol)

	// TODO: add non-negative check once the type fixed
	return nil
}

func (token Token) String() string {
	return fmt.Sprintf("{Name: %v, Symbol: %v, Supply: %v, Decimal: %v}", token.Name, token.Symbol, token.Supply, token.Decimal)
}
