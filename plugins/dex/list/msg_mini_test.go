package list

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/plugins/dex/order"
)

func TestMiniIdenticalBaseAssetAndQuoteAsset(t *testing.T) {
	msg := NewListMiniMsg(sdk.AccAddress{}, "BTC-000M", "BTC-000M", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "quote token is not valid")
}

func TestMiniWrongBaseAssetSymbol(t *testing.T) {
	msg := NewListMiniMsg(sdk.AccAddress{}, "BTC", "BTC-000", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "base token: suffixed token symbol must contain a hyphen ('-')")
}

func TestMiniWrongBaseAssetSymbolNotMiniToken(t *testing.T) {
	msg := NewListMiniMsg(sdk.AccAddress{}, "BTC-000", "BTC-000", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "base token: mini-token symbol suffix must be 4 chars in length, got 3")
}

func TestMiniWrongQuoteAssetSymbol(t *testing.T) {
	msg := NewListMiniMsg(sdk.AccAddress{}, "BTC-000M", "ETH-123", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "quote token is not valid")
}

func TestMiniWrongInitPrice(t *testing.T) {
	msg := NewListMiniMsg(sdk.AccAddress{}, "BTC-000M", "BNB", -1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "price should be positive")
}

func TestMiniRightMsg(t *testing.T) {
	msg := NewListMiniMsg(sdk.AccAddress{}, "BTC-000M", "BNB", 1000)
	err := msg.ValidateBasic()
	require.Nil(t, err, "msg should not be error")
}

func TestMiniBUSDQuote(t *testing.T) {
	msg := NewListMiniMsg(sdk.AccAddress{}, "BTC-000M", "BUSD-000", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "quote token is not valid")

	setChainVersion()
	defer resetChainVersion()
	order.BUSDSymbol = "BUSD-000"
	msg = NewListMiniMsg(sdk.AccAddress{}, "BTC-000M", "BUSD-000", 1000)
	err = msg.ValidateBasic()
	require.Nil(t, err, "msg should not be error")

}
