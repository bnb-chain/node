package issue

import (
	"encoding/json"
	"fmt"

	"github.com/BiJie/BinanceChain/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const Route  = "tokens/issue"
const Route = "tokensIssue"

var _ sdk.Msg = (*Msg)(nil)

type Msg struct {
	From    sdk.Address `json:"from"`
	Name    string      `json:"Name"`
	Symbol  string      `json:"Symbol"`
	Supply  int64       `json:"Supply"`
	Decimal int8        `json:"Decimal"`
}

func NewMsg(from sdk.Address, name, symbol string, supply int64, decimal int8) Msg {
	return Msg{
		From:    from,
		Name:    name,
		Symbol:  symbol,
		Supply:  supply,
		Decimal: decimal,
	}
}

func (msg Msg) Type() string { return Route }

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg Msg) ValidateBasic() sdk.Error {
	if msg.From == nil {
		return sdk.ErrInvalidAddress("sender address cannot be empty")
	}

	if err := types.ValidateSymbol(msg.Symbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	// TODO: check supply and decimal

	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("IssueMsg{%#v}", msg)
}

func (msg Msg) Get(key interface{}) (value interface{}) {
	return nil
}

func (msg Msg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

// Implements Msg.
func (msg Msg) GetSigners() []sdk.Address {
	return []sdk.Address{msg.From}
}
