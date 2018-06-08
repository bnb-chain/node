package burn

import (
	"fmt"

	"github.com/BiJie/BinanceChain/plugins/tokens/base"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const Route = "tokens/burn"
const Route = "tokensBurn"

var _ sdk.Msg = (*Msg)(nil)

type Msg struct {
	base.MsgBase
}

func NewMsg(owner sdk.Address, symbol string, amount int64) Msg {
	return Msg{base.MsgBase{Owner: owner, Symbol: symbol, Amount: amount}}
}

func (msg Msg) Type() string {
	return Route
}

func (msg Msg) String() string {
	return fmt.Sprintf("BurnMsg{%v#%v%v}", msg.Owner, msg.Amount, msg.Symbol)
}
