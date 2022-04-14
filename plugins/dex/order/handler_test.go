package order

import (
	"fmt"
	"math"
	"testing"

	cstore "github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/bnb-chain/node/common"
	"github.com/bnb-chain/node/plugins/dex/store"
	"github.com/bnb-chain/node/plugins/dex/types"
	dextypes "github.com/bnb-chain/node/plugins/dex/types"
	"github.com/bnb-chain/node/wire"
)

func setupMultiStore() (sdk.MultiStore, *sdk.KVStoreKey, *sdk.KVStoreKey, *sdk.KVStoreKey) {
	db := dbm.NewMemDB()
	key := sdk.NewKVStoreKey("pair") // TODO: can this be "pairs" as in the constant?
	key2 := sdk.NewKVStoreKey(common.AccountStoreName)
	key3 := sdk.NewKVStoreKey(common.DexStoreName)
	ms := cstore.NewCommitMultiStore(db)
	ms.MountStoreWithDB(key, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(key2, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	return ms, key, key2, key3
}

func setupMappers() (store.TradingPairMapper, auth.AccountKeeper, sdk.Context, *DexKeeper) {
	ms, key, key2, key3 := setupMultiStore()
	var cdc = wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	cdc.RegisterConcrete(types.TradingPair{}, "dex/TradingPair", nil)
	pairMapper := store.NewTradingPairMapper(cdc, key)
	accMapper := auth.NewAccountKeeper(cdc, key2, auth.ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, key2)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	keeper := NewDexKeeper(key3, accMapper, pairMapper, sdk.NewCodespacer().RegisterNext(dextypes.DefaultCodespace), 2, cdc, false)
	return pairMapper, accMapper, ctx, keeper
}

func setupAccount(ctx sdk.Context, accMapper auth.AccountKeeper) (sdk.Account, sdk.AccAddress) {
	saddr := "cosmos1a4y3tjwzgemg0g05fl8ucea0ftkj28l3cfes6q" // TODO: temporary
	addr, err := sdk.AccAddressFromBech32(saddr)
	if err != nil {
		panic(err)
	}
	acc := accMapper.NewAccountWithAddress(ctx, addr)
	err = acc.SetSequence(0)
	if err != nil {
		panic(err)
	}
	accMapper.SetAccount(ctx, acc)
	if err != nil {
		panic(err)
	}
	return acc, addr
}

func TestHandler_ValidateOrder_OrderNotExist(t *testing.T) {
	pairMapper, accMapper, ctx, keeper := setupMappers()
	pair := types.NewTradingPair("AAA-000", "BNB", 1e8)
	err := pairMapper.AddTradingPair(ctx, pair)
	require.NoError(t, err)

	acc, _ := setupAccount(ctx, accMapper)

	msg := NewOrderMsg{
		Symbol:   "BBB-000_BNB",
		Sender:   acc.GetAddress(),
		Price:    1e8,
		Quantity: 1e8,
		Id:       fmt.Sprintf("%X-0", acc.GetAddress()),
	}

	err = validateOrder(ctx, keeper, acc, msg)
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("trading pair not found: %s", msg.Symbol), err.Error())
}

func TestHandler_ValidateOrder_WrongSymbol(t *testing.T) {
	_, _, ctx, keeper := setupMappers()

	msgs := []NewOrderMsg{
		{
			Symbol:   "BNB",
			Price:    1e3,
			Quantity: 1e6,
		},
		{
			Symbol:   "_BNB",
			Price:    1e3,
			Quantity: 1e6,
		},
		{
			Symbol:   "BNB_",
			Price:    1e3,
			Quantity: 1e6,
		},
	}

	for _, msg := range msgs {
		err := validateOrder(ctx, keeper, nil, msg)
		require.Error(t, err)
		require.Equal(t, fmt.Sprintf("Failed to parse trading pair symbol:%s into assets", msg.Symbol), err.Error())
	}
}

func TestHandler_ValidateOrder_WrongPrice(t *testing.T) {
	pairMapper, accMapper, ctx, keeper := setupMappers()
	pair := types.NewTradingPair("AAA-000", "BNB", 1e8)
	err := pairMapper.AddTradingPair(ctx, pair)
	require.NoError(t, err)

	acc, _ := setupAccount(ctx, accMapper)

	msg := NewOrderMsg{
		Symbol:   "AAA-000_BNB",
		Sender:   acc.GetAddress(),
		Price:    1e3 + 1e2,
		Quantity: 1e6,
		Id:       fmt.Sprintf("%X-0", acc.GetAddress()),
	}

	err = validateOrder(ctx, keeper, acc, msg)
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("price(%v) is not rounded to tickSize(%v)", msg.Price, pair.TickSize.ToInt64()), err.Error())
}

func TestHandler_ValidateOrder_WrongQuantity(t *testing.T) {
	pairMapper, accMapper, ctx, keeper := setupMappers()
	pair := types.NewTradingPair("AAA-000", "BNB", 1e8)
	err := pairMapper.AddTradingPair(ctx, pair)
	require.NoError(t, err)

	acc, _ := setupAccount(ctx, accMapper)

	msg := NewOrderMsg{
		Symbol:   "AAA-000_BNB",
		Sender:   acc.GetAddress(),
		Price:    1e3,
		Quantity: 1e5 + 1e4,
		Id:       fmt.Sprintf("%X-0", acc.GetAddress()),
	}

	err = validateOrder(ctx, keeper, acc, msg)
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("quantity(%v) is not rounded to lotSize(%v)", msg.Quantity, pair.LotSize.ToInt64()), err.Error())
}

func TestHandler_ValidateOrder_Normal(t *testing.T) {
	pairMapper, accMapper, ctx, keeper := setupMappers()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA-000", "BNB", 1e8))
	require.NoError(t, err)

	acc, _ := setupAccount(ctx, accMapper)

	msg := NewOrderMsg{
		Symbol:   "AAA-000_BNB",
		Sender:   acc.GetAddress(),
		Price:    1e3,
		Quantity: 1e5,
		Id:       fmt.Sprintf("%X-0", acc.GetAddress()),
	}

	err = validateOrder(ctx, keeper, acc, msg)
	require.NoError(t, err)
}

func TestHandler_ValidateOrder_MaxNotional(t *testing.T) {
	pairMapper, accMapper, ctx, keeper := setupMappers()
	err := pairMapper.AddTradingPair(ctx, types.NewTradingPair("AAA-000", "BNB", 1e8))
	require.NoError(t, err)

	acc, _ := setupAccount(ctx, accMapper)

	msg := NewOrderMsg{
		Symbol:   "AAA-000_BNB",
		Sender:   acc.GetAddress(),
		Price:    (math.MaxInt64 - 4) - ((int64)(math.MaxInt64-4) % 1e3),
		Quantity: (math.MaxInt64 - 4) - ((int64)(math.MaxInt64-4) % 1e5),
		Id:       fmt.Sprintf("%X-0", acc.GetAddress()),
	}

	err = validateOrder(ctx, keeper, acc, msg)
	require.Error(t, err)
	require.Equal(t, "notional value of the order is too large(cannot fit in int64)", err.Error())
}
