package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/types"
)

func TestParseChannelID(t *testing.T) {
	channelID, err := types.ParseChannelID("12")
	require.NoError(t, err)
	require.Equal(t, types.ChannelID(12), channelID)

	_, err = types.ParseChannelID("1024")
	require.Error(t, err)
}

func TestParseCrossChainID(t *testing.T) {
	chainID, err := types.ParseChainID("12")
	require.NoError(t, err)
	require.Equal(t, types.ChainID(12), chainID)

	chainID, err = types.ParseChainID("10000")
	require.NoError(t, err)
	require.Equal(t, types.ChainID(10000), chainID)

	_, err = types.ParseChainID("65536")
	require.Error(t, err)
}
