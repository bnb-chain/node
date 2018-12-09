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
	if len(symbol) > ((types.TokenSymbolMaxLen + types.TokenSymbolTxHashSuffixLen) * 2) + 2 {
		return errors.New("symbol pair is too long")
	}
	if !strings.Contains(symbol, "_") {
		return errors.New("symbol pair must contain `_`")
	}
	tokenSymbols := strings.SplitN(strings.ToUpper(symbol), "_", 2)
	if len(tokenSymbols) != 2 {
		return errors.New("invalid symbol")
	}
	if strings.Contains(tokenSymbols[1], "_") {
		return errors.New("pair must contain only one underscore ('_')")
	}

	for _, tokenSymbol := range tokenSymbols {
		if err := types.ValidateMapperTokenSymbol(tokenSymbol); err != nil {
			return err
		}
	}
	return nil
}
