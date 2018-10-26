package list

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

const Route = "dexList"

var _ sdk.Msg = ListMsg{}

type ListMsg struct {
	Version          byte           `json:"version"`
	From             sdk.AccAddress `json:"from"`
	BaseAssetSymbol  string         `json:"base_asset_symbol"`
	QuoteAssetSymbol string         `json:"quote_asset_symbol"`
	InitPrice        int64          `json:"init_price"`
}

func NewMsg(from sdk.AccAddress, baseAssetSymbol string, quoteAssetSymbol string, initPrice int64) ListMsg {
	return ListMsg{
		Version:          0x01,
		From:             from,
		BaseAssetSymbol:  baseAssetSymbol,
		QuoteAssetSymbol: quoteAssetSymbol,
		InitPrice:        initPrice,
	}
}

func (msg ListMsg) Route() string                { return Route }
func (msg ListMsg) Type() string                 { return Route }
func (msg ListMsg) String() string               { return fmt.Sprintf("MsgList{%#v}", msg) }
func (msg ListMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg ListMsg) ValidateBasic() sdk.Error {
	if msg.Version != 0x01 {
		return sdk.ErrInternal("Invalid version. Expected 0x01")
	}
	err := types.ValidateSymbol(msg.BaseAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins("base token: " + err.Error())
	}
	err = types.ValidateSymbol(msg.QuoteAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins("quote token: " + err.Error())
	}
	if msg.InitPrice <= 0 {
		return sdk.ErrInvalidCoins("price should be positive")
	}
	return nil
}

func (msg ListMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
