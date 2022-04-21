package paramHub

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/bank"
	param "github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

const AbciQueryPrefix = "param"

func RegisterUpgradeBeginBlocker(paramHub *ParamHub) {
	sdk.UpgradeMgr.RegisterBeginBlocker(sdk.BEP9, func(ctx sdk.Context) {
		timeLockFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: "timeLock", Fee: TimeLockFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "timeUnlock", Fee: TimeUnlockFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "timeRelock", Fee: TimeRelockFee, FeeFor: sdk.FeeForProposer},
		}
		paramHub.UpdateFeeParams(ctx, timeLockFeeParams)
	})
	sdk.UpgradeMgr.RegisterBeginBlocker(sdk.BEP12, func(ctx sdk.Context) {
		accountFlagsFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: "setAccountFlags", Fee: SetAccountFlagsFee, FeeFor: sdk.FeeForProposer},
		}
		paramHub.UpdateFeeParams(ctx, accountFlagsFeeParams)
	})
	sdk.UpgradeMgr.RegisterBeginBlocker(sdk.BEP3, func(ctx sdk.Context) {
		swapFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: "HTLT", Fee: HTLTFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "depositHTLT", Fee: DepositHTLTFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "claimHTLT", Fee: ClaimHTLTFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "refundHTLT", Fee: RefundHTLTFee, FeeFor: sdk.FeeForProposer},
		}
		paramHub.UpdateFeeParams(ctx, swapFeeParams)
	})
	sdk.UpgradeMgr.RegisterBeginBlocker(sdk.LaunchBscUpgrade, func(ctx sdk.Context) {
		updateFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: "side_create_validator", Fee: CreateSideChainValidatorFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "side_edit_validator", Fee: EditSideChainValidatorFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "side_delegate", Fee: SideChainDelegateFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "side_redelegate", Fee: SideChainRedelegateFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "side_undelegate", Fee: SideChainUndelegateFee, FeeFor: sdk.FeeForProposer},

			&param.FixedFeeParams{MsgType: "bsc_submit_evidence", Fee: BscSubmitEvidenceFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "side_chain_unjail", Fee: SideChainUnjail, FeeFor: sdk.FeeForProposer},

			&param.FixedFeeParams{MsgType: "side_submit_proposal", Fee: SideProposeFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "side_deposit", Fee: SideDepositFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "side_vote", Fee: SideVoteFee, FeeFor: sdk.FeeForProposer},

			&param.FixedFeeParams{MsgType: "crossBind", Fee: CrossBindFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "crossUnbind", Fee: CrossUnbindFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "crossTransferOut", Fee: CrossTransferOutFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "oracleClaim", Fee: sdk.ZeroFee, FeeFor: sdk.FeeFree},

			// Following fees are charged on BC, and received at BSC, they are still fees in a broad sense, so still
			// decide to put it here, rather than in paramset.
			&param.FixedFeeParams{MsgType: "crossBindRelayFee", Fee: CrossBindRelayFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "crossUnbindRelayFee", Fee: CrossUnbindRelayFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "crossTransferOutRelayFee", Fee: CrossTransferOutRelayFee, FeeFor: sdk.FeeForProposer},
		}
		paramHub.UpdateFeeParams(ctx, updateFeeParams)
	})
	sdk.UpgradeMgr.RegisterBeginBlocker(sdk.BEP8, func(ctx sdk.Context) {
		miniTokenFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: "tinyIssueMsg", Fee: TinyIssueFee, FeeFor: sdk.FeeForAll},
			&param.FixedFeeParams{MsgType: "miniIssueMsg", Fee: MiniIssueFee, FeeFor: sdk.FeeForAll},
			&param.FixedFeeParams{MsgType: "miniTokensSetURI", Fee: MiniSetUriFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: "dexListMini", Fee: MiniListingFee, FeeFor: sdk.FeeForAll},
		}
		paramHub.UpdateFeeParams(ctx, miniTokenFeeParams)
	})
	sdk.UpgradeMgr.RegisterBeginBlocker(sdk.BEP82, func(ctx sdk.Context) {
		updateFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: "transferOwnership", Fee: TransferOwnershipFee, FeeFor: sdk.FeeForProposer},
		}
		paramHub.UpdateFeeParams(ctx, updateFeeParams)
	})
}

