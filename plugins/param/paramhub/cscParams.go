package paramhub

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/plugins/param/types"
)

const SafeToleratePeriod = 2 * 7 * 24 * 60 * 60 * time.Second // 2 weeks

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
		}, nil, nil, nil,
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
	// It can still find the valid proposal if the block chain stop for SafeToleratePeriod time
	backPeriod := SafeToleratePeriod + gov.MaxVotingPeriod
	keeper.govKeeper.Iterate(ctx, nil, nil, gov.StatusNil, 0, true, func(proposal gov.Proposal) bool {
		if proposal.GetProposalType() == gov.ProposalTypeCSCParamsChange {
			if ctx.BlockHeader().Time.Sub(proposal.GetVotingStartTime()) > backPeriod {
				return true
			}
			if proposal.GetStatus() != gov.StatusPassed {
				return false
			}

			proposal.SetStatus(gov.StatusExecuted)
			keeper.govKeeper.SetProposal(ctx, proposal)

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
