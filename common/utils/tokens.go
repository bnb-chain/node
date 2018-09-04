package utils

import (
	"errors"
	"fmt"
	"strings"
)

const DELIMITER = "_"

func TradeSymbol2Ccy(symbol string) (baseCcy, quoteCcy string, err error) {
	ccy := strings.Split(symbol, DELIMITER)
	if len(ccy) != 2 || ccy[0] == "" || ccy[1] == "" {
		return symbol, "", errors.New("Failed to parse trade symbol into currencies")
	}
	return ccy[0], ccy[1], nil
}

func Ccy2TradeSymbol(baseCcy, quoteCcy string) (symbol string) {
	return fmt.Sprintf("%s_%s", baseCcy, quoteCcy)
}
