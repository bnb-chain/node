package gov_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/x/mock"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

func TestTickExpiredDepositPeriod(t *testing.T) {
	mapp, ck, keeper, stakeKeeper, addrs, pubKeys, _ := getMockApp(t, 10)

	_, feeAccount := mock.GeneratePrivKeyAddressPairs(1)
	validator := stake.NewValidatorWithFeeAddr(feeAccount[0], sdk.ValAddress(addrs[0]), pubKeys[0], stake.Description{})

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{ProposerAddress: pubKeys[0].Address()})

	// create validator
	stakeKeeper.SetValidator(ctx, validator)
	stakeKeeper.SetValidatorByConsAddr(ctx, validator)

	govHandler := gov.NewHandler(keeper)

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))

	newProposalMsg := gov.NewMsgSubmitProposal("Test", "test", gov.ProposalTypeText, addrs[1], sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, 1000)

	res := govHandler(ctx, newProposalMsg)
	require.True(t, res.IsOK())

	gov.EndBlocker(ctx, keeper)
	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	gov.EndBlocker(ctx, keeper)
	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(keeper.GetDepositParams(ctx).MaxDepositPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.True(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	gov.EndBlocker(ctx, keeper)

	validatorCoins := ck.GetCoins(ctx, feeAccount[0])
	// check distribute deposits to proposer
	require.Equal(t, validatorCoins, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)})
	require.Equal(t, sdk.Coins(nil), ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
}

func TestTickMultipleExpiredDepositPeriod(t *testing.T) {
	mapp, ck, keeper, stakeKeeper, addrs, pubKeys, _ := getMockApp(t, 10)

	_, feeAccount := mock.GeneratePrivKeyAddressPairs(1)
	validator := stake.NewValidatorWithFeeAddr(feeAccount[0], sdk.ValAddress(addrs[0]), pubKeys[0], stake.Description{})

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{ProposerAddress: pubKeys[0].Address()})

	// create validator
	stakeKeeper.SetValidator(ctx, validator)
	stakeKeeper.SetValidatorByConsAddr(ctx, validator)

	govHandler := gov.NewHandler(keeper)

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))

	newProposalMsg := gov.NewMsgSubmitProposal("Test", "test", gov.ProposalTypeText, addrs[0], sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, 1000)

	res := govHandler(ctx, newProposalMsg)
	require.True(t, res.IsOK())

	gov.EndBlocker(ctx, keeper)
	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(2) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	gov.EndBlocker(ctx, keeper)
	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))

	newProposalMsg2 := gov.NewMsgSubmitProposal("Test2", "test2", gov.ProposalTypeText, addrs[1], sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5)}, 1000)
	res = govHandler(ctx, newProposalMsg2)
	require.True(t, res.IsOK())

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(keeper.GetDepositParams(ctx).MaxDepositPeriod).Add(time.Duration(-1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.True(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	gov.EndBlocker(ctx, keeper)
	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(5) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.True(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	gov.EndBlocker(ctx, keeper)
	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Equal(t, sdk.Coins(nil), ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))
}

func TestTickPassedDepositPeriod(t *testing.T) {
	mapp, ck, keeper, _, addrs, _, _ := getMockApp(t, 10)
	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{})
	govHandler := gov.NewHandler(keeper)

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))

	newProposalMsg := gov.NewMsgSubmitProposal("Test", "test", gov.ProposalTypeText, addrs[0], sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, 1000)

	res := govHandler(ctx, newProposalMsg)
	require.True(t, res.IsOK())

	proposalID, _ := strconv.Atoi(string(res.Data))

	gov.EndBlocker(ctx, keeper)
	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	gov.EndBlocker(ctx, keeper)
	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))

	newDepositMsg := gov.NewMsgDeposit(addrs[1], int64(proposalID), sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)})
	res = govHandler(ctx, newDepositMsg)
	require.True(t, res.IsOK())

	require.NotNil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.True(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.NotNil(t, keeper.ActiveProposalQueuePeek(ctx))
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 2000e8)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	gov.EndBlocker(ctx, keeper)

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.NotNil(t, keeper.ActiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))
}

