package utils

import (
	"errors"
	"regexp"
	"strconv"
)

var (
	isAlphaNumFunc = regexp.MustCompile(`^[[:alnum:]]+$`).MatchString
)

func IsAlphaNum(s string) bool {
	return isAlphaNumFunc(s)
}

func ParsePrice(priceStr string) (int64, error) {
	if len(priceStr) == 0 {
		return 0, errors.New("Input number should be provided")
	}

	price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil {
		return 0, err
	}

	if price <= 0 {
		return price, errors.New("string-to-parse should be greater than 0")
	}

	return price, nil
}
