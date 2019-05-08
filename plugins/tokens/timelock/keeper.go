package timelock

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/tendermint/tendermint/crypto"
	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/binance-chain/node/common/log"
)

const InitialRecordId = 1

var (
	TimeLockCoinsAccAddr = sdk.AccAddress(crypto.AddressHash([]byte("BinanceChainTimeLockCoins")))
)

type Keeper struct {
	ck        bank.Keeper
	storeKey  sdk.StoreKey // The key used to access the store from the Context.
	codespace sdk.CodespaceType
	cdc       *codec.Codec
	logger    tmlog.Logger
	pool      *sdk.Pool
}

func NewKeeper(cdc *codec.Codec, key sdk.StoreKey, ck bank.Keeper, codespace sdk.CodespaceType, pool *sdk.Pool) Keeper {
	logger := bnclog.With("module", "timelock")
	return Keeper{
		ck:        ck,
		storeKey:  key,
		codespace: codespace,
		cdc:       cdc,
		logger:    logger,
		pool:      pool,
	}
}

func (keeper Keeper) getNextRecordId(ctx sdk.Context, addr sdk.AccAddress) (recordId int64) {
	store := ctx.KVStore(keeper.storeKey)
	key := KeyNextRecordId(addr)
	bz := store.Get(key)
	if bz == nil {
		recordId = InitialRecordId
	} else {
		keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &recordId)
	}

	bz = keeper.cdc.MustMarshalBinaryLengthPrefixed(recordId + 1)
	store.Set(key, bz)

	return recordId
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

func (keeper Keeper) GetTimeLockRecords(ctx sdk.Context, addr sdk.AccAddress, recordId int64) []TimeLockRecord {
	var records []TimeLockRecord
	iterator := keeper.getTimeLockRecordsIterator(ctx, addr)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		record := TimeLockRecord{}
		keeper.cdc.MustUnmarshalBinaryLengthPrefixed(iterator.Value(), &record)
		records = append(records, record)
	}
	// TODO need to ensure they are sorted
	return records
}

func (keeper Keeper) TimeLock(ctx sdk.Context, from sdk.AccAddress, description string, amount sdk.Coins, lockTime time.Time) error {
	_, err := keeper.ck.SendCoins(ctx, from, TimeLockCoinsAccAddr, amount)
	if err != nil {
		return err
	}

	recordId := keeper.getNextRecordId(ctx, from)
	record := TimeLockRecord{
		Id:          recordId,
		Description: description,
		Amount:      amount,
		LockTime:    lockTime,
	}
	keeper.setTimeLockRecord(ctx, from, record)
	return nil
}

func (keeper Keeper) TimeUnlock(ctx sdk.Context, from sdk.AccAddress, recordId int64) error {
	record, found := keeper.GetTimeLockRecord(ctx, from, recordId)
	if !found {
		return fmt.Errorf("time lock record does not exist, addr=%s, recordId=%d", from.String(), recordId)
	}

	_, err := keeper.ck.SendCoins(ctx, TimeLockCoinsAccAddr, from, record.Amount)
	if err != nil {
		return err
	}

	keeper.deleteTimeLockRecord(ctx, from, recordId)
	return nil
}

func (keeper Keeper) TimeRelock(ctx sdk.Context, from sdk.AccAddress, recordId int64, newRecord TimeLockRecord) error {
	record, found := keeper.GetTimeLockRecord(ctx, from, recordId)
	if !found {
		return fmt.Errorf("time lock record does not exist, addr=%s, recordId=%d", from.String(), recordId)
	}

	if newRecord.Description != "" {
		record.Description = newRecord.Description
	}

	if !newRecord.Amount.IsZero() {
		if newRecord.Amount.IsEqual(record.Amount) || newRecord.Amount.IsLT(record.Amount) {
			return fmt.Errorf("new locked coins(%s) should be more than original locked coins(%s)",
				newRecord.Amount.String(), record.Amount.String())
		}
		record.Amount = newRecord.Amount
	}

	if !newRecord.LockTime.IsZero() {
		if !newRecord.LockTime.After(record.LockTime) {
			return fmt.Errorf("new lock time(%s) should after original lock time(%s)",
				newRecord.LockTime.String(), record.LockTime.String())
		}
		record.LockTime = newRecord.LockTime
	}

	keeper.setTimeLockRecord(ctx, from, record)

	return nil
}
