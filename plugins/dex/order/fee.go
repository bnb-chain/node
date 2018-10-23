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
	cdc       *wire.Codec
	storeKey  sdk.StoreKey
	feeConfig FeeConfig
}

func NewFeeManager(cdc *wire.Codec, storeKey sdk.StoreKey) *FeeManager {
	return &FeeManager{
		cdc:       cdc,
		storeKey:  storeKey,
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
	feeConfig.ExpireFee = data.ExpireFee
	feeConfig.ExpireFeeNative = data.ExpireFeeNative
	feeConfig.IOCExpireFee = data.IOCExpireFee
	feeConfig.IOCExpireFeeNative = data.IOCExpireFeeNative
	feeConfig.CancelFee = data.CancelFee
	feeConfig.CancelFeeNative = data.CancelFeeNative
	feeConfig.FeeRate = data.FeeRate
	feeConfig.FeeRateNative = data.FeeRateNative
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
		feeRate = m.feeConfig.FeeRateNative
	} else if feeType == FeeByTradeToken {
		feeRate = m.feeConfig.FeeRate
	}

	var fee big.Int
	return fee.Div(fee.Mul(big.NewInt(amount), big.NewInt(feeRate)), FeeRateMultiplier).Int64()
}

func (m *FeeManager) ExpireFees() (int64, int64) {
	return m.feeConfig.ExpireFeeNative, m.feeConfig.ExpireFee
}

func (m *FeeManager) IOCExpireFees() (int64, int64) {
	return m.feeConfig.IOCExpireFeeNative, m.feeConfig.IOCExpireFee
}

func (m *FeeManager) CancelFees() (int64, int64) {
	return m.feeConfig.CancelFeeNative, m.feeConfig.CancelFee
}

func (m *FeeManager) ExpireFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.feeConfig.ExpireFeeNative
	} else if feeType == FeeByTradeToken {
		return m.feeConfig.ExpireFee
	}

	panic(fmt.Sprintf("invalid feeType: %v", feeType))
}

func (m *FeeManager) IOCExpireFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.feeConfig.IOCExpireFeeNative
	} else if feeType == FeeByTradeToken {
		return m.feeConfig.IOCExpireFee
	}

	panic(fmt.Sprintf("invalid feeType: %v", feeType))
}

func (m *FeeManager) CancelFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.feeConfig.CancelFeeNative
	} else if feeType == FeeByTradeToken {
		return m.feeConfig.CancelFee
	}

	panic(fmt.Sprintf("invalid feeType: %v", feeType))
}

type FeeConfig struct {
	ExpireFee          int64 `json:"expire_fee"`
	ExpireFeeNative    int64 `json:"expire_fee_native"`
	IOCExpireFee       int64 `json:"ioc_expire_fee"`
	IOCExpireFeeNative int64 `json:"ioc_expire_fee_native"`
	CancelFee          int64 `json:"cancel_fee"`
	CancelFeeNative    int64 `json:"cancel_fee_native"`
	FeeRate            int64 `json:"fee_rate"`
	FeeRateNative      int64 `json:"fee_rate_native"`
}

func NewFeeConfig() FeeConfig {
	return FeeConfig{
		ExpireFee:          nilFeeValue,
		ExpireFeeNative:    nilFeeValue,
		IOCExpireFee:       nilFeeValue,
		IOCExpireFeeNative: nilFeeValue,
		CancelFee:          nilFeeValue,
		CancelFeeNative:    nilFeeValue,
		FeeRate:            nilFeeValue,
		FeeRateNative:      nilFeeValue,
	}
}

func TestFeeConfig() FeeConfig {
	feeConfig := NewFeeConfig()
	feeConfig.FeeRateNative = 500
	feeConfig.FeeRate = 1000
	feeConfig.ExpireFeeNative = 2e4
	feeConfig.ExpireFee = 1e5
	feeConfig.IOCExpireFeeNative = 1e4
	feeConfig.IOCExpireFee = 5e4
	feeConfig.CancelFeeNative = 2e4
	feeConfig.CancelFee = 1e5
	return feeConfig
}

func (config FeeConfig) anyEmpty() bool {
	if config.ExpireFee == nilFeeValue ||
		config.ExpireFeeNative == nilFeeValue ||
		config.IOCExpireFee == nilFeeValue ||
		config.IOCExpireFeeNative == nilFeeValue ||
		config.CancelFee == nilFeeValue ||
		config.CancelFeeNative == nilFeeValue ||
		config.FeeRate == nilFeeValue ||
		config.FeeRateNative == nilFeeValue {
		return true
	}

	return false
}
