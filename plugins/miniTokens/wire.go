package miniTokens

import (
	"github.com/binance-chain/node/plugins/miniTokens/issue"
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(issue.IssueMsg{}, "miniTokens/IssueMsg", nil)
	cdc.RegisterConcrete(issue.MintMsg{}, "miniTokens/MintMsg", nil)
}
