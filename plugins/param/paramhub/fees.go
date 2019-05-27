package paramhub

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/param/types"
)

func (keeper *Keeper) initFeeGenesis(ctx sdk.Context, state types.GenesisState) {
	keeper.SetFeeParams(ctx, state.FeeGenesis)
}

func (keeper *Keeper) GetFeeParams(ctx sdk.Context) []types.FeeParam {
	feeParams := make([]types.FeeParam, 0)
	keeper.paramSpace.Get(ctx, ParamStoreKeyFees, &feeParams)
	return feeParams
}

func (keeper *Keeper) SetFeeParams(ctx sdk.Context, fp []types.FeeParam) {
	keeper.paramSpace.Set(ctx, ParamStoreKeyFees, fp)
	return
}

func (keeper *Keeper) UpdateFeeParams(ctx sdk.Context, updates []types.FeeParam) {
	origin := keeper.GetFeeParams(ctx)
	opFeeMap := make(map[string]int, len(updates))
	dexFeeLoc := 0
	for index, update := range origin {
		switch update := update.(type) {
		case types.MsgFeeParams:
			opFeeMap[update.GetMsgType()] = index
		case *types.DexFeeParam:
			dexFeeLoc = index
		default:
			keeper.logger.Debug("Origin Fee param not supported ", "feeParam", update)
		}
	}
	for _, update := range updates {
		switch update := update.(type) {
		case types.MsgFeeParams:
			if index, exist := opFeeMap[update.GetMsgType()]; exist {
				origin[index] = update
			} else {
				opFeeMap[update.GetMsgType()] = len(origin)
				origin = append(origin, update)
			}
		case *types.DexFeeParam:
			origin[dexFeeLoc] = update
		default:
			keeper.logger.Info("Update fee param not supported ", "feeParam", update)
		}
	}
	keeper.updateFeeCalculator(origin)
	keeper.SetFeeParams(ctx, origin)
	return
}

func (keeper *Keeper) loadFeeParam(ctx sdk.Context) {
	fp := keeper.GetFeeParams(ctx)
	keeper.notifyOnLoad(ctx, fp)
}

func (keeper *Keeper) registerFeeParamCallBack() {
	keeper.SubscribeParamChange(
		func(context sdk.Context, changes []interface{}) {
			for _, c := range changes {
				switch change := c.(type) {
				case []types.FeeParam:
					keeper.UpdateFeeParams(context, change)
				default:
					keeper.logger.Debug("Receive param changes that not interested.")
				}
			}
		},
		func(context sdk.Context, state types.GenesisState) {
			keeper.SetFeeParams(context, state.FeeGenesis)
			keeper.updateFeeCalculator(state.FeeGenesis)
		},
		func(context sdk.Context, iLoad interface{}) {
			switch load := iLoad.(type) {
			case []types.FeeParam:
				keeper.updateFeeCalculator(load)
			default:
				keeper.logger.Debug("Receive load param that not interested.")
			}

		},
	)
}

func (keeper *Keeper) updateFeeCalculator(updates []types.FeeParam) {
	println("update_calculator ", upgrade.Mgr.GetHeight())
	fees.UnsetAllCalculators()
	for _, u := range updates {
		if u, ok := u.(types.MsgFeeParams); ok {
			generator := fees.GetCalculatorGenerator(u.GetMsgType())
			if generator == nil {
				continue
			} else {
				err := u.Check()
				if err != nil {
					panic(err)
				}
				fees.RegisterCalculator(u.GetMsgType(), generator(u))
			}
		}
	}
}
