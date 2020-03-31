package issue

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/minitokens/store"
	"github.com/binance-chain/node/wire"
)

func setup() (sdk.Context, sdk.Handler, auth.AccountKeeper, store.MiniTokenMapper) {
	ms, capKey1, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	tokenMapper := store.NewMiniTokenMapper(cdc, capKey1)
	accountKeeper := auth.NewAccountKeeper(cdc, capKey2, auth.ProtoBaseAccount)
	bankKeeper := bank.NewBaseKeeper(accountKeeper)
	handler := NewHandler(tokenMapper, bankKeeper)

	accountStore := ms.GetKVStore(capKey2)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1},
		sdk.RunTxModeDeliver, log.NewNopLogger()).
		WithAccountCache(auth.NewAccountCache(accountStoreCache))
	return ctx, handler, accountKeeper, tokenMapper
}

func TestHandleIssueToken(t *testing.T) {
	ctx, handler, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg := NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 100001e8, 100000e8, false, "http://www.xyz.com/nnb.json")
	sdkResult := handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "max total supply is too large")

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg = NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 100001e8, 100000e8-100, false, "http://www.xyz.com/nnb.json")
	sdkResult = handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "total supply should be a multiple of 100000000")

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg = NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 100000e8, 100000e8, false, "http://www.xyz.com/nnb.json")
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err := types.NewMiniToken("New BNB", "NNB-000M", 100000e8, 100000e8, acc.GetAddress(), false, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, token)

	sdkResult = handler(ctx, msg)
	require.Contains(t, sdkResult.Log, "symbol(NNB) already exists")
}

func TestHandleMintToken(t *testing.T) {
	ctx, handler, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)
	mintMsg := NewMintMsg(acc.GetAddress(), "NNB-000M", 10000e8)
	sdkResult := handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "symbol(NNB-000M) does not exist")

	issueMsg := NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 100000e8, 90000e8, true, "http://www.xyz.com/nnb.json")
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = handler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	sdkResult = handler(ctx, mintMsg)
	token, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err := types.NewMiniToken("New BNB", "NNB-000M", 100000e8, 100000e8, acc.GetAddress(), true, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, token)

	sdkResult = handler(ctx, mintMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "mint amount is too large")

	invalidMintMsg := NewMintMsg(acc.GetAddress(), "NNB-000M", types.MiniTokenMaxTotalSupplyUpperBound)
	sdkResult = handler(ctx, invalidMintMsg)
	require.Contains(t, sdkResult.Log, "mint amount is too large")

	_, acc2 := testutils.NewAccount(ctx, accountKeeper, 100e8)
	invalidMintMsg = NewMintMsg(acc2.GetAddress(), "NNB-000M", types.MiniTokenMaxTotalSupplyUpperBound)
	sdkResult = handler(ctx, invalidMintMsg)
	require.Contains(t, sdkResult.Log, "only the owner can mint token NNB")

	// issue a non-mintable token
	issueMsg = NewIssueMsg(acc.GetAddress(), "New BNB2", "NNB2", 100000e8, 100000e8, false, "http://www.xyz.com/nnb.json")
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = handler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	mintMsg = NewMintMsg(acc.GetAddress(), "NNB2-000M", 10000e8)
	sdkResult = handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "token(NNB2-000M) cannot be minted")

	// mint native token
	invalidMintMsg = NewMintMsg(acc.GetAddress(), "BNB", 10000e8)
	require.Contains(t, invalidMintMsg.ValidateBasic().Error(), "suffixed mini-token symbol must contain a hyphen")
}
