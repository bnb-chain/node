package types

import (
	"fmt"
)

const (
	keyBindRequest = "bindReq:%s"
)

func GetBindRequestKey(symbol string) []byte {
	return []byte(fmt.Sprintf(keyBindRequest, symbol))
}
