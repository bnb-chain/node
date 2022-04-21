package fees

import (
	"github.com/cosmos/cosmos-sdk/types"
	param "github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

type FeeCalculator func(msg types.Msg) types.Fee
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

func FixedFeeCalculator(amount int64, feeType types.FeeDistributeType) FeeCalculator {
	if feeType == types.FeeFree {
		return FreeFeeCalculator()
	}
	return func(msg types.Msg) types.Fee {
		return types.NewFee(append(types.Coins{}, types.NewCoin(types.NativeTokenSymbol, amount)), feeType)
	}
}

func FreeFeeCalculator() FeeCalculator {
	return func(msg types.Msg) types.Fee {
		return types.NewFee(append(types.Coins{}), types.FeeFree)
	}
}

var FixedFeeCalculatorGen = func(params param.FeeParam) FeeCalculator {
	if defaultParam, ok := params.(*param.FixedFeeParams); ok {
		if defaultParam.Fee <= 0 || defaultParam.FeeFor == types.FeeFree {
			return FreeFeeCalculator()
		} else {
			return FixedFeeCalculator(defaultParam.Fee, defaultParam.FeeFor)
		}
	} else {
		panic("Generator receive unexpected param type")
	}
}
