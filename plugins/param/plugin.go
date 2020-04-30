package param

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/oracle"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/binance-chain/go-sdk/common/types"

	"github.com/binance-chain/node/common/fees"
	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/account"
	"github.com/binance-chain/node/plugins/bridge"
	"github.com/binance-chain/node/plugins/dex/list"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/param/paramhub"
	param "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/plugins/tokens/timelock"
)

const AbciQueryPrefix = "param"

// InitPlugin initializes the param plugin.
func InitPlugin(app app.ChainApp, hub *paramhub.Keeper) {
	handler := createQueryHandler(hub)
	app.RegisterQueryHandler(AbciQueryPrefix, handler)
	RegisterUpgradeBeginBlocker(hub)
}

func createQueryHandler(keeper *paramhub.Keeper) app.AbciQueryHandler {
	return createAbciQueryHandler(keeper)
}

func RegisterUpgradeBeginBlocker(paramHub *ParamHub) {
	upgrade.Mgr.RegisterBeginBlocker(upgrade.BEP9, func(ctx sdk.Context) {
		timeLockFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: timelock.TimeLockMsg{}.Type(), Fee: TimeLockFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: timelock.TimeUnlockMsg{}.Type(), Fee: TimeUnlockFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: timelock.TimeRelockMsg{}.Type(), Fee: TimeRelockFee, FeeFor: sdk.FeeForProposer},
		}
		paramHub.UpdateFeeParams(ctx, timeLockFeeParams)
	})
	upgrade.Mgr.RegisterBeginBlocker(upgrade.BEP12, func(ctx sdk.Context) {
		accountFlagsFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: account.SetAccountFlagsMsg{}.Type(), Fee: SetAccountFlagsFee, FeeFor: sdk.FeeForProposer},
		}
		paramHub.UpdateFeeParams(ctx, accountFlagsFeeParams)
	})
	upgrade.Mgr.RegisterBeginBlocker(upgrade.BEP3, func(ctx sdk.Context) {
		swapFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: swap.HTLTMsg{}.Type(), Fee: HTLTFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: swap.DepositHTLTMsg{}.Type(), Fee: DepositHTLTFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: swap.ClaimHTLTMsg{}.Type(), Fee: ClaimHTLTFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: swap.RefundHTLTMsg{}.Type(), Fee: RefundHTLTFee, FeeFor: sdk.FeeForProposer},
		}
		paramHub.UpdateFeeParams(ctx, swapFeeParams)
	})
	upgrade.Mgr.RegisterBeginBlocker(upgrade.LaunchBscUpgrade, func(ctx sdk.Context) {
		stakingFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: stake.MsgCreateSideChainValidator{}.Type(), Fee: CreateSideChainValidatorFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: stake.MsgEditSideChainValidator{}.Type(), Fee: EditSideChainValidatorFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: stake.MsgSideChainDelegate{}.Type(), Fee: SideChainDelegateFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: stake.MsgSideChainRedelegate{}.Type(), Fee: SideChainRedelegateFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: stake.MsgSideChainUndelegate{}.Type(), Fee: SideChainUndelegateFee, FeeFor: sdk.FeeForProposer},

			&param.FixedFeeParams{MsgType: slashing.MsgBscSubmitEvidence{}.Type(), Fee: BscSubmitEvidenceFee, FeeFor: sdk.FeeForProposer},
			&param.FixedFeeParams{MsgType: slashing.MsgSideChainUnjail{}.Type(), Fee: SideChainUnjail, FeeFor: sdk.FeeForProposer},

			&param.FixedFeeParams{MsgType: bridge.BindMsg{}.Type(), Fee: CrossBindFee, FeeFor: types.FeeForProposer},
			&param.FixedFeeParams{MsgType: bridge.TransferOutMsg{}.Type(), Fee: CrossTransferOutFee, FeeFor: types.FeeForProposer},
			&param.FixedFeeParams{MsgType: bridge.BindRelayFeeName, Fee: CrossBindRelayFee, FeeFor: types.FeeForProposer},
			&param.FixedFeeParams{MsgType: bridge.TransferOutFeeName, Fee: CrossTransferOutRelayFee, FeeFor: types.FeeForProposer},
			&param.FixedFeeParams{MsgType: oracle.ClaimMsg{}.Type(), Fee: app.ZeroFee, FeeFor: types.FeeFree},
		}
		paramHub.UpdateFeeParams(ctx, stakingFeeParams)
	})
}