func TestTickPassedVotingPeriodRejected(t *testing.T) {
	mapp, ck, keeper, stakeKeeper, addrs, pubKeys, _ := getMockApp(t, 10)

	_, feeAccount := mock.GeneratePrivKeyAddressPairs(1)
	validator := stake.NewValidatorWithFeeAddr(feeAccount[0], sdk.ValAddress(addrs[0]), pubKeys[0], stake.Description{})

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{ProposerAddress: pubKeys[0].Address()})

	// create validator
	stakeKeeper.SetValidator(ctx, validator)
	stakeKeeper.SetValidatorByConsAddr(ctx, validator)
	stakeKeeper.Delegate(ctx, sdk.AccAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 1000), validator, true)
	stakeKeeper.ApplyAndReturnValidatorSetUpdates(ctx)

	govHandler := gov.NewHandler(keeper)

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))

	votingPeriod := 1000 * time.Second
	newProposalMsg := gov.NewMsgSubmitProposal("Test", "test", gov.ProposalTypeText, addrs[0], sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, votingPeriod)

	res := govHandler(ctx, newProposalMsg)
	require.True(t, res.IsOK())

	proposalID, _ := strconv.Atoi(string(res.Data))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	newDepositMsg := gov.NewMsgDeposit(addrs[1], int64(proposalID), sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)})
	res = govHandler(ctx, newDepositMsg)
	require.True(t, res.IsOK())
	gov.EndBlocker(ctx, keeper)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)
	newVoteMsg := gov.NewMsgVote(addrs[0], int64(proposalID), gov.OptionNo)
	res = govHandler(ctx, newVoteMsg)
	require.True(t, res.IsOK())
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 2000e8)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	gov.EndBlocker(ctx, keeper)

	// pass voting period
	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(votingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	require.True(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))
	depositsIterator := keeper.GetDeposits(ctx, int64(proposalID))
	require.True(t, depositsIterator.Valid())
	depositsIterator.Close()
	require.Equal(t, gov.StatusVotingPeriod, keeper.GetProposal(ctx, int64(proposalID)).GetStatus())

	gov.EndBlocker(ctx, keeper)

	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	depositsIterator = keeper.GetDeposits(ctx, int64(proposalID))
	require.False(t, depositsIterator.Valid())
	depositsIterator.Close()
	require.Equal(t, gov.StatusRejected, keeper.GetProposal(ctx, int64(proposalID)).GetStatus())

	// check distribute deposits to proposer
	validatorCoins := ck.GetCoins(ctx, feeAccount[0])
	require.Equal(t, validatorCoins, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 2000e8)})
	require.Equal(t, sdk.Coins(nil), ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))
}

