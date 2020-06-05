package fees

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
	param "github.com/binance-chain/node/plugins/param/types"
)

type FeeCalculator func(msg sdk.Msg) sdk.Fee
type FeeCalculatorGenerator func(params param.FeeParam) FeeCalculator

var calculators = make(map[string]FeeCalculator)
var CalculatorsGen = make(map[string]FeeCalculatorGenerator)

func RegisterCalculator(msgType string, feeCalc FeeCalculator) {
	calculators[msgType] = feeCalc
}

func GetCalculatorGenerator(msgType string) FeeCalculatorGenerator {
	return CalculatorsGen[msgType]
}

func GetCalculator(msgType string) FeeCalculator {
	return calculators[msgType]
}

func UnsetAllCalculators() {
	for key := range calculators {
		delete(calculators, key)
	}
}

func FixedFeeCalculator(amount int64, feeType sdk.FeeDistributeType) FeeCalculator {
	if feeType == sdk.FeeFree {
		return FreeFeeCalculator()
	}
	return func(msg sdk.Msg) sdk.Fee {
		return sdk.NewFee(append(sdk.Coins{}, sdk.NewCoin(types.NativeTokenSymbol, amount)), feeType)
	}
}

func FreeFeeCalculator() FeeCalculator {
	return func(msg sdk.Msg) sdk.Fee {
		return sdk.NewFee(append(sdk.Coins{}), sdk.FeeFree)
	}
}

var FixedFeeCalculatorGen = func(params param.FeeParam) FeeCalculator {
	if defaultParam, ok := params.(*param.FixedFeeParams); ok {
		if defaultParam.Fee <= 0 || defaultParam.FeeFor == sdk.FeeFree {
			return FreeFeeCalculator()
		} else {
			return FixedFeeCalculator(defaultParam.Fee, defaultParam.FeeFor)
		}
	} else {
		panic("Generator receive unexpected param type")
	}

}
