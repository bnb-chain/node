package utils

import (
	"errors"
	"fmt"
	"strings"
)

const DELIMITER = "_"

func TradeSymbol2Ccy(symbol string) (tradeCcy, quoteCcy string, err error) {
	ccy := strings.Split(symbol, DELIMITER)
	if len(ccy) != 2 {
		return symbol, "", errors.New("Failed to parse trade symbol into currencies")
	}
	return ccy[0], ccy[1], nil
}

func Ccy2TradeSymbol(tradeCcy, quoteCcy string) (symbol string) {
	return fmt.Sprintf("%s_%s", tradeCcy, quoteCcy)
}
