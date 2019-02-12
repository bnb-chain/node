package burn

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

// TODO: "route expressions can only contain alphanumeric characters", we need to change the cosmos sdk to support slash
// const BurnRoute = "tokens/burn"
const BurnRoute = "tokensBurn"

var _ sdk.Msg = BurnMsg{}

type BurnMsg struct {
	From   sdk.AccAddress `json:"from"`
	Symbol string         `json:"symbol"`
	Amount int64          `json:"amount"`
}

func NewMsg(from sdk.AccAddress, symbol string, amount int64) BurnMsg {
	return BurnMsg{
		From:   from,
		Symbol: symbol,
		Amount: amount,
	}
}

func (msg BurnMsg) Route() string { return BurnRoute }
func (msg BurnMsg) Type() string  { return BurnRoute }
func (msg BurnMsg) String() string {
	return fmt.Sprintf("BurnMsg{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}
func (msg BurnMsg) GetInvolvedAddresses() []sdk.AccAddress { return msg.GetSigners() }
func (msg BurnMsg) GetSigners() []sdk.AccAddress           { return []sdk.AccAddress{msg.From} }

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg BurnMsg) ValidateBasic() sdk.Error {
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

func (msg BurnMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}
