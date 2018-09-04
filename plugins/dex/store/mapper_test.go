package store

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
)

func setup() (TradingPairMapper, sdk.Context) {
	ms, key := setupMultiStore()
	ctx := sdk.NewContext(ms, abci.Header{}, false, log.NewNopLogger())
	var cdc = wire.NewCodec()
	cdc.RegisterConcrete(types.TradingPair{}, "dex/TradingPair", nil)
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

	baseAsset := "XYZ"
	quoteAsset := "BNB"
	pair, err := pairMapper.GetTradingPair(ctx, baseAsset, quoteAsset)
	require.Empty(t, pair)
	require.Error(t, err)

	pair = types.NewTradingPair(baseAsset, quoteAsset, 1e8)
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

	baseAsset := "XYZ"
	quoteAsset := "BNB"
	exists := pairMapper.Exists(ctx, baseAsset, quoteAsset)
	require.False(t, exists)
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair(baseAsset, quoteAsset, 1e8))
	require.NoError(t, err)
	exists = pairMapper.Exists(ctx, baseAsset, quoteAsset)
	require.True(t, exists)
}

func TestMapper_ListAllTradingPairs(t *testing.T) {
	pairMapper, ctx := setup()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)
	pairMapper.AddTradingPair(ctx, types.NewTradingPair("BBB", "BNB", 1e8))
	require.NoError(t, err)
	pairMapper.AddTradingPair(ctx, types.NewTradingPair("CCC", "BNB", 1e8))
	require.NoError(t, err)

	pairs := pairMapper.ListAllTradingPairs(ctx)
	require.Len(t, pairs, 3)
	require.Equal(t, "AAA", pairs[0].BaseAssetSymbol)
	require.Equal(t, "BBB", pairs[1].BaseAssetSymbol)
	require.Equal(t, "CCC", pairs[2].BaseAssetSymbol)
}
