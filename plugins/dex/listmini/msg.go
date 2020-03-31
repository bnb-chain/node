package list

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

const ListMiniRoute = "dexListMini"

var _ sdk.Msg = ListMiniMsg{}

type ListMiniMsg struct {
	From             sdk.AccAddress `json:"from"`
	BaseAssetSymbol  string         `json:"base_asset_symbol"`
	QuoteAssetSymbol string         `json:"quote_asset_symbol"`
	InitPrice        int64          `json:"init_price"`
}

func NewMsg(from sdk.AccAddress, baseAssetSymbol string, quoteAssetSymbol string, initPrice int64) ListMiniMsg {
	return ListMiniMsg{
		From:             from,
		BaseAssetSymbol:  baseAssetSymbol,
		QuoteAssetSymbol: quoteAssetSymbol,
		InitPrice:        initPrice,
	}
}

func (msg ListMiniMsg) Route() string                { return ListMiniRoute }
func (msg ListMiniMsg) Type() string                 { return ListMiniRoute }
func (msg ListMiniMsg) String() string               { return fmt.Sprintf("MsgListMini{%#v}", msg) }
func (msg ListMiniMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg ListMiniMsg) ValidateBasic() sdk.Error {

	err := types.ValidateMapperMiniTokenSymbol(msg.BaseAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins("base token: " + err.Error())
	}
	if types.NativeTokenSymbol != msg.QuoteAssetSymbol {//todo permit BUSD
		return sdk.ErrInvalidCoins("quote token: " + err.Error())
	}
	if msg.InitPrice <= 0 {
		return sdk.ErrInvalidCoins("price should be positive")
	}
	return nil
}

func (msg ListMiniMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg ListMiniMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
