package order

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/types"
	cmnUtils "github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/utils"
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

func (m *FeeManager) CalcTradesFee(balances sdk.Coins, tradeTransfers TradeTransfers, engines map[string]*matcheng.MatchEng) sdk.Fee {
	var fees sdk.Fee
	if tradeTransfers == nil {
		return fees
	}
	tradeTransfers.Sort()
	for _, tran := range tradeTransfers {
		fee := m.calcTradeFeeForSingleTransfer(balances, tran, engines)
		tran.Fee = fee
		if tran.IsBuyer() {
			tran.Trade.BuyerFee = &fee
		} else {
			tran.Trade.SellerFee = &fee
		}
		fees.AddFee(fee)
		balances = balances.Minus(fee.Tokens)
	}
	return fees
}

func (m *FeeManager) CalcExpiresFee(balances sdk.Coins, expireType transferEventType, expireTransfers ExpireTransfers, engines map[string]*matcheng.MatchEng, expireTransferHandler func(tran Transfer)) sdk.Fee {
	var fees sdk.Fee
	if expireTransfers == nil {
		return fees
	}
	expireTransfers.Sort()
	for _, tran := range expireTransfers {
		fee := m.CalcFixedFee(balances, expireType, tran.inAsset, engines)
		tran.Fee = fee
		if expireTransferHandler != nil {
			expireTransferHandler(*tran)
		}
		fees.AddFee(fee)
		balances = balances.Minus(fee.Tokens)
	}
	return fees
}

func (m *FeeManager) calcTradeFeeForSingleTransfer(balances sdk.Coins, tran *Transfer, engines map[string]*matcheng.MatchEng) sdk.Fee {
	var feeToken sdk.Coin

	var nativeFee int64
	var isOverflow bool
	if tran.IsNativeIn() {
		// always have enough balance to pay the fee.
		nativeFee = m.calcTradeFee(big.NewInt(tran.in), FeeByNativeToken).Int64()
		return sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, nativeFee)}, sdk.FeeForProposer)
	} else if tran.IsNativeOut() {
		nativeFee, isOverflow = m.calcNativeFee(types.NativeTokenSymbol, tran.out, engines)
	} else {
		nativeFee, isOverflow = m.calcNativeFee(tran.inAsset, tran.in, engines)
	}

	if isOverflow || nativeFee == 0 || nativeFee > balances.AmountOf(types.NativeTokenSymbol) {
		// 1. if the fee is too low and round to 0, we charge by inAsset
		// 2. no enough NativeToken, use the received tokens as fee
		feeToken = sdk.NewCoin(tran.inAsset, m.calcTradeFee(big.NewInt(tran.in), FeeByTradeToken).Int64())
		m.logger.Debug("No enough native token to pay trade fee", "feeToken", feeToken)
	} else {
		// have sufficient native token to pay the fees
		feeToken = sdk.NewCoin(types.NativeTokenSymbol, nativeFee)
	}
	return sdk.NewFee(sdk.Coins{feeToken}, sdk.FeeForProposer)
}

func (m *FeeManager) calcNativeFee(inSymbol string, inQty int64, engines map[string]*matcheng.MatchEng) (fee int64, isOverflow bool) {
	var nativeNotional *big.Int
	if isNativeToken(inSymbol) {
		nativeNotional = big.NewInt(inQty)
	} else {
		// price against native token,
		// both `nativeNotional` and `feeByNativeToken` may overflow when it's a non-BNB pair like ABC_XYZ
		if engine, ok := engines[utils.Assets2TradingPair(inSymbol, types.NativeTokenSymbol)]; ok {
			// XYZ_BNB
			nativeNotional = utils.CalBigNotional(engine.LastTradePrice, inQty)
		} else {
			// BNB_XYZ
			engine := engines[utils.Assets2TradingPair(types.NativeTokenSymbol, inSymbol)]
			var amount big.Int
			nativeNotional = amount.Div(
				amount.Mul(
					big.NewInt(inQty),
					big.NewInt(cmnUtils.Fixed8One.ToInt64())),
				big.NewInt(engine.LastTradePrice))
		}
	}

	nativeFee := m.calcTradeFee(nativeNotional, FeeByNativeToken)
	if nativeFee.IsInt64() {
		return nativeFee.Int64(), false
	}
	return 0, true
}

