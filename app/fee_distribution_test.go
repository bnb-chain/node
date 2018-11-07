package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/val"
	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

func setup() (am auth.AccountKeeper, valMapper val.Mapper, ctx sdk.Context) {
	ms, capKey, cap2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	am = auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	valMapper = val.NewMapper(cap2)
	ctx = sdk.NewContext(ms, abci.Header{}, false, log.NewNopLogger())
	// setup proposer and other validators
	_, proposerAcc := testutils.NewAccount(ctx, am, 100)
	_, valAcc1 := testutils.NewAccount(ctx, am, 100)
	_, valAcc2 := testutils.NewAccount(ctx, am, 100)
	_, valAcc3 := testutils.NewAccount(ctx, am, 100)
	proposerValAddr := ed25519.GenPrivKey().PubKey().Address()
	val1ValAddr := ed25519.GenPrivKey().PubKey().Address()
	val2ValAddr := ed25519.GenPrivKey().PubKey().Address()
	val3ValAddr := ed25519.GenPrivKey().PubKey().Address()

	valMapper.SetVal(ctx, proposerAcc.GetAddress(), proposerValAddr)
	valMapper.SetVal(ctx, valAcc1.GetAddress(), val1ValAddr)
	valMapper.SetVal(ctx, valAcc2.GetAddress(), val2ValAddr)
	valMapper.SetVal(ctx, valAcc3.GetAddress(), val3ValAddr)

	proposer := abci.Validator{Address: proposerValAddr, Power: 10}
	ctx = ctx.WithBlockHeader(abci.Header{ProposerAddress: proposerValAddr}).WithVoteInfos([]abci.VoteInfo{
		{Validator: proposer, SignedLastBlock: true},
		{Validator: abci.Validator{Address: val1ValAddr, Power: 10}, SignedLastBlock: true},
		{Validator: abci.Validator{Address: val2ValAddr, Power: 10}, SignedLastBlock: true},
		{Validator: abci.Validator{Address: val3ValAddr, Power: 10}, SignedLastBlock: true},
	})

	return
}

func checkBalance(t *testing.T, ctx sdk.Context, am auth.AccountKeeper, valMapper val.Mapper, balances []int64) {
	for i, voteInfo := range ctx.VoteInfos() {
		accAddr := getAccAddr(ctx, valMapper, voteInfo.Validator.Address)
		valAcc := am.GetAccount(ctx, accAddr)
		require.Equal(t, balances[i], valAcc.GetCoins().AmountOf(types.NativeToken).Int64())
	}
}

func TestNoFeeDistribution(t *testing.T) {
	// setup
	am, valMapper, ctx := setup()
	fee := tx.Fee(ctx)
	require.True(t, true, fee.IsEmpty())

	distributeFee(ctx, am, valMapper)
	checkBalance(t, ctx, am, valMapper, []int64{100, 100, 100, 100})
}

func TestFeeDistribution2Proposer(t *testing.T) {
	// setup
	am, valMapper, ctx := setup()
	ctx = tx.WithFee(ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 10)}, types.FeeForProposer))
	distributeFee(ctx, am, valMapper)
	checkBalance(t, ctx, am, valMapper, []int64{110, 100, 100, 100})
}

func TestFeeDistribution2AllValidators(t *testing.T) {
	// setup
	am, valMapper, ctx := setup()
	// fee amount can be divided evenly
	ctx = tx.WithFee(ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 40)}, types.FeeForAll))
	distributeFee(ctx, am, valMapper)
	checkBalance(t, ctx, am, valMapper, []int64{110, 110, 110, 110})

	// cannot be divided evenly
	ctx = tx.WithFee(ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 50)}, types.FeeForAll))
	distributeFee(ctx, am, valMapper)
	checkBalance(t, ctx, am, valMapper, []int64{124, 122, 122, 122})
}
