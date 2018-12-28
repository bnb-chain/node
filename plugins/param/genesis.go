package param

import (
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"

	sdk "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	param "github.com/BiJie/BinanceChain/plugins/param/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
)

const (
	// Operate fee
	GovFee      = 1e6
	ListingFee  = 1e12
	IssueFee    = 2000e8
	MintFee     = 500e8
	BurnFee     = 1e6
	FreezeFee   = 1e6
	TransferFee = 1e6

	// Dex fee
	ExpireFee          = 1e5
	ExpireFeeNative    = 2e4
	CancelFee          = 1e5
	CancelFeeNative    = 2e4
	FeeRate            = 500
	FeeRateNative      = 250
	IOCExpireFee       = 5e4
	IOCExpireFeeNative = 1e4
)

var DefaultGenesisState = param.GenesisState{
	FeeGenesis: FeeGenesisState,

	//Add other param genesis here
}

// ---------   Definition about fee prams  ------------------- //
var FeeGenesisState = []param.FeeParam{
	// Operate
	&param.FixedFeeParams{gov.MsgSubmitProposal{}.Type(), GovFee, sdk.FeeForProposer},
	&param.FixedFeeParams{gov.MsgDeposit{}.Type(), GovFee, sdk.FeeForProposer},
	&param.FixedFeeParams{gov.MsgVote{}.Type(), sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{list.Route, ListingFee, sdk.FeeForAll},
	&param.FixedFeeParams{order.RouteNewOrder, sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{order.RouteCancelOrder, sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{issue.IssueMsgType, IssueFee, sdk.FeeForAll},
	&param.FixedFeeParams{issue.MintMsgType, MintFee, sdk.FeeForAll},
	&param.FixedFeeParams{burn.BurnRoute, BurnFee, sdk.FeeForProposer},
	&param.FixedFeeParams{freeze.FreezeRoute, FreezeFee, sdk.FeeForProposer},
	&param.FixedFeeParams{bank.MsgSend{}.Type(), TransferFee, sdk.FeeForProposer},

	// Dex
	&param.DexFeeParam{
		DexFeeFields: []param.DexFeeField{
			{order.ExpireFeeField, ExpireFee},
			{order.ExpireFeeNativeField, ExpireFeeNative},
			{order.CancelFeeField, CancelFee},
			{order.CancelFeeNativeField, CancelFeeNative},
			{order.FeeRateField, FeeRate},
			{order.FeeRateNativeField, FeeRateNative},
			{order.IOCExpireFee, IOCExpireFee},
			{order.IOCExpireFeeNative, IOCExpireFeeNative},
		},
	},
}

//----------  End definition about fee param ---------------- //
