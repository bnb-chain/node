package slashing

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/bsc"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/stretchr/testify/require"
)

func TestSideChainSlashDoubleSign(t *testing.T) {
	slashParams := DefaultParams()
	slashParams.DoubleSignUnbondDuration = 5 * time.Second
	slashParams.MaxEvidenceAge = math.MaxInt64
	slashParams.DoubleSignSlashAmount = 6000e8
	slashParams.SubmitterReward = 3000e8
	submitter := sdk.AccAddress(addrs[2])
	ctx, sideCtx, bankKeeper, stakeKeeper, _, keeper := createSideTestInput(t, slashParams)

	// create a malicious validator
	ctx = ctx.WithBlockHeight(100)
	bondAmount := int64(10000e8)
	mValAddr := addrs[0]
	mSideConsAddr, err := sdk.HexDecode("0x625448c3f21AB4636bBCef84Baaf8D6cCdE13c3F")
	require.Nil(t, err)
	mSideFeeAddr := createSideAddr(20)
	msgCreateVal := newTestMsgCreateSideValidator(mValAddr, mSideConsAddr, mSideFeeAddr, bondAmount)
	got := stake.NewHandler(stakeKeeper, gov.Keeper{})(ctx, msgCreateVal)
	require.True(t, got.IsOK(), "expected create validator msg to be ok, got: %v", got)
	// end block
	stake.EndBreatheBlock(ctx, stakeKeeper)

	ctx = ctx.WithBlockHeight(200)
	ValAddr1 := addrs[1]
	sideConsAddr1, sideFeeAddr1 := createSideAddr(20), createSideAddr(20)
	msgCreateVal1 := newTestMsgCreateSideValidator(ValAddr1, sideConsAddr1, sideFeeAddr1, bondAmount)
	got1 := stake.NewHandler(stakeKeeper, gov.Keeper{})(ctx, msgCreateVal1)
	require.True(t, got1.IsOK(), "expected create validator msg to be ok, got: %v", got1)
	// end block
	stake.EndBreatheBlock(ctx, stakeKeeper)
	stakingPoolBalance := bankKeeper.GetCoins(ctx, stake.DelegationAccAddr).AmountOf("steak")
	require.EqualValues(t, bondAmount*2, stakingPoolBalance)

	ctx = ctx.WithBlockHeight(300)
	headers := make([]bsc.Header, 0)
	headersJson := `[{"parentHash":"0x6116de25352c93149542e950162c7305f207bbc17b0eb725136b78c80aed79cc","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","miner":"0x0000000000000000000000000000000000000000","stateRoot":"0xe7cb9d2fd449f7bd11126bff55266e7b74936f2f230e21d44d75c04b7780dfeb","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","difficulty":"0x20000","number":"0x1","gasLimit":"0x47e7c4","gasUsed":"0x0","timestamp":"0x5ea6a002","extraData":"0x0000000000000000000000000000000000000000000000000000000000000000bb4a77b57c2a82de97b557442883ee19d481a415fc76d3833de83ba37f2d8674375f85fd96affd603244e3448a2b101c40511aa18ce8c1edf4e940dec648ac1300","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","hash":"0x1532065752393ff2f6e7ef9b64f80d6e10efe42a4d9bdd8149fcbac6f86b365b"},{"parentHash":"0x6116de25352c93149542e950162c7305f207bbc17b0eb725136b78c80aed79cc","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","miner":"0x0000000000000000000000000000000000000000","stateRoot":"0xe7cb9d2fd449f7bd11126bff55266e7b74936f2f230e21d44d75c04b7780dfeb","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","difficulty":"0x20000","number":"0x1","gasLimit":"0x47e7c4","gasUsed":"0x64","timestamp":"0x5ea6a002","extraData":"0x000000000000000000000000000000000000000000000000000000000000000055a9a47820e18c025d0b98a722c3fb83d28e4547e0090cbe5cc17683b7f25d5e18c6e359631ec10d9c08ceaafc9e9847de3de18694d073af9515638eee73c58e00","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","hash":"0x811a42453f826f05e9d85998551636f59eb740d5b03fe2416700058a4f31ca1e"}]`
	err = json.Unmarshal([]byte(headersJson), &headers)
	require.Nil(t, err)

	feesInPoolBefore := fees.Pool.BlockFees().Tokens.AmountOf("steak")
	msgSubmitEvidence := NewMsgBscSubmitEvidence(submitter, headers)
	got = NewHandler(keeper)(ctx, msgSubmitEvidence)
	require.True(t, got.IsOK(), "expected submit evidence msg to be ok, got: %v", got)

	mValidator, found := stakeKeeper.GetValidator(sideCtx, mValAddr)
	require.True(t, found)
	require.True(t, mValidator.Jailed)
	require.EqualValues(t, bondAmount-slashParams.DoubleSignSlashAmount, mValidator.Tokens.RawInt())
	require.EqualValues(t, bondAmount-slashParams.DoubleSignSlashAmount, mValidator.DelegatorShares.RawInt())

	submitterBalance := bankKeeper.GetCoins(ctx, submitter).AmountOf("steak")
	require.EqualValues(t, initCoins+slashParams.SubmitterReward, submitterBalance)

	require.EqualValues(t, slashParams.DoubleSignSlashAmount-slashParams.SubmitterReward, fees.Pool.BlockFees().Tokens.AmountOf("steak")-feesInPoolBefore)

	slashRecord, found := keeper.getSlashRecord(sideCtx, mSideConsAddr, DoubleSign, 1)
	require.True(t, found)
	require.EqualValues(t, slashParams.DoubleSignSlashAmount, slashRecord.SlashAmt)
	require.EqualValues(t, ctx.BlockHeader().Time.Add(slashParams.DoubleSignUnbondDuration).Unix(), slashRecord.JailUntil.Unix())

	expectedStakingPoolBalance := stakingPoolBalance - slashParams.DoubleSignSlashAmount
	stakingPoolBalance = bankKeeper.GetCoins(ctx, stake.DelegationAccAddr).AmountOf("steak")
	require.EqualValues(t, expectedStakingPoolBalance, stakingPoolBalance)
	// end block
	stake.EndBreatheBlock(ctx, stakeKeeper)

	realSlashedAmt := sdk.MinInt64(slashParams.DoubleSignSlashAmount, mValidator.Tokens.RawInt())
	realSubmitterReward := sdk.MinInt64(slashParams.SubmitterReward, mValidator.Tokens.RawInt())
	expectedAfterValTokensLeft := mValidator.Tokens.RawInt() - realSlashedAmt
	expectedAfterSubmitterBalance := submitterBalance + realSubmitterReward
	// send submit evidence tx
	ctx = ctx.WithBlockHeight(350).WithBlockTime(time.Now())
	headersJson = `[{"parentHash":"0x9dc70cfc956472119b82b6bbc1e6be139a68d03e99a4dcec1ccd0d9b4fd9c822","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","miner":"0x0000000000000000000000000000000000000000","stateRoot":"0x0988fe1673073b5e1c5f052e5a9a30ec871f90768041a7bfed5ee03f6304b138","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","difficulty":"0x20000","number":"0x2","gasLimit":"0x47e7c4","gasUsed":"0x0","timestamp":"0x5eb8fc64","extraData":"0x0000000000000000000000000000000000000000000000000000000000000000e0c6926949e84f0adc499a615795e78114f994a6bb8e2861a24ed2d875e78a2971a87054001132fec64dfce5bbed65af5802d416ccfcb981856f8aa9e7edaa4d01","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","hash":"0x132a6caa72f3e5e98b086c5bcf2d7fe95ac612152114caca3e95bc8ec8e068a0"},{"parentHash":"0x9dc70cfc956472119b82b6bbc1e6be139a68d03e99a4dcec1ccd0d9b4fd9c822","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","miner":"0x0000000000000000000000000000000000000000","stateRoot":"0x0988fe1673073b5e1c5f052e5a9a30ec871f90768041a7bfed5ee03f6304b138","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","difficulty":"0x20000","number":"0x2","gasLimit":"0x47e7c4","gasUsed":"0x64","timestamp":"0x5eb8fc64","extraData":"0x0000000000000000000000000000000000000000000000000000000000000000afb8c1b842fcbdd306415fb6efdd34bcfa2f03f68add76ef3a651e0c975a3b176d3f37b8819537cba69e79ac91926e8f43753c49daa113a6c42268b64f01f8f901","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","hash":"0x9c4f11247e697a75ba633f87112895d537156265bf52b6e85e43e551b4d1cb78"}]`
	err = json.Unmarshal([]byte(headersJson), &headers)
	require.Nil(t, err)
	msgSubmitEvidence = NewMsgBscSubmitEvidence(submitter, headers)
	got = NewHandler(keeper)(ctx, msgSubmitEvidence)
	require.True(t, got.IsOK(), "expected submit evidence msg to be ok, got: %v", got)

	// check balance
	expectedStakingPoolBalance = stakingPoolBalance - realSlashedAmt
	stakingPoolBalance = bankKeeper.GetCoins(ctx, stake.DelegationAccAddr).AmountOf("steak")
	require.EqualValues(t, expectedStakingPoolBalance, stakingPoolBalance)

	mValidator, found = stakeKeeper.GetValidator(sideCtx, mValAddr)
	require.True(t, found)
	require.True(t, mValidator.Jailed)
	require.EqualValues(t, expectedAfterValTokensLeft, mValidator.Tokens.RawInt())
	require.EqualValues(t, expectedAfterValTokensLeft, mValidator.DelegatorShares.RawInt())

	submitterBalance = bankKeeper.GetCoins(ctx, submitter).AmountOf("steak")
	require.EqualValues(t, expectedAfterSubmitterBalance, submitterBalance)

	validator1, found := stakeKeeper.GetValidator(sideCtx, ValAddr1)
	require.True(t, found)
	distributionAddr1 := validator1.DistributionAddr
	distributionAddr1Balance := bankKeeper.GetCoins(ctx, distributionAddr1).AmountOf("steak")
	require.EqualValues(t, realSlashedAmt-realSubmitterReward, distributionAddr1Balance)

	slashRecord, found = keeper.getSlashRecord(sideCtx, mSideConsAddr, DoubleSign, 2)
	require.True(t, found)
	require.EqualValues(t, realSlashedAmt, slashRecord.SlashAmt)
	require.EqualValues(t, ctx.BlockHeader().Time.Add(slashParams.DoubleSignUnbondDuration).Unix(), slashRecord.JailUntil.Unix())
}

