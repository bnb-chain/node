package sidechain

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestInitCrossChainID(t *testing.T) {
	sourceChainID := sdk.ChainID(0x0001)
	_, keeper := CreateTestInput(t, true)
	keeper.SetSrcChainID(sourceChainID)

	require.Equal(t, sourceChainID, keeper.GetSrcChainID())
}

func TestRegisterCrossChainChannel(t *testing.T) {
	_, keeper := CreateTestInput(t, true)
	require.NoError(t, keeper.RegisterChannel("bind", sdk.ChannelID(1), nil))
	require.NoError(t, keeper.RegisterChannel("transfer", sdk.ChannelID(2), nil))
	require.NoError(t, keeper.RegisterChannel("timeout", sdk.ChannelID(3), nil))
	require.NoError(t, keeper.RegisterChannel("staking", sdk.ChannelID(4), nil))
	require.Error(t, keeper.RegisterChannel("staking", sdk.ChannelID(5), nil))
	require.Error(t, keeper.RegisterChannel("staking-new", sdk.ChannelID(4), nil))

	channeID, err := keeper.GetChannelID("transfer")
	require.NoError(t, err)
	require.Equal(t, sdk.ChannelID(2), channeID)

	channeID, err = keeper.GetChannelID("staking")
	require.NoError(t, err)
	require.Equal(t, sdk.ChannelID(4), channeID)
}

func TestRegisterDestChainID(t *testing.T) {
	_, keeper := CreateTestInput(t, true)
	require.NoError(t, keeper.RegisterDestChain("bsc", sdk.ChainID(1)))
	require.NoError(t, keeper.RegisterDestChain("ethereum", sdk.ChainID(2)))
	require.NoError(t, keeper.RegisterDestChain("btc", sdk.ChainID(3)))
	require.NoError(t, keeper.RegisterDestChain("cosmos", sdk.ChainID(4)))
	require.Error(t, keeper.RegisterDestChain("cosmos", sdk.ChainID(5)))
	require.Error(t, keeper.RegisterDestChain("mock", sdk.ChainID(4)))
	require.Error(t, keeper.RegisterDestChain("cosmos::", sdk.ChainID(5)))

	destChainID, err := keeper.GetDestChainID("bsc")
	require.NoError(t, err)
	require.Equal(t, sdk.ChainID(1), destChainID)

	destChainID, err = keeper.GetDestChainID("btc")
	require.NoError(t, err)
	require.Equal(t, sdk.ChainID(3), destChainID)
}

func TestCrossChainID(t *testing.T) {
	chainID, err := sdk.ParseChainID("123")
	require.NoError(t, err)
	require.Equal(t, sdk.ChainID(123), chainID)

	_, err = sdk.ParseChainID("65537")
	require.Error(t, err)
}
