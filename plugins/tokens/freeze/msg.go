package freeze

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const FreezeRoute = "tokens/freeze"
const FreezeRoute = "tokensFreeze"

var _ sdk.Msg = FreezeMsg{}

type FreezeMsg struct {
	From   sdk.AccAddress `json:"from"`
	Symbol string         `json:"symbol"`
	Amount int64          `json:"amount"`
}

func NewFreezeMsg(from sdk.AccAddress, symbol string, amount int64) FreezeMsg {
	return FreezeMsg{
		From:   from,
		Symbol: symbol,
		Amount: amount,
	}
}

func (msg FreezeMsg) Route() string { return FreezeRoute }
func (msg FreezeMsg) Type() string  { return FreezeRoute }
func (msg FreezeMsg) String() string {
	return fmt.Sprintf("Freeze{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}
func (msg FreezeMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg FreezeMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg FreezeMsg) ValidateBasic() sdk.Error {
	// expect all msgs that reference a token after issue to use the suffixed form (e.g. "BNB-ABC")
	err := types.ValidateMapperTokenSymbol(msg.Symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}
	if msg.Amount <= 0 {
		// TODO: maybe we need to define our own errors
		return sdk.ErrInsufficientFunds("amount should be more than 0")
	}
	return nil
}

func (msg FreezeMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

var _ sdk.Msg = UnfreezeMsg{}

type UnfreezeMsg struct {
	From   sdk.AccAddress `json:"from"`
	Symbol string         `json:"symbol"`
	Amount int64          `json:"amount"`
}

func NewUnfreezeMsg(from sdk.AccAddress, symbol string, amount int64) UnfreezeMsg {
	return UnfreezeMsg{
		From: from, Symbol: symbol, Amount: amount}
}

func (msg UnfreezeMsg) Route() string { return FreezeRoute }
func (msg UnfreezeMsg) Type() string  { return FreezeRoute }
func (msg UnfreezeMsg) String() string {
	return fmt.Sprintf("Unfreeze{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}
func (msg UnfreezeMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg UnfreezeMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

func (msg UnfreezeMsg) ValidateBasic() sdk.Error {
	// expect all msgs that reference a token after issue to use the suffixed form (e.g. "BNB-ABC")
	err := types.ValidateMapperTokenSymbol(msg.Symbol)
	if err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}
	if msg.Amount <= 0 {
		// TODO: maybe we need to define our own errors
		return sdk.ErrInsufficientFunds("amount should be more than 0")
	}
	return nil
}

func (msg UnfreezeMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}
