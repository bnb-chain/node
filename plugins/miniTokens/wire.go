package minitokens

import (
	"github.com/binance-chain/node/plugins/minitokens/burn"
	"github.com/binance-chain/node/plugins/minitokens/freeze"
	"github.com/binance-chain/node/plugins/minitokens/issue"
	"github.com/binance-chain/node/plugins/minitokens/uri"
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(issue.IssueMsg{}, "minitokens/IssueMsg", nil)
	cdc.RegisterConcrete(issue.MintMsg{}, "minitokens/MintMsg", nil)
	cdc.RegisterConcrete(freeze.FreezeMsg{}, "minitokens/FreezeMsg", nil)
	cdc.RegisterConcrete(freeze.UnfreezeMsg{}, "minitokens/UnFreezeMsg", nil)
	cdc.RegisterConcrete(burn.BurnMsg{}, "minitokens/BurnMsg", nil)
	cdc.RegisterConcrete(uri.SetURIMsg{}, "minitokens/SetURIMsg", nil)
}
