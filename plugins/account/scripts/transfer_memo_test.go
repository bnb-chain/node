package scripts

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	bankclient "github.com/cosmos/cosmos-sdk/x/bank/client"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/bnb-chain/node/common/testutils"
	"github.com/bnb-chain/node/common/upgrade"
	"github.com/bnb-chain/node/wire"
)

func setup() (sdk.Context, auth.AccountKeeper) {
	ms, _, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	accountKeeper := auth.NewAccountKeeper(cdc, capKey2, auth.ProtoBaseAccount)

	accountStore := ms.GetKVStore(capKey2)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1},
		sdk.RunTxModeDeliver, log.NewNopLogger()).
		WithAccountCache(auth.NewAccountCache(accountStoreCache))
	return ctx, accountKeeper
}

func TestTransferMemoScript(t *testing.T) {
	ctx, accountKeeper := setup()

	_, acc0 := testutils.NewNamedAccount(ctx, accountKeeper, 100e8)
	_, acc1 := testutils.NewNamedAccount(ctx, accountKeeper, 100e8)

	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP12, 10)
	transferMemoScript := generateTransferMemoCheckScript(accountKeeper)

	var tx sdk.Tx
	msg := bankclient.CreateMsg(acc0.GetAddress(), acc1.GetAddress(), testutils.NewNativeTokens(10000))

	// Before BEP12 upgrade
	upgrade.Mgr.SetHeight(5)
	// receiver account flags is zero
	// memo is empty
	tx = auth.StdTx{Memo: "", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err := transferMemoScript(ctx, msg)
	require.NoError(t, err)

	// After BEP12 upgrade
	upgrade.Mgr.SetHeight(11)

	// receiver account flags is zero
	// memo is empty
	tx = auth.StdTx{Memo: "", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.NoError(t, err)
	// memo contains both letters and numbers
	tx = auth.StdTx{Memo: "123456abc", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.NoError(t, err)
	// memo only contains numbers
	tx = auth.StdTx{Memo: "1234567890", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.NoError(t, err)

	// receiver account flags enable a non-existing script
	acc1.SetFlags(0x0000000000000002)
	accountKeeper.SetAccount(ctx, acc1)
	// memo is empty
	tx = auth.StdTx{Memo: "", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.NoError(t, err)
	// memo contains both letters and numbers
	tx = auth.StdTx{Memo: "123456abc", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.NoError(t, err)
	// memo only contains numbers
	tx = auth.StdTx{Memo: "1234567890", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.NoError(t, err)

	// receiver account flags enable transfer memo checker
	acc1.SetFlags(TransferMemoCheckerFlag)
	accountKeeper.SetAccount(ctx, acc1)
	// memo is empty
	tx = auth.StdTx{Memo: "", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.Error(t, err)
	// memo contains both letters and numbers
	tx = auth.StdTx{Memo: "123456abc", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.Error(t, err)
	// memo only contains numbers
	tx = auth.StdTx{Memo: "1234567890", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.NoError(t, err)

	// receiver account flags enable many flags and transfer memo checker flag is included
	acc1.SetFlags(0x0000000000000003)
	accountKeeper.SetAccount(ctx, acc1)
	// memo is empty
	tx = auth.StdTx{Memo: "", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.Error(t, err)
	// memo contains both letters and numbers
	tx = auth.StdTx{Memo: "123456abc", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.Error(t, err)
	// memo only contains numbers
	tx = auth.StdTx{Memo: "1234567890", Msgs: []sdk.Msg{msg}}
	ctx = ctx.WithTx(tx)
	err = transferMemoScript(ctx, msg)
	require.NoError(t, err)

}
