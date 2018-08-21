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

	"math"

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

	tradeAsset := "XYZ"
	quoteAsset := "BNB"
	pair, err := pairMapper.GetTradingPair(ctx, tradeAsset, quoteAsset)
	require.Empty(t, pair)
	require.Error(t, err)

	pair = types.NewTradingPair(tradeAsset, quoteAsset, 1e8)
	pair.TickSize = 1
	pair.LotSize = 1e8
	err = pairMapper.AddTradingPair(ctx, pair)
	require.NoError(t, err)

	pair, err = pairMapper.GetTradingPair(ctx, tradeAsset, quoteAsset)
	require.NoError(t, err)
	require.NotEmpty(t, pair)
	require.Equal(t, tradeAsset, pair.TradeAsset)
	require.Equal(t, quoteAsset, pair.QuoteAsset)
	require.Equal(t, int64(1e8), pair.Price)
	require.Equal(t, int64(1), pair.TickSize)
	require.Equal(t, int64(1e8), pair.LotSize)
}

func TestMapper_Exists(t *testing.T) {
	pairMapper, ctx := setup()

	tradeAsset := "XYZ"
	quoteAsset := "BNB"
	exists := pairMapper.Exists(ctx, tradeAsset, quoteAsset)
	require.False(t, exists)
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair(tradeAsset, quoteAsset, 1e8))
	require.NoError(t, err)
	exists = pairMapper.Exists(ctx, tradeAsset, quoteAsset)
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
	require.Equal(t, "AAA", pairs[0].TradeAsset)
	require.Equal(t, "BBB", pairs[1].TradeAsset)
	require.Equal(t, "CCC", pairs[2].TradeAsset)
}

func TestMapper_ValidateOrder_OrderNotExist(t *testing.T) {
	pairMapper, ctx := setup()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	err = pairMapper.ValidateOrder(ctx, "BBB_BNB", 1e8, 1e8)
	require.Error(t, err)
}

func TestMapper_ValidateOrder_WrongPrice(t *testing.T) {
	pairMapper, ctx := setup()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	err = pairMapper.ValidateOrder(ctx, "AAA_BNB", 1e3+1e2, 1e6)
	require.Error(t, err)
}

func TestMapper_ValidateOrder_WrongQuantity(t *testing.T) {
	pairMapper, ctx := setup()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	err = pairMapper.ValidateOrder(ctx, "AAA_BNB", 1e3, 1e5+1e4)
	require.Error(t, err)
}

func TestMapper_ValidateOrder_Normal(t *testing.T) {
	pairMapper, ctx := setup()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	err = pairMapper.ValidateOrder(ctx, "AAA_BNB", 1e3, 1e5)
	require.NoError(t, err)
}

func TestMapper_ValidateOrder_MaxNotional(t *testing.T) {
	pairMapper, ctx := setup()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	err = pairMapper.ValidateOrder(ctx, "AAA_BNB", (math.MaxInt64-4)-((int64)(math.MaxInt64-4)%1e3),
		(math.MaxInt64-4)-((int64)(math.MaxInt64-4)%1e5))
	require.Error(t, err)
}
