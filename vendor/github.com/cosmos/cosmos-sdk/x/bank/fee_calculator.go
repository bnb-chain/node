package bank

import (
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	param "github.com/cosmos/cosmos-sdk/x/paramHub/types"
	"github.com/tendermint/tendermint/libs/common"
)

var TransferFeeCalculatorGen = fees.FeeCalculatorGenerator(func(params param.FeeParam) fees.FeeCalculator {
	transferFeeParam, ok := params.(*param.TransferFeeParam)
	if !ok {
		panic("Generator received unexpected param type")
	}

	return fees.FeeCalculator(func(msg types.Msg) types.Fee {
		transferMsg, ok := msg.(MsgSend)
		if !ok {
			panic("unexpected msg for TransferFeeCalculator")
		}

		totalFee := transferFeeParam.Fee
		var inputNum int64 = 0
		for _, input := range transferMsg.Inputs {
			inputNum += int64(len(input.Coins))
		}
		var outputNum int64 = 0
		for _, output := range transferMsg.Outputs {
			outputNum += int64(len(output.Coins))
		}
		num := common.MaxInt64(inputNum, outputNum)
		if num >= transferFeeParam.LowerLimitAsMulti {
			if num > types.TokenMaxTotalSupply/transferFeeParam.MultiTransferFee {
				totalFee = types.TokenMaxTotalSupply
			} else {
				totalFee = transferFeeParam.MultiTransferFee * num
			}
		}
		return types.NewFee(types.Coins{types.NewCoin(types.NativeTokenSymbol, totalFee)}, transferFeeParam.FeeFor)
	})
})
