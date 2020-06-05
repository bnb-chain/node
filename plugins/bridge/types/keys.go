package types

import (
	"fmt"
)

const (
	keyBindRequest      = "bindReq:%s"
	keyContractDecimals = "decs:"
)

func GetBindRequestKey(symbol string) []byte {
	return []byte(fmt.Sprintf(keyBindRequest, symbol))
}

func GetContractDecimalsKey(contractAddr []byte) []byte {
	return append([]byte(keyContractDecimals), contractAddr...)
}
