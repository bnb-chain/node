package freeze

import (
	"fmt"

	"github.com/BiJie/BinanceChain/plugins/tokens/base"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const RouteFreeze = "tokens/freeze"
// const RouteFreeze = "tokens/unfreeze"
const RouteFreeze = "tokensFreeze"
const RouteUnfreeze = "tokensUnfreeze"

var _ sdk.Msg = (*FreezeMsg)(nil)

type FreezeMsg struct {
	base.MsgBase
}

func NewFreezeMsg(owner sdk.Address, symbol string, amount int64) FreezeMsg {
	return FreezeMsg{base.MsgBase{Owner: owner, Symbol: symbol, Amount: amount}}
}

func (msg FreezeMsg) Type() string { return RouteFreeze }

func (msg FreezeMsg) String() string {
	return fmt.Sprintf("Freeze{%v#%v}", msg.Owner, msg.Symbol)
}

var _ sdk.Msg = (*UnfreezeMsg)(nil)

type UnfreezeMsg struct {
	base.MsgBase
}

func NewUnfreezeMsg(owner sdk.Address, symbol string, amount int64) UnfreezeMsg {
	return UnfreezeMsg{base.MsgBase{Owner: owner, Symbol: symbol, Amount: amount}}
}

func (msg UnfreezeMsg) Type() string { return RouteUnfreeze }

func (msg UnfreezeMsg) String() string {
	return fmt.Sprintf("Unfreeze{%v#%v%v}", msg.Owner, msg.Amount, msg.Symbol)
}
