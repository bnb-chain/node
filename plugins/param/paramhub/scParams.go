package paramhub

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/plugins/param/types"
)

func (keeper *Keeper) getLastSCParamChanges(ctx sdk.Context) []types.SCChangeParams {
	changes := make([]types.SCChangeParams, 0)
	lastProposalId := keeper.GetLastSCParamChangeProposalId(ctx)
	first := true
	keeper.govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, lastProposalId.ProposalID, true, func(proposal gov.Proposal) bool {
		if proposal.GetProposalType() == gov.ProposalTypeSCParamsChange {
			if first {
				keeper.SetLastSCParamChangeProposalId(ctx, types.LastProposalID{ProposalID: proposal.GetProposalID()})
				first = false
			}
			var changeParam types.SCChangeParams
			strProposal := proposal.GetDescription()
			err := keeper.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
			if err != nil {
				keeper.logger.Error("Get broken data when unmarshal SCParamsChange msg, will skip.", "proposalId", proposal.GetProposalID(), "err", err)
				return false
			}
			if err := changeParam.Check(); err != nil {
				keeper.logger.Error("The SCParamsChange proposal is invalid, will skip.", "proposalId", proposal.GetProposalID(), "param", changeParam, "err", err)
				return false
			}
			changes = append(changes, changeParam)
		}
		return false
	})
	return changes
}

func (keeper *Keeper) GetLastSCParamChangeProposalId(ctx sdk.Context) types.LastProposalID {
	var id types.LastProposalID
	keeper.sideParamSpace.GetIfExists(ctx, ParamStoreKeySCLastParamsChangeProposalID, &id)
	return id
}

func (keeper *Keeper) SetLastSCParamChangeProposalId(ctx sdk.Context, id types.LastProposalID) {
	keeper.sideParamSpace.Set(ctx, ParamStoreKeySCLastParamsChangeProposalID, &id)
	return
}
