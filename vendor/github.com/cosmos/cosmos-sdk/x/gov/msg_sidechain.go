package gov

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/sidechain/types"
)

const (
	MsgTypeSideSubmitProposal = "side_submit_proposal"
	MsgTypeSideDeposit        = "side_deposit"
	MsgTypeSideVote           = "side_vote"
)

var _, _, _ sdk.Msg = MsgSideChainSubmitProposal{}, MsgSideChainDeposit{}, MsgSideChainVote{}

//-----------------------------------------------------------
// MsgSideChainSubmitProposal
type MsgSideChainSubmitProposal struct {
	Title          string         `json:"title"`           //  Title of the proposal
	Description    string         `json:"description"`     //  Description of the proposal
	ProposalType   ProposalKind   `json:"proposal_type"`   //  Type of proposal. Initial set {PlainTextProposal, SoftwareUpgradeProposal}
	Proposer       sdk.AccAddress `json:"proposer"`        //  Address of the proposer
	InitialDeposit sdk.Coins      `json:"initial_deposit"` //  Initial deposit paid by sender. Must be strictly positive.
	VotingPeriod   time.Duration  `json:"voting_period"`   //  Length of the voting period (s)
	SideChainId    string         `json:"side_chain_id"`
}

func NewMsgSideChainSubmitProposal(title string, description string, proposalType ProposalKind, proposer sdk.AccAddress, initialDeposit sdk.Coins, votingPeriod time.Duration, sideChainId string) MsgSideChainSubmitProposal {
	return MsgSideChainSubmitProposal{
		Title:          title,
		Description:    description,
		ProposalType:   proposalType,
		Proposer:       proposer,
		InitialDeposit: initialDeposit,
		VotingPeriod:   votingPeriod,
		SideChainId:    sideChainId,
	}
}

//nolint
func (msg MsgSideChainSubmitProposal) Route() string { return MsgRoute }
func (msg MsgSideChainSubmitProposal) Type() string  { return MsgTypeSideSubmitProposal }

// Implements Msg.
func (msg MsgSideChainSubmitProposal) ValidateBasic() sdk.Error {
	if len(msg.SideChainId) == 0 || len(msg.SideChainId) > types.MaxSideChainIdLength {
		return ErrInvalidSideChainId(DefaultCodespace, msg.SideChainId)
	}
	if len(msg.Title) == 0 {
		return ErrInvalidTitle(DefaultCodespace, "No title present in proposal")
	}
	if len(msg.Title) > MaxTitleLength {
		return ErrInvalidTitle(DefaultCodespace, fmt.Sprintf("Proposal title is longer than max length of %d", MaxTitleLength))
	}
	if len(msg.Description) == 0 {
		return ErrInvalidDescription(DefaultCodespace, "No description present in proposal")
	}
	if len(msg.Description) > MaxDescriptionLength {
		return ErrInvalidDescription(DefaultCodespace, fmt.Sprintf("Proposal description is longer than max length of %d", MaxDescriptionLength))
	}
	if !validSideProposalType(msg.ProposalType) {
		return ErrInvalidProposalType(DefaultCodespace, msg.ProposalType)
	}
	if len(msg.Proposer) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("length of address(%s) should be %d", string(msg.Proposer), sdk.AddrLen))
	}
	if !msg.InitialDeposit.IsValid() {
		return sdk.ErrInvalidCoins(msg.InitialDeposit.String())
	}
	if !msg.InitialDeposit.IsNotNegative() {
		return sdk.ErrInvalidCoins(msg.InitialDeposit.String())
	}
	if msg.VotingPeriod <= 0 || msg.VotingPeriod > MaxVotingPeriod {
		return ErrInvalidVotingPeriod(DefaultCodespace, msg.VotingPeriod)
	}
	return nil
}

func (msg MsgSideChainSubmitProposal) String() string {
	return fmt.Sprintf("MsgSideChainSubmitProposal{%s, %s, %s, %v, %s, %s}", msg.Title, msg.Description, msg.ProposalType, msg.InitialDeposit, msg.VotingPeriod, msg.SideChainId)
}

// Implements Msg.
func (msg MsgSideChainSubmitProposal) GetSignBytes() []byte {
	b, err := msgCdc.MarshalJSON(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

// Implements Msg. Identical to MsgSubmitProposal, keep here for code readability.
func (msg MsgSideChainSubmitProposal) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Proposer}
}

// Implements Msg. Identical to MsgSubmitProposal, keep here for code readability.
func (msg MsgSideChainSubmitProposal) GetInvolvedAddresses() []sdk.AccAddress {
	// Better include DepositedCoinsAccAddr, before further discussion, follow the old rule.
	return msg.GetSigners()
}

