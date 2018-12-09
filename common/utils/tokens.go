package utils

import (
	"fmt"
	"strings"
)

const DELIMITER = "_"

func TradingPair2Assets(symbol string) (baseAsset, quoteAsset string, err error) {
	assets := strings.SplitN(symbol, DELIMITER, 2)
	if len(assets) != 2 || assets[0] == "" || assets[1] == "" {
		return symbol, "", fmt.Errorf("Failed to parse trading pair symbol:%s into assets", symbol)
	}
	if strings.Contains(assets[1], DELIMITER) {
		return symbol, "", fmt.Errorf("Failed to parse trading pair symbol:%s into assets", symbol)
	}
	return assets[0], assets[1], nil
}

func TradingPair2AssetsSafe(symbol string) (baseAsset, quoteAsset string) {
	baseAsset, quoteAsset, err := TradingPair2Assets(symbol)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse trading pair symbol:%s into assets", symbol))
	}
	return
}

func Assets2TradingPair(baseAsset, quoteAsset string) (symbol string) {
	return fmt.Sprintf("%s_%s", baseAsset, quoteAsset)
}
