package gov

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// event hooks for governance
type GovHooks interface {
	OnProposalSubmitted(ctx sdk.Context, proposal Proposal) error // Must be called when a proposal submitted
}

func (keeper Keeper) OnProposalSubmitted(ctx sdk.Context, proposal Proposal) error {
	hs := keeper.hooks[proposal.GetProposalType()]
	for _, hooks := range hs {
		err := hooks.OnProposalSubmitted(ctx, proposal)
		if err != nil {
			return err
		}
	}
	return nil
}
