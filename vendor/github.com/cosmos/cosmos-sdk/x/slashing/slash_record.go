package slashing

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
)

const (
	DoubleSign byte = iota
	Downtime
)

type SlashRecord struct {
	ConsAddr         []byte
	InfractionType   byte
	InfractionHeight uint64
	SlashHeight      int64
	JailUntil        time.Time
	SlashAmt         int64
	SideChainId      string
}

func (r SlashRecord) HumanReadableString() (string, error) {
	var infraType string
	if r.InfractionType == 0 {
		infraType = "DoubleSign"
	} else if r.InfractionType == 1 {
		infraType = "Downtime"
	}

	var consAddr string
	if len(r.SideChainId) == 0 {
		pk, err := cryptoAmino.PubKeyFromBytes(r.ConsAddr)
		if err != nil {
			return "", err
		}
		consAddr, err = sdk.Bech32ifyConsPub(pk)
		if err != nil {
			return "", err
		}
	} else {

		consAddr = sdk.HexEncode(r.ConsAddr)
	}

	resp := "SlashRecord \n"
	resp += fmt.Sprintf("Consensus Address: %s\n", consAddr)
	resp += fmt.Sprintf("Infraction Type : %s\n", infraType)
	resp += fmt.Sprintf("Infraction Height: %d\n", r.InfractionHeight)
	resp += fmt.Sprintf("Slash Height: %d\n", r.SlashHeight)
	resp += fmt.Sprintf("Jail Until: %v\n", r.JailUntil)
	resp += fmt.Sprintf("Slash Amount: %d\n", r.SlashAmt)
	if len(r.SideChainId) != 0 {
		resp += fmt.Sprintf("Side Chain id: %s\n", r.SideChainId)
	}
	return resp, nil
}

type slashRecordValue struct {
	SlashHeight int64
	JailUntil   time.Time
	SlashAmt    int64
	SideChainId string
}

func MustMarshalSlashRecord(cdc *codec.Codec, record SlashRecord) []byte {
	bz, err := MarshalSlashRecord(cdc, record)
	if err != nil {
		panic(err)
	}
	return bz
}

func MarshalSlashRecord(cdc *codec.Codec, record SlashRecord) ([]byte, error) {
	srv := slashRecordValue{
		SlashHeight: record.SlashHeight,
		JailUntil:   record.JailUntil,
		SlashAmt:    record.SlashAmt,
		SideChainId: record.SideChainId,
	}
	return cdc.MarshalBinaryLengthPrefixed(srv)
}

func MustUnmarshalSlashRecord(cdc *codec.Codec, key []byte, value []byte) SlashRecord {
	sr, err := UnmarshalSlashRecord(cdc, key, value)
	if err != nil {
		panic(err)
	}
	return sr
}

func UnmarshalSlashRecord(cdc *codec.Codec, key []byte, value []byte) (SlashRecord, error) {
	var storeValue slashRecordValue
	if err := cdc.UnmarshalBinaryLengthPrefixed(value, &storeValue); err != nil {
		return SlashRecord{}, err
	}
	keys := key[1:] // remove prefix bytes
	consAddr := keys[:sdk.AddrLen]
	infractionType := keys[sdk.AddrLen : sdk.AddrLen+1]
	infractionHeightBz := keys[sdk.AddrLen+1:]

	infractionHeight := binary.BigEndian.Uint64(infractionHeightBz)
	return SlashRecord{
		ConsAddr:         consAddr,
		InfractionType:   infractionType[0],
		InfractionHeight: infractionHeight,
		SlashHeight:      storeValue.SlashHeight,
		JailUntil:        storeValue.JailUntil,
		SlashAmt:         storeValue.SlashAmt,
		SideChainId:      storeValue.SideChainId,
	}, nil
}

func (k Keeper) setSlashRecord(ctx sdk.Context, record SlashRecord) {
	store := ctx.KVStore(k.storeKey)
	bz := MustMarshalSlashRecord(k.cdc, record)
	store.Set(GetSlashRecordKey(record.ConsAddr, record.InfractionType, record.InfractionHeight), bz)
}

func (k Keeper) getSlashRecord(ctx sdk.Context, consAddr []byte, infractionType byte, infractionHeight uint64) (sr SlashRecord, found bool) {
	store := ctx.KVStore(k.storeKey)
	key := GetSlashRecordKey(consAddr, infractionType, infractionHeight)
	bz := store.Get(key)
	if bz == nil {
		return sr, false
	}
	return MustUnmarshalSlashRecord(k.cdc, key, bz), true
}

func (k Keeper) hasSlashRecord(ctx sdk.Context, consAddr []byte, infractionType byte, infractionHeight uint64) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Get(GetSlashRecordKey(consAddr, infractionType, infractionHeight)) != nil
}

func (k Keeper) getSlashRecordsByConsAddr(ctx sdk.Context, consAddr []byte) (slashRecords []SlashRecord) {
	store := ctx.KVStore(k.storeKey)
	consAddrPrefixKey := GetSlashRecordsByAddrIndexKey(consAddr)
	iterator := sdk.KVStorePrefixIterator(store, consAddrPrefixKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		slashRecord := MustUnmarshalSlashRecord(k.cdc, iterator.Key(), iterator.Value())
		slashRecords = append(slashRecords, slashRecord)
	}
	return
}

func (k Keeper) getSlashRecordsByConsAddrAndType(ctx sdk.Context, consAddr []byte, infractionType byte) (slashRecords []SlashRecord) {
	store := ctx.KVStore(k.storeKey)
	consAddrPrefixKey := GetSlashRecordsByAddrAndTypeIndexKey(consAddr, infractionType)
	iterator := sdk.KVStorePrefixIterator(store, consAddrPrefixKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		slashRecord := MustUnmarshalSlashRecord(k.cdc, iterator.Key(), iterator.Value())
		slashRecords = append(slashRecords, slashRecord)
	}
	return
}
