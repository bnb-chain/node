package gov_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
)

func TestGetSetProposal(t *testing.T) {
	mapp, _, keeper, _, _, _, _ := getMockApp(t, 0)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()
	keeper.SetProposal(ctx, proposal)

	gotProposal := keeper.GetProposal(ctx, proposalID)
	require.True(t, gov.ProposalEqual(proposal, gotProposal))
}

func TestIncrementProposalNumber(t *testing.T) {
	mapp, _, keeper, _, _, _, _ := getMockApp(t, 0)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})

	keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposal6 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)

	require.Equal(t, int64(6), proposal6.GetProposalID())
}

func TestActivateVotingPeriod(t *testing.T) {
	mapp, _, keeper, _, _, _, _ := getMockApp(t, 0)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)

	require.True(t, proposal.GetVotingStartTime().Equal(time.Time{}))
	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))

	keeper.ActivateVotingPeriod(ctx, proposal)

	require.True(t, proposal.GetVotingStartTime().Equal(ctx.BlockHeader().Time))
	require.Equal(t, proposal.GetProposalID(), keeper.ActiveProposalQueuePeek(ctx).GetProposalID())
}

func TestPushActiveProposalQueue(t *testing.T) {
	mapp, _, keeper, _, _, _, _ := getMockApp(t, 0)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})

	// insert one proposal
	proposal1 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)

	keeper.ActivateVotingPeriod(ctx, proposal1)

	require.Equal(t, proposal1.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())

	require.Nil(t, keeper.ActiveProposalQueuePop(ctx))

	// insert two proposal
	proposal2 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposal3 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 2000*time.Second)

	keeper.ActivateVotingPeriod(ctx, proposal2)
	keeper.ActivateVotingPeriod(ctx, proposal3)

	require.Equal(t, proposal2.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())
	require.Equal(t, proposal3.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())

	// insert two proposal
	proposal4 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 2000*time.Second)
	proposal5 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)

	keeper.ActivateVotingPeriod(ctx, proposal4)
	keeper.ActivateVotingPeriod(ctx, proposal5)

	require.Equal(t, proposal5.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())
	require.Equal(t, proposal4.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())

	require.Nil(t, keeper.ActiveProposalQueuePop(ctx))

	// insert multiple proposals in order
	proposal6 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposal7 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 2000*time.Second)
	proposal8 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 3000*time.Second)

	keeper.ActivateVotingPeriod(ctx, proposal6)
	keeper.ActivateVotingPeriod(ctx, proposal7)
	keeper.ActivateVotingPeriod(ctx, proposal8)

	require.Equal(t, proposal6.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())
	require.Equal(t, proposal7.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())
	require.Equal(t, proposal8.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())

	require.Nil(t, keeper.ActiveProposalQueuePop(ctx))

	// insert multiple proposals in random order
	proposal9 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposal10 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 2000*time.Second)
	proposal11 := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 3000*time.Second)

	keeper.ActivateVotingPeriod(ctx, proposal11)
	keeper.ActivateVotingPeriod(ctx, proposal9)
	keeper.ActivateVotingPeriod(ctx, proposal10)

	require.Equal(t, proposal9.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())
	require.Equal(t, proposal10.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())
	require.Equal(t, proposal11.GetProposalID(), keeper.ActiveProposalQueuePop(ctx).GetProposalID())

	require.Nil(t, keeper.ActiveProposalQueuePop(ctx))
}

