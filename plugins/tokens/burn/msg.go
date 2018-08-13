package burn

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/plugins/tokens/base"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const Route = "tokens/burn"
const Route = "tokensBurn"

var _ tx.Msg = (*Msg)(nil)

type Msg struct {
	base.MsgBase
}

func NewMsg(from sdk.AccAddress, symbol string, amount int64) Msg {
	return Msg{base.MsgBase{From: from, Symbol: symbol, Amount: amount}}
}

func (msg Msg) Type() string {
	return Route
}

func (msg Msg) String() string {
	return fmt.Sprintf("BurnMsg{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}
