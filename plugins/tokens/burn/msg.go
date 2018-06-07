package burn

import (
	"encoding/json"
	"fmt"

	"github.com/BiJie/BinanceChain/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const Route = "tokens/burn"
const Route = "tokensBurn"

var _ sdk.Msg = (*Msg)(nil)

type Msg struct {
	Owner  sdk.Address `json:"owner"`
	Symbol string      `json:"symbol"`
	Amount int64       `json:"amount"`
}

func NewMsg(owner sdk.Address, symbol string, amount int64) Msg {
	return Msg{Owner: owner, Symbol: symbol, Amount: amount}
}

func (msg Msg) Type() string {
	return Route
}

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg Msg) ValidateBasic() sdk.Error {
	err := types.ValidateSymbol(msg.Symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	if msg.Amount <= 0 {
		// TODO: maybe we need to define our own errors
		return sdk.ErrInsufficientFunds("amount should be more than 0")
	}

	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("BurnMsg{%v#%v%v}", msg.Owner, msg.Amount, msg.Symbol)
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

func (msg Msg) GetSigners() []sdk.Address {
	return []sdk.Address{msg.Owner}
}
