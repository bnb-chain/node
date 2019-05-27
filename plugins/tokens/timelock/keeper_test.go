package timelock

import (
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkstore "github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/stretchr/testify/require"

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
	ms.MountStoreWithDB(common.TimeLockStoreKey, sdk.StoreTypeIAVL, nil)
	ms.LoadLatestVersion()
	cms := ms.CacheMultiStore()
	return cms
}

func MakeKeeper(cdc *wire.Codec) (auth.AccountKeeper, Keeper) {
	accKeeper := auth.NewAccountKeeper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	ck := bank.NewBaseKeeper(accKeeper)
	codespacer := sdk.NewCodespacer()
	keeper := NewKeeper(cdc, common.TimeLockStoreKey, ck, accKeeper,
		codespacer.RegisterNext(DefaultCodespace), &sdk.Pool{})
	return accKeeper, keeper
}

func TestKeeper_TimeLock(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	ctx.WithBlockTime(time.Now())

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)
	_ = acc.SetCoins(sdk.Coins{
		sdk.NewCoin("BNB", 1000e8),
	}.Sort())
	accKeeper.SetAccount(ctx, acc)

	lockCoins := sdk.Coins{
		sdk.NewCoin("BNB", 900e8),
	}.Sort()

	record, err := keeper.TimeLock(ctx, acc.GetAddress(), "Test", lockCoins, time.Now().Add(1000*time.Second))
	require.Nil(t, err)

	queryRecord, found := keeper.GetTimeLockRecord(ctx, acc.GetAddress(), record.Id)

	require.True(t, found)
	require.Equal(t, queryRecord.Id, record.Id)
	require.Equal(t, queryRecord.LockTime.Unix(), record.LockTime.Unix())
	require.Equal(t, queryRecord.Description, record.Description)
	require.Equal(t, record.Amount, queryRecord.Amount)
}

func TestKeeper_TimeLock_ErrorLockTime(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)
	_ = acc.SetCoins(sdk.Coins{
		sdk.NewCoin("BNB", 1000e8),
	}.Sort())
	accKeeper.SetAccount(ctx, acc)

	lockCoins := sdk.Coins{
		sdk.NewCoin("BNB", 900e8),
	}.Sort()

	_, err := keeper.TimeLock(ctx, acc.GetAddress(), "Test", lockCoins, time.Now().Add(-1000*time.Second))
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "Invalid lock time")
}

func TestKeeper_TimeLock_InsufficentFund(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)

	lockCoins := sdk.Coins{
		sdk.NewCoin("BNB", 900e8),
	}.Sort()

	_, err := keeper.TimeLock(ctx, acc.GetAddress(), "Test", lockCoins, time.Now().Add(1000*time.Second))
	require.NotNil(t, err)
	require.Equal(t, err.Code(), sdk.CodeInsufficientCoins)
}

func TestKeeper_TimeUnlock_RecordNotExist(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)

	err := keeper.TimeUnlock(ctx, acc.GetAddress(), 1)
	require.NotNil(t, err)
	require.Equal(t, err.Code(), CodeTimeLockRecordDoesNotExist)
}

func TestKeeper_TimeUnlock_ErrLockTime(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	ctx.WithBlockTime(time.Now())

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)
	_ = acc.SetCoins(sdk.Coins{
		sdk.NewCoin("BNB", 1000e8),
	}.Sort())
	accKeeper.SetAccount(ctx, acc)

	lockCoins := sdk.Coins{
		sdk.NewCoin("BNB", 900e8),
	}.Sort()

	record, err := keeper.TimeLock(ctx, acc.GetAddress(), "Test", lockCoins, time.Now().Add(1000*time.Second))
	require.Nil(t, err)

	err = keeper.TimeUnlock(ctx, acc.GetAddress(), record.Id)
	require.NotNil(t, err)
	require.Equal(t, err.Code(), CodeCanNotUnlock)
}

