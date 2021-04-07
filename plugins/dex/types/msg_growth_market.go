package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

const ListGrowthMarketMsgType = "dexListGrowthMarket"

var _ sdk.Msg = ListGrowthMarketMsg{}

type ListGrowthMarketMsg struct {
	From             sdk.AccAddress `json:"from"`
	BaseAssetSymbol  string         `json:"base_asset_symbol"`
	QuoteAssetSymbol string         `json:"quote_asset_symbol"`
	InitPrice        int64          `json:"init_price"`
}

func NewListGrowthMarketMsg(from sdk.AccAddress, baseAssetSymbol string, quoteAssetSymbol string, initPrice int64) ListGrowthMarketMsg {
	return ListGrowthMarketMsg{
		From:             from,
		BaseAssetSymbol:  baseAssetSymbol,
		QuoteAssetSymbol: quoteAssetSymbol,
		InitPrice:        initPrice,
	}
}

func (msg ListGrowthMarketMsg) Route() string                { return ListRoute }
func (msg ListGrowthMarketMsg) Type() string                 { return ListGrowthMarketMsgType }
func (msg ListGrowthMarketMsg) String() string               { return fmt.Sprintf("MsgListGrowthMarket{%#v}", msg) }
func (msg ListGrowthMarketMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg ListGrowthMarketMsg) ValidateBasic() sdk.Error {
	if msg.BaseAssetSymbol == msg.QuoteAssetSymbol {
		return sdk.ErrInvalidCoins("base token and quote token should not be the same")
	}

	if !types.IsValidMiniTokenSymbol(msg.BaseAssetSymbol) {
		err := types.ValidateTokenSymbol(msg.BaseAssetSymbol)
		if err != nil {
			return sdk.ErrInvalidCoins("base token: " + err.Error())
		}
	}

	if !types.IsValidMiniTokenSymbol(msg.QuoteAssetSymbol) {
		err := types.ValidateTokenSymbol(msg.QuoteAssetSymbol)
		if err != nil {
			return sdk.ErrInvalidCoins("quote token: " + err.Error())
		}
	}

	if msg.InitPrice <= 0 {
		return sdk.ErrInvalidCoins("price should be positive")
	}
	return nil
}

func (msg ListGrowthMarketMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

func (msg ListGrowthMarketMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
