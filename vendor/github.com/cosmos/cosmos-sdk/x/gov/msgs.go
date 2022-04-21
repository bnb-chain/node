package gov

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// name to identify transaction types
const (
	MsgRoute = "gov"

	MaxTitleLength           = 128
	MaxDescriptionLength int = 2048
	MaxVotingPeriod          = 2 * 7 * 24 * 60 * 60 * time.Second // 2 weeks
)

var _, _, _ sdk.Msg = MsgSubmitProposal{}, MsgDeposit{}, MsgVote{}

//-----------------------------------------------------------
type ListTradingPairParams struct {
	BaseAssetSymbol  string    `json:"base_asset_symbol"`  // base asset symbol
	QuoteAssetSymbol string    `json:"quote_asset_symbol"` // quote asset symbol
	InitPrice        int64     `json:"init_price"`         // init price
	Description      string    `json:"description"`        // description
	ExpireTime       time.Time `json:"expire_time"`        // expire time
}

//-----------------------------------------------------------
type DelistTradingPairParams struct {
	BaseAssetSymbol  string `json:"base_asset_symbol"`  // base asset symbol
	QuoteAssetSymbol string `json:"quote_asset_symbol"` // quote asset symbol
	Justification    string `json:"justification"`      // justification
	IsExecuted       bool   `json:"is_executed"`        // is this proposal executed
}

//-----------------------------------------------------------
// MsgSubmitProposal
type MsgSubmitProposal struct {
	Title          string         `json:"title"`           //  Title of the proposal
	Description    string         `json:"description"`     //  Description of the proposal
	ProposalType   ProposalKind   `json:"proposal_type"`   //  Type of proposal. Initial set {PlainTextProposal, SoftwareUpgradeProposal}
	Proposer       sdk.AccAddress `json:"proposer"`        //  Address of the proposer
	InitialDeposit sdk.Coins      `json:"initial_deposit"` //  Initial deposit paid by sender. Must be strictly positive.
	VotingPeriod   time.Duration  `json:"voting_period"`   //  Length of the voting period (s)
}

func NewMsgSubmitProposal(title string, description string, proposalType ProposalKind, proposer sdk.AccAddress, initialDeposit sdk.Coins, votingPeriod time.Duration) MsgSubmitProposal {
	return MsgSubmitProposal{
		Title:          title,
		Description:    description,
		ProposalType:   proposalType,
		Proposer:       proposer,
		InitialDeposit: initialDeposit,
		VotingPeriod:   votingPeriod,
	}
}

//nolint
func (msg MsgSubmitProposal) Route() string { return MsgRoute }
func (msg MsgSubmitProposal) Type() string  { return "submit_proposal" }

// Implements Msg.
func (msg MsgSubmitProposal) ValidateBasic() sdk.Error {
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
	if !validProposalType(msg.ProposalType) {
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

func (msg MsgSubmitProposal) String() string {
	return fmt.Sprintf("MsgSubmitProposal{%s, %s, %s, %v, %s}", msg.Title, msg.Description, msg.ProposalType, msg.InitialDeposit, msg.VotingPeriod)
}

// Implements Msg.
func (msg MsgSubmitProposal) Get(key interface{}) (value interface{}) {
	return nil
}

// Implements Msg.
func (msg MsgSubmitProposal) GetSignBytes() []byte {
	b, err := msgCdc.MarshalJSON(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

// Implements Msg.
func (msg MsgSubmitProposal) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Proposer}
}

func (msg MsgSubmitProposal) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

//-----------------------------------------------------------
// MsgDeposit
type MsgDeposit struct {
	ProposalID int64          `json:"proposal_id"` // ID of the proposal
	Depositer  sdk.AccAddress `json:"depositer"`   // Address of the depositer
	Amount     sdk.Coins      `json:"amount"`      // Coins to add to the proposal's deposit
}

func NewMsgDeposit(depositer sdk.AccAddress, proposalID int64, amount sdk.Coins) MsgDeposit {
	return MsgDeposit{
		ProposalID: proposalID,
		Depositer:  depositer,
		Amount:     amount,
	}
}

// Implements Msg.
// nolint
func (msg MsgDeposit) Route() string { return MsgRoute }
func (msg MsgDeposit) Type() string  { return "deposit" }

// Implements Msg.
func (msg MsgDeposit) ValidateBasic() sdk.Error {
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

func (msg MsgDeposit) String() string {
	return fmt.Sprintf("MsgDeposit{%s=>%v: %v}", msg.Depositer, msg.ProposalID, msg.Amount)
}

// Implements Msg.
func (msg MsgDeposit) Get(key interface{}) (value interface{}) {
	return nil
}

// Implements Msg.
func (msg MsgDeposit) GetSignBytes() []byte {
	b, err := msgCdc.MarshalJSON(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

// Implements Msg.
func (msg MsgDeposit) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Depositer}
}

func (msg MsgDeposit) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}

//-----------------------------------------------------------
// MsgVote
type MsgVote struct {
	ProposalID int64          `json:"proposal_id"` // ID of the proposal
	Voter      sdk.AccAddress `json:"voter"`       //  address of the voter
	Option     VoteOption     `json:"option"`      //  option from OptionSet chosen by the voter
}

func NewMsgVote(voter sdk.AccAddress, proposalID int64, option VoteOption) MsgVote {
	return MsgVote{
		ProposalID: proposalID,
		Voter:      voter,
		Option:     option,
	}
}

// Implements Msg.
// nolint
func (msg MsgVote) Route() string { return MsgRoute }
func (msg MsgVote) Type() string  { return "vote" }

// Implements Msg.
func (msg MsgVote) ValidateBasic() sdk.Error {
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

func (msg MsgVote) String() string {
	return fmt.Sprintf("MsgVote{%v - %s}", msg.ProposalID, msg.Option)
}

// Implements Msg.
func (msg MsgVote) Get(key interface{}) (value interface{}) {
	return nil
}

// Implements Msg.
func (msg MsgVote) GetSignBytes() []byte {
	b, err := msgCdc.MarshalJSON(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(b)
}

// Implements Msg.
func (msg MsgVote) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Voter}
}

func (msg MsgVote) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.GetSigners()
}
