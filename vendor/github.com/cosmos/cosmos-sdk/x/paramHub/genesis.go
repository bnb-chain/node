package paramHub

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	param "github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

const (
	// Operate fee
	ProposeFee           = 10e8
	DepositFee           = 125e3
	SideProposeFee       = 10e8
	SideDepositFee       = 125e3
	SideVoteFee          = 1e8
	ListingFee           = 2000e8
	IssueFee             = 1000e8
	MintFee              = 200e8
	BurnFee              = 1e8
	FreezeFee            = 1e6
	TimeLockFee          = 1e6
	TimeUnlockFee        = 1e6
	TimeRelockFee        = 1e6
	TransferOwnershipFee = 1e6

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

	// cross chain
	CrossBindFee        = 1e8
	CrossUnbindFee      = 1e8
	CrossTransferOutFee = 2e4

	CrossTransferOutRelayFee = 1e5
	CrossBindRelayFee        = 2e6
	CrossUnbindRelayFee      = 2e6

	//MiniToken fee
	TinyIssueFee   = 2e8
	MiniIssueFee   = 3e8
	MiniSetUriFee  = 37500
	MiniListingFee = 8e8
)

var DefaultGenesisState = param.GenesisState{
	FeeGenesis: FeeGenesisState,

	//Add other param genesis here
}

// ---------   Definition about fee prams  ------------------- //
var FeeGenesisState = []param.FeeParam{
	// Operate
	&param.FixedFeeParams{"submit_proposal", ProposeFee, sdk.FeeForProposer},
	&param.FixedFeeParams{"deposit", DepositFee, sdk.FeeForProposer},
	&param.FixedFeeParams{"vote", sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{"create_validator", CreateValidatorFee, sdk.FeeForProposer},
	&param.FixedFeeParams{"remove_validator", RemoveValidatorFee, sdk.FeeForProposer},
	&param.FixedFeeParams{"dexList", ListingFee, sdk.FeeForAll},
	&param.FixedFeeParams{"orderNew", sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{"orderCancel", sdk.ZeroFee, sdk.FeeFree},
	&param.FixedFeeParams{"issueMsg", IssueFee, sdk.FeeForAll},
	&param.FixedFeeParams{"mintMsg", MintFee, sdk.FeeForAll},
	&param.FixedFeeParams{"tokensBurn", BurnFee, sdk.FeeForProposer},
	&param.FixedFeeParams{"tokensFreeze", FreezeFee, sdk.FeeForProposer},

	// Transfer
	&param.TransferFeeParam{
		FixedFeeParams: param.FixedFeeParams{
			MsgType: "send",
			Fee:     TransferFee,
			FeeFor:  sdk.FeeForProposer},
		MultiTransferFee:  MultiTransferFee,
		LowerLimitAsMulti: LowerLimitAsMulti,
	},

	// Dex
	&param.DexFeeParam{
		DexFeeFields: []param.DexFeeField{
			{"ExpireFee", ExpireFee},
			{"ExpireFeeNative", ExpireFeeNative},
			{"CancelFee", CancelFee},
			{"CancelFeeNative", CancelFeeNative},
			{"FeeRate", FeeRate},
			{"FeeRateNative", FeeRateNative},
			{"IOCExpireFee", IOCExpireFee},
			{"IOCExpireFeeNative", IOCExpireFeeNative},
		},
	},
}

//----------  End definition about fee param ---------------- //
