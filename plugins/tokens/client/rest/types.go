package rest

import "github.com/binance-chain/node/common/types"

type TokenWrap struct {
	*types.Token
	Height int64 `json:"height"`
}
