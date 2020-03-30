package miniTokens

import (
	"github.com/binance-chain/node/plugins/miniTokens/burn"
	"github.com/binance-chain/node/plugins/miniTokens/freeze"
	"github.com/binance-chain/node/plugins/miniTokens/issue"
	"github.com/binance-chain/node/plugins/miniTokens/uri"
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(issue.IssueMsg{}, "miniTokens/IssueMsg", nil)
	cdc.RegisterConcrete(issue.MintMsg{}, "miniTokens/MintMsg", nil)
	cdc.RegisterConcrete(freeze.FreezeMsg{}, "miniTokens/FreezeMsg", nil)
	cdc.RegisterConcrete(freeze.UnfreezeMsg{}, "miniTokens/UnFreezeMsg", nil)
	cdc.RegisterConcrete(burn.BurnMsg{}, "miniTokens/BurnMsg", nil)
	cdc.RegisterConcrete(uri.SetURIMsg{}, "miniTokens/SetURIMsg", nil)
}
