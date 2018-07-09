package dex

import (
	"encoding/binary"

	me "github.com/BiJie/BinanceChain/matcheng"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

// Keeper - handlers sets/gets of custom variables for your module
type Keeper struct {
	ck        bank.Keeper
	storeKey  sdk.StoreKey // The key used to access the store from the Context.
	codespace sdk.CodespaceType
	engines   map[string]*me.MatchEng
	cancels   map[string][]string
}

// NewKeeper - Returns the Keeper
func NewKeeper(key sdk.StoreKey, bankKeeper bank.Keeper, codespace sdk.CodespaceType) Keeper {
	return Keeper{ck: bankKeeper, storeKey: key, codespace: codespace,
		engines: make(map[string]*me.MatchEng)}
}

func CreateMatchEng(symbol string) *me.MatchEng {
	//TODO: read lot size
	return me.NewMatchEng(1000, 1, 0.05)
}

func (kp *Keeper) AddOrder(id string, symbol string, side int8, price int64, qty int64, height int64) (err error) {
	eng, ok := kp.engines[symbol]
	if !ok {
		eng = CreateMatchEng(symbol)
		kp.engines[symbol] = eng
	}
	_, err := eng.Book.InsertOrder(id, side, height, price, qty)
	return err
}

// Key to knowing the trend on the streets!
var makerFeeKey = []byte("MakerFee")
var takerFeeKey = []byte("TakerFee")
var feeFactorKey = []byte("FeeFactor")
var maxFeeKey = []byte("MaxFee")
var nativeTokenDiscountKey = []byte("NativeTokenDiscount")
var volumeBucketDurationKey = []byte("VolumeBucketDuration")

func itob(num int64) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(buf, num)
	b := buf[:n]
	return b
}

func btoi(bytes []byte) int64 {
	x, _ := binary.Varint(bytes)
	return x
}

// GetFees - returns the current fees settings
func (k Keeper) GetFees(ctx sdk.Context) (
	makerFee int64, takerFee int64, feeFactor int64, maxFee int64, nativeTokenDiscount int64, volumeBucketDuration int64,
) {
	store := ctx.KVStore(k.storeKey)
	makerFee = btoi(store.Get(makerFeeKey))
	takerFee = btoi(store.Get(takerFeeKey))
	feeFactor = btoi(store.Get(feeFactorKey))
	maxFee = btoi(store.Get(maxFeeKey))
	nativeTokenDiscount = btoi(store.Get(nativeTokenDiscountKey))
	volumeBucketDuration = btoi(store.Get(volumeBucketDurationKey))
	return makerFee, takerFee, feeFactor, maxFee, nativeTokenDiscount, volumeBucketDuration
}

func (k Keeper) setMakerFee(ctx sdk.Context, makerFee int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(makerFee)
	store.Set(makerFeeKey, b)
}

func (k Keeper) setTakerFee(ctx sdk.Context, takerFee int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(takerFee)
	store.Set(takerFeeKey, b)
}

func (k Keeper) setFeeFactor(ctx sdk.Context, feeFactor int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(feeFactor)
	store.Set(feeFactorKey, b)
}

func (k Keeper) setMaxFee(ctx sdk.Context, maxFee int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(maxFee)
	store.Set(maxFeeKey, b)
}

func (k Keeper) setNativeTokenDiscount(ctx sdk.Context, nativeTokenDiscount int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(nativeTokenDiscount)
	store.Set(nativeTokenDiscountKey, b)
}

func (k Keeper) setVolumeBucketDuration(ctx sdk.Context, volumeBucketDuration int64) {
	store := ctx.KVStore(k.storeKey)
	b := itob(volumeBucketDuration)
	store.Set(volumeBucketDurationKey, b)
}

// InitGenesis - store the genesis trend
// func (k Keeper) InitGenesis(ctx sdk.Context, data DexGenesis) error {
// 	k.setMakerFee(ctx, data.MakerFee)
// 	k.setTakerFee(ctx, data.TakerFee)
// 	k.setFeeFactor(ctx, data.FeeFactor)
// 	k.setMaxFee(ctx, data.MaxFee)
// 	k.setNativeTokenDiscount(ctx, data.NativeTokenDiscount)
// 	k.setVolumeBucketDuration(ctx, data.VolumeBucketDuration)
// 	return nil
// }
