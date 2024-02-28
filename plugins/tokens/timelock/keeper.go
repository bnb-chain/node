package timelock

import (
	"fmt"
	"sort"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/tendermint/tendermint/crypto"
	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/bnb-chain/node/common/log"
)

const InitialRecordId = 1

var (
	// bnb prefix address:  bnb1hn8ym9xht925jkncjpf7lhjnax6z8nv24fv2yq
	// tbnb prefix address: tbnb1hn8ym9xht925jkncjpf7lhjnax6z8nv2mu9wy3
	TimeLockCoinsAccAddr = sdk.AccAddress(crypto.AddressHash([]byte("BinanceChainTimeLockCoins")))
)

type Keeper struct {
	ck        bank.Keeper
	ak        auth.AccountKeeper
	storeKey  sdk.StoreKey // The key used to access the store from the Context.
	codespace sdk.CodespaceType
	cdc       *codec.Codec
	logger    tmlog.Logger
}

func NewKeeper(cdc *codec.Codec, key sdk.StoreKey, ck bank.Keeper, ak auth.AccountKeeper, codespace sdk.CodespaceType) Keeper {
	logger := bnclog.With("module", "timelock")
	return Keeper{
		ck:        ck,
		ak:        ak,
		storeKey:  key,
		codespace: codespace,
		cdc:       cdc,
		logger:    logger,
	}
}

func (keeper Keeper) setTimeLockRecord(ctx sdk.Context, addr sdk.AccAddress, record TimeLockRecord) {
	store := ctx.KVStore(keeper.storeKey)
	bz := keeper.cdc.MustMarshalBinaryLengthPrefixed(record)
	store.Set(KeyRecord(addr, record.Id), bz)
}

func (keeper Keeper) deleteTimeLockRecord(ctx sdk.Context, addr sdk.AccAddress, recordId int64) {
	store := ctx.KVStore(keeper.storeKey)
	store.Delete(KeyRecord(addr, recordId))
}

func (keeper Keeper) getTimeLockRecordsIterator(ctx sdk.Context, addr sdk.AccAddress) sdk.Iterator {
	store := ctx.KVStore(keeper.storeKey)
	return sdk.KVStorePrefixIterator(store, KeyRecordSubSpace(addr))
}

func (keeper Keeper) GetTimeLockRecord(ctx sdk.Context, addr sdk.AccAddress, recordId int64) (TimeLockRecord, bool) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyRecord(addr, recordId))
	if bz == nil {
		return TimeLockRecord{}, false
	}

	var record TimeLockRecord
	keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &record)
	return record, true
}

func (keeper Keeper) GetTimeLockRecords(ctx sdk.Context, addr sdk.AccAddress) []TimeLockRecord {
	var records []TimeLockRecord
	iterator := keeper.getTimeLockRecordsIterator(ctx, addr)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		record := TimeLockRecord{}
		keeper.cdc.MustUnmarshalBinaryLengthPrefixed(iterator.Value(), &record)
		records = append(records, record)
	}

	sort.Sort(TimeLockRecords(records))

	return records
}

func (kp *Keeper) GetTimeLockRecordIterator(ctx sdk.Context) (iterator store.Iterator) {
	kvStore := ctx.KVStore(kp.storeKey)
	return sdk.KVStorePrefixIterator(kvStore, []byte{})
}

func (keeper Keeper) getTimeLockId(ctx sdk.Context, from sdk.AccAddress) int64 {
	acc := keeper.ak.GetAccount(ctx, from)
	return acc.GetSequence()
}

