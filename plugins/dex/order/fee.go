package order

import (
	"math"
	"math/big"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
	expireFeeKey              = []byte("ExpireFee")
	iocExpireFeeKey           = []byte("IOCExpireFee")
	feeRateWithNativeTokenKey = []byte("FeeRateWithNativeToken")
	feeRateKey                = []byte("FeeRate")

	FeeRateMultiplier = big.NewInt(int64(math.Pow(10, float64(feeRateDecimals))))
)

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
	ctx.Logger().Info("Set Expire Fee", "fee", expireFee)
	config.expireFee = expireFee
}

func (config *FeeConfig) SetIOCExpireFee(ctx sdk.Context, iocExpireFee int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(iocExpireFee)
	store.Set(iocExpireFeeKey, b)
	ctx.Logger().Info("Set IOCExpire Fee", "fee", iocExpireFee)
	config.iocExpireFee = iocExpireFee
}

func (config *FeeConfig) SetFeeRateWithNativeToken(ctx sdk.Context, feeRateWithNativeToken int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(feeRateWithNativeToken)
	store.Set(feeRateWithNativeTokenKey, b)
	ctx.Logger().Info("Set Fee Rate with native token", "rate", feeRateWithNativeToken)
	config.feeRateWithNativeToken = feeRateWithNativeToken
}

func (config *FeeConfig) SetFeeRate(ctx sdk.Context, feeRate int64) {
	store := ctx.KVStore(config.storeKey)
	b := config.itob(feeRate)
	store.Set(feeRateKey, b)
	ctx.Logger().Info("Set Fee Rate for tokens", "rate", feeRate)
	config.feeRate = feeRate
}

func (config FeeConfig) ExpireFee(ctx sdk.Context) int64 {
	return config.expireFee
}

func (config FeeConfig) IOCExpireFee(ctx sdk.Context) int64 {
	return config.iocExpireFee
}

func (config FeeConfig) FeeRateWithNativeToken(ctx sdk.Context) int64 {
	return config.feeRateWithNativeToken
}

func (config FeeConfig) FeeRate(ctx sdk.Context) int64 {
	return config.feeRate
}

// either init fee by Init, or by InitGenesis.
func (config *FeeConfig) Init(ctx sdk.Context) {
	store := ctx.KVStore(config.storeKey)
	if bz := store.Get(expireFeeKey); bz != nil {
		config.expireFee = config.btoi(bz)
		config.iocExpireFee = config.btoi(store.Get(iocExpireFeeKey))
		config.feeRateWithNativeToken = config.btoi(store.Get(feeRateWithNativeTokenKey))
		config.feeRate = config.btoi(store.Get(feeRateKey))
	}
	// otherwise, the chain first starts up and InitGenesis would be called.
}

// InitGenesis - store the genesis trend
func (config *FeeConfig) InitGenesis(ctx sdk.Context, data TradingGenesis) {
	ctx.Logger().Info("Setting Genesis Fee/Rate")
	config.SetExpireFee(ctx, data.ExpireFee)
	config.SetIOCExpireFee(ctx, data.IOCExpireFee)
	config.SetFeeRateWithNativeToken(ctx, data.FeeRateWithNativeToken)
	config.SetFeeRate(ctx, data.FeeRate)
}

func (config *FeeConfig) CalcFee(amount int64, feeType FeeType) int64 {
	var feeRate int64
	if feeType == FeeByNativeToken {
		feeRate = config.feeRateWithNativeToken
	} else if feeType == FeeByTradeToken {
		feeRate = config.feeRate
	}

	var fee big.Int
	return fee.Div(fee.Mul(big.NewInt(amount), big.NewInt(feeRate)), FeeRateMultiplier).Int64()
}
