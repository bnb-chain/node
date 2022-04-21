package gov_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/mock"
)

var (
	coinsPos         = sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000)}
	coinsZero        = sdk.Coins{}
	coinsNeg         = sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, -10000)}
	coinsPosNotAtoms = sdk.Coins{sdk.NewCoin("foo", 10000)}
	coinsMulti       = sdk.Coins{sdk.NewCoin("foo", 10000), sdk.NewCoin(gov.DefaultDepositDenom, 1000)}
)

// test ValidateBasic for MsgCreateValidator
func TestMsgSubmitProposal(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		title, description string
		proposalType       gov.ProposalKind
		proposerAddr       sdk.AccAddress
		initialDeposit     sdk.Coins
		votingPeriod       time.Duration
		expectPass         bool
	}{
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeText, addrs[0], coinsPos, 1000 * time.Second, true},
		{"", "the purpose of this proposal is to test", gov.ProposalTypeText, addrs[0], coinsPos, 1000 * time.Second, false},
		{"Test Proposal", "", gov.ProposalTypeText, addrs[0], coinsPos, 1000 * time.Second, false},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeParameterChange, addrs[0], coinsPos, 1000 * time.Second, true},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeSoftwareUpgrade, addrs[0], coinsPos, 1000 * time.Second, true},
		{"Test Proposal", "the purpose of this proposal is to test", 0x10, addrs[0], coinsPos, 1, false},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeText, sdk.AccAddress{}, coinsPos, 1000 * time.Second, false},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeText, addrs[0], coinsZero, 1000 * time.Second, true},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeText, addrs[0], coinsNeg, 1000 * time.Second, false},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeText, addrs[0], coinsMulti, 1000 * time.Second, true},
		{strings.Repeat("#", gov.MaxTitleLength*2), "the purpose of this proposal is to test", gov.ProposalTypeText, addrs[0], coinsMulti, 1000 * time.Second, false},
		{"Test Proposal", strings.Repeat("#", gov.MaxDescriptionLength*2), gov.ProposalTypeText, addrs[0], coinsMulti, 1000 * time.Second, false},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeParameterChange, addrs[0], coinsPos, 0, false},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeParameterChange, addrs[0], coinsPos, 2 * gov.MaxVotingPeriod, false},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeText, sdk.AccAddress{0, 1}, coinsZero, 1000 * time.Second, false},
	}

	for i, tc := range tests {
		msg := gov.NewMsgSubmitProposal(tc.title, tc.description, tc.proposalType, tc.proposerAddr, tc.initialDeposit, tc.votingPeriod)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

// test ValidateBasic for MsgDeposit
func TestMsgDeposit(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		proposalID    int64
		depositerAddr sdk.AccAddress
		depositAmount sdk.Coins
		expectPass    bool
	}{
		{0, addrs[0], coinsPos, true},
		{-1, addrs[0], coinsPos, false},
		{1, sdk.AccAddress{}, coinsPos, false},
		{1, sdk.AccAddress{0, 1}, coinsPos, false},
		{1, addrs[0], coinsZero, true},
		{1, addrs[0], coinsNeg, false},
		{1, addrs[0], coinsMulti, true},
	}

	for i, tc := range tests {
		msg := gov.NewMsgDeposit(tc.depositerAddr, tc.proposalID, tc.depositAmount)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

// test ValidateBasic for MsgDeposit
func TestMsgVote(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		proposalID int64
		voterAddr  sdk.AccAddress
		option     gov.VoteOption
		expectPass bool
	}{
		{0, addrs[0], gov.OptionYes, true},
		{-1, addrs[0], gov.OptionYes, false},
		{0, sdk.AccAddress{}, gov.OptionYes, false},
		{0, sdk.AccAddress{1, 2}, gov.OptionYes, false},
		{0, addrs[0], gov.OptionNo, true},
		{0, addrs[0], gov.OptionNoWithVeto, true},
		{0, addrs[0], gov.OptionAbstain, true},
		{0, addrs[0], gov.VoteOption(0x13), false},
	}

	for i, tc := range tests {
		msg := gov.NewMsgVote(tc.voterAddr, tc.proposalID, tc.option)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgSideChainSubmitProposal(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		title, description string
		proposalType       gov.ProposalKind
		proposerAddr       sdk.AccAddress
		initialDeposit     sdk.Coins
		votingPeriod       time.Duration
		sideChainId        string
		expectPass         bool
	}{
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeSCParamsChange, addrs[0], coinsPos, 1000 * time.Second, "bsc", true},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeCSCParamsChange, addrs[0], coinsPos, 1000 * time.Second, "rialto", true},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeSCParamsChange, addrs[0], coinsPos, 1000 * time.Second, "", false},
		{"Test Proposal", "the purpose of this proposal is to test", gov.ProposalTypeParameterChange, addrs[0], coinsPos, 1000 * time.Second, "", false},
	}

	for i, tc := range tests {
		msg := gov.NewMsgSideChainSubmitProposal(tc.title, tc.description, tc.proposalType, tc.proposerAddr, tc.initialDeposit, tc.votingPeriod, tc.sideChainId)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

func TestMsgSideChainDeposit(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		proposalID    int64
		depositerAddr sdk.AccAddress
		depositAmount sdk.Coins
		sideChain     string
		expectPass    bool
	}{
		{0, addrs[0], coinsPos, "bsc", true},
		{0, addrs[0], coinsPos, "", false},
	}

	for i, tc := range tests {
		msg := gov.NewMsgSideChainDeposit(tc.depositerAddr, tc.proposalID, tc.depositAmount, tc.sideChain)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}

// test ValidateBasic for MsgDeposit
func TestMsgSideChainVote(t *testing.T) {
	_, addrs, _, _ := mock.CreateGenAccounts(1, sdk.Coins{})
	tests := []struct {
		proposalID int64
		voterAddr  sdk.AccAddress
		option     gov.VoteOption
		sideChain  string
		expectPass bool
	}{
		{0, addrs[0], gov.OptionYes, "bsc", true},
		{0, addrs[0], gov.OptionYes, "", false},
	}

	for i, tc := range tests {
		msg := gov.NewMsgSideChainVote(tc.voterAddr, tc.proposalID, tc.option, tc.sideChain)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test: %v", i)
		}
	}
}
