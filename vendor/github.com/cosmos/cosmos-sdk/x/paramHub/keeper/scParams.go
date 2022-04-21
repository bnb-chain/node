package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

func (keeper *Keeper) getLastSCParamChanges(ctx sdk.Context) *types.SCChangeParams {
	var latestProposal *gov.Proposal
	lastProposalId := keeper.GetLastSCParamChangeProposalId(ctx)
	keeper.govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, lastProposalId.ProposalID, true, func(proposal gov.Proposal) bool {
		if proposal.GetProposalType() == gov.ProposalTypeSCParamsChange {
			latestProposal = &proposal
			return true
		}
		return false
	})

	if latestProposal != nil {
		var changeParam types.SCChangeParams
		strProposal := (*latestProposal).GetDescription()
		err := keeper.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
		if err != nil {
			keeper.Logger(ctx).Error("Get broken data when unmarshal SCParamsChange msg, will skip.", "proposalId", (*latestProposal).GetProposalID(), "err", err)
			return nil
		}
		// SetLastSCParamChangeProposalId first. If invalid, the proposal before it will not been processed too.
		keeper.SetLastSCParamChangeProposalId(ctx, types.LastProposalID{ProposalID: (*latestProposal).GetProposalID()})
		if err := changeParam.Check(); err != nil {
			keeper.Logger(ctx).Error("The SCParamsChange proposal is invalid, will skip.", "proposalId", (*latestProposal).GetProposalID(), "param", changeParam, "err", err)
			return nil
		}
		return &changeParam
	}
	return nil
}

func (keeper *Keeper) GetLastSCParamChangeProposalId(ctx sdk.Context) types.LastProposalID {
	var id types.LastProposalID
	keeper.paramSpace.GetIfExists(ctx, ParamStoreKeySCLastParamsChangeProposalID, &id)
	return id
}

func (keeper *Keeper) SetLastSCParamChangeProposalId(ctx sdk.Context, id types.LastProposalID) {
	keeper.paramSpace.Set(ctx, ParamStoreKeySCLastParamsChangeProposalID, &id)
	return
}

func (keeper *Keeper) GetSCParams(ctx sdk.Context, sideChainId string) ([]types.SCParam, sdk.Error) {
	storePrefix := keeper.ScKeeper.GetSideChainStorePrefix(ctx, sideChainId)
	if len(storePrefix) == 0 {
		return nil, types.ErrInvalidSideChainId(types.DefaultCodespace, "the side chain id is not registered")
	}
	newCtx := ctx.WithSideChainKeyPrefix(storePrefix)
	params := make([]types.SCParam, 0)
	for _, subSpace := range keeper.GetSubscriberParamSpace() {
		param := subSpace.Proto()
		if _, native := param.GetParamAttribute(); native {
			subSpace.ParamSpace.GetParamSet(ctx, param)
		} else {
			subSpace.ParamSpace.GetParamSet(newCtx, param)
		}
		params = append(params, param)
	}
	return params, nil
}
