package tokens

import (
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/fees"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
)

const (
	IssueFee    = 1e10
	BurnFee     = 1e6
	FreezeFee   = 1e6
	TransferFee = 1e6
)

func init() {
	fees.RegisterCalculator(issue.Route, fees.FixedFeeCalculator(IssueFee, types.FeeForAll))
	fees.RegisterCalculator(burn.BurnRoute, fees.FixedFeeCalculator(BurnFee, types.FeeForProposer))
	fees.RegisterCalculator(freeze.FreezeRoute, fees.FixedFeeCalculator(FreezeFee, types.FeeForProposer))
	// TODO: we will rewrite Transfer fees, so put it here temporarily
	fees.RegisterCalculator(bank.MsgSend{}.Type(), fees.FixedFeeCalculator(TransferFee, types.FeeForProposer))
}
