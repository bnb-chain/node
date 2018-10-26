package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

func setup() (mapper auth.AccountKeeper, ctx sdk.Context) {
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper = auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	ctx = sdk.NewContext(ms, abci.Header{}, false, log.NewNopLogger())
	// setup proposer and other validators
	_, proposerAcc := testutils.NewAccount(ctx, mapper, 100)
	_, valAcc1 := testutils.NewAccount(ctx, mapper, 100)
	_, valAcc2 := testutils.NewAccount(ctx, mapper, 100)
	_, valAcc3 := testutils.NewAccount(ctx, mapper, 100)

	proposer := abci.Validator{Address: proposerAcc.GetAddress(), Power: 10}
	ctx = ctx.WithBlockHeader(abci.Header{Proposer: proposer}).WithSigningValidators([]abci.SigningValidator{
		{proposer, true},
		{abci.Validator{Address: valAcc1.GetAddress(), Power: 10}, true},
		{abci.Validator{Address: valAcc2.GetAddress(), Power: 10}, true},
		{abci.Validator{Address: valAcc3.GetAddress(), Power: 10}, true},
	})

	return
}

func checkBalance(t *testing.T, ctx sdk.Context, am auth.AccountKeeper, vals []int64) {
	for i, val := range ctx.SigningValidators() {
		valAcc := am.GetAccount(ctx, val.Validator.Address)
		require.Equal(t, vals[i], valAcc.GetCoins().AmountOf(types.NativeToken).Int64())
	}
}

func TestNoFeeDistribution(t *testing.T) {
	// setup
	am, ctx := setup()
	fee := tx.Fee(ctx)
	require.True(t, true, fee.IsEmpty())

	distributeFee(ctx, am)
	checkBalance(t, ctx, am, []int64{100, 100, 100, 100})
}

func TestFeeDistribution2Proposer(t *testing.T) {
	// setup
	am, ctx := setup()
	ctx = tx.WithFee(ctx, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeToken, 10)}, types.FeeForProposer))
	distributeFee(ctx, am)
	checkBalance(t, ctx, am, []int64{110, 100, 100, 100})
}

func TestFeeDistribution2AllValidators(t *testing.T) {
	// setup
	am, ctx := setup()
	// fee amount can be divided evenly
	ctx = tx.WithFee(ctx, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeToken, 40)}, types.FeeForAll))
	distributeFee(ctx, am)
	checkBalance(t, ctx, am, []int64{110, 110, 110, 110})

	// cannot be divided evenly
	ctx = tx.WithFee(ctx, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeToken, 50)}, types.FeeForAll))
	distributeFee(ctx, am)
	checkBalance(t, ctx, am, []int64{124, 122, 122, 122})
}
