package order

import (
	"encoding/binary"
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	feeRateDecimals int64 = 6
)

var expireFeeKey = []byte("ExpireFee")
var iocExpireFeeKey = []byte("IocExpireFee")
var feeRateWithNativeTokenKey = []byte("FeeRateWithNativeToken")
var feeRateKey = []byte("FeeRate")

const nilFeeValue = -1

type FeeConfig struct {
	storeKey               sdk.StoreKey
	expireFee              int64
	iocExpireFee           int64
	feeRateWithNativeToken int64
	feeRate                int64
}

func NewFeeConfig(storeKey sdk.StoreKey) FeeConfig {
	return FeeConfig{
		storeKey:               storeKey,
		expireFee:              nilFeeValue,
		iocExpireFee:           nilFeeValue,
		feeRateWithNativeToken: nilFeeValue,
		feeRate:                nilFeeValue,
	}
}

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

func (config *FeeConfig) setExpireFee(ctx sdk.Context, expireFee int64) {
	store := ctx.KVStore(config.storeKey)
	b := itob(expireFee)
	store.Set(expireFeeKey, b)
	config.expireFee = expireFee
}

func (config *FeeConfig) setIocExpireFee(ctx sdk.Context, iocExpireFee int64) {
	store := ctx.KVStore(config.storeKey)
	b := itob(iocExpireFee)
	store.Set(iocExpireFeeKey, b)
	config.iocExpireFee = iocExpireFee
}

func (config *FeeConfig) setFeeRateWithNativeToken(ctx sdk.Context, feeRateWithNativeToken int64) {
	store := ctx.KVStore(config.storeKey)
	b := itob(feeRateWithNativeToken)
	store.Set(feeRateWithNativeTokenKey, b)
	config.feeRateWithNativeToken = feeRateWithNativeToken
}

func (config *FeeConfig) setFeeRate(ctx sdk.Context, feeRate int64) {
	store := ctx.KVStore(config.storeKey)
	b := itob(feeRate)
	store.Set(feeRateKey, b)
	config.feeRate = feeRate
}

func (config FeeConfig) getExpireFee(ctx sdk.Context) int64 {
	if config.expireFee == nilFeeValue {
		store := ctx.KVStore(config.storeKey)
		config.expireFee = btoi(store.Get(expireFeeKey))
	}

	return config.expireFee
}

func (config FeeConfig) getIocExpireFee(ctx sdk.Context) int64 {
	if config.iocExpireFee == nilFeeValue {
		store := ctx.KVStore(config.storeKey)
		config.iocExpireFee = btoi(store.Get(iocExpireFeeKey))
	}

	return config.iocExpireFee
}

func (config FeeConfig) getFeeRateWithNativeToken(ctx sdk.Context) int64 {
	if config.feeRateWithNativeToken == nilFeeValue {
		store := ctx.KVStore(config.storeKey)
		config.feeRateWithNativeToken = btoi(store.Get(feeRateWithNativeTokenKey))
	}

	return config.feeRateWithNativeToken
}

func (config FeeConfig) getFeeRate(ctx sdk.Context) int64 {
	if config.feeRate == nilFeeValue {
		store := ctx.KVStore(config.storeKey)
		config.feeRate = btoi(store.Get(feeRateKey))
	}

	return config.feeRate
}

// InitGenesis - store the genesis trend
func (config *FeeConfig) InitGenesis(ctx sdk.Context, data TradingGenesis) {
	config.setExpireFee(ctx, data.ExpireFee)
	config.setIocExpireFee(ctx, data.IocExpireFee)
	config.setFeeRateWithNativeToken(ctx, data.FeeRateWithNativeToken)
	config.setFeeRate(ctx, data.FeeRate)
}

func calcFee(amount int64, feeRate int64) int64 {
	var fee big.Int
	return fee.Div(fee.Mul(big.NewInt(amount), big.NewInt(feeRate)), big.NewInt(int64(math.Pow(10, float64(feeRateDecimals))))).Int64()
}
