package tokens

import (
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/tx"
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
	tx.RegisterCalculator(issue.Route, tx.FixedFeeCalculator(IssueFee, types.FeeForAll))
	tx.RegisterCalculator(burn.Route, tx.FixedFeeCalculator(BurnFee, types.FeeForProposer))
	tx.RegisterCalculator(freeze.RouteFreeze, tx.FixedFeeCalculator(FreezeFee, types.FeeForProposer))
	// TODO: we will rewrite Transfer tx, so put it here temporarily
	tx.RegisterCalculator(bank.MsgSend{}.Type(), tx.FixedFeeCalculator(TransferFee, types.FeeForProposer))
}
