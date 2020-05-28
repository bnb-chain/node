package list

import (
	"encoding/json"
	"fmt"

	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/order"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

const MiniMsg = "dexListMini"

var _ sdk.Msg = ListMiniMsg{}

type ListMiniMsg struct {
	From             sdk.AccAddress `json:"from"`
	BaseAssetSymbol  string         `json:"base_asset_symbol"`
	QuoteAssetSymbol string         `json:"quote_asset_symbol"`
	InitPrice        int64          `json:"init_price"`
}

func NewMiniMsg(from sdk.AccAddress, baseAssetSymbol string, quoteAssetSymbol string, initPrice int64) ListMiniMsg {
	return ListMiniMsg{
		From:             from,
		BaseAssetSymbol:  baseAssetSymbol,
		QuoteAssetSymbol: quoteAssetSymbol,
		InitPrice:        initPrice,
	}
}

func (msg ListMiniMsg) Route() string                { return Route }
func (msg ListMiniMsg) Type() string                 { return MiniMsg }
func (msg ListMiniMsg) String() string               { return fmt.Sprintf("MsgListMini{%#v}", msg) }
func (msg ListMiniMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg ListMiniMsg) ValidateBasic() sdk.Error {
	err := types.ValidateMiniTokenSymbol(msg.BaseAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins("base token: " + err.Error())
	}
	if len(msg.QuoteAssetSymbol) == 0 {
		return sdk.ErrInvalidCoins("quote token is empty ")
	}

	// before BEP70 upgraded, we only support listing mini token against NativeToken
	if sdk.IsUpgrade(upgrade.BEP70) {
		if types.NativeTokenSymbol != msg.QuoteAssetSymbol && order.BUSDSymbol != msg.QuoteAssetSymbol {
			return sdk.ErrInvalidCoins("quote token is not valid ")
		}
	} else {
		if types.NativeTokenSymbol != msg.QuoteAssetSymbol {
			return sdk.ErrInvalidCoins("quote token is not valid ")
		}
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