func TestDeposits(t *testing.T) {
	mapp, ck, keeper, _, addrs, _, _ := getMockApp(t, 2)
	SortAddresses(addrs)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()

	fiveHundredSteak := sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 500e8)}
	oneThousandSteak := sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}

	addr0Initial := ck.GetCoins(ctx, addrs[0])
	addr1Initial := ck.GetCoins(ctx, addrs[1])

	// require.True(t, addr0Initial.IsEqual(sdk.Coins{sdk.NewCoin("steak", 42)}))
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5000e8)}, addr0Initial)

	require.True(t, proposal.GetTotalDeposit().IsEqual(sdk.Coins{}))

	// Check no deposits at beginning
	deposit, found := keeper.GetDeposit(ctx, proposalID, addrs[1])
	require.False(t, found)
	require.True(t, keeper.GetProposal(ctx, proposalID).GetVotingStartTime().Equal(time.Time{}))
	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))

	// Check first deposit
	err, votingStarted := keeper.AddDeposit(ctx, proposalID, addrs[0], fiveHundredSteak)
	require.Nil(t, err)
	require.False(t, votingStarted)
	deposit, found = keeper.GetDeposit(ctx, proposalID, addrs[0])
	require.True(t, found)
	require.Equal(t, fiveHundredSteak, deposit.Amount)
	require.Equal(t, addrs[0], deposit.Depositer)
	require.Equal(t, fiveHundredSteak, keeper.GetProposal(ctx, proposalID).GetTotalDeposit())
	require.Equal(t, addr0Initial.Minus(fiveHundredSteak), ck.GetCoins(ctx, addrs[0]))
	require.Equal(t, fiveHundredSteak, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	// Check a second deposit from same address
	err, votingStarted = keeper.AddDeposit(ctx, proposalID, addrs[0], oneThousandSteak)
	require.Nil(t, err)
	require.False(t, votingStarted)
	deposit, found = keeper.GetDeposit(ctx, proposalID, addrs[0])
	require.True(t, found)
	require.Equal(t, fiveHundredSteak.Plus(oneThousandSteak), deposit.Amount)
	require.Equal(t, addrs[0], deposit.Depositer)
	require.Equal(t, fiveHundredSteak.Plus(oneThousandSteak), keeper.GetProposal(ctx, proposalID).GetTotalDeposit())
	require.Equal(t, addr0Initial.Minus(fiveHundredSteak).Minus(oneThousandSteak), ck.GetCoins(ctx, addrs[0]))
	require.Equal(t, fiveHundredSteak.Plus(oneThousandSteak), ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	// Check third deposit from a new address
	err, votingStarted = keeper.AddDeposit(ctx, proposalID, addrs[1], fiveHundredSteak)
	require.Nil(t, err)
	require.True(t, votingStarted)
	deposit, found = keeper.GetDeposit(ctx, proposalID, addrs[1])
	require.True(t, found)
	require.Equal(t, addrs[1], deposit.Depositer)
	require.Equal(t, fiveHundredSteak, deposit.Amount)
	require.Equal(t, fiveHundredSteak.Plus(oneThousandSteak).Plus(fiveHundredSteak), keeper.GetProposal(ctx, proposalID).GetTotalDeposit())
	require.Equal(t, addr1Initial.Minus(fiveHundredSteak), ck.GetCoins(ctx, addrs[1]))
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 2000e8)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	// Check that proposal moved to voting period
	require.True(t, keeper.GetProposal(ctx, proposalID).GetVotingStartTime().Equal(ctx.BlockHeader().Time))
	require.NotNil(t, keeper.ActiveProposalQueuePeek(ctx))
	require.Equal(t, proposalID, keeper.ActiveProposalQueuePeek(ctx).GetProposalID())

	// Test deposit iterator
	depositsIterator := keeper.GetDeposits(ctx, proposalID)
	require.True(t, depositsIterator.Valid())
	mapp.Cdc.MustUnmarshalBinaryLengthPrefixed(depositsIterator.Value(), &deposit)
	require.Equal(t, addrs[0], deposit.Depositer)
	require.Equal(t, fiveHundredSteak.Plus(oneThousandSteak), deposit.Amount)
	depositsIterator.Next()
	mapp.Cdc.MustUnmarshalBinaryLengthPrefixed(depositsIterator.Value(), &deposit)
	require.Equal(t, addrs[1], deposit.Depositer)
	require.Equal(t, fiveHundredSteak, deposit.Amount)
	depositsIterator.Next()
	require.False(t, depositsIterator.Valid())
	depositsIterator.Close()

	// Test Refund Deposits
	deposit, found = keeper.GetDeposit(ctx, proposalID, addrs[1])
	require.True(t, found)
	require.Equal(t, fiveHundredSteak, deposit.Amount)
	keeper.RefundDeposits(ctx, proposalID)
	deposit, found = keeper.GetDeposit(ctx, proposalID, addrs[1])
	require.False(t, found)
	require.Equal(t, addr0Initial, ck.GetCoins(ctx, addrs[0]))
	require.Equal(t, addr1Initial, ck.GetCoins(ctx, addrs[1]))
	require.Equal(t, sdk.Coins(nil), ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))
}

