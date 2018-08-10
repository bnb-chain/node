package store

import (
	"errors"
	"strings"

	"github.com/BiJie/BinanceChain/common/types"
)

// ValidatePairSymbol validates the given trading pair.
func ValidatePairSymbol(symbol string) error {
	tokenSymbols := strings.Split(symbol, "_")
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
