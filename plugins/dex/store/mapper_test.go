package store

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	dextypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

func setup() (TradingPairMapper, sdk.Context) {
	ms, key := setupMultiStore()
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	var cdc = wire.NewCodec()
	cdc.RegisterConcrete(dextypes.TradingPair{}, "dex/TradingPair", nil)
	return NewTradingPairMapper(cdc, key), ctx
}

func setupMultiStore() (sdk.MultiStore, *sdk.KVStoreKey) {
	db := dbm.NewMemDB()
	key := sdk.NewKVStoreKey("pair")
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(key, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	return ms, key
}

func TestMapper_GetAddTradingPair(t *testing.T) {
	pairMapper, ctx := setup()

	baseAsset := "XYZ-000"
	quoteAsset := types.NativeTokenSymbol
	pair, err := pairMapper.GetTradingPair(ctx, baseAsset, quoteAsset)
	require.Empty(t, pair)
	require.Error(t, err)

	pair = dextypes.NewTradingPair(baseAsset, quoteAsset, 1e8)
	pair.TickSize = 1
	pair.LotSize = 1e8
	err = pairMapper.AddTradingPair(ctx, pair)
	require.NoError(t, err)

	pair, err = pairMapper.GetTradingPair(ctx, baseAsset, quoteAsset)
	require.NoError(t, err)
	require.NotEmpty(t, pair)
	require.Equal(t, baseAsset, pair.BaseAssetSymbol)
	require.Equal(t, quoteAsset, pair.QuoteAssetSymbol)
	require.Equal(t, utils.Fixed8(1e8), pair.Price)
	require.Equal(t, utils.Fixed8(1), pair.TickSize)
	require.Equal(t, utils.Fixed8(1e8), pair.LotSize)
}

func TestMapper_Exists(t *testing.T) {
	pairMapper, ctx := setup()

	baseAsset := "XYZ-000"
	quoteAsset := types.NativeTokenSymbol
	exists := pairMapper.Exists(ctx, baseAsset, quoteAsset)
	require.False(t, exists)
	err := pairMapper.AddTradingPair(ctx, dextypes.NewTradingPair(baseAsset, quoteAsset, 1e8))
	require.NoError(t, err)
	exists = pairMapper.Exists(ctx, baseAsset, quoteAsset)
	require.True(t, exists)
}

func TestMapper_ListAllTradingPairs(t *testing.T) {
	pairMapper, ctx := setup()
	err := pairMapper.AddTradingPair(ctx, dextypes.NewTradingPair("AAA-000", "BNB", 1e8))
	require.NoError(t, err)
	pairMapper.AddTradingPair(ctx, dextypes.NewTradingPair("BBB-000", types.NativeTokenSymbol, 1e8))
	require.NoError(t, err)
	pairMapper.AddTradingPair(ctx, dextypes.NewTradingPair("CCC-000", types.NativeTokenSymbol, 1e8))
	require.NoError(t, err)

	pairs := pairMapper.ListAllTradingPairs(ctx)
	require.Len(t, pairs, 3)
	require.Equal(t, "AAA-000", pairs[0].BaseAssetSymbol)
	require.Equal(t, "BBB-000", pairs[1].BaseAssetSymbol)
	require.Equal(t, "CCC-000", pairs[2].BaseAssetSymbol)
}
