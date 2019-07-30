package swap

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/tendermint/tendermint/crypto"
	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/binance-chain/node/common/log"
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
		panic("nil empty swap pointer")
	}

	swapHashKey := GetSwapHashKey(swap.RandomNumberHash)
	if kvStore.Get(swapHashKey) != nil {
		return ErrDuplicatedRandomNumberHash(fmt.Sprintf("Duplicated random number hash %v", swap.RandomNumberHash))
	}
	kvStore.Set(swapHashKey, EncodeAtomicSwap(kp.cdc, *swap))

	swapCreatorKey := GetSwapFromKey(swap.From, swap.RandomNumberHash)
	kvStore.Set(swapCreatorKey, swap.RandomNumberHash)

	swapReceiverKey := GetSwapToKey(swap.To, swap.RandomNumberHash)
	kvStore.Set(swapReceiverKey, swap.RandomNumberHash)

	if swap.ClosedTime > 0 {
		timeKey := GetTimeKey(swap.ClosedTime, swap.RandomNumberHash)
		kvStore.Set(timeKey, swap.RandomNumberHash)
	}

	return nil
}

func (kp *Keeper) UpdateSwap(ctx sdk.Context, swap *AtomicSwap) sdk.Error {
	kvStore := ctx.KVStore(kp.storeKey)
	if swap == nil {
		panic("nil atomic swap pointer")
	}

	swapHashKey := GetSwapHashKey(swap.RandomNumberHash)
	if !kvStore.Has(swapHashKey) {
		return sdk.ErrInternal(fmt.Sprintf("Trying to update non-exist swap %v", swap.RandomNumberHash))
	}
	kvStore.Set(swapHashKey, EncodeAtomicSwap(kp.cdc, *swap))

	if swap.ClosedTime > 0 {
		timeKey := GetTimeKey(swap.ClosedTime, swap.RandomNumberHash)
		kvStore.Set(timeKey, swap.RandomNumberHash)
	}

	return nil
}

func (kp *Keeper) DeleteSwap(ctx sdk.Context, swap *AtomicSwap) sdk.Error {
	kvStore := ctx.KVStore(kp.storeKey)
	if swap == nil {
		panic("nil atomic swap pointer")
	}
	swapHashKey := GetSwapHashKey(swap.RandomNumberHash)
	kvStore.Delete(swapHashKey)

	swapCreatorKey := GetSwapFromKey(swap.From, swap.RandomNumberHash)
	kvStore.Delete(swapCreatorKey)

	swapReceiverKey := GetSwapToKey(swap.To, swap.RandomNumberHash)
	kvStore.Delete(swapReceiverKey)

	if swap.ClosedTime > 0 {
		timeKey := GetTimeKey(swap.ClosedTime, swap.RandomNumberHash)
		kvStore.Delete(timeKey)
	}

	return nil
}

func (kp *Keeper) QuerySwap(ctx sdk.Context, randomNumberHash []byte) *AtomicSwap {
	kvStore := ctx.KVStore(kp.storeKey)

	swapHashKey := GetSwapHashKey(randomNumberHash)
	bz := kvStore.Get(swapHashKey)
	if bz == nil {
		return nil
	}
	swap := DecodeAtomicSwap(kp.cdc, bz)
	return &swap
}

func (kp *Keeper) GetSwapFromIterator(ctx sdk.Context, addr sdk.AccAddress) (iterator store.Iterator) {
	kvStore := ctx.KVStore(kp.storeKey)
	return sdk.KVStorePrefixIterator(kvStore, GetSwapFromQueueKey(addr))
}

func (kp *Keeper) GetSwapToIterator(ctx sdk.Context, addr sdk.AccAddress) (iterator store.Iterator) {
	kvStore := ctx.KVStore(kp.storeKey)
	return sdk.KVStorePrefixIterator(kvStore, GetSwapToQueueKey(addr))
}

func (kp *Keeper) GetSwapTimerIterator(ctx sdk.Context) (iterator store.Iterator) {
	kvStore := ctx.KVStore(kp.storeKey)
	return sdk.KVStorePrefixIterator(kvStore, GetTimeQueueKey())
}

func EncodeAtomicSwap(cdc *codec.Codec, swap AtomicSwap) []byte {
	bz, err := cdc.MarshalBinaryBare(swap)
	if err != nil {
		panic(err)
	}
	return bz
}

func DecodeAtomicSwap(cdc *codec.Codec, bz []byte) (swap AtomicSwap) {
	err := cdc.UnmarshalBinaryBare(bz, &swap)
	if err != nil {
		panic(err)
	}
	return
}
