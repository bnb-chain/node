package store

import (
	"errors"
	"strings"

	"github.com/BiJie/BinanceChain/common/types"
)

// ValidatePairSymbol validates the given trading pair.
func ValidatePairSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("symbol pair must not be empty")
	}
	if !strings.Contains(symbol, "_") {
		return errors.New("symbol pair must contain `_`")
	}
	tokenSymbols := strings.Split(strings.ToUpper(symbol), "_")
	if len(tokenSymbols) != 2 {
		return errors.New("Invalid symbol")
	}
	for _, tokenSymbol := range tokenSymbols {
		err := types.ValidateSymbol(tokenSymbol)
		if err != nil {
			return err
		}
	}
	return nil
}
