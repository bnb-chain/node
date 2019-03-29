package param

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/binance-chain/node/common/fees"
	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex/list"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/param/paramhub"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
)

const AbciQueryPrefix = "param"

// InitPlugin initializes the param plugin.
func InitPlugin(app app.ChainApp, hub *paramhub.Keeper) {
	handler := createQueryHandler(hub)
	app.RegisterQueryHandler(AbciQueryPrefix, handler)
}

func createQueryHandler(keeper *paramhub.Keeper) app.AbciQueryHandler {
	return createAbciQueryHandler(keeper)
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
		freeze.FreezeRoute:                fees.FixedFeeCalculatorGen,
		bank.MsgSend{}.Type():             tokens.TransferFeeCalculatorGen,
	}
}
