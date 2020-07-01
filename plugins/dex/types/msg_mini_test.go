package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
