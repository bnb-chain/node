package param

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/types"
	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/dex/list"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/param/paramhub"
	"github.com/binance-chain/node/plugins/account"
	param "github.com/binance-chain/node/plugins/param/types"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
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
			&param.FixedFeeParams{MsgType: timelock.TimeLockMsg{}.Type(), Fee: TimeLockFee, FeeFor: types.FeeForProposer},
			&param.FixedFeeParams{MsgType: timelock.TimeUnlockMsg{}.Type(), Fee: TimeUnlockFee, FeeFor: types.FeeForProposer},
			&param.FixedFeeParams{MsgType: timelock.TimeRelockMsg{}.Type(), Fee: TimeRelockFee, FeeFor: types.FeeForProposer},
		}
		paramHub.UpdateParams(ctx, []interface{}{timeLockFeeParams})
	})
	upgrade.Mgr.RegisterBeginBlocker(upgrade.BEP12, func(ctx sdk.Context) {
		accountFlagsFeeParams := []param.FeeParam{
			&param.FixedFeeParams{MsgType: account.SetAccountFlagsMsg{}.Type(), Fee: SetAccountFlagsFee, FeeFor: types.FeeForProposer},
		}
		paramHub.UpdateParams(ctx, []interface{}{accountFlagsFeeParams})
	})
	upgrade.Mgr.RegisterBeginBlocker(upgrade.MakerTakerFee, func(ctx sdk.Context) {
		dexFeeParams := []param.FeeParam{
			&param.DexFeeParam{
				DexFeeFields: []param.DexFeeField{
					{order.ExpireFeeField, 25000},
					{order.ExpireFeeNativeField, 5000},
					{order.CancelFeeField, 25000},
					{order.CancelFeeNativeField, 5000},
					{order.FeeRateField, 1000},
					{order.FeeRateNativeField, 400},
					{order.IOCExpireFee, 10000},
					{order.IOCExpireFeeNative, 2500},

					{order.MakerFeeRateField, 1000 },
					{order.MakerFeeRateNativeField, 400},
					{order.TakerFeeRateField, 1000},
					{order.TakerFeeRateNativeField, 400},
				},
			},
		}
		paramHub.UpdateParams(ctx, []interface{}{dexFeeParams})
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
		gov.MsgSubmitProposal{}.Type():    fees.FixedFeeCalculatorGen,
		gov.MsgDeposit{}.Type():           fees.FixedFeeCalculatorGen,
		gov.MsgVote{}.Type():              fees.FixedFeeCalculatorGen,
		stake.MsgCreateValidator{}.Type(): fees.FixedFeeCalculatorGen,
		stake.MsgRemoveValidator{}.Type(): fees.FixedFeeCalculatorGen,
		list.Route:                        fees.FixedFeeCalculatorGen,
		order.RouteNewOrder:               fees.FixedFeeCalculatorGen,
		order.RouteCancelOrder:            fees.FixedFeeCalculatorGen,
		issue.IssueMsgType:                fees.FixedFeeCalculatorGen,
		issue.MintMsgType:                 fees.FixedFeeCalculatorGen,
		burn.BurnRoute:                    fees.FixedFeeCalculatorGen,
		account.SetAccountFlagsMsgType:    fees.FixedFeeCalculatorGen,
		freeze.FreezeRoute:                fees.FixedFeeCalculatorGen,
		timelock.TimeLockMsg{}.Type():     fees.FixedFeeCalculatorGen,
		timelock.TimeUnlockMsg{}.Type():   fees.FixedFeeCalculatorGen,
		timelock.TimeRelockMsg{}.Type():   fees.FixedFeeCalculatorGen,
		bank.MsgSend{}.Type():             tokens.TransferFeeCalculatorGen,
	}
}