func TestTickPassedVotingPeriodPassed(t *testing.T) {
	mapp, ck, keeper, stakeKeeper, addrs, pubKeys, _ := getMockApp(t, 3)

	_, feeAccount := mock.GeneratePrivKeyAddressPairs(1)
	validator0 := stake.NewValidatorWithFeeAddr(feeAccount[0], sdk.ValAddress(addrs[0]), pubKeys[0], stake.Description{})

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{ProposerAddress: pubKeys[0].Address()})

	// create and delegate validator
	stakeKeeper.SetValidator(ctx, validator0)
	stakeKeeper.SetValidatorByConsAddr(ctx, validator0)
	stakeKeeper.Delegate(ctx, sdk.AccAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 1000), validator0, true)

	stakeKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	validator0, _ = stakeKeeper.GetValidator(ctx, validator0.OperatorAddr)

	govHandler := gov.NewHandler(keeper)

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))

	votingPeriod := 1000 * time.Second
	newProposalMsg := gov.NewMsgSubmitProposal("Test", "test", gov.ProposalTypeText, addrs[0], sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, votingPeriod)

	res := govHandler(ctx, newProposalMsg)
	require.True(t, res.IsOK())

	proposalID, _ := strconv.Atoi(string(res.Data))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	newDepositMsg := gov.NewMsgDeposit(addrs[1], int64(proposalID), sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)})
	res = govHandler(ctx, newDepositMsg)
	require.True(t, res.IsOK())
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 2000e8)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))
	gov.EndBlocker(ctx, keeper)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)
	newVoteMsg := gov.NewMsgVote(addrs[0], int64(proposalID), gov.OptionYes)
	res = govHandler(ctx, newVoteMsg)
	require.True(t, res.IsOK())
	gov.EndBlocker(ctx, keeper)

	// pass voting period
	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(votingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	require.True(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))
	depositsIterator := keeper.GetDeposits(ctx, int64(proposalID))
	require.True(t, depositsIterator.Valid())
	depositsIterator.Close()
	require.Equal(t, gov.StatusVotingPeriod, keeper.GetProposal(ctx, int64(proposalID)).GetStatus())

	gov.EndBlocker(ctx, keeper)

	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	depositsIterator = keeper.GetDeposits(ctx, int64(proposalID))
	require.False(t, depositsIterator.Valid())
	depositsIterator.Close()
	require.Equal(t, gov.StatusPassed, keeper.GetProposal(ctx, int64(proposalID)).GetStatus())

	// check refund deposits
	validatorCoins := ck.GetCoins(ctx, addrs[0])
	require.Equal(t, validatorCoins, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5000e8)})
	require.Equal(t, sdk.Coins(nil), ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))
}

func TestTickPassedVotingPeriodUnreachedQuorum(t *testing.T) {
	mapp, ck, keeper, stakeKeeper, addrs, pubKeys, _ := getMockApp(t, 3)

	validator0 := stake.NewValidator(sdk.ValAddress(addrs[0]), pubKeys[0], stake.Description{})
	validator1 := stake.NewValidator(sdk.ValAddress(addrs[1]), pubKeys[1], stake.Description{})

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{ProposerAddress: pubKeys[0].Address()})

	// create and delegate validator
	stakeKeeper.SetValidator(ctx, validator0)
	stakeKeeper.SetValidatorByConsAddr(ctx, validator0)
	stakeKeeper.SetValidator(ctx, validator1)
	stakeKeeper.SetValidatorByConsAddr(ctx, validator1)
	stakeKeeper.Delegate(ctx, sdk.AccAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 1000), validator0, true)
	stakeKeeper.Delegate(ctx, sdk.AccAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 2000), validator1, true)

	stakeKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	validator0, _ = stakeKeeper.GetValidator(ctx, validator0.OperatorAddr)

	govHandler := gov.NewHandler(keeper)

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))

	votingPeriod := 1000 * time.Second
	newProposalMsg := gov.NewMsgSubmitProposal("Test", "test", gov.ProposalTypeText, addrs[0], sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, votingPeriod)

	res := govHandler(ctx, newProposalMsg)
	require.True(t, res.IsOK())

	proposalID, _ := strconv.Atoi(string(res.Data))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	newDepositMsg := gov.NewMsgDeposit(addrs[1], int64(proposalID), sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)})
	res = govHandler(ctx, newDepositMsg)
	require.True(t, res.IsOK())
	gov.EndBlocker(ctx, keeper)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)
	newVoteMsg := gov.NewMsgVote(addrs[0], int64(proposalID), gov.OptionYes)
	res = govHandler(ctx, newVoteMsg)
	require.True(t, res.IsOK())
	require.Equal(t, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 2000e8)}, ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))

	gov.EndBlocker(ctx, keeper)

	// pass voting period
	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(votingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	require.True(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))
	depositsIterator := keeper.GetDeposits(ctx, int64(proposalID))
	require.True(t, depositsIterator.Valid())
	depositsIterator.Close()
	require.Equal(t, gov.StatusVotingPeriod, keeper.GetProposal(ctx, int64(proposalID)).GetStatus())

	gov.EndBlocker(ctx, keeper)

	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	depositsIterator = keeper.GetDeposits(ctx, int64(proposalID))
	require.False(t, depositsIterator.Valid())
	depositsIterator.Close()
	require.Equal(t, gov.StatusRejected, keeper.GetProposal(ctx, int64(proposalID)).GetStatus())

	// check refund deposits
	validatorCoins := ck.GetCoins(ctx, addrs[0])
	require.Equal(t, validatorCoins, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5000e8)})
	require.Equal(t, sdk.Coins(nil), ck.GetCoins(ctx, gov.DepositedCoinsAccAddr))
}

