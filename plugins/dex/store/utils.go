package store

import (
	"errors"
	"strings"

	"github.com/bnb-chain/node/common/types"
)

// ValidatePairSymbol validates the given trading pair.
func ValidatePairSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("symbol pair must not be empty")
	}

	tokenSymbols := strings.SplitN(strings.ToUpper(symbol), "_", 2)
	if len(tokenSymbols) != 2 {
		return errors.New("invalid symbol: trading pair must contain an underscore ('_')")
	}
	for _, tokenSymbol := range tokenSymbols {
		if types.IsValidMiniTokenSymbol(tokenSymbol) {
			continue
		}
		if err := types.ValidateTokenSymbol(tokenSymbol); err != nil {
			return err
		}
	}
	return nil
}
