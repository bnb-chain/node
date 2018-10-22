package order

import (
	"fmt"
	"github.com/pkg/errors"
	"math"
	"math/big"

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
	feeConfigKey = []byte("FeeConfig")

	FeeRateMultiplier = big.NewInt(int64(math.Pow(10, float64(feeRateDecimals))))
)

type FeeManager struct {
	cdc      *wire.Codec
	storeKey sdk.StoreKey
	feeConfig FeeConfig
}

func NewFeeManager(cdc *wire.Codec, storeKey sdk.StoreKey) *FeeManager {
	return &FeeManager{
		cdc:      cdc,
		storeKey: storeKey,
		feeConfig: NewFeeConfig(),
	}
}

func (m *FeeManager) InitFeeConfig(ctx sdk.Context) {
	feeConfig, err := m.getConfigFromCtx(ctx)
	if err != nil {
		// this will only happen when the chain first starts up, and InitGenesis would be called.
	}

	m.feeConfig = feeConfig
}

func (m *FeeManager) InitGenesis(ctx sdk.Context, data TradingGenesis) {
	feeConfig := NewFeeConfig()
	feeConfig.expireFee = data.ExpireFee
	feeConfig.expireFeeNative = data.ExpireFeeNative
	feeConfig.iocExpireFee = data.IOCExpireFee
	feeConfig.iocExpireFeeNative = data.IOCExpireFeeNative
	feeConfig.cancelFee = data.CancelFee
	feeConfig.cancelFeeNative = data.CancelFeeNative
	feeConfig.feeRate = data.FeeRate
	feeConfig.feeRateNative = data.FeeRateNative
	log.With("module", "dex").Info("Setting Genesis Fee/Rate", "feeConfig", feeConfig)
	err := m.UpdateConfig(ctx, feeConfig)
	if err != nil {
		panic(err)
	}
}

// UpdateConfig should only happen when Init or in BreatheBlock
func (m *FeeManager) UpdateConfig(ctx sdk.Context, feeConfig FeeConfig) error {
	if feeConfig.anyEmpty() {
		return errors.New("invalid feeConfig")
	}

	store := ctx.KVStore(m.storeKey)
	store.Set(feeConfigKey, m.encodeConfig(feeConfig))
	m.feeConfig = feeConfig
	return nil
}

func (m *FeeManager) GetConfig() FeeConfig {
	return m.feeConfig
}

func (m *FeeManager) getConfigFromCtx(ctx sdk.Context) (FeeConfig, error) {
	store := ctx.KVStore(m.storeKey)
	bz := store.Get(feeConfigKey)
	if bz == nil {
		return NewFeeConfig(), errors.New("feeConfig does not exist")
	}

	return m.decodeConfig(bz), nil
}

func (m *FeeManager) encodeConfig(config FeeConfig) []byte {
	bz, err := m.cdc.MarshalBinaryBare(config)
	if err != nil {
		panic(err)
	}

	return bz
}

func (m *FeeManager) decodeConfig(bz []byte) (config FeeConfig) {
	err := m.cdc.UnmarshalBinaryBare(bz, &config)
	if err != nil {
		panic(err)
	}

	return
}

func (m *FeeManager) CalcTradeFee(amount int64, feeType FeeType) int64 {
	var feeRate int64
	if feeType == FeeByNativeToken {
		feeRate = m.feeConfig.feeRateNative
	} else if feeType == FeeByTradeToken {
		feeRate = m.feeConfig.feeRate
	}

	var fee big.Int
	return fee.Div(fee.Mul(big.NewInt(amount), big.NewInt(feeRate)), FeeRateMultiplier).Int64()
}

func (m *FeeManager) ExpireFees() (int64, int64) {
	return m.feeConfig.expireFeeNative, m.feeConfig.expireFee
}

func (m *FeeManager) IOCExpireFees() (int64, int64) {
	return m.feeConfig.iocExpireFeeNative, m.feeConfig.iocExpireFee
}

func (m *FeeManager) CancelFees() (int64, int64) {
	return m.feeConfig.cancelFeeNative, m.feeConfig.cancelFee
}

func (m *FeeManager) ExpireFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.feeConfig.expireFeeNative
	} else if feeType == FeeByTradeToken {
		return m.feeConfig.expireFee
	}

	panic(fmt.Sprintf("invalid feeType: %v", feeType))
}

func (m *FeeManager) IOCExpireFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.feeConfig.iocExpireFeeNative
	} else if feeType == FeeByTradeToken {
		return m.feeConfig.iocExpireFee
	}

	panic(fmt.Sprintf("invalid feeType: %v", feeType))
}

func (m *FeeManager) CancelFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.feeConfig.cancelFeeNative
	} else if feeType == FeeByTradeToken {
		return m.feeConfig.cancelFee
	}

	panic(fmt.Sprintf("invalid feeType: %v", feeType))
}

type FeeConfig struct {
	expireFee          int64
	expireFeeNative    int64
	iocExpireFee       int64
	iocExpireFeeNative int64
	cancelFee          int64
	cancelFeeNative    int64
	feeRate            int64
	feeRateNative      int64
}

func NewFeeConfig() FeeConfig {
	return FeeConfig{
		expireFee: nilFeeValue,
		expireFeeNative: nilFeeValue,
		iocExpireFee: nilFeeValue,
		iocExpireFeeNative:nilFeeValue,
		cancelFee:nilFeeValue,
		cancelFeeNative:nilFeeValue,
		feeRate:nilFeeValue,
		feeRateNative:nilFeeValue,
	}
}

func TestFeeConfig() FeeConfig {
	feeConfig := NewFeeConfig()
	feeConfig.feeRateNative = 500
	feeConfig.feeRate = 1000
	feeConfig.expireFeeNative = 2e4
	feeConfig.expireFee = 1e5
	feeConfig.iocExpireFeeNative = 1e4
	feeConfig.iocExpireFee = 5e4
	feeConfig.cancelFeeNative = 2e4
	feeConfig.cancelFee = 1e5
	return feeConfig
}

func (config FeeConfig) anyEmpty() bool {
	if config.expireFee == nilFeeValue ||
		config.expireFeeNative == nilFeeValue ||
		config.iocExpireFee == nilFeeValue ||
		config.iocExpireFeeNative == nilFeeValue ||
		config.cancelFee == nilFeeValue ||
		config.cancelFeeNative == nilFeeValue ||
		config.feeRate == nilFeeValue ||
		config.feeRateNative == nilFeeValue {
		return true
	}

	return false
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
