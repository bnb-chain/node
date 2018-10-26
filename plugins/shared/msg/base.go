package msg

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

type MsgBase struct {
	Version byte           `json:"version"`
	From    sdk.AccAddress `json:"from"`
	Symbol  string         `json:"symbol"`
	Amount  int64          `json:"amount"`
}

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg MsgBase) ValidateBasic() sdk.Error {
	if msg.Version != 0x01 {
		return sdk.ErrInternal("Invalid version. Expected 0x01")
	}
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

func (msg MsgBase) String() string {
	return fmt.Sprintf("MsgBase{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}

func (msg MsgBase) Get(key interface{}) (value interface{}) {
	return nil
}

func (msg MsgBase) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

func (msg MsgBase) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.From}
}
