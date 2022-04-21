package gov_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

var (
	pubkeys = []crypto.PubKey{ed25519.GenPrivKey().PubKey(), ed25519.GenPrivKey().PubKey(), ed25519.GenPrivKey().PubKey()}

	testDescription   = stake.NewDescription("T", "E", "S", "T")
	testCommissionMsg = stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec())
)

func createValidators(t *testing.T, stakeHandler sdk.Handler, ctx sdk.Context, addrs []sdk.ValAddress, coinAmt []int64) {
	require.True(t, len(addrs) <= len(pubkeys), "Not enough pubkeys specified at top of file.")

	for i := 0; i < len(addrs); i++ {
		valCreateMsg := stake.NewMsgCreateValidator(
			addrs[i], pubkeys[i], sdk.NewCoin(gov.DefaultDepositDenom, coinAmt[i]), testDescription, testCommissionMsg,
		)

		res := stakeHandler(ctx, valCreateMsg)
		require.True(t, res.IsOK())
	}
}

func TestTallyNoOneVotes(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:2]))
	for i, addr := range addrs[:2] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 5})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.False(t, passes)
	tallyResults.Total = sdk.ZeroDec()
	require.True(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyOnlyValidatorsAllYes(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:2]))
	for i, addr := range addrs[:2] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 5})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionYes)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.True(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyOnlyValidators51No(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:2]))
	for i, addr := range addrs[:2] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 6})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionNo)
	require.Nil(t, err)

	passes, _, _ := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.False(t, passes)
}

func TestTallyOnlyValidators51Yes(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{6, 6, 7})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionNo)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.True(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyOnlyValidatorsVetoed(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{6, 6, 7})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionNoWithVeto)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.False(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyOnlyValidatorsAbstainPasses(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{6, 6, 7})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionAbstain)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionNo)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionYes)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.True(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyOnlyValidatorsAbstainFails(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{6, 6, 7})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionAbstain)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionNo)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.False(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyOnlyValidatorsUnreachedQuorum(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{6, 6, 7})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionYes)
	require.Nil(t, err)

	passes, refundDeposits, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.True(t, refundDeposits)
	require.False(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyOnlyValidatorsAllAbstain(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{6, 6, 7})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionAbstain)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionAbstain)
	require.Nil(t, err)

	passes, refundDeposits, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.True(t, refundDeposits)
	require.False(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyOnlyValidatorsNonVoter(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{6, 6, 7})
	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionNo)
	require.Nil(t, err)

	passes, refundDeposits, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.False(t, refundDeposits)
	require.False(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyDelgatorOverride(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 6, 7})
	stake.EndBlocker(ctx, sk)

	delegator1Msg := stake.NewMsgDelegate(addrs[3], sdk.ValAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 30))
	stakeHandler(ctx, delegator1Msg)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[3], gov.OptionNo)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.False(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyDelegatorInherit(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 6, 7})
	stake.EndBlocker(ctx, sk)

	delegator1Msg := stake.NewMsgDelegate(addrs[3], sdk.ValAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 30))
	stakeHandler(ctx, delegator1Msg)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionNo)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionNo)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionYes)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.True(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyDelegatorMultipleOverride(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{5, 6, 7})
	stake.EndBlocker(ctx, sk)

	delegator1Msg := stake.NewMsgDelegate(addrs[3], sdk.ValAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 10))
	stakeHandler(ctx, delegator1Msg)
	delegator1Msg2 := stake.NewMsgDelegate(addrs[3], sdk.ValAddress(addrs[1]), sdk.NewCoin(gov.DefaultDepositDenom, 10))
	stakeHandler(ctx, delegator1Msg2)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[3], gov.OptionNo)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.False(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyDelegatorMultipleInherit(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	val1CreateMsg := stake.NewMsgCreateValidator(
		sdk.ValAddress(addrs[0]), ed25519.GenPrivKey().PubKey(), sdk.NewCoin(gov.DefaultDepositDenom, 25), testDescription, testCommissionMsg,
	)
	stakeHandler(ctx, val1CreateMsg)

	val2CreateMsg := stake.NewMsgCreateValidator(
		sdk.ValAddress(addrs[1]), ed25519.GenPrivKey().PubKey(), sdk.NewCoin(gov.DefaultDepositDenom, 6), testDescription, testCommissionMsg,
	)
	stakeHandler(ctx, val2CreateMsg)

	val3CreateMsg := stake.NewMsgCreateValidator(
		sdk.ValAddress(addrs[2]), ed25519.GenPrivKey().PubKey(), sdk.NewCoin(gov.DefaultDepositDenom, 7), testDescription, testCommissionMsg,
	)
	stakeHandler(ctx, val3CreateMsg)

	delegator1Msg := stake.NewMsgDelegate(addrs[3], sdk.ValAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 10))
	stakeHandler(ctx, delegator1Msg)

	delegator1Msg2 := stake.NewMsgDelegate(addrs[3], sdk.ValAddress(addrs[1]), sdk.NewCoin(gov.DefaultDepositDenom, 10))
	stakeHandler(ctx, delegator1Msg2)

	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionNo)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionNo)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.False(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}

func TestTallyJailedValidator(t *testing.T) {
	mapp, _, keeper, sk, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	stakeHandler := stake.NewStakeHandler(sk)

	valAddrs := make([]sdk.ValAddress, len(addrs[:3]))
	for i, addr := range addrs[:3] {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	createValidators(t, stakeHandler, ctx, valAddrs, []int64{25, 6, 7})
	stake.EndBlocker(ctx, sk)

	delegator1Msg := stake.NewMsgDelegate(addrs[3], sdk.ValAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 10))
	stakeHandler(ctx, delegator1Msg)

	delegator1Msg2 := stake.NewMsgDelegate(addrs[3], sdk.ValAddress(addrs[1]), sdk.NewCoin(gov.DefaultDepositDenom, 10))
	stakeHandler(ctx, delegator1Msg2)

	val2, found := sk.GetValidator(ctx, sdk.ValAddress(addrs[1]))
	require.True(t, found)
	sk.Jail(ctx, sdk.ConsAddress(val2.ConsPubKey.Address()))

	stake.EndBlocker(ctx, sk)

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	err := keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionNo)
	require.Nil(t, err)
	err = keeper.AddVote(ctx, proposalID, addrs[2], gov.OptionNo)
	require.Nil(t, err)

	passes, _, tallyResults := gov.Tally(ctx, keeper, keeper.GetProposal(ctx, proposalID))

	require.True(t, passes)
	require.False(t, tallyResults.Equals(gov.EmptyTallyResult()))
}
