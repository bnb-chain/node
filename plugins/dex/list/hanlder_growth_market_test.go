package list

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/binance-chain/node/common/types"
	dextypes "github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/tokens"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

func setupForGrowthMarket(ctx sdk.Context, tokenMapper tokens.Mapper, t *testing.T) {
	err := tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Bitcoin",
		Symbol:      "BTC-000",
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Native Token",
		Symbol:      types.NativeTokenSymbol,
		OrigSymbol:  types.NativeTokenSymbol,
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	miniToken := types.NewMiniToken("Bitcoin Mini", "BTC", "BTC-000M", types.MiniRangeType, 100000e8, sdk.AccAddress("testacc"), false, "")
	err = tokenMapper.NewToken(ctx, miniToken)
	require.Nil(t, err, "new token error")
}

func TestHandler(t *testing.T) {
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForGrowthMarket(ctx, tokenMapper, t)

	// test invalid quote
	result := handleListGrowthMarket(ctx, orderKeeper, tokenMapper,
		dextypes.NewListGrowthMarketMsg(sdk.AccAddress("testacc"), "BTC-000", "AAA-000", 100000))
	require.Contains(t, result.Log, "quote token is not valid")

	// test non-owner address
	result = handleListGrowthMarket(ctx, orderKeeper, tokenMapper,
		dextypes.NewListGrowthMarketMsg(sdk.AccAddress("testacc1"), "BTC-000", types.NativeTokenSymbol, 100000))
	require.Contains(t, result.Log, "only the owner of the base asset or quote asset can list the trading pair")

	// test identical
	result = handleListGrowthMarket(ctx, orderKeeper, tokenMapper,
		dextypes.NewListGrowthMarketMsg(sdk.AccAddress("testacc"), types.NativeTokenSymbol, types.NativeTokenSymbol, 100000))
	require.Contains(t, result.Log, "base asset symbol should not be identical to quote asset symbol")

	// test positive case
	result = handleListGrowthMarket(ctx, orderKeeper, tokenMapper,
		dextypes.NewListGrowthMarketMsg(sdk.AccAddress("testacc"), "BTC-000M", types.NativeTokenSymbol, 100000))
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	result = handleListGrowthMarket(ctx, orderKeeper, tokenMapper,
		dextypes.NewListGrowthMarketMsg(sdk.AccAddress("testacc"), "BTC-000", types.NativeTokenSymbol, 100000))
	require.Equal(t, result.Code, sdk.ABCICodeOK)

	// test duplicated pair
	result = handleListGrowthMarket(ctx, orderKeeper, tokenMapper,
		dextypes.NewListGrowthMarketMsg(sdk.AccAddress("testacc"), "BTC-000", types.NativeTokenSymbol, 100000))
	require.Contains(t, result.Log, "trading pair exists")

	// test not exist symbol
	result = handleListGrowthMarket(ctx, orderKeeper, tokenMapper,
		dextypes.NewListGrowthMarketMsg(sdk.AccAddress("testacc"), "AAA-000", types.NativeTokenSymbol, 100000))
	require.Contains(t, result.Log, "not found")
}
