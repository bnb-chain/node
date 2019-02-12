package order

import (
	"errors"
	"fmt"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/pkg/errors"
	"math"
	"math/big"

	tmlog "github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/matcheng"
	param "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/wire"
)

type FeeType uint8

const (
	FeeByNativeToken = FeeType(0x01)
	FeeByTradeToken  = FeeType(0x02)

	feeRateDecimals int64 = 6
	nilFeeValue     int64 = -1

	ExpireFeeField       = "ExpireFee"
	ExpireFeeNativeField = "ExpireFeeNative"
	CancelFeeField       = "CancelFee"
	CancelFeeNativeField = "CancelFeeNative"
	FeeRateField         = "FeeRate"
	FeeRateNativeField   = "FeeRateNative"
	IOCExpireFee         = "IOCExpireFee"
	IOCExpireFeeNative   = "IOCExpireFeeNative"
)

var (
	FeeRateMultiplier = big.NewInt(int64(math.Pow10(int(feeRateDecimals))))
)

type FeeManager struct {
	cdc       *wire.Codec
	logger    tmlog.Logger
	FeeConfig FeeConfig
}

func NewFeeManager(cdc *wire.Codec, storeKey sdk.StoreKey, logger tmlog.Logger) *FeeManager {
	return &FeeManager{
		cdc:       cdc,
		logger:    logger,
		FeeConfig: NewFeeConfig(),
	}
}

// UpdateConfig should only happen when Init or in BreatheBlock
func (m *FeeManager) UpdateConfig(feeConfig FeeConfig) error {
	if feeConfig.anyEmpty() {
		return errors.New("invalid FeeConfig")
	}
	m.FeeConfig = feeConfig
	return nil
}

func (m *FeeManager) GetConfig() FeeConfig {
	return m.FeeConfig
}

// Note: the result of `CalcOrderFee` depends on the balances of the acc,
// so the right way of allocation is:
// 1. transfer the "inAsset" to the balance, i.e. call doTransfer()
// 2. call this method
// 3. deduct the fee right away
func (m *FeeManager) CalcOrderFee(balances sdk.Coins, tradeIn sdk.Coin, engines map[string]*matcheng.MatchEng) types.Fee {
	var feeToken sdk.Coin
	inSymbol := tradeIn.Denom
	inAmt := tradeIn.Amount
	if inSymbol == types.NativeTokenSymbol {
		feeToken = sdk.NewCoin(types.NativeTokenSymbol, m.calcTradeFee(inAmt, FeeByNativeToken))
	} else {
		// price against native token
		var amountOfNativeToken int64
		if pair, ok := engines[utils.Assets2TradingPair(inSymbol, types.NativeTokenSymbol)]; ok {
			// XYZ_BNB
			amountOfNativeToken = utils.CalBigNotional(pair.LastTradePrice, inAmt)
		} else {
			// BNB_XYZ
			pair := engines[utils.Assets2TradingPair(types.NativeTokenSymbol, inSymbol)]
			var amount big.Int
			amountOfNativeToken = amount.Div(
				amount.Mul(
					big.NewInt(inAmt),
					big.NewInt(utils.Fixed8One.ToInt64())),
				big.NewInt(pair.LastTradePrice)).Int64()
		}
		feeByNativeToken := m.calcTradeFee(amountOfNativeToken, FeeByNativeToken)
		if balances.AmountOf(types.NativeTokenSymbol) >= feeByNativeToken {
			// have sufficient native token to pay the fees
			feeToken = sdk.NewCoin(types.NativeTokenSymbol, feeByNativeToken)
		} else {
			// no enough NativeToken, use the received tokens as fee
			feeToken = sdk.NewCoin(inSymbol, m.calcTradeFee(inAmt, FeeByTradeToken))
			m.logger.Debug("Not enough native token to pay trade fee", "feeToken", feeToken)
		}
	}

	return types.NewFee(sdk.Coins{feeToken}, types.FeeForProposer)
}

