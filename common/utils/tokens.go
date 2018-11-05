package utils

import (
	"errors"
	"fmt"
	"strings"
)

const DELIMITER = "_"

func TradingPair2Assets(symbol string) (baseAsset, quoteAsset string, err error) {
	assets := strings.Split(symbol, DELIMITER)
	if len(assets) != 2 || assets[0] == "" || assets[1] == "" {
		return symbol, "", errors.New(fmt.Sprintf("Failed to parse trading pair symbol:%s into assets", symbol))
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