func TestTickPassedVotingPeriodAllAbstain(t *testing.T) {
	mapp, ck, keeper, stakeKeeper, addrs, pubKeys, _ := getMockApp(t, 3)

	validator0 := stake.NewValidator(sdk.ValAddress(addrs[0]), pubKeys[0], stake.Description{})
	validator1 := stake.NewValidator(sdk.ValAddress(addrs[1]), pubKeys[1], stake.Description{})

	mapp.BeginBlock(abci.RequestBeginBlock{})
	ctx := mapp.BaseApp.NewContext(sdk.RunTxModeDeliver, abci.Header{ProposerAddress: pubKeys[0].Address()})

	// create and delegate validator
	stakeKeeper.SetValidator(ctx, validator0)
	stakeKeeper.SetValidatorByConsAddr(ctx, validator0)
	stakeKeeper.SetValidator(ctx, validator1)
	stakeKeeper.SetValidatorByConsAddr(ctx, validator1)
	stakeKeeper.Delegate(ctx, sdk.AccAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 1000), validator0, true)
	stakeKeeper.Delegate(ctx, sdk.AccAddress(addrs[2]), sdk.NewCoin(gov.DefaultDepositDenom, 2000), validator1, true)

	stakeKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	validator0, _ = stakeKeeper.GetValidator(ctx, validator0.OperatorAddr)

	govHandler := gov.NewHandler(keeper)

	require.Nil(t, keeper.InactiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopInactiveProposalQueue(ctx, keeper))
	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	require.False(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))

	votingPeriod := 1000 * time.Second
	newProposalMsg := gov.NewMsgSubmitProposal("Test", "test", gov.ProposalTypeText, addrs[0], sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)}, votingPeriod)

	res := govHandler(ctx, newProposalMsg)
	require.True(t, res.IsOK())

	proposalID, _ := strconv.Atoi(string(res.Data))

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)

	newDepositMsg := gov.NewMsgDeposit(addrs[1], int64(proposalID), sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 1000e8)})
	res = govHandler(ctx, newDepositMsg)
	require.True(t, res.IsOK())
	gov.EndBlocker(ctx, keeper)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Duration(1) * time.Second)
	ctx = ctx.WithBlockHeader(newHeader)
	newVoteMsg := gov.NewMsgVote(addrs[1], int64(proposalID), gov.OptionAbstain)
	res = govHandler(ctx, newVoteMsg)
	require.True(t, res.IsOK())
	gov.EndBlocker(ctx, keeper)

	// pass voting period
	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(votingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	require.True(t, gov.ShouldPopActiveProposalQueue(ctx, keeper))
	depositsIterator := keeper.GetDeposits(ctx, int64(proposalID))
	require.True(t, depositsIterator.Valid())
	depositsIterator.Close()
	require.Equal(t, gov.StatusVotingPeriod, keeper.GetProposal(ctx, int64(proposalID)).GetStatus())

	gov.EndBlocker(ctx, keeper)

	require.Nil(t, keeper.ActiveProposalQueuePeek(ctx))
	depositsIterator = keeper.GetDeposits(ctx, int64(proposalID))
	require.False(t, depositsIterator.Valid())
	depositsIterator.Close()
	require.Equal(t, gov.StatusRejected, keeper.GetProposal(ctx, int64(proposalID)).GetStatus())

	// check refund deposits
	validatorCoins := ck.GetCoins(ctx, addrs[0])
	require.Equal(t, validatorCoins, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5000e8)})
}
