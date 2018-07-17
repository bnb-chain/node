package list

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

const Route = "dexList"

type Msg struct {
	From        sdk.AccAddress `json:"from"`
	Symbol      string         `json:"symbol"`
	QuoteSymbol string         `json:"quote_symbol"`
	InitPrice   int64          `json:"init_price"`
}

func NewMsg(from sdk.AccAddress, symbol string, quoteSymbol string, initPrice int64) Msg {
	return Msg{
		From:        from,
		Symbol:      symbol,
		QuoteSymbol: quoteSymbol,
		InitPrice:   initPrice,
	}
}

func (msg Msg) Type() string                            { return Route }
func (msg Msg) String() string                          { return fmt.Sprintf("MsgList{%#v}", msg) }
func (msg Msg) Get(key interface{}) (value interface{}) { return nil }
func (msg Msg) GetSigners() []sdk.AccAddress            { return []sdk.AccAddress{msg.From} }

func (msg Msg) ValidateBasic() sdk.Error {
	err := types.ValidateSymbol(msg.Symbol)
	if err != nil {
		return sdk.ErrInvalidCoins("base token: " + err.Error())
	}

	err = types.ValidateSymbol(msg.QuoteSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins("quote token: " + err.Error())
	}

	if msg.InitPrice <= 0 {
		return sdk.ErrInvalidCoins("price should be positive")
	}

	return nil
}

func (msg Msg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