func EndBreatheBlock(ctx sdk.Context, paramHub *ParamHub) {
	paramHub.EndBreatheBlock(ctx)
	return
}

func EndBlock(ctx sdk.Context, paramHub *ParamHub) {
	paramHub.EndBlock(ctx)
	return
}

func init() {
	// CalculatorsGen is defined in a common package which can't import app package.
	// Reasonable to init here, since fee param drive the calculator.
	fees.CalculatorsGen = map[string]fees.FeeCalculatorGenerator{
		"submit_proposal":          fees.FixedFeeCalculatorGen,
		"deposit":                  fees.FixedFeeCalculatorGen,
		"vote":                     fees.FixedFeeCalculatorGen,
		"side_submit_proposal":     fees.FixedFeeCalculatorGen,
		"side_deposit":             fees.FixedFeeCalculatorGen,
		"side_vote":                fees.FixedFeeCalculatorGen,
		"create_validator":         fees.FixedFeeCalculatorGen,
		"remove_validator":         fees.FixedFeeCalculatorGen,
		"side_create_validator":    fees.FixedFeeCalculatorGen,
		"side_edit_validator":      fees.FixedFeeCalculatorGen,
		"side_delegate":            fees.FixedFeeCalculatorGen,
		"side_redelegate":          fees.FixedFeeCalculatorGen,
		"side_undelegate":          fees.FixedFeeCalculatorGen,
		"bsc_submit_evidence":      fees.FixedFeeCalculatorGen,
		"side_chain_unjail":        fees.FixedFeeCalculatorGen,
		"dexList":                  fees.FixedFeeCalculatorGen,
		"orderNew":                 fees.FixedFeeCalculatorGen,
		"orderCancel":              fees.FixedFeeCalculatorGen,
		"issueMsg":                 fees.FixedFeeCalculatorGen,
		"mintMsg":                  fees.FixedFeeCalculatorGen,
		"tokensBurn":               fees.FixedFeeCalculatorGen,
		"setAccountFlags":          fees.FixedFeeCalculatorGen,
		"tokensFreeze":             fees.FixedFeeCalculatorGen,
		"timeLock":                 fees.FixedFeeCalculatorGen,
		"timeUnlock":               fees.FixedFeeCalculatorGen,
		"timeRelock":               fees.FixedFeeCalculatorGen,
		"transferOwnership":        fees.FixedFeeCalculatorGen,
		"send":                     bank.TransferFeeCalculatorGen,
		"HTLT":                     fees.FixedFeeCalculatorGen,
		"depositHTLT":              fees.FixedFeeCalculatorGen,
		"claimHTLT":                fees.FixedFeeCalculatorGen,
		"refundHTLT":               fees.FixedFeeCalculatorGen,
		"crossBind":                fees.FixedFeeCalculatorGen,
		"crossUnbind":              fees.FixedFeeCalculatorGen,
		"crossTransferOut":         fees.FixedFeeCalculatorGen,
		"crossBindRelayFee":        fees.FixedFeeCalculatorGen,
		"crossUnbindRelayFee":      fees.FixedFeeCalculatorGen,
		"crossTransferOutRelayFee": fees.FixedFeeCalculatorGen,
		"oracleClaim":              fees.FixedFeeCalculatorGen,
		"miniTokensSetURI":         fees.FixedFeeCalculatorGen,
		"dexListMini":              fees.FixedFeeCalculatorGen,
		"tinyIssueMsg":             fees.FixedFeeCalculatorGen,
		"miniIssueMsg":             fees.FixedFeeCalculatorGen,
	}
}
