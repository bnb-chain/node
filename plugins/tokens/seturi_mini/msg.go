package seturi_mini

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

const SetURIRoute = "miniTokensSetURI"

var _ sdk.Msg = SetURIMsg{}

type SetURIMsg struct {
	From     sdk.AccAddress `json:"from"`
	Symbol   string         `json:"symbol"`
	TokenURI string         `json:"token_uri"`
}

func NewSetUriMsg(from sdk.AccAddress, symbol string, tokenURI string) SetURIMsg {
	return SetURIMsg{
		From:     from,
		Symbol:   symbol,
		TokenURI: tokenURI,
	}
}

func (msg SetURIMsg) ValidateBasic() sdk.Error {
	if msg.From == nil || len(msg.From) == 0 {
		return sdk.ErrInvalidAddress("sender address cannot be empty")
	}

	if err := types.ValidateMapperMiniTokenSymbol(msg.Symbol); err != nil {
		return sdk.ErrInvalidCoins(err.Error())
	}

	return nil
}

// Implements MintMsg.
func (msg SetURIMsg) Route() string                { return SetURIRoute }
func (msg SetURIMsg) Type() string                 { return SetURIRoute }
func (msg SetURIMsg) String() string               { return fmt.Sprintf("SetURI{%#v}", msg) }
func (msg SetURIMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{msg.From} }
func (msg SetURIMsg) GetSignBytes() []byte {
	b, err := json.Marshal(msg) // XXX: ensure some canonical form
	if err != nil {
		panic(err)
	}
	return b
}
func (msg SetURIMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
