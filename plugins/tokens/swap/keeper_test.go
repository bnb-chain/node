package swap

import (
	"encoding/binary"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkstore "github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/wire"
)

func getAccountCache(cdc *codec.Codec, ms sdk.MultiStore) sdk.AccountCache {
	accountStore := ms.GetKVStore(common.AccountStoreKey)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	return auth.NewAccountCache(accountStoreCache)
}

func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	wire.RegisterCrypto(cdc) // Register crypto.
	bank.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc) // Register Msgs
	types.RegisterWire(cdc)

	return cdc
}

func MakeCMS(memDB *db.MemDB) sdk.CacheMultiStore {
	if memDB == nil {
		memDB = db.NewMemDB()
	}
	ms := sdkstore.NewCommitMultiStore(memDB)
	ms.MountStoreWithDB(common.AccountStoreKey, sdk.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(common.AtomicSwapStoreKey, sdk.StoreTypeIAVL, nil)
	ms.LoadLatestVersion()
	cms := ms.CacheMultiStore()
	return cms
}

func MakeKeeper(cdc *wire.Codec) (auth.AccountKeeper, Keeper) {
	accKeeper := auth.NewAccountKeeper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	ck := bank.NewBaseKeeper(accKeeper)
	codespacer := sdk.NewCodespacer()
	keeper := NewKeeper(cdc, common.AtomicSwapStoreKey, ck, nil, codespacer.RegisterNext(DefaultCodespace))
	return accKeeper, keeper
}

func TestKeeper_CreateSwap(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	ctx = ctx.WithBlockTime(time.Now())

	_, acc1 := testutils.NewAccount(ctx, accKeeper, 10000e8)
	_, acc2 := testutils.NewAccount(ctx, accKeeper, 10000e8)

	toOnOtherChain, _ := hex.DecodeString("491e71b619878c083eaf2894718383c7eb15eb17")
	randomNumberHash, _ := hex.DecodeString("be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167")
	swap := &AtomicSwap{
		From:             acc1.GetAddress(),
		To:               acc2.GetAddress(),
		OutAmount:        sdk.Coin{"BNB", 10000},
		InAmount:         10000,
		ToOnOtherChain:   toOnOtherChain,
		RandomNumberHash: randomNumberHash,
		RandomNumber:     nil,
		Timestamp:        1564471835,
		ExpireHeight:     1000,
		ClosedTime:       0,
		Status:           Open,
	}
	err := keeper.CreateSwap(ctx, swap)
	require.NoError(t, err)
	// Create duplicated swap will tiger error
	err = keeper.CreateSwap(ctx, swap)
	require.Error(t, err)

	querySwap := keeper.QuerySwap(ctx, randomNumberHash)

	require.Equal(t, querySwap.RandomNumberHash, swap.RandomNumberHash)
	require.Equal(t, querySwap.Timestamp, swap.Timestamp)
	require.Equal(t, querySwap.From, swap.From)
	require.Equal(t, querySwap.To, swap.To)
	require.Equal(t, querySwap.Index, int64(0))

	iteratorFrom := keeper.GetSwapFromIterator(ctx, acc1.GetAddress())
	require.True(t, iteratorFrom.Valid())
	require.Equal(t, []byte(swap.RandomNumberHash), iteratorFrom.Value())
	iteratorFrom.Next()
	require.False(t, iteratorFrom.Valid())
	iteratorFrom.Close()

	iteratorTo := keeper.GetSwapToIterator(ctx, acc2.GetAddress())
	require.True(t, iteratorTo.Valid())
	require.Equal(t, []byte(swap.RandomNumberHash), iteratorTo.Value())
	iteratorTo.Next()
	require.False(t, iteratorTo.Valid())
	iteratorTo.Close()

	iteratorTime := keeper.GetSwapTimerIterator(ctx)
	require.False(t, iteratorTime.Valid())
	iteratorTime.Close()
}

func TestKeeper_UpdateSwap(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	ctx.WithBlockTime(time.Now())

	_, acc1 := testutils.NewAccount(ctx, accKeeper, 10000e8)
	_, acc2 := testutils.NewAccount(ctx, accKeeper, 10000e8)

	toOnOtherChain, _ := hex.DecodeString("491e71b619878c083eaf2894718383c7eb15eb17")
	randomNumberHash, _ := hex.DecodeString("be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167")
	swap := &AtomicSwap{
		From:             acc1.GetAddress(),
		To:               acc2.GetAddress(),
		OutAmount:        sdk.Coin{"BNB", 10000},
		InAmount:         10000,
		ToOnOtherChain:   toOnOtherChain,
		RandomNumberHash: randomNumberHash,
		RandomNumber:     nil,
		Timestamp:        1564471835,
		ExpireHeight:     1000,
		ClosedTime:       0,
		Status:           Open,
	}
	err := keeper.CreateSwap(ctx, swap)
	require.NoError(t, err)

	querySwap := keeper.QuerySwap(ctx, randomNumberHash)

	require.Equal(t, querySwap.RandomNumberHash, swap.RandomNumberHash)

	querySwap.RandomNumber, _ = hex.DecodeString("52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649")
	querySwap.ClosedTime = time.Now().Unix()
	querySwap.Status = Completed

	err = keeper.CloseSwap(ctx, querySwap)
	require.NoError(t, err)

	querySwap = keeper.QuerySwap(ctx, randomNumberHash)
	require.Equal(t, querySwap.Status, Completed)

	iteratorTime := keeper.GetSwapTimerIterator(ctx)
	require.True(t, iteratorTime.Valid())
	key := iteratorTime.Key()
	require.Equal(t, 1+Int64Size+Int64Size, len(key))
	swapClosedTime := int64(binary.BigEndian.Uint64(key[1:1+Int64Size]))
	require.Equal(t, querySwap.ClosedTime, swapClosedTime)
	require.Equal(t, []byte(swap.RandomNumberHash), iteratorTime.Value())
	iteratorTime.Next()
	require.False(t, iteratorTime.Valid())
	iteratorTime.Close()
}

func TestKeeper_DeleteSwap(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	ctx.WithBlockTime(time.Now())

	_, acc1 := testutils.NewAccount(ctx, accKeeper, 10000e8)
	_, acc2 := testutils.NewAccount(ctx, accKeeper, 10000e8)

	toOnOtherChain, _ := hex.DecodeString("491e71b619878c083eaf2894718383c7eb15eb17")
	randomNumberHash, _ := hex.DecodeString("be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167")
	swap1 := &AtomicSwap{
		From:             acc1.GetAddress(),
		To:               acc2.GetAddress(),
		OutAmount:        sdk.Coin{"BNB", 10000},
		InAmount:         10000,
		ToOnOtherChain:   toOnOtherChain,
		RandomNumberHash: randomNumberHash,
		RandomNumber:     nil,
		Timestamp:        1564471835,
		ExpireHeight:     1000,
		ClosedTime:       0,
		Status:           Open,
		Index:            0,
	}
	err := keeper.CreateSwap(ctx, swap1)
	require.NoError(t, err)

	toOnOtherChain, _ = hex.DecodeString("491e71b619878c083eaf2894718383c7eb15eb17")
	randomNumberHash, _ = hex.DecodeString("0xba624f3a2c2909f26c9c9ac06d24ae6cab8483ca79cd95e073a8b7bbfc246701")
	swap2 := &AtomicSwap{
		From:             acc1.GetAddress(),
		To:               acc2.GetAddress(),
		OutAmount:        sdk.Coin{"BNB", 10000},
		InAmount:         10000,
		ToOnOtherChain:   toOnOtherChain,
		RandomNumberHash: randomNumberHash,
		RandomNumber:     nil,
		Timestamp:        1564471835,
		ExpireHeight:     1000,
		ClosedTime:       0,
		Status:           Open,
		Index:            1,
	}
	err = keeper.CreateSwap(ctx, swap2)
	require.NoError(t, err)
	require.Equal(t, int64(2), keeper.GetIndex(ctx))

	err = keeper.DeleteSwap(ctx, swap1)
	require.NoError(t, err)
	err = keeper.DeleteSwap(ctx, swap2)
	require.NoError(t, err)

	require.Nil(t, keeper.QuerySwap(ctx, swap1.RandomNumberHash))
	require.Nil(t, keeper.QuerySwap(ctx, swap2.RandomNumberHash))

	iteratorFrom := keeper.GetSwapFromIterator(ctx, acc1.GetAddress())
	require.False(t, iteratorFrom.Valid())
	iteratorFrom.Close()

	iteratorTo := keeper.GetSwapToIterator(ctx, acc2.GetAddress())
	require.False(t, iteratorTo.Valid())
	iteratorTo.Close()

	iteratorTime := keeper.GetSwapTimerIterator(ctx)
	require.False(t, iteratorTime.Valid())
	iteratorTime.Close()

}
