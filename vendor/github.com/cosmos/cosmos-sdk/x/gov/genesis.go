package gov

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultDepositDenom = "steak"
)

// GenesisState - all staking state that must be provided at genesis
type GenesisState struct {
	StartingProposalID int64         `json:"starting_proposalID"`
	DepositParams      DepositParams `json:"deposit_params"`
	TallyParams        TallyParams   `json:"tally_params"`
}

func NewGenesisState(startingProposalID int64, dp DepositParams, tp TallyParams) GenesisState {
	return GenesisState{
		StartingProposalID: startingProposalID,
		DepositParams:      dp,
		TallyParams:        tp,
	}
}

// get raw genesis raw message for testing
func DefaultGenesisState() GenesisState {
	return GenesisState{
		StartingProposalID: 1,
		DepositParams: DepositParams{
			MinDeposit:       sdk.Coins{sdk.NewCoin(DefaultDepositDenom, 2000e8)},
			MaxDepositPeriod: time.Duration(2*24) * time.Hour, // 2 days
		},
		TallyParams: TallyParams{
			Quorum:    sdk.NewDecWithPrec(5, 1),
			Threshold: sdk.NewDecWithPrec(5, 1),
			Veto:      sdk.NewDecWithPrec(334, 3),
		},
	}
}

// InitGenesis - store genesis parameters
func InitGenesis(ctx sdk.Context, k Keeper, data GenesisState) {
	err := k.SetInitialProposalID(ctx, data.StartingProposalID)
	if err != nil {
		// TODO: Handle this with #870
		panic(err)
	}
	k.SetDepositParams(ctx, data.DepositParams)
	k.SetTallyParams(ctx, data.TallyParams)
}

// WriteGenesis - output genesis parameters
func WriteGenesis(ctx sdk.Context, k Keeper) GenesisState {
	startingProposalID, _ := k.getNewProposalID(ctx)
	depositParams := k.GetDepositParams(ctx)
	tallyingParams := k.GetTallyParams(ctx)

	return GenesisState{
		StartingProposalID: startingProposalID,
		DepositParams:      depositParams,
		TallyParams:        tallyingParams,
	}
}
