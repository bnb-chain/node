package order

import (
	"math"
	"testing"

	cstore "github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/wire"
)

func newTradingPairMapper() (store.TradingPairMapper, sdk.Context) {
	ms, key := setupMultiStore()
	ctx := sdk.NewContext(ms, abci.Header{}, false, log.NewNopLogger())
	var cdc = wire.NewCodec()
	cdc.RegisterConcrete(types.TradingPair{}, "dex/TradingPair", nil)
	return store.NewTradingPairMapper(cdc, key), ctx
}

func setupMultiStore() (sdk.MultiStore, *sdk.KVStoreKey) {
	db := dbm.NewMemDB()
	key := sdk.NewKVStoreKey("pair")
	ms := cstore.NewCommitMultiStore(db)
	ms.MountStoreWithDB(key, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	return ms, key
}

func TestHandler_ValidateOrder_OrderNotExist(t *testing.T) {
	pairMapper, ctx := newTradingPairMapper()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	msg := NewOrderMsg{
		Symbol:   "BBB_BNB",
		Price:    1e8,
		Quantity: 1e8,
	}

	err = validateOrder(ctx, pairMapper, msg)
	require.Error(t, err)
}

func TestHandler_ValidateOrder_WrongPrice(t *testing.T) {
	pairMapper, ctx := newTradingPairMapper()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	msg := NewOrderMsg{
		Symbol:   "AAA_BNB",
		Price:    1e3 + 1e2,
		Quantity: 1e6,
	}

	err = validateOrder(ctx, pairMapper, msg)
	require.Error(t, err)
}

func TestHandler_ValidateOrder_WrongQuantity(t *testing.T) {
	pairMapper, ctx := newTradingPairMapper()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	msg := NewOrderMsg{
		Symbol:   "AAA_BNB",
		Price:    1e3,
		Quantity: 1e5 + 1e4,
	}

	err = validateOrder(ctx, pairMapper, msg)
	require.Error(t, err)
}

func TestHandler_ValidateOrder_Normal(t *testing.T) {
	pairMapper, ctx := newTradingPairMapper()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	msg := NewOrderMsg{
		Symbol:   "AAA_BNB",
		Price:    1e3,
		Quantity: 1e5,
	}

	err = validateOrder(ctx, pairMapper, msg)
	require.NoError(t, err)
}

func TestHandler_ValidateOrder_MaxNotional(t *testing.T) {
	pairMapper, ctx := newTradingPairMapper()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA", "BNB", 1e8))
	require.NoError(t, err)

	msg := NewOrderMsg{
		Symbol:   "AAA_BNB",
		Price:    (math.MaxInt64 - 4) - ((int64)(math.MaxInt64-4) % 1e3),
		Quantity: (math.MaxInt64 - 4) - ((int64)(math.MaxInt64-4) % 1e5),
	}

	err = validateOrder(ctx, pairMapper, msg)
	require.Error(t, err)
}
