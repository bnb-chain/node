package swap

import (
	"encoding/binary"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/tendermint/tendermint/crypto"
	cmm "github.com/tendermint/tendermint/libs/common"
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

func (kp *Keeper) CreateSwap(ctx sdk.Context, swapID cmm.HexBytes, swap *AtomicSwap) sdk.Error {
	if swap == nil {
		return sdk.ErrInternal("empty atomic swap pointer")
	}
	kvStore := ctx.KVStore(kp.storeKey)
	hashKey := BuildHashKey(swapID)
	if kvStore.Get(hashKey) != nil {
		return ErrDuplicatedSwapID(fmt.Sprintf("Duplicated swapID %v", swapID))
	}
	kvStore.Set(hashKey, kp.cdc.MustMarshalBinaryBare(*swap))

	swapCreatorKey := BuildSwapCreatorKey(swap.From, swap.Index)
	kvStore.Set(swapCreatorKey, swapID)

	swapRecipientKey := BuildSwapRecipientKey(swap.To, swap.Index)
	kvStore.Set(swapRecipientKey, swapID)

	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, uint64(swap.Index+1))
	kvStore.Set(SwapIndexKey, indexBytes)

	return nil
}

func (kp *Keeper) UpdateSwap(ctx sdk.Context, swapID cmm.HexBytes, swap *AtomicSwap) sdk.Error {
	if swap == nil {
		return sdk.ErrInternal("empty atomic swap pointer")
	}
	kvStore := ctx.KVStore(kp.storeKey)
	hashKey := BuildHashKey(swapID)
	if !kvStore.Has(hashKey) {
		return sdk.ErrInternal(fmt.Sprintf("Trying to close non-exist swapID %v", swapID))
	}
	kvStore.Set(hashKey, kp.cdc.MustMarshalBinaryBare(*swap))

	return nil
}

func (kp *Keeper) CloseSwap(ctx sdk.Context, swapID cmm.HexBytes, swap *AtomicSwap) sdk.Error {
	if swap == nil {
		return sdk.ErrInternal("empty atomic swap pointer")
	}
	if swap.ClosedTime <= 0 {
		return sdk.ErrInternal("Missing swap close time")
	}
	kvStore := ctx.KVStore(kp.storeKey)
	hashKey := BuildHashKey(swapID)
	if !kvStore.Has(hashKey) {
		return sdk.ErrInternal(fmt.Sprintf("Trying to close non-exist swapID %v", swapID))
	}
	kvStore.Set(hashKey, kp.cdc.MustMarshalBinaryBare(*swap))

	closeTimeKey := BuildCloseTimeKey(swap.ClosedTime, swap.Index)
	kvStore.Set(closeTimeKey, swapID)

	return nil
}

func (kp *Keeper) DeleteSwap(ctx sdk.Context, swapID cmm.HexBytes, swap *AtomicSwap) sdk.Error {
	if swap == nil {
		return sdk.ErrInternal("empty atomic swap pointer")
	}
	kvStore := ctx.KVStore(kp.storeKey)
	hashKey := BuildHashKey(swapID)
	kvStore.Delete(hashKey)

	swapCreatorKey := BuildSwapCreatorKey(swap.From, swap.Index)
	kvStore.Delete(swapCreatorKey)

	swapRecipientKey := BuildSwapRecipientKey(swap.To, swap.Index)
	kvStore.Delete(swapRecipientKey)

	closeTimeKey := BuildCloseTimeKey(swap.ClosedTime, swap.Index)
	kvStore.Delete(closeTimeKey)

	return nil
}

func (kp *Keeper) DeleteKey(ctx sdk.Context, key []byte) {
	kvStore := ctx.KVStore(kp.storeKey)
	kvStore.Delete(key)
}

func (kp *Keeper) GetSwap(ctx sdk.Context, swapID cmm.HexBytes) *AtomicSwap {
	kvStore := ctx.KVStore(kp.storeKey)

	hashKey := BuildHashKey(swapID)
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

func (kp *Keeper) getIndex(ctx sdk.Context) int64 {
	kvStore := ctx.KVStore(kp.storeKey)
	bz := kvStore.Get(SwapIndexKey)
	if bz == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}
