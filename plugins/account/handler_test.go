package account

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/bnb-chain/node/common/testutils"
	"github.com/bnb-chain/node/plugins/account/scripts"
	"github.com/bnb-chain/node/wire"
)

func setup() (sdk.Context, sdk.Handler, auth.AccountKeeper) {
	ms, _, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	accountKeeper := auth.NewAccountKeeper(cdc, capKey2, auth.ProtoBaseAccount)
	handler := NewHandler(accountKeeper)

	accountStore := ms.GetKVStore(capKey2)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1},
		sdk.RunTxModeDeliver, log.NewNopLogger()).
		WithAccountCache(auth.NewAccountCache(accountStoreCache))
	return ctx, handler, accountKeeper
}

func TestHandleIssueToken(t *testing.T) {
	ctx, handler, accountKeeper := setup()

	// Invalid account
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)
	msg := NewSetAccountFlagsMsg(acc.GetAddress(), scripts.TransferMemoCheckerFlag)
	sdkResult := handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())

	// Invalid address
	msg = NewSetAccountFlagsMsg(acc.GetAddress()[0:10], scripts.TransferMemoCheckerFlag)
	sdkResult = handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())

	_, acc = testutils.NewNamedAccount(ctx, accountKeeper, 100e8)
	msg = NewSetAccountFlagsMsg(acc.GetAddress(), scripts.TransferMemoCheckerFlag)
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())
}