func TestSideChainSlashDoubleSignUBD(t *testing.T) {

	slashParams := DefaultParams()
	slashParams.MaxEvidenceAge = math.MaxInt64
	slashParams.DoubleSignSlashAmount = 6000e8
	slashParams.SubmitterReward = 3000e8
	submitter := sdk.AccAddress(addrs[2])
	ctx, sideCtx, bankKeeper, stakeKeeper, _, keeper := createSideTestInput(t, slashParams)

	// create a malicious validator
	ctx = ctx.WithBlockHeight(100)
	bondAmount := int64(10000e8)
	mValAddr := addrs[0]
	mSideConsAddr, err := sdk.HexDecode("0x625448c3f21AB4636bBCef84Baaf8D6cCdE13c3F")
	require.Nil(t, err)
	mSideFeeAddr := createSideAddr(20)
	msgCreateVal := newTestMsgCreateSideValidator(mValAddr, mSideConsAddr, mSideFeeAddr, bondAmount)
	got := stake.NewHandler(stakeKeeper, gov.Keeper{})(ctx, msgCreateVal)
	require.True(t, got.IsOK(), "expected create validator msg to be ok, got: %v", got)
	// end block
	stake.EndBreatheBlock(ctx, stakeKeeper)

	ctx = ctx.WithBlockHeight(150)
	msgUnDelegate := newTestMsgSideUnDelegate(sdk.AccAddress(mValAddr), mValAddr, 5000e8)
	got = stake.NewHandler(stakeKeeper, gov.Keeper{})(ctx, msgUnDelegate)
	require.True(t, got.IsOK(), "expected unDelegate msg to be ok, got: %v", got)
	ubd, found := stakeKeeper.GetUnbondingDelegation(sideCtx, sdk.AccAddress(mValAddr), mValAddr)
	require.True(t, found)
	require.EqualValues(t, 5000e8, ubd.Balance.Amount)

	ctx = ctx.WithBlockHeight(200)
	// end block
	stake.EndBreatheBlock(ctx, stakeKeeper)

	ctx = ctx.WithBlockHeight(201)
	headers := make([]bsc.Header, 0)
	headersJson := `[{"parentHash":"0x6116de25352c93149542e950162c7305f207bbc17b0eb725136b78c80aed79cc","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","miner":"0x0000000000000000000000000000000000000000","stateRoot":"0xe7cb9d2fd449f7bd11126bff55266e7b74936f2f230e21d44d75c04b7780dfeb","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","difficulty":"0x20000","number":"0x1","gasLimit":"0x47e7c4","gasUsed":"0x0","timestamp":"0x5ea6a002","extraData":"0x0000000000000000000000000000000000000000000000000000000000000000bb4a77b57c2a82de97b557442883ee19d481a415fc76d3833de83ba37f2d8674375f85fd96affd603244e3448a2b101c40511aa18ce8c1edf4e940dec648ac1300","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","hash":"0x1532065752393ff2f6e7ef9b64f80d6e10efe42a4d9bdd8149fcbac6f86b365b"},{"parentHash":"0x6116de25352c93149542e950162c7305f207bbc17b0eb725136b78c80aed79cc","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","miner":"0x0000000000000000000000000000000000000000","stateRoot":"0xe7cb9d2fd449f7bd11126bff55266e7b74936f2f230e21d44d75c04b7780dfeb","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","difficulty":"0x20000","number":"0x1","gasLimit":"0x47e7c4","gasUsed":"0x64","timestamp":"0x5ea6a002","extraData":"0x000000000000000000000000000000000000000000000000000000000000000055a9a47820e18c025d0b98a722c3fb83d28e4547e0090cbe5cc17683b7f25d5e18c6e359631ec10d9c08ceaafc9e9847de3de18694d073af9515638eee73c58e00","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","hash":"0x811a42453f826f05e9d85998551636f59eb740d5b03fe2416700058a4f31ca1e"}]`
	err = json.Unmarshal([]byte(headersJson), &headers)
	require.Nil(t, err)

	feesInPoolBefore := fees.Pool.BlockFees().Tokens.AmountOf("steak")
	msgSubmitEvidence := NewMsgBscSubmitEvidence(submitter, headers)
	got = NewHandler(keeper)(ctx, msgSubmitEvidence)
	require.True(t, got.IsOK(), "expected submit evidence msg to be ok, got: %v", got)

	mValidator, found := stakeKeeper.GetValidator(sideCtx, mValAddr)
	require.True(t, found)
	require.True(t, mValidator.Jailed)
	require.EqualValues(t, 0, mValidator.Tokens.RawInt())
	require.EqualValues(t, 0, mValidator.DelegatorShares.RawInt())

	ubd, found = stakeKeeper.GetUnbondingDelegation(sideCtx, sdk.AccAddress(mValAddr), mValAddr)
	require.True(t, found)
	require.EqualValues(t, 4000e8, ubd.Balance.Amount)

	submitterBalance := bankKeeper.GetCoins(ctx, submitter).AmountOf("steak")
	require.EqualValues(t, initCoins+slashParams.SubmitterReward, submitterBalance)

	require.EqualValues(t, slashParams.DoubleSignSlashAmount-slashParams.SubmitterReward, fees.Pool.BlockFees().Tokens.AmountOf("steak")-feesInPoolBefore)

	stakingPoolBalance := bankKeeper.GetCoins(ctx, stake.DelegationAccAddr).AmountOf("steak")
	require.EqualValues(t, 4000e8, stakingPoolBalance)

}
