package gov

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// validatorGovInfo used for tallying
type validatorGovInfo struct {
	Address             sdk.ValAddress // address of the validator operator
	Power               sdk.Dec        // Power of a Validator
	DelegatorShares     sdk.Dec        // Total outstanding delegator shares
	DelegatorDeductions sdk.Dec        // Delegator deductions from validator's delegators voting independently
	Vote                VoteOption     // Vote of the validator
}

func Tally(ctx sdk.Context, keeper Keeper, proposal Proposal) (passes bool, refundDeposits bool, tallyResults TallyResult) {
	results := make(map[VoteOption]sdk.Dec)
	results[OptionYes] = sdk.ZeroDec()
	results[OptionAbstain] = sdk.ZeroDec()
	results[OptionNo] = sdk.ZeroDec()
	results[OptionNoWithVeto] = sdk.ZeroDec()

	totalVotingPower := sdk.ZeroDec()
	currValidators := make(map[string]validatorGovInfo)

	keeper.vs.IterateValidatorsBonded(ctx, func(index int64, validator sdk.Validator) (stop bool) {
		currValidators[validator.GetOperator().String()] = validatorGovInfo{
			Address:             validator.GetOperator(),
			Power:               validator.GetPower(),
			DelegatorShares:     validator.GetDelegatorShares(),
			DelegatorDeductions: sdk.ZeroDec(),
			Vote:                OptionEmpty,
		}
		return false
	})

	// iterate over all the votes
	votesIterator := keeper.GetVotes(ctx, proposal.GetProposalID())
	defer votesIterator.Close()
	for ; votesIterator.Valid(); votesIterator.Next() {
		vote := &Vote{}
		keeper.cdc.MustUnmarshalBinaryLengthPrefixed(votesIterator.Value(), vote)

		// if validator, just record it in the map
		// if delegator tally voting power
		valAddrStr := sdk.ValAddress(vote.Voter).String()
		if val, ok := currValidators[valAddrStr]; ok {
			val.Vote = vote.Option
			currValidators[valAddrStr] = val
		} else {

			keeper.ds.IterateDelegations(ctx, vote.Voter, func(index int64, delegation sdk.Delegation) (stop bool) {
				valAddrStr := delegation.GetValidatorAddr().String()

				if val, ok := currValidators[valAddrStr]; ok {
					val.DelegatorDeductions = val.DelegatorDeductions.Add(delegation.GetShares())
					currValidators[valAddrStr] = val

					delegatorShare := delegation.GetShares().Quo(val.DelegatorShares)
					votingPower := val.Power.Mul(delegatorShare)

					results[vote.Option] = results[vote.Option].Add(votingPower)
					totalVotingPower = totalVotingPower.Add(votingPower)
				}

				return false
			})
		}

		keeper.deleteVote(ctx, vote.ProposalID, vote.Voter)
	}

	// iterate over the validators again to tally their voting power
	for _, val := range currValidators {
		if val.Vote == OptionEmpty {
			continue
		}

		sharesAfterMinus := val.DelegatorShares.Sub(val.DelegatorDeductions)
		percentAfterMinus := sharesAfterMinus.Quo(val.DelegatorShares)
		votingPower := val.Power.Mul(percentAfterMinus)

		results[val.Vote] = results[val.Vote].Add(votingPower)
		totalVotingPower = totalVotingPower.Add(votingPower)
	}

	tallyingParams := keeper.GetTallyParams(ctx)
	totalPower := keeper.vs.TotalPower(ctx)
	tallyResults = TallyResult{
		Yes:        results[OptionYes],
		Abstain:    results[OptionAbstain],
		No:         results[OptionNo],
		NoWithVeto: results[OptionNoWithVeto],
		Total:      totalPower,
	}

	// If there is no staked coins, the proposal fails
	if keeper.vs.TotalPower(ctx).IsZero() {
		return false, true, tallyResults
	}
	// If there is not enough quorum of votes, the proposal fails
	percentVoting := totalVotingPower.Quo(totalPower)
	if percentVoting.LT(tallyingParams.Quorum) {
		return false, true, tallyResults
	}
	// If no one votes, proposal fails
	if totalVotingPower.Sub(results[OptionAbstain]).Equal(sdk.ZeroDec()) {
		return false, true, tallyResults
	}
	// If more than 1/3 of voters veto, proposal fails
	if results[OptionNoWithVeto].Quo(totalVotingPower).GT(tallyingParams.Veto) {
		return false, false, tallyResults
	}
	// If more than 1/2 of non-abstaining voters vote Yes, proposal passes
	if results[OptionYes].Quo(totalVotingPower.Sub(results[OptionAbstain])).GT(tallyingParams.Threshold) {
		return true, true, tallyResults
	}
	// If more than 1/2 of non-abstaining voters vote No, proposal fails

	return false, false, tallyResults
}
