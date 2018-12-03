package fees

import (
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/BiJie/BinanceChain/common/types"
)

const (
	GovFee = 1e6
)

func init() {
	RegisterCalculator(gov.MsgSubmitProposal{}.Type(), FixedFeeCalculator(GovFee, types.FeeForProposer))
	RegisterCalculator(gov.MsgDeposit{}.Type(), FixedFeeCalculator(GovFee, types.FeeForProposer))
	RegisterCalculator(gov.MsgVote{}.Type(), FreeFeeCalculator())
}