// Note: the result of `CalcFixedFee` depends on the balances of the acc,
// so the right way of allocation is:
// 1. transfer the "inAsset" to the balance, i.e. call doTransfer()
// 2. call this method
// 3. deduct the fee right away
func (m *FeeManager) CalcFixedFee(balances sdk.Coins, eventType transferEventType, inAsset string, engines map[string]*matcheng.MatchEng) types.Fee {
	var feeAmountNative int64
	var feeAmount int64
	if eventType == eventFullyExpire {
		feeAmountNative, feeAmount = m.ExpireFees()
	} else if eventType == eventIOCFullyExpire {
		feeAmountNative, feeAmount = m.IOCExpireFees()
	} else if eventType == eventFullyCancel {
		feeAmountNative, feeAmount = m.CancelFees()
	} else {
		// should not be here
		m.logger.Error("Invalid expire eventType", "eventType", eventType)
		return types.Fee{}
	}

	var feeToken sdk.Coin
	nativeTokenBalance := balances.AmountOf(types.NativeTokenSymbol)
	if nativeTokenBalance >= feeAmountNative || inAsset == types.NativeTokenSymbol {
		feeToken = sdk.NewCoin(types.NativeTokenSymbol, utils.MinInt(feeAmountNative, nativeTokenBalance))
	} else {
		if pair, ok := engines[utils.Assets2TradingPair(inAsset, types.NativeTokenSymbol)]; ok {
			// XYZ_BNB
			var amount big.Int
			feeAmount = amount.Div(
				amount.Mul(
					big.NewInt(feeAmount),
					big.NewInt(utils.Fixed8One.ToInt64())),
				big.NewInt(pair.LastTradePrice)).Int64()
		} else {
			// BNB_XYZ
			pair = engines[utils.Assets2TradingPair(types.NativeTokenSymbol, inAsset)]
			feeAmount = utils.CalBigNotional(pair.LastTradePrice, feeAmount)
		}

		feeAmount = utils.MinInt(feeAmount, balances.AmountOf(inAsset))
		feeToken = sdk.NewCoin(inAsset, feeAmount)
	}

	return types.NewFee(sdk.Coins{feeToken}, types.FeeForProposer)
}

func (m *FeeManager) calcTradeFee(amount int64, feeType FeeType) int64 {
	var feeRate int64
	if feeType == FeeByNativeToken {
		feeRate = m.FeeConfig.FeeRateNative
	} else if feeType == FeeByTradeToken {
		feeRate = m.FeeConfig.FeeRate
	}

	var fee big.Int
	return fee.Div(fee.Mul(big.NewInt(amount), big.NewInt(feeRate)), FeeRateMultiplier).Int64()
}

func (m *FeeManager) ExpireFees() (int64, int64) {
	return m.FeeConfig.ExpireFeeNative, m.FeeConfig.ExpireFee
}

func (m *FeeManager) IOCExpireFees() (int64, int64) {
	return m.FeeConfig.IOCExpireFeeNative, m.FeeConfig.IOCExpireFee
}

func (m *FeeManager) CancelFees() (int64, int64) {
	return m.FeeConfig.CancelFeeNative, m.FeeConfig.CancelFee
}

func (m *FeeManager) ExpireFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.FeeConfig.ExpireFeeNative
	} else if feeType == FeeByTradeToken {
		return m.FeeConfig.ExpireFee
	}

	panic(fmt.Sprintf("invalid feeType: %v", feeType))
}

func (m *FeeManager) IOCExpireFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.FeeConfig.IOCExpireFeeNative
	} else if feeType == FeeByTradeToken {
		return m.FeeConfig.IOCExpireFee
	}

	panic(fmt.Sprintf("invalid feeType: %v", feeType))
}

func (m *FeeManager) CancelFee(feeType FeeType) int64 {
	if feeType == FeeByNativeToken {
		return m.FeeConfig.CancelFeeNative
	} else if feeType == FeeByTradeToken {
		return m.FeeConfig.CancelFee
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

func (config FeeConfig) anyEmpty() bool {
	if config.ExpireFee < 0 ||
		config.ExpireFeeNative < 0 ||
		config.IOCExpireFee < 0 ||
		config.IOCExpireFeeNative < 0 ||
		config.CancelFee < 0 ||
		config.CancelFeeNative < 0 ||
		config.FeeRate < 0 ||
		config.FeeRateNative < 0 {
		return true
	}

	return false
}

func ParamToFeeConfig(feeParams []param.FeeParam) *FeeConfig {
	for _, p := range feeParams {
		if u, ok := p.(*param.DexFeeParam); ok {
			config := FeeConfig{}
			for _, d := range u.DexFeeFields {
				switch d.FeeName {
				case ExpireFeeField:
					config.ExpireFee = d.FeeValue
				case ExpireFeeNativeField:
					config.ExpireFeeNative = d.FeeValue
				case CancelFeeField:
					config.CancelFee = d.FeeValue
				case CancelFeeNativeField:
					config.CancelFeeNative = d.FeeValue
				case FeeRateField:
					config.FeeRate = d.FeeValue
				case FeeRateNativeField:
					config.FeeRateNative = d.FeeValue
				case IOCExpireFee:
					config.IOCExpireFee = d.FeeValue
				case IOCExpireFeeNative:
					config.IOCExpireFeeNative = d.FeeValue
				}
			}
			return &config
		}
	}
	return nil
}
