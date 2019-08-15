package swap

import (
	"encoding/binary"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/tendermint/tendermint/crypto"
	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/binance-chain/node/common/log"
)

const (
	OneHour = 3600
	TwoHour = 7200
)

var (
	// bnb prefix address:  bnb1wxeplyw7x8aahy93w96yhwm7xcq3ke4f8ge93u
	// tbnb prefix address: tbnb1wxeplyw7x8aahy93w96yhwm7xcq3ke4ffasp3d
	AtomicSwapCoinsAccAddr = sdk.AccAddress(crypto.AddressHash([]byte("BinanceChainAtomicSwapCoins")))
)

type Keeper struct {
	ck        bank.Keeper
	storeKey  sdk.StoreKey // The key used to access the store from the Context.
	codespace sdk.CodespaceType
	cdc       *codec.Codec
	addrPool  *sdk.Pool
	logger    tmlog.Logger
}

func NewKeeper(cdc *codec.Codec, key sdk.StoreKey, ck bank.Keeper, addrPool *sdk.Pool, codespace sdk.CodespaceType) Keeper {
	logger := bnclog.With("module", "atomicswap")
	return Keeper{
		ck:        ck,
		storeKey:  key,
		codespace: codespace,
		cdc:       cdc,
		addrPool:  addrPool,
		logger:    logger,
	}
}

func (kp *Keeper) CreateSwap(ctx sdk.Context, swap *AtomicSwap) sdk.Error {
	kvStore := ctx.KVStore(kp.storeKey)
	if swap == nil {
		panic("empty atomic swap pointer")
	}

	hashKey := BuildHashKey(swap.RandomNumberHash)
	if kvStore.Get(hashKey) != nil {
		return ErrDuplicatedRandomNumberHash(fmt.Sprintf("Duplicated random number hash %v", swap.RandomNumberHash))
	}
	kvStore.Set(hashKey, kp.cdc.MustMarshalBinaryBare(*swap))

	swapCreatorKey := BuildSwapCreatorKey(swap.From, swap.Index)
	kvStore.Set(swapCreatorKey, swap.RandomNumberHash)

	swapRecipientKey := BuildSwapRecipientKey(swap.To, swap.Index)
	kvStore.Set(swapRecipientKey, swap.RandomNumberHash)

	kp.SetIndex(ctx, swap.Index+1)

	return nil
}

func (kp *Keeper) UpdateSwap(ctx sdk.Context, swap *AtomicSwap) sdk.Error {
	kvStore := ctx.KVStore(kp.storeKey)
	if swap == nil {
		panic("empty atomic swap pointer")
	}

	hashKey := BuildHashKey(swap.RandomNumberHash)
	if !kvStore.Has(hashKey) {
		return sdk.ErrInternal(fmt.Sprintf("Trying to close non-exist swap %v", swap.RandomNumberHash))
	}
	kvStore.Set(hashKey, kp.cdc.MustMarshalBinaryBare(*swap))

	if swap.ClosedTime > 0 {
		closeTimeKey := BuildCloseTimeKey(swap.ClosedTime, swap.Index)
		kvStore.Set(closeTimeKey, swap.RandomNumberHash)
	}

	return nil
}

func (kp *Keeper) CloseSwap(ctx sdk.Context, swap *AtomicSwap) sdk.Error {
	kvStore := ctx.KVStore(kp.storeKey)
	if swap == nil {
		panic("empty atomic swap pointer")
	}
	if swap.ClosedTime <= 0 {
		return sdk.ErrInternal("Missing swap close time")
	}

	hashKey := BuildHashKey(swap.RandomNumberHash)
	if !kvStore.Has(hashKey) {
		return sdk.ErrInternal(fmt.Sprintf("Trying to close non-exist swap %v", swap.RandomNumberHash))
	}
	kvStore.Set(hashKey, kp.cdc.MustMarshalBinaryBare(*swap))

	closeTimeKey := BuildCloseTimeKey(swap.ClosedTime, swap.Index)
	kvStore.Set(closeTimeKey, swap.RandomNumberHash)

	return nil
}

func (kp *Keeper) DeleteSwap(ctx sdk.Context, swap *AtomicSwap) sdk.Error {
	kvStore := ctx.KVStore(kp.storeKey)
	if swap == nil {
		panic("nil atomic swap pointer")
	}
	hashKey := BuildHashKey(swap.RandomNumberHash)
	kvStore.Delete(hashKey)

	swapCreatorKey := BuildSwapCreatorKey(swap.From, swap.Index)
	kvStore.Delete(swapCreatorKey)

	swapRecipientKey := BuildSwapRecipientKey(swap.To, swap.Index)
	kvStore.Delete(swapRecipientKey)

	closeTimeKey := BuildCloseTimeKey(swap.ClosedTime, swap.Index)
	kvStore.Delete(closeTimeKey)

	return nil
}

func (kp *Keeper) QuerySwap(ctx sdk.Context, randomNumberHash []byte) *AtomicSwap {
	kvStore := ctx.KVStore(kp.storeKey)

	hashKey := BuildHashKey(randomNumberHash)
	bz := kvStore.Get(hashKey)
	if bz == nil {
		return nil
	}
	var swap AtomicSwap
	kp.cdc.MustUnmarshalBinaryBare(bz, &swap)
	return &swap
}

func (kp *Keeper) GetSwapCreatorIterator(ctx sdk.Context, addr sdk.AccAddress) (iterator store.Iterator) {
	kvStore := ctx.KVStore(kp.storeKey)
	return sdk.KVStorePrefixIterator(kvStore, BuildSwapCreatorQueueKey(addr))
}

func (kp *Keeper) GetSwapRecipientIterator(ctx sdk.Context, addr sdk.AccAddress) (iterator store.Iterator) {
	kvStore := ctx.KVStore(kp.storeKey)
	return sdk.KVStorePrefixIterator(kvStore, BuildSwapRecipientQueueKey(addr))
}

func (kp *Keeper) GetSwapCloseTimeIterator(ctx sdk.Context) (iterator store.Iterator) {
	kvStore := ctx.KVStore(kp.storeKey)
	return sdk.KVStorePrefixIterator(kvStore, BuildCloseTimeQueueKey())
}

func (kp *Keeper) GetIndex(ctx sdk.Context) int64 {
	kvStore := ctx.KVStore(kp.storeKey)
	bz := kvStore.Get(SwapIndexKey)
	if bz == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}

func (kp *Keeper) SetIndex(ctx sdk.Context, index int64) {
	kvStore := ctx.KVStore(kp.storeKey)
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, uint64(index))
	kvStore.Set(SwapIndexKey, value)
}
