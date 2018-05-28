package tokens

import (
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/cosmos/cosmos-sdk/wire"
)

// Register concrete types on wire codec
func RegisterTypes(cdc *wire.Codec) {
	cdc.RegisterConcrete(issue.Msg{}, "tokens/IssueMsg", nil)
	cdc.RegisterConcrete(burn.Msg{}, "tokens/BurnMsg", nil)
	cdc.RegisterConcrete(freeze.FreezeMsg{}, "tokens/FreezeMsg", nil)
	cdc.RegisterConcrete(freeze.UnfreezeMsg{}, "tokens/UnfreezeMsg", nil)
}
