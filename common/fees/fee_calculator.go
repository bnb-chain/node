package fees

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

type FeeCalculator func(msg sdk.Msg) types.Fee

var calculators = make(map[string]FeeCalculator)

func RegisterCalculator(msgType string, feeCalc FeeCalculator) {
	calculators[msgType] = feeCalc
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

	return func(msg sdk.Msg) types.Fee {
		return types.NewFee(append(sdk.Coins{}, sdk.NewCoin(types.NativeToken, amount)), feeType)
	}
}

func FreeFeeCalculator() FeeCalculator {
	return func(msg sdk.Msg) types.Fee {
		return types.NewFee(append(sdk.Coins{}), types.FeeFree)
	}
}
