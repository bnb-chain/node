package order

import (
	"math"
	"math/big"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/wire"
)

const (
	feeRateDecimals int64 = 6
)

var expireFeeKey = []byte("ExpireFee")
var iocExpireFeeKey = []byte("IOCExpireFee")
var feeRateWithNativeTokenKey = []byte("FeeRateWithNativeToken")
var feeRateKey = []byte("FeeRate")

const nilFeeValue = -1

type FeeConfig struct {
	cdc                    *wire.Codec
	mtx                    sync.Mutex
	storeKey               sdk.StoreKey
	expireFee              int64
	iocExpireFee           int64
	feeRateWithNativeToken int64
	feeRate                int64
}

func NewFeeConfig(cdc *wire.Codec, storeKey sdk.StoreKey) FeeConfig {
	return FeeConfig{
		cdc:                    cdc,
		storeKey:               storeKey,
		expireFee:              nilFeeValue,
		iocExpireFee:           nilFeeValue,
		feeRateWithNativeToken: nilFeeValue,
		feeRate:                nilFeeValue,
	}
}

func (config *FeeConfig) itob(num int64) []byte {
	bz, err := config.cdc.MarshalBinary(num)
	if err != nil {
		panic(err)
	}
	return bz
}

func (config *FeeConfig) btoi(bz []byte) (i int64) {
	err := config.cdc.UnmarshalBinaryBare(bz, &i)
	if err != nil {
		panic(err)
	}
	return
}

// warning: all set methods are not thread safe. They would only be called in DeliverTx
func (config *FeeConfig) SetExpireFee(ctx sdk.Context, expireFee int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(expireFee)
	store.Set(expireFeeKey, b)
	config.expireFee = expireFee
}

func (config *FeeConfig) SetIOCExpireFee(ctx sdk.Context, iocExpireFee int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(iocExpireFee)
	store.Set(iocExpireFeeKey, b)
	config.iocExpireFee = iocExpireFee
}

func (config *FeeConfig) SetFeeRateWithNativeToken(ctx sdk.Context, feeRateWithNativeToken int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(feeRateWithNativeToken)
	store.Set(feeRateWithNativeTokenKey, b)
	config.feeRateWithNativeToken = feeRateWithNativeToken
}

func (config *FeeConfig) SetFeeRate(ctx sdk.Context, feeRate int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(feeRate)
	store.Set(feeRateKey, b)
	config.feeRate = feeRate
}

func (config FeeConfig) ExpireFee(ctx sdk.Context) int64 {
	if config.expireFee == nilFeeValue {
		config.mtx.Lock()
		defer config.mtx.Unlock()
		if config.expireFee == nilFeeValue {
			store := ctx.KVStore(config.storeKey)
			config.expireFee = config.btoi(store.Get(expireFeeKey))
		}
	}
	return config.expireFee
}

func (config FeeConfig) IOCExpireFee(ctx sdk.Context) int64 {
	if config.iocExpireFee == nilFeeValue {
		config.mtx.Lock()
		defer config.mtx.Unlock()
		if config.iocExpireFee == nilFeeValue {
			store := ctx.KVStore(config.storeKey)
			config.iocExpireFee = config.btoi(store.Get(iocExpireFeeKey))
		}
	}
	return config.iocExpireFee
}

func (config FeeConfig) FeeRateWithNativeToken(ctx sdk.Context) int64 {
	if config.feeRateWithNativeToken == nilFeeValue {
		config.mtx.Lock()
		defer config.mtx.Unlock()
		if config.feeRateWithNativeToken == nilFeeValue {
			store := ctx.KVStore(config.storeKey)
			config.feeRateWithNativeToken = config.btoi(store.Get(feeRateWithNativeTokenKey))
		}
	}
	return config.feeRateWithNativeToken
}

func (config FeeConfig) FeeRate(ctx sdk.Context) int64 {
	if config.feeRate == nilFeeValue {
		config.mtx.Lock()
		defer config.mtx.Unlock()
		if config.feeRate == nilFeeValue {
			store := ctx.KVStore(config.storeKey)
			config.feeRate = config.btoi(store.Get(feeRateKey))
		}
	}
	return config.feeRate
}

// InitGenesis - store the genesis trend
func (config *FeeConfig) InitGenesis(ctx sdk.Context, data TradingGenesis) {
	config.SetExpireFee(ctx, data.ExpireFee)
	config.SetIOCExpireFee(ctx, data.IOCExpireFee)
	config.SetFeeRateWithNativeToken(ctx, data.FeeRateWithNativeToken)
	config.SetFeeRate(ctx, data.FeeRate)
}

func calcFee(amount int64, feeRate int64) int64 {
	var fee big.Int
	return fee.Div(fee.Mul(big.NewInt(amount), big.NewInt(feeRate)), big.NewInt(int64(math.Pow(10, float64(feeRateDecimals))))).Int64()
}
