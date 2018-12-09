package app

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/pub"
	"github.com/BiJie/BinanceChain/app/val"
	"github.com/BiJie/BinanceChain/common/fees"
	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

func getAccountCache(cdc *codec.Codec, ms sdk.MultiStore, accountKey *sdk.KVStoreKey) sdk.AccountCache {
	accountStore := ms.GetKVStore(accountKey)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	return auth.NewAccountCache(accountStoreCache)
}

func setup() (am auth.AccountKeeper, valMapper val.Mapper, ctx sdk.Context, proposerAcc, valAcc1, valAcc2, valAcc3 sdk.Account) {
	ms, capKey, cap2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	am = auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	valMapper = val.NewMapper(cap2)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx = sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	// setup proposer and other validators
	_, proposerAcc = testutils.NewAccount(ctx, am, 100)
	_, valAcc1 = testutils.NewAccount(ctx, am, 100)
	_, valAcc2 = testutils.NewAccount(ctx, am, 100)
	_, valAcc3 = testutils.NewAccount(ctx, am, 100)
	proposerValAddr := ed25519.GenPrivKey().PubKey().Address()
	val1ValAddr := ed25519.GenPrivKey().PubKey().Address()
	val2ValAddr := ed25519.GenPrivKey().PubKey().Address()
	val3ValAddr := ed25519.GenPrivKey().PubKey().Address()

	valMapper.SetVal(ctx, proposerValAddr, proposerAcc.GetAddress())
	valMapper.SetVal(ctx, val1ValAddr, valAcc1.GetAddress())
	valMapper.SetVal(ctx, val2ValAddr, valAcc2.GetAddress())
	valMapper.SetVal(ctx, val3ValAddr, valAcc3.GetAddress())

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
		require.Equal(t, balances[i], valAcc.GetCoins().AmountOf(types.NativeTokenSymbol))
	}
}

func TestNoFeeDistribution(t *testing.T) {
	// setup
	am, valMapper, ctx, _, _, _, _ := setup()
	fee := fees.Pool.BlockFees()
	require.True(t, true, fee.IsEmpty())

	blockFee := distributeFee(ctx, am, valMapper, true)
	require.Equal(t, pub.BlockFee{0, "", nil}, blockFee)
	checkBalance(t, ctx, am, valMapper, []int64{100, 100, 100, 100})
}

func TestFeeDistribution2Proposer(t *testing.T) {
	// setup
	am, valMapper, ctx, proposerAcc, _, _, _ := setup()
	fees.Pool.AddFee(types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, types.FeeForProposer))
	blockFee := distributeFee(ctx, am, valMapper, true)
	require.Equal(t, pub.BlockFee{0, "BNB:10", []string{string(proposerAcc.GetAddress())}}, blockFee)
	checkBalance(t, ctx, am, valMapper, []int64{110, 100, 100, 100})
}

func TestFeeDistribution2AllValidators(t *testing.T) {
	// setup
	am, valMapper, ctx, proposerAcc, valAcc1, valAcc2, valAcc3 := setup()
	// fee amount can be divided evenly
	fees.Pool.AddFee(types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 40)}, types.FeeForAll))
	blockFee := distributeFee(ctx, am, valMapper, true)
	require.Equal(t, pub.BlockFee{0, "BNB:40", []string{string(proposerAcc.GetAddress()), string(valAcc1.GetAddress()), string(valAcc2.GetAddress()), string(valAcc3.GetAddress())}}, blockFee)
	checkBalance(t, ctx, am, valMapper, []int64{110, 110, 110, 110})

	// cannot be divided evenly
	fees.Pool.AddFee(types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 50)}, types.FeeForAll))
	blockFee = distributeFee(ctx, am, valMapper, true)
	require.Equal(t, pub.BlockFee{0, "BNB:50", []string{string(proposerAcc.GetAddress()), string(valAcc1.GetAddress()), string(valAcc2.GetAddress()), string(valAcc3.GetAddress())}}, blockFee)
	checkBalance(t, ctx, am, valMapper, []int64{124, 122, 122, 122})
}
