package msg

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

type TokenOpMsgBase struct {
	From   sdk.AccAddress `json:"from"`
	Symbol string         `json:"symbol"`
	Amount int64          `json:"amount"`
}

// ValidateBasic does a simple validation check that
// doesn't require access to any other information.
func (msg TokenOpMsgBase) ValidateBasic() sdk.Error {
	// expect all msgs that reference a token after issue to use the suffixed form (e.g. "BNB-ABCDEF")
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

func (msg TokenOpMsgBase) String() string {
	return fmt.Sprintf("TokenOpMsgBase{%v#%v%v}", msg.From, msg.Amount, msg.Symbol)
}

func (msg TokenOpMsgBase) Get(key interface{}) (value interface{}) {
	return nil
}

func (msg TokenOpMsgBase) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}

func (msg TokenOpMsgBase) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.From}
}
