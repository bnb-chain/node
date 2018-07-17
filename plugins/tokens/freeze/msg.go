package freeze

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/plugins/tokens/base"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const RouteFreeze = "tokens/freeze"
const RouteFreeze = "tokensFreeze"

var _ sdk.Msg = (*FreezeMsg)(nil)

type FreezeMsg struct {
	base.MsgBase
}

func NewFreezeMsg(from sdk.AccAddress, symbol string, amount int64) FreezeMsg {
	return FreezeMsg{base.MsgBase{From: from, Symbol: symbol, Amount: amount}}
}

func (msg FreezeMsg) Type() string { return RouteFreeze }

func (msg FreezeMsg) String() string {
	return fmt.Sprintf("Freeze{%v#%v}", msg.From, msg.Symbol)
}

var _ sdk.Msg = (*UnfreezeMsg)(nil)

type UnfreezeMsg struct {
	base.MsgBase
}

func NewUnfreezeMsg(from sdk.AccAddress, symbol string, amount int64) UnfreezeMsg {
	return UnfreezeMsg{base.MsgBase{From: from, Symbol: symbol, Amount: amount}}
}

func (msg UnfreezeMsg) Type() string { return RouteFreeze }

func (msg UnfreezeMsg) String() string {
	return fmt.Sprintf("Unfreeze{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}
