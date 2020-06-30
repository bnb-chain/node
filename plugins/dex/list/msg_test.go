package list

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/binance-chain/node/plugins/dex/types"
)

func TestIdenticalBaseAssetAndQuoteAsset(t *testing.T) {
	msg := types.NewMsg(sdk.AccAddress{}, 1, "BTC-000", "BTC-000", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "base token and quote token should not be the same")
}

func TestWrongProposalId(t *testing.T) {
	msg := types.NewMsg(sdk.AccAddress{}, -1, "BTC-000", "BTC-000", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "proposal id should be positive")
}

func TestWrongBaseAssetSymbol(t *testing.T) {
	msg := types.NewMsg(sdk.AccAddress{}, 1, "BTC", "BTC-000", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "base token: suffixed token symbol")
}

func TestWrongQuoteAssetSymbol(t *testing.T) {
	msg := types.NewMsg(sdk.AccAddress{}, 1, "BTC-000", "ETH", 1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "quote token: suffixed token symbol")
}

func TestWrongInitPrice(t *testing.T) {
	msg := types.NewMsg(sdk.AccAddress{}, 1, "BTC-000", "BNB", -1000)
	err := msg.ValidateBasic()
	require.NotNil(t, err, "msg should be error")
	require.Contains(t, err.Error(), "price should be positive")
}

func TestRightMsg(t *testing.T) {
	msg := types.NewMsg(sdk.AccAddress{}, 1, "BTC-000", "BNB", 1000)
	err := msg.ValidateBasic()
	require.Nil(t, err, "msg should not be error")
}