func TestKeeper_TimeUnlock_Success(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	ctx.WithBlockTime(time.Now())

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)
	_ = acc.SetCoins(sdk.Coins{
		sdk.NewCoin("BNB", 1000e8),
	}.Sort())
	accKeeper.SetAccount(ctx, acc)

	lockCoins := sdk.Coins{
		sdk.NewCoin("BNB", 900e8),
	}.Sort()

	record, err := keeper.TimeLock(ctx, acc.GetAddress(), "Test", lockCoins, time.Now().Add(1000*time.Second))
	require.Nil(t, err)

	ctx = ctx.WithBlockTime(time.Now().Add(2000 * time.Second))
	err = keeper.TimeUnlock(ctx, acc.GetAddress(), record.Id)
	require.Nil(t, err)
}

func TestKeeper_TimeRelock_RecordNotExist(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)

	err := keeper.TimeRelock(ctx, acc.GetAddress(), 1, TimeLockRecord{})
	require.NotNil(t, err)
	require.Equal(t, err.Code(), CodeTimeLockRecordDoesNotExist)
}

func TestKeeper_TimeRelock_Error(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	ctx.WithBlockTime(time.Now())

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)
	_ = acc.SetCoins(sdk.Coins{
		sdk.NewCoin("BNB", 2000e8),
	}.Sort())
	accKeeper.SetAccount(ctx, acc)

	lockCoins := sdk.Coins{
		sdk.NewCoin("BNB", 900e8),
	}.Sort()

	record, err := keeper.TimeLock(ctx, acc.GetAddress(), "Test", lockCoins, time.Now().Add(1000*time.Second))
	require.Nil(t, err)

	errAmount := sdk.Coins{
		sdk.NewCoin("BNB", 800e8),
	}.Sort()

	err = keeper.TimeRelock(ctx, acc.GetAddress(), record.Id, TimeLockRecord{Amount: errAmount})
	require.NotNil(t, err)
	require.Equal(t, err.Code(), CodeInvalidLockAmount)

	err = keeper.TimeRelock(ctx, acc.GetAddress(), record.Id, TimeLockRecord{LockTime: time.Now().Add(500 * time.Second)})
	require.NotNil(t, err)
	require.Equal(t, err.Code(), CodeInvalidLockTime)

	ctx = ctx.WithBlockTime(time.Now().Add(2000 * time.Second))
	err = keeper.TimeRelock(ctx, acc.GetAddress(), record.Id, TimeLockRecord{LockTime: time.Now().Add(1500 * time.Second)})
	require.NotNil(t, err)
	require.Equal(t, err.Code(), CodeInvalidLockTime)
}

func TestKeeper_TimeRelock(t *testing.T) {
	cdc := MakeCodec()
	accKeeper, keeper := MakeKeeper(cdc)
	cms := MakeCMS(nil)
	logger := log.NewTMLogger(os.Stdout)
	accountCache := getAccountCache(cdc, cms)
	ctx := sdk.NewContext(cms, abci.Header{Time: time.Now()}, sdk.RunTxModeDeliver, logger).WithAccountCache(accountCache)
	ctx.WithBlockTime(time.Now())

	_, acc := testutils.NewAccount(ctx, accKeeper, 0)
	_ = acc.SetCoins(sdk.Coins{
		sdk.NewCoin("BNB", 2000e8),
	}.Sort())
	accKeeper.SetAccount(ctx, acc)

	lockCoins := sdk.Coins{
		sdk.NewCoin("BNB", 900e8),
	}.Sort()

	record, err := keeper.TimeLock(ctx, acc.GetAddress(), "Test", lockCoins, time.Now().Add(1000*time.Second))
	require.Nil(t, err)

	newAmount := sdk.Coins{
		sdk.NewCoin("BNB", 1000e8),
	}.Sort()

	newRecord := TimeLockRecord{
		Description: "New Description",
		Amount:      newAmount,
		LockTime:    time.Now().Add(2000 * time.Second),
	}
	err = keeper.TimeRelock(ctx, acc.GetAddress(), record.Id, newRecord)
	require.Nil(t, err)

	queryRecord, found := keeper.GetTimeLockRecord(ctx, acc.GetAddress(), record.Id)
	require.True(t, found)
	require.Equal(t, newRecord.Description, queryRecord.Description)
	require.Equal(t, newRecord.LockTime.UTC(), queryRecord.LockTime.UTC())
	require.Equal(t, newRecord.Amount, queryRecord.Amount)
}
