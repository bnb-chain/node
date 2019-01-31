package burn

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	shared "github.com/binance-chain/node/plugins/shared/msg"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const BurnRoute = "tokens/burn"
const BurnRoute = "tokensBurn"

var _ sdk.Msg = BurnMsg{}

type BurnMsg struct {
	shared.TokenOpMsgBase
}

func NewMsg(from sdk.AccAddress, symbol string, amount int64) BurnMsg {
	return BurnMsg{shared.TokenOpMsgBase{From: from, Symbol: symbol, Amount: amount}}
}

func (msg BurnMsg) Route() string {
	return BurnRoute
}

func (msg BurnMsg) Type() string {
	return BurnRoute
}

func (msg BurnMsg) String() string {
	return fmt.Sprintf("BurnMsg{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}

func (msg BurnMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
