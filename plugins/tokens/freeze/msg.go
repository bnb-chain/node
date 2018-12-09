package freeze

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	shared "github.com/BiJie/BinanceChain/plugins/shared/msg"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const FreezeRoute = "tokens/freeze"
const FreezeRoute = "tokensFreeze"

var _ sdk.Msg = FreezeMsg{}

type FreezeMsg struct {
	shared.TokenOpMsgBase
}

func NewFreezeMsg(from sdk.AccAddress, symbol string, amount int64) FreezeMsg {
	return FreezeMsg{shared.TokenOpMsgBase{From: from, Symbol: symbol, Amount: amount}}
}

func (msg FreezeMsg) Route() string {
	return FreezeRoute
}

func (msg FreezeMsg) Type() string {
	return FreezeRoute
}

func (msg FreezeMsg) String() string {
	return fmt.Sprintf("Freeze{%v#%v}", msg.From, msg.Symbol)
}

func (msg FreezeMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

var _ sdk.Msg = UnfreezeMsg{}

type UnfreezeMsg struct {
	shared.TokenOpMsgBase
}

func NewUnfreezeMsg(from sdk.AccAddress, symbol string, amount int64) UnfreezeMsg {
	return UnfreezeMsg{shared.TokenOpMsgBase{From: from, Symbol: symbol, Amount: amount}}
}

func (msg UnfreezeMsg) Route() string {
	return FreezeRoute
}

func (msg UnfreezeMsg) Type() string { return FreezeRoute }

func (msg UnfreezeMsg) String() string {
	return fmt.Sprintf("Unfreeze{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}

func (msg UnfreezeMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