func TestVotes(t *testing.T) {
	mapp, _, keeper, _, addrs, _, _ := getMockApp(t, 2)
	SortAddresses(addrs)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})

	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposalID := proposal.GetProposalID()

	proposal.SetStatus(gov.StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)

	// Test first vote
	keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionAbstain)
	vote, found := keeper.GetVote(ctx, proposalID, addrs[0])
	require.True(t, found)
	require.Equal(t, addrs[0], vote.Voter)
	require.Equal(t, proposalID, vote.ProposalID)
	require.Equal(t, gov.OptionAbstain, vote.Option)

	// Test change of vote
	keeper.AddVote(ctx, proposalID, addrs[0], gov.OptionYes)
	vote, found = keeper.GetVote(ctx, proposalID, addrs[0])
	require.True(t, found)
	require.Equal(t, addrs[0], vote.Voter)
	require.Equal(t, proposalID, vote.ProposalID)
	require.Equal(t, gov.OptionYes, vote.Option)

	// Test second vote
	keeper.AddVote(ctx, proposalID, addrs[1], gov.OptionNoWithVeto)
	vote, found = keeper.GetVote(ctx, proposalID, addrs[1])
	require.True(t, found)
	require.Equal(t, addrs[1], vote.Voter)
	require.Equal(t, proposalID, vote.ProposalID)
	require.Equal(t, gov.OptionNoWithVeto, vote.Option)

	// Test vote iterator
	votesIterator := keeper.GetVotes(ctx, proposalID)
	require.True(t, votesIterator.Valid())
	mapp.Cdc.MustUnmarshalBinaryLengthPrefixed(votesIterator.Value(), &vote)
	require.True(t, votesIterator.Valid())
	require.Equal(t, addrs[0], vote.Voter)
	require.Equal(t, proposalID, vote.ProposalID)
	require.Equal(t, gov.OptionYes, vote.Option)
	votesIterator.Next()
	require.True(t, votesIterator.Valid())
	mapp.Cdc.MustUnmarshalBinaryLengthPrefixed(votesIterator.Value(), &vote)
	require.True(t, votesIterator.Valid())
	require.Equal(t, addrs[1], vote.Voter)
	require.Equal(t, proposalID, vote.ProposalID)
	require.Equal(t, gov.OptionNoWithVeto, vote.Option)
	votesIterator.Next()
	require.False(t, votesIterator.Valid())
	votesIterator.Close()
}

func TestProposalQueues(t *testing.T) {
	mapp, _, keeper, _, _, _, _ := getMockApp(t, 0)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	mapp.InitChainer(ctx, abci.RequestInitChain{})

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))

	// create test proposals
	proposal := keeper.NewTextProposal(ctx, "Test", "description", gov.ProposalTypeText, 1000*time.Second)
	proposal2 := keeper.NewTextProposal(ctx, "Test2", "description", gov.ProposalTypeText, 1000*time.Second)
	proposal3 := keeper.NewTextProposal(ctx, "Test3", "description", gov.ProposalTypeText, 1000*time.Second)
	proposal4 := keeper.NewTextProposal(ctx, "Test4", "description", gov.ProposalTypeText, 1000*time.Second)

	// test pushing to inactive proposal queue
	keeper.InactiveProposalQueuePush(ctx, proposal)
	keeper.InactiveProposalQueuePush(ctx, proposal2)
	keeper.InactiveProposalQueuePush(ctx, proposal3)
	keeper.InactiveProposalQueuePush(ctx, proposal4)

	// test peeking and popping from inactive proposal queue
	require.Equal(t, keeper.InactiveProposalQueuePeek(ctx).GetProposalID(), proposal.GetProposalID())
	require.Equal(t, keeper.InactiveProposalQueuePop(ctx).GetProposalID(), proposal.GetProposalID())
	require.Equal(t, keeper.InactiveProposalQueuePeek(ctx).GetProposalID(), proposal2.GetProposalID())
	require.Equal(t, keeper.InactiveProposalQueuePop(ctx).GetProposalID(), proposal2.GetProposalID())
	require.Equal(t, keeper.InactiveProposalQueuePeek(ctx).GetProposalID(), proposal3.GetProposalID())
	require.Equal(t, keeper.InactiveProposalQueuePop(ctx).GetProposalID(), proposal3.GetProposalID())
	require.Equal(t, keeper.InactiveProposalQueuePeek(ctx).GetProposalID(), proposal4.GetProposalID())
	require.Equal(t, keeper.InactiveProposalQueuePop(ctx).GetProposalID(), proposal4.GetProposalID())

	// test pushing to active proposal queue
	keeper.ActiveProposalQueuePush(ctx, proposal)
	keeper.ActiveProposalQueuePush(ctx, proposal2)
	keeper.ActiveProposalQueuePush(ctx, proposal3)
	keeper.ActiveProposalQueuePush(ctx, proposal4)

	// test peeking and popping from active proposal queue
	require.Equal(t, keeper.ActiveProposalQueuePeek(ctx).GetProposalID(), proposal.GetProposalID())
	require.Equal(t, keeper.ActiveProposalQueuePop(ctx).GetProposalID(), proposal.GetProposalID())
	require.Equal(t, keeper.ActiveProposalQueuePeek(ctx).GetProposalID(), proposal2.GetProposalID())
	require.Equal(t, keeper.ActiveProposalQueuePop(ctx).GetProposalID(), proposal2.GetProposalID())
	require.Equal(t, keeper.ActiveProposalQueuePeek(ctx).GetProposalID(), proposal3.GetProposalID())
	require.Equal(t, keeper.ActiveProposalQueuePop(ctx).GetProposalID(), proposal3.GetProposalID())
	require.Equal(t, keeper.ActiveProposalQueuePeek(ctx).GetProposalID(), proposal4.GetProposalID())
	require.Equal(t, keeper.ActiveProposalQueuePop(ctx).GetProposalID(), proposal4.GetProposalID())
}
