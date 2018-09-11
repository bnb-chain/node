package tx_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/common/types"

	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/tx"
)

var (
	emptyCoins = sdk.Coins{}
	oneCoin    = sdk.Coins{sdk.NewInt64Coin("foocoin", 1)}
	twoCoins   = sdk.Coins{sdk.NewInt64Coin("foocoin", 2)}
)

func TestFeeCollectionKeeperGetSet(t *testing.T) {
	ms, _, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()

	// make context and keeper
	ctx := types.NewContext(ms, abci.Header{}, false, log.NewNopLogger())
	fck := tx.NewFeeCollectionKeeper(cdc, capKey2)

	// no coins initially
	currFees := fck.GetCollectedFees(ctx)
	require.True(t, currFees.IsEqual(emptyCoins))

	// set feeCollection to oneCoin
	fck.SetCollectedFees(ctx, oneCoin)

	// check that it is equal to oneCoin
	require.True(t, fck.GetCollectedFees(ctx).IsEqual(oneCoin))
}

func TestFeeCollectionKeeperAdd(t *testing.T) {
	ms, _, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()

	// make context and keeper
	ctx := types.NewContext(ms, abci.Header{}, false, log.NewNopLogger())
	fck := tx.NewFeeCollectionKeeper(cdc, capKey2)

	// no coins initially
	require.True(t, fck.GetCollectedFees(ctx).IsEqual(emptyCoins))

	// add oneCoin and check that pool is now oneCoin
	fck.AddCollectedFees(ctx, oneCoin)
	require.True(t, fck.GetCollectedFees(ctx).IsEqual(oneCoin))

	// add oneCoin again and check that pool is now twoCoins
	fck.AddCollectedFees(ctx, oneCoin)
	require.True(t, fck.GetCollectedFees(ctx).IsEqual(twoCoins))
}

func TestFeeCollectionKeeperClear(t *testing.T) {
	ms, _, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()

	// make context and keeper
	ctx := types.NewContext(ms, abci.Header{}, false, log.NewNopLogger())
	fck := tx.NewFeeCollectionKeeper(cdc, capKey2)

	// set coins initially
	fck.SetCollectedFees(ctx, twoCoins)
	require.True(t, fck.GetCollectedFees(ctx).IsEqual(twoCoins))

	// clear fees and see that pool is now empty
	fck.ClearCollectedFees(ctx)
	require.True(t, fck.GetCollectedFees(ctx).IsEqual(emptyCoins))
}
