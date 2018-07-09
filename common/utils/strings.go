package utils

import "regexp"

var (
	isAlphaNumFunc = regexp.MustCompile(`^[[:alnum:]]+$`).MatchString
)

func IsAlphaNum(s string) bool {
	return isAlphaNumFunc(s)
}
