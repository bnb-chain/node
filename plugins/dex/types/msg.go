package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/node/common/types"
)

const ListRoute = "dexList"

var _ sdk.Msg = ListMsg{}

type ListMsg struct {
	From             sdk.AccAddress `json:"from"`
	ProposalId       int64          `json:"proposal_id"`
	BaseAssetSymbol  string         `json:"base_asset_symbol"`
	QuoteAssetSymbol string         `json:"quote_asset_symbol"`
	InitPrice        int64          `json:"init_price"`
}

func NewListMsg(from sdk.AccAddress, proposalId int64, baseAssetSymbol string, quoteAssetSymbol string, initPrice int64) ListMsg {
	return ListMsg{
		From:             from,
		ProposalId:       proposalId,
		BaseAssetSymbol:  baseAssetSymbol,
		QuoteAssetSymbol: quoteAssetSymbol,
		InitPrice:        initPrice,
	}
}

func (msg ListMsg) Route() string                { return ListRoute }
func (msg ListMsg) Type() string                 { return ListRoute }
func (msg ListMsg) String() string               { return fmt.Sprintf("MsgList{%#v}", msg) }
func (msg ListMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }

func (msg ListMsg) ValidateBasic() sdk.Error {
	if msg.ProposalId <= 0 {
		return sdk.ErrInvalidCoins("proposal id should be positive")
	}
	err := types.ValidateTokenSymbol(msg.BaseAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins("base token: " + err.Error())
	}
	err = types.ValidateTokenSymbol(msg.QuoteAssetSymbol)
	if err != nil {
		return sdk.ErrInvalidCoins("quote token: " + err.Error())
	}
	if msg.BaseAssetSymbol == msg.QuoteAssetSymbol {
		return sdk.ErrInvalidCoins("base token and quote token should not be the same")
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

func (msg ListMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
