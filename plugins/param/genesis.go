package param

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/binance-chain/node/plugins/dex/list"
	"github.com/binance-chain/node/plugins/dex/order"
	param "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
)

const (
	// Operate fee
	ProposeFee     = 10e8
	DepositFee     = 125e3
	SideProposeFee = 10e8
	SideDepositFee = 125e3
	SideVoteFee    = 1e8
	ListingFee     = 2000e8
	IssueFee       = 1000e8
	MintFee        = 200e8
	BurnFee        = 1e8
	FreezeFee      = 1e6
	TimeLockFee    = 1e6
	TimeUnlockFee  = 1e6
	TimeRelockFee  = 1e6

	SetAccountFlagsFee = 1e8

	HTLTFee        = 37500
	DepositHTLTFee = 37500
	ClaimHTLTFee   = 37500
	RefundHTLTFee  = 37500

	// stake fee
	CreateValidatorFee          = 10e8
	RemoveValidatorFee          = 1e8
	CreateSideChainValidatorFee = 10e8
	EditSideChainValidatorFee   = 1e8
	SideChainDelegateFee        = 1e5
	SideChainRedelegateFee      = 3e5
	SideChainUndelegateFee      = 2e5

	// slashing fee
	BscSubmitEvidenceFee = 10e8
	SideChainUnjail      = 1e8

	// Transfer fee
	TransferFee       = 62500
	MultiTransferFee  = 50000 // discount 80%
	LowerLimitAsMulti = 2

	// Dex fee
	ExpireFee          = 5e4
	ExpireFeeNative    = 1e4
	CancelFee          = 5e4
	CancelFeeNative    = 1e4
	FeeRate            = 1000
	FeeRateNative      = 400
	IOCExpireFee       = 25e3
	IOCExpireFeeNative = 5e3
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
