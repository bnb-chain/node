package tokens

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/binance-chain/node/common/utils"

	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/types"
	param "github.com/binance-chain/node/plugins/param/types"
)

var TransferFeeCalculatorGen = fees.FeeCalculatorGenerator(func(params param.FeeParam) fees.FeeCalculator {
	transferFeeParam, ok := params.(*param.TransferFeeParam)
	if !ok {
		panic("Generator received unexpected param type")
	}

	return fees.FeeCalculator(func(msg sdk.Msg) types.Fee {
		transferMsg, ok := msg.(bank.MsgSend)
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
		num := utils.MaxInt(inputNum, outputNum)
		if num >= transferFeeParam.LowerLimitAsMulti {
			if num > types.TokenMaxTotalSupply/transferFeeParam.MultiTransferFee {
				totalFee = types.TokenMaxTotalSupply
			} else {
				totalFee = transferFeeParam.MultiTransferFee * num
			}
		}
		return types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, totalFee)}, transferFeeParam.FeeFor)
	})
})
