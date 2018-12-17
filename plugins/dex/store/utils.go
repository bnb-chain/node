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
	// TokenSymbolMaxLen: BTC00000
	// TokenSymbolTxHashSuffixLen: 000
	// + 2: ".B"
	// * 2: BTC00000.B, ETH00000.B
	// + 3: 2x `-` and 1x `_`
	if len(symbol) > ((types.TokenSymbolMaxLen+types.TokenSymbolTxHashSuffixLen+2)*2)+3 {
		return errors.New("symbol pair is too long")
	}
	tokenSymbols := strings.SplitN(strings.ToUpper(symbol), "_", 2)
	if len(tokenSymbols) != 2 {
		return errors.New("invalid symbol: trading pair must contain an underscore ('_')")
	}
	for _, tokenSymbol := range tokenSymbols {
		if err := types.ValidateMapperTokenSymbol(tokenSymbol); err != nil {
			return err
		}
	}
	return nil
}
