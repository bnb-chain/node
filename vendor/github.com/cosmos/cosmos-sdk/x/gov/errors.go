//nolint
package gov

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = 5

	CodeUnknownProposal         sdk.CodeType = 1
	CodeInactiveProposal        sdk.CodeType = 2
	CodeAlreadyActiveProposal   sdk.CodeType = 3
	CodeAlreadyFinishedProposal sdk.CodeType = 4
	CodeAddressNotStaked        sdk.CodeType = 5
	CodeInvalidTitle            sdk.CodeType = 6
	CodeInvalidDescription      sdk.CodeType = 7
	CodeInvalidProposalType     sdk.CodeType = 8
	CodeInvalidVote             sdk.CodeType = 9
	CodeInvalidGenesis          sdk.CodeType = 10
	CodeInvalidProposalStatus   sdk.CodeType = 11
	CodeInvalidProposal         sdk.CodeType = 12
	CodeInvalidVotingPeriod     sdk.CodeType = 13
	CodeInvalidSideChainId      sdk.CodeType = 14
)

//----------------------------------------
// Error constructors

func ErrInvalidProposal(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidProposal, fmt.Sprintf("Invalid proposal: %s", msg))
}

func ErrUnknownProposal(codespace sdk.CodespaceType, proposalID int64) sdk.Error {
	return sdk.NewError(codespace, CodeUnknownProposal, fmt.Sprintf("Unknown proposal with id %d", proposalID))
}

func ErrInactiveProposal(codespace sdk.CodespaceType, proposalID int64) sdk.Error {
	return sdk.NewError(codespace, CodeInactiveProposal, fmt.Sprintf("Inactive proposal with id %d", proposalID))
}

func ErrAlreadyActiveProposal(codespace sdk.CodespaceType, proposalID int64) sdk.Error {
	return sdk.NewError(codespace, CodeAlreadyActiveProposal, fmt.Sprintf("Proposal %d has been already active", proposalID))
}

func ErrAlreadyFinishedProposal(codespace sdk.CodespaceType, proposalID int64) sdk.Error {
	return sdk.NewError(codespace, CodeAlreadyFinishedProposal, fmt.Sprintf("Proposal %d has already passed its voting period", proposalID))
}

func ErrAddressNotStaked(codespace sdk.CodespaceType, address sdk.AccAddress) sdk.Error {
	return sdk.NewError(codespace, CodeAddressNotStaked, fmt.Sprintf("Address %s is not staked and is thus ineligible to vote", address))
}

func ErrInvalidTitle(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidTitle, msg)
}

func ErrInvalidDescription(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidDescription, msg)
}

func ErrInvalidProposalType(codespace sdk.CodespaceType, proposalType ProposalKind) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidProposalType, fmt.Sprintf("Proposal Type '%s' is not valid", proposalType))
}

func ErrInvalidVote(codespace sdk.CodespaceType, voteOption VoteOption) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidVote, fmt.Sprintf("'%v' is not a valid voting option", voteOption))
}

func ErrInvalidGenesis(codespace sdk.CodespaceType, msg string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidVote, msg)
}

func ErrInvalidVotingPeriod(codespace sdk.CodespaceType, votingPeriod time.Duration) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidVotingPeriod, fmt.Sprintf("Voting period '%d' should larger than 0 and less than %s", votingPeriod, MaxVotingPeriod))
}

func ErrInvalidSideChainId(codespace sdk.CodespaceType, sideChain string) sdk.Error {
	return sdk.NewError(codespace, CodeInvalidSideChainId, fmt.Sprintf("Invalid side chain id: %s", sideChain))
}
