package list

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/types"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/order"
	dextypes "github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/tokens"
)

func setChainVersion() {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP8, -1)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP70, -1)
}

func resetChainVersion() {
	upgrade.Mgr.Config.HeightMap = nil
}

func setupForMini(ctx sdk.Context, tokenMapper tokens.Mapper, t *testing.T) {
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

	tinyToken := types.NewMiniToken("Bitcoin Mini", "ETH", "ETH-000M", types.TinyRangeType, 10000e8, sdk.AccAddress("testacc"), true, "abc")
	err = tokenMapper.NewToken(ctx, tinyToken)
	require.Nil(t, err, "new token error")
}

func TestHandleListMiniIdenticalSymbols(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, dextypes.ListMiniMsg{
		From:             sdk.AccAddress("testacc"),
		BaseAssetSymbol:  "BTC-000M",
		QuoteAssetSymbol: "BTC-000M",
		InitPrice:        1000,
	})
	require.Contains(t, result.Log, "quote token is not valid")
}

func TestMiniWrongQuoteAssetSymbol(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, dextypes.ListMiniMsg{
		From:             sdk.AccAddress("testacc"),
		BaseAssetSymbol:  "BTC-000M",
		QuoteAssetSymbol: "ETH-000M",
		InitPrice:        1000,
	})
	require.Contains(t, result.Log, "quote token is not valid")
}

func TestMiniBUSDQuote(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, dextypes.ListMiniMsg{
		From:             sdk.AccAddress("testacc"),
		BaseAssetSymbol:  "BTC-000M",
		QuoteAssetSymbol: "BUSD-000",
		InitPrice:        1000,
	})
	require.Contains(t, result.Log, "quote token is not valid")

	order.BUSDSymbol = "BUSD-000"
	busd, _ := common.NewToken("BUSD", "BUSD-000", 10000000000, nil, false)
	tokenMapper.NewToken(ctx, busd)
	pair := dextypes.NewTradingPair(types.NativeTokenSymbol, "BUSD-000", 1000)
	orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	result = handleListMini(ctx, orderKeeper, tokenMapper, dextypes.ListMiniMsg{
		From:             sdk.AccAddress("testacc"),
		BaseAssetSymbol:  "BTC-000M",
		QuoteAssetSymbol: "BUSD-000",
		InitPrice:        1000,
	})
	require.Equal(t, true, result.IsOK())
}

func TestHandleListMiniWrongBaseSymbol(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, dextypes.ListMiniMsg{
		From:             sdk.AccAddress("testacc"),
		BaseAssetSymbol:  "BTC",
		QuoteAssetSymbol: "BNB",
		InitPrice:        1000,
	})
	//require.Equal(t, result.Code, sdk.ABCICodeOK)
	require.Contains(t, result.Log, "token(BTC) not found")
}

func TestHandleListMiniRight(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, dextypes.ListMiniMsg{
		From:             sdk.AccAddress("testacc"),
		BaseAssetSymbol:  "BTC-000M",
		QuoteAssetSymbol: "BNB",
		InitPrice:        1000,
	})
	require.Equal(t, result.Code, sdk.ABCICodeOK)
}

func TestHandleListTinyRight(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	setupForMini(ctx, tokenMapper, t)
	result := handleListMini(ctx, orderKeeper, tokenMapper, dextypes.ListMiniMsg{
		From:             sdk.AccAddress("testacc"),
		BaseAssetSymbol:  "ETH-000M",
		QuoteAssetSymbol: "BNB",
		InitPrice:        1000,
	})
	require.Equal(t, result.Code, sdk.ABCICodeOK)
}
