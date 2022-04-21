package tags

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	ActionSubmitProposal = []byte("submit-proposal")
	ActionDeposit        = []byte("deposit")
	ActionVote           = []byte("vote")

	Action            = sdk.TagAction
	Proposer          = "proposer"
	ProposalID        = "proposal-id"
	VotingPeriodStart = "voting-period-start"
	Depositer         = "depositer"
	Voter             = "voter"
)
