package types

import (
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"testing"
)

func TestGrowthIdenticalBaseAssetAndQuoteAsset(t *testing.T) {
	msg := NewListGrowthMarketMsg(sdk.AccAddress{}, "BTC-000", "BTC-000", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "base token and quote token should not be the same")
}

func TestGrowthWrongBaseAssetAndQuoteAssetSymbol(t *testing.T) {
	msg := NewListGrowthMarketMsg(sdk.AccAddress{}, "BTC", "BUSD", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "base token: suffixed token symbol")

	msg = NewListGrowthMarketMsg(sdk.AccAddress{}, "BTC-000", "BUSD", 1000)
	err = msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "quote token must be BNB or")
}

func TestGrowthWrongInitPrice(t *testing.T) {
	msg := NewListGrowthMarketMsg(sdk.AccAddress{}, "BTC-000", "BNB", -1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "price should be positive")
}

func TestGrowthRightMsg(t *testing.T) {
	msg := NewListGrowthMarketMsg(sdk.AccAddress{}, "BTC-000", "BNB", 1000)
	err := msg.ValidateBasic()
	require.Nil(t, err)

	msg = NewListGrowthMarketMsg(sdk.AccAddress{}, "AAA-000M", "BNB", 1000)
	err = msg.ValidateBasic()
	require.Nil(t, err)
}
