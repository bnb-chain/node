package paramhub

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/plugins/param/types"
)

func (keeper *Keeper) registerCSCParamsCallBack() {
	keeper.SubscribeParamChange(
		func(contexts []sdk.Context, changes []interface{}) {
			for idx, c := range changes {
				switch changes := c.(type) {
				case types.CSCParamChanges:
					keeper.updateCSCParams(contexts[idx], changes)
				default:
					keeper.logger.Debug("Receive param changes that not interested.")
				}
			}
		}, nil,nil, nil,
	)
}

func (keeper *Keeper) updateCSCParams(ctx sdk.Context, updates types.CSCParamChanges) {
	// write package in reverse order
	for j := len(updates.Changes) - 1; j >= 0; j-- {
		change := updates.Changes[j]
		_, err := keeper.SaveParamChangeToIbc(ctx, updates.ChainID, change)
		if err != nil {
			keeper.logger.Error("failed to save param change to ibc", "err", err, "change", change)
		}
	}
}

func (keeper *Keeper) getLastCSCParamChanges(ctx sdk.Context) []types.CSCParamChange {
	changes := make([]types.CSCParamChange, 0)
	lastProposalId := keeper.GetLastCSCParamChangeProposalId(ctx)
	first := true
	keeper.govKeeper.Iterate(ctx, nil, nil, gov.StatusPassed, lastProposalId.ProposalID, true, func(proposal gov.Proposal) bool {
		if proposal.GetProposalType() == gov.ProposalTypeCSCParamsChange {
			if first {
				keeper.SetLastCSCParamChangeProposalId(ctx, types.LastProposalID{ProposalID: proposal.GetProposalID()})
				first = false
			}
			var changeParam types.CSCParamChange
			strProposal := proposal.GetDescription()
			err := keeper.cdc.UnmarshalJSON([]byte(strProposal), &changeParam)
			if err != nil {
				keeper.logger.Error("Get broken data when unmarshal CSCParamChange msg, will skip.", "proposalId", proposal.GetProposalID(), "err", err)
				return false
			}
			if err := changeParam.Check(); err != nil {
				keeper.logger.Error("The CSCParamChange proposal is invalid, will skip.", "proposalId", proposal.GetProposalID(), "param", changeParam, "err", err)
				return false
			}
			changes = append(changes, changeParam)
		}
		return false
	})
	return changes
}

func (keeper *Keeper) GetLastCSCParamChangeProposalId(ctx sdk.Context) types.LastProposalID {
	var id types.LastProposalID
	keeper.sideParamSpace.GetIfExists(ctx, ParamStoreKeyCSCLastParamsChangeProposalID, &id)
	return id
}

func (keeper *Keeper) SetLastCSCParamChangeProposalId(ctx sdk.Context, id types.LastProposalID) {
	keeper.sideParamSpace.Set(ctx, ParamStoreKeyCSCLastParamsChangeProposalID, &id)
	return
}
