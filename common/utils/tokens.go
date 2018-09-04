package utils

import (
	"errors"
	"fmt"
	"strings"
)

const DELIMITER = "_"

func TradingPair2Asset(symbol string) (baseAsset, quoteAsset string, err error) {
	assets := strings.Split(symbol, DELIMITER)
	if len(assets) != 2 || assets[0] == "" || assets[1] == "" {
		return symbol, "", errors.New("Failed to parse trading pair symbol into assets")
	}
	return assets[0], assets[1], nil
}

func Asset2TradingPair(baseAsset, quoteAsset string) (symbol string) {
	return fmt.Sprintf("%s_%s", baseAsset, quoteAsset)
}