func EndBreatheBlock(ctx sdk.Context, paramHub *ParamHub) {
	paramHub.EndBreatheBlock(ctx)
	return
}

func init() {
	// CalculatorsGen is defined in a common package which can't import app package.
	// Reasonable to init here, since fee param drive the calculator.
	fees.CalculatorsGen = map[string]fees.FeeCalculatorGenerator{
		gov.MsgSubmitProposal{}.Type():             fees.FixedFeeCalculatorGen,
		gov.MsgDeposit{}.Type():                    fees.FixedFeeCalculatorGen,
		gov.MsgVote{}.Type():                       fees.FixedFeeCalculatorGen,
		stake.MsgCreateValidator{}.Type():          fees.FixedFeeCalculatorGen,
		stake.MsgRemoveValidator{}.Type():          fees.FixedFeeCalculatorGen,
		stake.MsgCreateSideChainValidator{}.Type(): fees.FixedFeeCalculatorGen,
		stake.MsgEditSideChainValidator{}.Type():   fees.FixedFeeCalculatorGen,
		stake.MsgSideChainDelegate{}.Type():        fees.FixedFeeCalculatorGen,
		stake.MsgSideChainRedelegate{}.Type():      fees.FixedFeeCalculatorGen,
		stake.MsgSideChainUndelegate{}.Type():      fees.FixedFeeCalculatorGen,
		slashing.MsgBscSubmitEvidence{}.Type():     fees.FixedFeeCalculatorGen,
		slashing.MsgSideChainUnjail{}.Type():       fees.FixedFeeCalculatorGen,
		list.Route:                                 fees.FixedFeeCalculatorGen,
		order.RouteNewOrder:                        fees.FixedFeeCalculatorGen,
		order.RouteCancelOrder:                     fees.FixedFeeCalculatorGen,
		issue.IssueMsgType:                         fees.FixedFeeCalculatorGen,
		issue.MintMsgType:                          fees.FixedFeeCalculatorGen,
		burn.BurnRoute:                             fees.FixedFeeCalculatorGen,
		account.SetAccountFlagsMsgType:             fees.FixedFeeCalculatorGen,
		freeze.FreezeRoute:                         fees.FixedFeeCalculatorGen,
		timelock.TimeLockMsg{}.Type():              fees.FixedFeeCalculatorGen,
		timelock.TimeUnlockMsg{}.Type():            fees.FixedFeeCalculatorGen,
		timelock.TimeRelockMsg{}.Type():            fees.FixedFeeCalculatorGen,
		bank.MsgSend{}.Type():                      tokens.TransferFeeCalculatorGen,
		swap.HTLT:                                  fees.FixedFeeCalculatorGen,
		swap.DepositHTLT:                           fees.FixedFeeCalculatorGen,
		swap.ClaimHTLT:                             fees.FixedFeeCalculatorGen,
		swap.RefundHTLT:                            fees.FixedFeeCalculatorGen,
		bridge.BindMsg{}.Type():                    fees.FixedFeeCalculatorGen,
		bridge.TransferOutMsg{}.Type():             fees.FixedFeeCalculatorGen,
		bridge.BindRelayFeeName:                    fees.FixedFeeCalculatorGen,
		bridge.TransferOutFeeName:                  fees.FixedFeeCalculatorGen,
		oracle.ClaimMsg{}.Type():                   fees.FixedFeeCalculatorGen,
	}
}
