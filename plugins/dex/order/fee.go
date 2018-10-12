package order

import (
	"math"
	"math/big"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/wire"
)

type FeeType uint8

const (
	FeeByNativeToken = FeeType(0x01)
	FeeByTradeToken  = FeeType(0x02)

	feeRateDecimals int64 = 6
	nilFeeValue     int64 = -1
)

var (
	expireFeeKey     = []byte("ExpireFee")
	iocExpireFeeKey  = []byte("IOCExpireFee")
	feeRateNativeKey = []byte("FeeRateNative")
	feeRateKey       = []byte("FeeRate")

	FeeRateMultiplier = big.NewInt(int64(math.Pow(10, float64(feeRateDecimals))))
)

type FeeConfig struct {
	cdc           *wire.Codec
	mtx           sync.Mutex
	storeKey      sdk.StoreKey
	expireFee     int64
	iocExpireFee  int64
	feeRateNative int64
	feeRate       int64
}

func NewFeeConfig(cdc *wire.Codec, storeKey sdk.StoreKey) FeeConfig {
	return FeeConfig{
		cdc:           cdc,
		storeKey:      storeKey,
		expireFee:     nilFeeValue,
		iocExpireFee:  nilFeeValue,
		feeRateNative: nilFeeValue,
		feeRate:       nilFeeValue,
	}
}

func (config *FeeConfig) itob(num int64) []byte {
	bz, err := config.cdc.MarshalBinaryBare(num)
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
	log.With("module", "dex").Info("Set Expire Fee", "fee", expireFee)
	config.expireFee = expireFee
}

func (config *FeeConfig) SetIOCExpireFee(ctx sdk.Context, iocExpireFee int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(iocExpireFee)
	store.Set(iocExpireFeeKey, b)
	log.With("module", "dex").Info("Set IOCExpire Fee", "fee", iocExpireFee)
	config.iocExpireFee = iocExpireFee
}

func (config *FeeConfig) SetFeeRateNative(ctx sdk.Context, feeRateNative int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(feeRateNative)
	store.Set(feeRateNativeKey, b)
	log.With("module", "dex").Info("Set Fee Rate with native token", "rate", feeRateNative)
	config.feeRateNative = feeRateNative
}

func (config *FeeConfig) SetFeeRate(ctx sdk.Context, feeRate int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(feeRate)
	store.Set(feeRateKey, b)
	log.With("module", "dex").Info("Set Fee Rate for tokens", "rate", feeRate)
	config.feeRate = feeRate
}

func (config FeeConfig) ExpireFee() int64 {
	return config.expireFee
}

func (config FeeConfig) IOCExpireFee() int64 {
	return config.iocExpireFee
}

func (config FeeConfig) FeeRateWithNativeToken() int64 {
	return config.feeRateNative
}

func (config FeeConfig) FeeRate() int64 {
	return config.feeRate
}

// either init fee by Init, or by InitGenesis.
func (config *FeeConfig) Init(ctx sdk.Context) {
	store := ctx.KVStore(config.storeKey)
	if bz := store.Get(expireFeeKey); bz != nil {
		config.expireFee = config.btoi(bz)
		config.iocExpireFee = config.btoi(store.Get(iocExpireFeeKey))
		config.feeRateNative = config.btoi(store.Get(feeRateNativeKey))
		config.feeRate = config.btoi(store.Get(feeRateKey))
		log.With("module", "dex").Info("Initialized fees from storage", "ExpireFee", config.expireFee,
			"IOCExpireFee", config.iocExpireFee, "FeeRateWithNativeToken", config.feeRateNative,
			"FeeRate", config.feeRate)
	}
	// otherwise, the chain first starts up and InitGenesis would be called.
}

// InitGenesis - store the genesis trend
func (config *FeeConfig) InitGenesis(ctx sdk.Context, data TradingGenesis) {
	log.With("module", "dex").Info("Setting Genesis Fee/Rate")
	config.SetExpireFee(ctx, data.ExpireFee)
	config.SetIOCExpireFee(ctx, data.IOCExpireFee)
	config.SetFeeRateNative(ctx, data.FeeRateNative)
	config.SetFeeRate(ctx, data.FeeRate)
}

func (config *FeeConfig) CalcFee(amount int64, feeType FeeType) int64 {
	var feeRate int64
	if feeType == FeeByNativeToken {
		feeRate = config.feeRateNative
	} else if feeType == FeeByTradeToken {
		feeRate = config.feeRate
	}

	var fee big.Int
	return fee.Div(fee.Mul(big.NewInt(amount), big.NewInt(feeRate)), FeeRateMultiplier).Int64()
}
