package store

import (
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/common/utils"
	dextypes "github.com/bnb-chain/node/plugins/dex/types"
	"github.com/bnb-chain/node/wire"
)

func setup() (TradingPairMapper, sdk.Context) {
	ms, key := setupMultiStore()
	ctx := sdk.NewContext(ms, abci.Header{Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger())
	var cdc = wire.NewCodec()
	cdc.RegisterConcrete(dextypes.TradingPair{}, "dex/TradingPair", nil)
	cdc.RegisterConcrete(RecentPrice{}, "dex/RecentPrice", nil)
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
	require.Equal(t, utils.Fixed8(1e8), pair.ListPrice)
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

func TestMapper_DeleteTradingPair_Failed(t *testing.T) {
	pairMapper, ctx := setup()

	baseAsset := "XYZ-000"
	quoteAsset := types.NativeTokenSymbol
	err := pairMapper.DeleteTradingPair(ctx, baseAsset, quoteAsset)
	require.Error(t, err)
}

func TestMapper_DeleteTradingPair_Succeed(t *testing.T) {
	pairMapper, ctx := setup()

	baseAsset := "XYZ-000"
	quoteAsset := types.NativeTokenSymbol
	err := pairMapper.AddTradingPair(ctx, dextypes.NewTradingPair(baseAsset, quoteAsset, 1e8))
	require.NoError(t, err)
	err = pairMapper.DeleteTradingPair(ctx, baseAsset, quoteAsset)
	require.NoError(t, err)
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

func TestMapper_UpdateRecentPrices(t *testing.T) {
	pairMapper, ctx := setup()
	for i := 0; i < 3000; i++ {
		lastPrices := make(map[string]int64, 1)
		lastPrices["ABC"] = 10
		ctx = ctx.WithBlockHeight(int64(2 * (i + 1)))
		pairMapper.UpdateRecentPrices(ctx, 2, 5, lastPrices)
	}

	allRecentPrices := pairMapper.GetRecentPrices(ctx, 2, 5)
	require.Equal(t, int64(5), allRecentPrices["ABC"].Count())
	require.Equal(t, []interface{}{int64(10), int64(10), int64(10), int64(10), int64(10)}, allRecentPrices["ABC"].Elements())
}

func TestMapper_DeleteRecentPrices(t *testing.T) {
	const numPricesStored = 5
	const pairNum = 3
	const priceStoreEvery = 2
	pairMapper, ctx := setup()
	for i := 0; i < 30; i++ {
		lastPrices := make(map[string]int64, pairNum)
		lastPrices["ABC_BNB"] = 10
		lastPrices["ABC_EFG"] = 3
		lastPrices["EFG_BNB"] = 3
		ctx = ctx.WithBlockHeight(int64(priceStoreEvery * (i + 1)))
		pairMapper.UpdateRecentPrices(ctx, priceStoreEvery, numPricesStored, lastPrices)
	}

	allRecentPrices := pairMapper.GetRecentPrices(ctx, priceStoreEvery, numPricesStored)
	require.Equal(t, 3, len(allRecentPrices))
	require.Equal(t, int64(5), allRecentPrices["ABC_BNB"].Count())
	require.Equal(t, []interface{}{int64(10), int64(10), int64(10), int64(10), int64(10)}, allRecentPrices["ABC_BNB"].Elements())
	require.Equal(t, int64(5), allRecentPrices["ABC_EFG"].Count())
	require.Equal(t, []interface{}{int64(3), int64(3), int64(3), int64(3), int64(3)}, allRecentPrices["ABC_EFG"].Elements())
	require.Equal(t, int64(5), allRecentPrices["ABC_EFG"].Count())
	require.Equal(t, []interface{}{int64(3), int64(3), int64(3), int64(3), int64(3)}, allRecentPrices["EFG_BNB"].Elements())

	pairMapper.DeleteRecentPrices(ctx, "ABC_EFG")
	allRecentPrices = pairMapper.GetRecentPrices(ctx, priceStoreEvery, numPricesStored)
	require.Equal(t, 2, len(allRecentPrices))
	require.Equal(t, int64(5), allRecentPrices["ABC_BNB"].Count())
	require.Equal(t, []interface{}{int64(10), int64(10), int64(10), int64(10), int64(10)}, allRecentPrices["ABC_BNB"].Elements())
	require.Equal(t, int64(5), allRecentPrices["EFG_BNB"].Count())
	require.Equal(t, []interface{}{int64(3), int64(3), int64(3), int64(3), int64(3)}, allRecentPrices["EFG_BNB"].Elements())
}

func TestMapper_DeleteOneRecentPrices(t *testing.T) {
	const numPricesStored = 10
	const pairNum = 1
	const priceStoreEvery = 2
	pairMapper, ctx := setup()
	for i := 0; i < numPricesStored; i++ {
		lastPrices := make(map[string]int64, pairNum)
		if i < 5 {
			lastPrices["ABC_BNB"] = 10
		}
		ctx = ctx.WithBlockHeight(int64(2 * (i + 1)))
		pairMapper.UpdateRecentPrices(ctx, priceStoreEvery, numPricesStored, lastPrices)
	}
	allRecentPrices := pairMapper.GetRecentPrices(ctx, priceStoreEvery, numPricesStored)
	require.Equal(t, 1, len(allRecentPrices))
	require.Equal(t, int64(5), allRecentPrices["ABC_BNB"].Count())
	require.Equal(t, []interface{}{int64(10), int64(10), int64(10), int64(10), int64(10)}, allRecentPrices["ABC_BNB"].Elements())

	pairMapper.DeleteRecentPrices(ctx, "ABC_BNB")
	allRecentPrices = pairMapper.GetRecentPrices(ctx, priceStoreEvery, numPricesStored)
	require.Equal(t, 0, len(allRecentPrices))

	//allowed to delete again
	pairMapper.DeleteRecentPrices(ctx, "ABC_BNB")
	allRecentPrices = pairMapper.GetRecentPrices(ctx, priceStoreEvery, numPricesStored)
	require.Equal(t, 0, len(allRecentPrices))
}

func BenchmarkMapper_DeleteRecentPrices(b *testing.B) {
	db, pairMapper, ctx := setupForBenchTest()
	defer db.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pairMapper.DeleteRecentPrices(ctx, strconv.Itoa(i)+"_"+strconv.Itoa(i))
	}
}

func setupForBenchTest() (dbm.DB, TradingPairMapper, sdk.Context) {
	const numPricesStored = 2000
	const pairNum = 500
	const priceStoreEvery = 1000
	db, ms, key := setupLevelDbMultiStore()
	ctx := sdk.NewContext(ms, abci.Header{Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger())
	var cdc = wire.NewCodec()
	cdc.RegisterConcrete(dextypes.TradingPair{}, "dex/TradingPair", nil)
	cdc.RegisterConcrete(RecentPrice{}, "dex/RecentPrice", nil)
	pairMapper := NewTradingPairMapper(cdc, key)
	for i := 0; i < pairNum; i++ {
		tradingPair := dextypes.NewTradingPair(strconv.Itoa(i), strconv.Itoa(i), 102000)
		pairMapper.AddTradingPair(ctx, tradingPair)
	}

	for i := 0; i < numPricesStored; i++ {
		lastPrices := make(map[string]int64, pairNum)
		for j := 0; j < pairNum; j++ {
			lastPrices[strconv.Itoa(j)+"_"+strconv.Itoa(j)] = 8
		}
		ctx = ctx.WithBlockHeight(int64(priceStoreEvery * (i + 1)))
		pairMapper.UpdateRecentPrices(ctx, priceStoreEvery, numPricesStored, lastPrices)
	}

	return db, pairMapper, ctx
}

func setupLevelDbMultiStore() (dbm.DB, sdk.MultiStore, *sdk.KVStoreKey) {
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	db, err := db.NewGoLevelDB("test", path.Join(basePath, "data"))
	if err != nil {
		panic(err)
	}
	key := sdk.NewKVStoreKey("pair")
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(key, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	return db, ms, key
}