//-----------------------------------------------------------
// MsgSideChainDeposit
type MsgSideChainDeposit struct {
	ProposalID  int64          `json:"proposal_id"` // ID of the proposal
	Depositer   sdk.AccAddress `json:"depositer"`   // Address of the depositer
	Amount      sdk.Coins      `json:"amount"`      // Coins to add to the proposal's deposit
	SideChainId string         `json:"side_chain_id"`
}

func NewMsgSideChainDeposit(depositer sdk.AccAddress, proposalID int64, amount sdk.Coins, sideChainId string) MsgSideChainDeposit {
	return MsgSideChainDeposit{
		ProposalID:  proposalID,
		Depositer:   depositer,
		Amount:      amount,
		SideChainId: sideChainId,
	}
}

// nolint
func (msg MsgSideChainDeposit) Route() string { return MsgRoute }
func (msg MsgSideChainDeposit) Type() string  { return MsgTypeSideDeposit }

// Implements Msg.
func (msg MsgSideChainDeposit) ValidateBasic() sdk.Error {
	if len(msg.SideChainId) == 0 || len(msg.SideChainId) > types.MaxSideChainIdLength {
		return ErrInvalidSideChainId(DefaultCodespace, msg.SideChainId)
	}
	if len(msg.Depositer) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("length of address(%s) should be %d", string(msg.Depositer), sdk.AddrLen))
	}
	if !msg.Amount.IsValid() {
		return sdk.ErrInvalidCoins(msg.Amount.String())
	}
	if !msg.Amount.IsNotNegative() {
		return sdk.ErrInvalidCoins(msg.Amount.String())
	}
	if msg.ProposalID < 0 {
		return ErrUnknownProposal(DefaultCodespace, msg.ProposalID)
	}
	return nil
}

func (msg MsgSideChainDeposit) String() string {
	return fmt.Sprintf("MsgSideChainDeposit{%s=>%v: %v, %s}", msg.Depositer, msg.ProposalID, msg.Amount, msg.SideChainId)
}

// Implements Msg.
func (msg MsgSideChainDeposit) GetSignBytes() []byte {
	b, err := msgCdc.MarshalJSON(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

// Implements Msg. Identical to MsgDeposit, keep here for code readability.
func (msg MsgSideChainDeposit) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Depositer}
}

// Implements Msg. Identical to MsgDeposit, keep here for code readability.
func (msg MsgSideChainDeposit) GetInvolvedAddresses() []sdk.AccAddress {
	// Better include DepositedCoinsAccAddr, before further discussion, follow the old rule.
	return msg.GetSigners()
}

//-----------------------------------------------------------
// MsgSideChainVote

type MsgSideChainVote struct {
	ProposalID  int64          `json:"proposal_id"` // ID of the proposal
	Voter       sdk.AccAddress `json:"voter"`       //  address of the voter
	Option      VoteOption     `json:"option"`      //  option from OptionSet chosen by the voter
	SideChainId string         `json:"side_chain_id"`
}

func NewMsgSideChainVote(voter sdk.AccAddress, proposalID int64, option VoteOption, sideChainId string) MsgSideChainVote {
	return MsgSideChainVote{
		ProposalID:  proposalID,
		Voter:       voter,
		Option:      option,
		SideChainId: sideChainId,
	}
}

func (msg MsgSideChainVote) Route() string { return MsgRoute }
func (msg MsgSideChainVote) Type() string  { return MsgTypeSideVote }

// Implements Msg.
func (msg MsgSideChainVote) ValidateBasic() sdk.Error {
	if len(msg.SideChainId) == 0 || len(msg.SideChainId) > types.MaxSideChainIdLength {
		return ErrInvalidSideChainId(DefaultCodespace, msg.SideChainId)
	}
	if len(msg.Voter) != sdk.AddrLen {
		return sdk.ErrInvalidAddress(fmt.Sprintf("length of address(%s) should be %d", string(msg.Voter), sdk.AddrLen))
	}
	if msg.ProposalID < 0 {
		return ErrUnknownProposal(DefaultCodespace, msg.ProposalID)
	}
	if !validVoteOption(msg.Option) {
		return ErrInvalidVote(DefaultCodespace, msg.Option)
	}
	return nil
}

func (msg MsgSideChainVote) String() string {
	return fmt.Sprintf("MsgSideChainVote{%v - %s, %s}", msg.ProposalID, msg.Option, msg.SideChainId)
}

// Implements Msg.
func (msg MsgSideChainVote) GetSignBytes() []byte {
	b, err := msgCdc.MarshalJSON(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

// Implements Msg. Identical to MsgVote, keep here for code readability.
func (msg MsgSideChainVote) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Voter}
}

// Implements Msg. Identical to MsgVote, keep here for code readability.
func (msg MsgSideChainVote) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
