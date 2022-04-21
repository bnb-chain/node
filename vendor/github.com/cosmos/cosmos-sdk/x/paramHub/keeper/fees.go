package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

func (keeper *Keeper) initFeeGenesis(ctx sdk.Context, state types.GenesisState) {
	keeper.SetFeeParams(ctx, state.FeeGenesis)
}

func (keeper *Keeper) UpdateFeeParams(ctx sdk.Context, updates []types.FeeParam) {
	log := keeper.Logger(ctx)
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
			log.Debug("Origin Fee param not supported ", "feeParam", update)
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
			log.Info("Update fee param not supported ", "feeParam", update)
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
		func(ctx sdk.Context, iChange interface{}) {
			switch change := iChange.(type) {
			case []types.FeeParam:
				keeper.UpdateFeeParams(ctx, change)
			default:
				keeper.Logger(ctx).Debug("Receive param changes that not interested.")
			}
		},
		nil,
		func(context sdk.Context, state interface{}) {
			switch genesisState := state.(type) {
			case types.GenesisState:
				keeper.SetFeeParams(context, genesisState.FeeGenesis)
				keeper.updateFeeCalculator(genesisState.FeeGenesis)
			default:
				keeper.Logger(context).Debug("Receive genesis param that not interested.")
			}
		},
		func(context sdk.Context, iLoad interface{}) {
			switch load := iLoad.(type) {
			case []types.FeeParam:
				keeper.updateFeeCalculator(load)
			default:
				keeper.Logger(context).Debug("Receive load param that not interested.")
			}
		},
	)
}

func (keeper *Keeper) updateFeeCalculator(updates []types.FeeParam) {
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

func (keeper *Keeper) getLastFeeChangeParam(ctx sdk.Context) []types.FeeParam {
	log := keeper.Logger(ctx)
	var latestProposal *gov.Proposal
	lastProposalId := keeper.getLastFeeChangeProposalId(ctx)
	keeper.govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, lastProposalId.ProposalID, true, func(proposal gov.Proposal) bool {
		if proposal.GetProposalType() == gov.ProposalTypeFeeChange {
			latestProposal = &proposal
			return true
		}
		return false
	})
	if latestProposal != nil {
		var changeParam types.FeeChangeParams
		strProposal := (*latestProposal).GetDescription()
		err := keeper.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
		if err != nil {
			log.Error("Get broken data when unmarshal FeeChangeParams msg, will skip", "proposalId", (*latestProposal).GetProposalID(), "err", err)
			return nil
		}
		// setLastFeeProposal first. If invalid, the proposal before it will not been processed too.
		keeper.setLastFeeChangeProposalId(ctx, types.LastProposalID{(*latestProposal).GetProposalID()})
		if err := changeParam.Check(); err != nil {
			log.Error("The latest fee param change proposal is invalid.", "proposalId", (*latestProposal).GetProposalID(), "param", changeParam, "err", err)
			return nil
		}
		return changeParam.FeeParams
	}
	return nil
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

func (keeper *Keeper) getLastFeeChangeProposalId(ctx sdk.Context) types.LastProposalID {
	var id types.LastProposalID
	keeper.paramSpace.Get(ctx, ParamStoreKeyLastFeeChangeProposalID, &id)
	return id
}

func (keeper *Keeper) setLastFeeChangeProposalId(ctx sdk.Context, id types.LastProposalID) {
	keeper.paramSpace.Set(ctx, ParamStoreKeyLastFeeChangeProposalID, &id)
	return
}