// DEPRECATED
// Note1: the result of `CalcTradeFeeDeprecated` depends on the balances of the acc,
// so the right way of allocation is:
// 1. transfer the "inAsset" to the balance, i.e. call doTransfer()
// 2. call this method
// 3. deduct the fee right away
//
// Note2: even though the function is called in multiple threads,
// `engines` map would stay the same as no other function may change it in fee calculation stage,
// so no race condition concern
func (m *FeeManager) CalcTradeFee(balances sdk.Coins, tradeIn sdk.Coin, engines map[string]*matcheng.MatchEng) sdk.Fee {
	var feeToken sdk.Coin
	inSymbol := tradeIn.Denom
	inAmt := tradeIn.Amount
	if inSymbol == types.NativeTokenSymbol {
		feeToken = sdk.NewCoin(types.NativeTokenSymbol, m.calcTradeFee(big.NewInt(inAmt), FeeByNativeToken).Int64())
	} else {
		// price against native token,
		// both `amountOfNativeToken` and `feeByNativeToken` may overflow when it's a non-BNB pair like ABC_XYZ
		var amountOfNativeToken *big.Int
		if market, ok := engines[utils.Assets2TradingPair(inSymbol, types.NativeTokenSymbol)]; ok {
			// XYZ_BNB
			amountOfNativeToken = utils.CalBigNotional(market.LastTradePrice, inAmt)
		} else {
			// BNB_XYZ
			market := engines[utils.Assets2TradingPair(types.NativeTokenSymbol, inSymbol)]
			var amount big.Int
			amountOfNativeToken = amount.Div(
				amount.Mul(
					big.NewInt(inAmt),
					big.NewInt(cmnUtils.Fixed8One.ToInt64())),
				big.NewInt(market.LastTradePrice))
		}
		feeByNativeToken := m.calcTradeFee(amountOfNativeToken, FeeByNativeToken)
		if feeByNativeToken.IsInt64() && feeByNativeToken.Int64() != 0 &&
			feeByNativeToken.Int64() <= balances.AmountOf(types.NativeTokenSymbol) {
			// 1. if the fee is too low and round to 0, we charge by inAsset
			// 2. have sufficient native token to pay the fees
			feeToken = sdk.NewCoin(types.NativeTokenSymbol, feeByNativeToken.Int64())
		} else {
			// no enough NativeToken, use the received tokens as fee
			feeToken = sdk.NewCoin(inSymbol, m.calcTradeFee(big.NewInt(inAmt), FeeByTradeToken).Int64())
			m.logger.Debug("No enough native token to pay trade fee", "feeToken", feeToken)
		}
	}

	return sdk.NewFee(sdk.Coins{feeToken}, sdk.FeeForProposer)
}

// Note: the result of `CalcFixedFee` depends on the balances of the acc,
// so the right way of allocation is:
// 1. transfer the "inAsset" to the balance, i.e. call doTransfer()
// 2. call this method
// 3. deduct the fee right away
func (m *FeeManager) CalcFixedFee(balances sdk.Coins, eventType transferEventType, inAsset string, engines map[string]*matcheng.MatchEng) sdk.Fee {
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
		return sdk.Fee{}
	}

	var feeToken sdk.Coin
	nativeTokenBalance := balances.AmountOf(types.NativeTokenSymbol)
	if nativeTokenBalance >= feeAmountNative || inAsset == types.NativeTokenSymbol {
		feeToken = sdk.NewCoin(types.NativeTokenSymbol, cmnUtils.MinInt(feeAmountNative, nativeTokenBalance))
	} else {
		// the amount may overflow int64, so use big.Int instead.
		// TODO: (perf) may remove the big.Int use to improve the performance
		var amount *big.Int
		if market, ok := engines[utils.Assets2TradingPair(inAsset, types.NativeTokenSymbol)]; ok {
			// XYZ_BNB
			var tmp big.Int
			amount = tmp.Div(tmp.Mul(
				big.NewInt(feeAmount),
				big.NewInt(cmnUtils.Fixed8One.ToInt64())),
				big.NewInt(market.LastTradePrice))
		} else {
			// BNB_XYZ
			market = engines[utils.Assets2TradingPair(types.NativeTokenSymbol, inAsset)]
			amount = utils.CalBigNotional(market.LastTradePrice, feeAmount)
		}

		if amount.IsInt64() {
			feeAmount = amount.Int64()
		} else {
			feeAmount = math.MaxInt64
		}
		feeAmount = cmnUtils.MinInt(feeAmount, balances.AmountOf(inAsset))
		feeToken = sdk.NewCoin(inAsset, feeAmount)
	}

	return sdk.NewFee(sdk.Coins{feeToken}, sdk.FeeForProposer)
}

func (m *FeeManager) calcTradeFee(amount *big.Int, feeType FeeType) *big.Int {
	var feeRate int64
	if feeType == FeeByNativeToken {
		feeRate = m.FeeConfig.FeeRateNative
	} else if feeType == FeeByTradeToken {
		feeRate = m.FeeConfig.FeeRate
	}

	// TODO: (Perf) find a more efficient way to replace the big.Int solution.
	var fee big.Int
	return fee.Div(fee.Mul(amount, big.NewInt(feeRate)), FeeRateMultiplier)
}

func isNativeToken(symbol string) bool {
	return symbol == types.NativeTokenSymbol
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
