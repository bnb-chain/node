package tokens

import (
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
	"github.com/binance-chain/node/wire"
)

// Register concrete types on wire codec
func RegisterWire(cdc *wire.Codec) {
	cdc.RegisterConcrete(issue.IssueMsg{}, "tokens/IssueMsg", nil)
	cdc.RegisterConcrete(issue.MintMsg{}, "tokens/MintMsg", nil)
	cdc.RegisterConcrete(burn.BurnMsg{}, "tokens/BurnMsg", nil)
	cdc.RegisterConcrete(freeze.FreezeMsg{}, "tokens/FreezeMsg", nil)
	cdc.RegisterConcrete(freeze.UnfreezeMsg{}, "tokens/UnfreezeMsg", nil)
}
