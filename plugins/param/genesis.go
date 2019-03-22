package param

import (
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"

	sdk "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex/list"
	"github.com/binance-chain/node/plugins/dex/order"
	param "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
)

const (
	// Operate fee
	ProposeFee = 10e8
	DepositFee = 125e3
	ListingFee = 10000e8
	IssueFee   = 2000e8
	MintFee    = 200e8
	BurnFee    = 1e8
	FreezeFee  = 1e6

	// stake fee
	CreateValidatorFee = 10e8
	RemoveValidatorFee = 1e8

	// Transfer fee
	TransferFee       = 125e3
	MultiTransferFee  = 100e3 // discount 80%
	LowerLimitAsMulti = 2

	// Dex fee
	ExpireFee          = 1e5
	ExpireFeeNative    = 2e4
	CancelFee          = 1e5
	CancelFeeNative    = 2e4
	FeeRate            = 1000
	FeeRateNative      = 400
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
	&param.FixedFeeParams{gov.MsgSubmitProposal{}.Type(), ProposeFee, sdk.FeeForProposer},
	&param.FixedFeeParams{gov.MsgDeposit{}.Type(), DepositFee, sdk.FeeForProposer},
	&param.FixedFeeParams{gov.MsgVote{}.Type(), sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{stake.MsgCreateValidator{}.Type(), CreateValidatorFee, sdk.FeeForProposer},
	&param.FixedFeeParams{stake.MsgRemoveValidator{}.Type(), RemoveValidatorFee, sdk.FeeForProposer},
	&param.FixedFeeParams{list.Route, ListingFee, sdk.FeeForAll},
	&param.FixedFeeParams{order.RouteNewOrder, sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{order.RouteCancelOrder, sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{issue.IssueMsgType, IssueFee, sdk.FeeForAll},
	&param.FixedFeeParams{issue.MintMsgType, MintFee, sdk.FeeForAll},
	&param.FixedFeeParams{burn.BurnRoute, BurnFee, sdk.FeeForProposer},
	&param.FixedFeeParams{freeze.FreezeRoute, FreezeFee, sdk.FeeForProposer},

	// Transfer
	&param.TransferFeeParam{
		FixedFeeParams: param.FixedFeeParams{
			MsgType: bank.MsgSend{}.Type(),
			Fee:     TransferFee,
			FeeFor:  sdk.FeeForProposer},
		MultiTransferFee:  MultiTransferFee,
		LowerLimitAsMulti: LowerLimitAsMulti,
	},

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