func (keeper Keeper) TimeLock(ctx sdk.Context, from sdk.AccAddress, description string, amount sdk.Coins, lockTime time.Time) (TimeLockRecord, sdk.Error) {
	if !lockTime.After(ctx.BlockHeader().Time.Add(MinLockTime)) {
		return TimeLockRecord{}, ErrInvalidLockTime(DefaultCodespace,
			fmt.Sprintf("lock time(%s) should be %d minute(s) after now(%s)", lockTime.UTC().String(),
				MinLockTime/time.Minute, ctx.BlockHeader().Time.Add(MinLockTime).UTC().String()))
	}

	_, err := keeper.ck.SendCoins(ctx, from, TimeLockCoinsAccAddr, amount)
	if err != nil {
		return TimeLockRecord{}, err
	}

	recordId := keeper.getTimeLockId(ctx, from)
	_, found := keeper.GetTimeLockRecord(ctx, from, recordId)
	if found {
		return TimeLockRecord{}, ErrTimeLockRecordAlreadyExist(DefaultCodespace, from, recordId)
	}

	record := TimeLockRecord{
		Id:          recordId,
		Description: description,
		Amount:      amount,
		LockTime:    lockTime,
	}
	keeper.setTimeLockRecord(ctx, from, record)
	return record, nil
}

func (keeper Keeper) TimeUnlock(ctx sdk.Context, from sdk.AccAddress, recordId int64, isBCFusionRefund bool) sdk.Error {
	record, found := keeper.GetTimeLockRecord(ctx, from, recordId)
	if !found {
		return ErrTimeLockRecordDoesNotExist(DefaultCodespace, from, recordId)
	}

	if !isBCFusionRefund && ctx.BlockHeader().Time.Before(record.LockTime) {
		return ErrCanNotUnlock(DefaultCodespace, fmt.Sprintf("lock time(%s) is after now(%s)",
			record.LockTime.UTC().String(), ctx.BlockHeader().Time.UTC().String()))
	}

	_, err := keeper.ck.SendCoins(ctx, TimeLockCoinsAccAddr, from, record.Amount)
	if err != nil {
		return err
	}

	keeper.deleteTimeLockRecord(ctx, from, recordId)
	return nil
}

func (keeper Keeper) TimeRelock(ctx sdk.Context, from sdk.AccAddress, recordId int64, newRecord TimeLockRecord) sdk.Error {
	record, found := keeper.GetTimeLockRecord(ctx, from, recordId)
	if !found {
		return ErrTimeLockRecordDoesNotExist(DefaultCodespace, from, recordId)
	}

	if newRecord.Description != "" {
		record.Description = newRecord.Description
	}

	if !newRecord.Amount.IsZero() {
		if newRecord.Amount.IsEqual(record.Amount) || newRecord.Amount.IsLT(record.Amount) {
			return ErrInvalidLockAmount(DefaultCodespace,
				fmt.Sprintf("new locked coins(%s) should be more than original locked coins(%s)",
					newRecord.Amount.String(), record.Amount.String()))
		}

		amountToIncrease := newRecord.Amount.Minus(record.Amount)
		_, err := keeper.ck.SendCoins(ctx, from, TimeLockCoinsAccAddr, amountToIncrease)
		if err != nil {
			return err
		}

		record.Amount = newRecord.Amount
	}

	if !newRecord.LockTime.Equal(time.Unix(0, 0)) {
		if !newRecord.LockTime.After(record.LockTime) {
			return ErrInvalidLockTime(DefaultCodespace,
				fmt.Sprintf("new lock time(%s) should after original lock time(%s)",
					newRecord.LockTime.UTC().String(), record.LockTime.UTC().String()))
		}

		if !newRecord.LockTime.After(ctx.BlockHeader().Time.Add(MinLockTime)) {
			return ErrInvalidLockTime(DefaultCodespace,
				fmt.Sprintf("new lock time(%s) should be %d minute(s) after now(%s)",
					newRecord.LockTime.UTC().String(), MinLockTime/time.Minute, ctx.BlockHeader().Time.Add(MinLockTime).UTC().String()))
		}

		record.LockTime = newRecord.LockTime
	}

	keeper.setTimeLockRecord(ctx, from, record)
	return nil
}
