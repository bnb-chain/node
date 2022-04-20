package ownership

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/bnb-chain/node/common/testutils"
	"github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/plugins/tokens/issue"
	"github.com/bnb-chain/node/plugins/tokens/store"
	"github.com/bnb-chain/node/wire"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/require"
)

func setup() (sdk.Context, sdk.Handler, sdk.Handler, auth.AccountKeeper, store.Mapper) {
	ms, capKey1, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	cdc.RegisterInterface((*types.IToken)(nil), nil)
	cdc.RegisterConcrete(&types.Token{}, "bnbchain/Token", nil)
	cdc.RegisterConcrete(&types.MiniToken{}, "bnbchain/MiniToken", nil)
	tokenMapper := store.NewMapper(cdc, capKey1)
	//app.AccountKeeper = auth.NewAccountKeeper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	accountKeeper := auth.NewAccountKeeper(cdc, capKey2, types.ProtoAppAccount)
	bankKeeper := bank.NewBaseKeeper(accountKeeper)
	handler := NewHandler(tokenMapper, bankKeeper)
	issueHandler := issue.NewHandler(tokenMapper, bankKeeper)

	accountStore := ms.GetKVStore(capKey2)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1},
		sdk.RunTxModeDeliver, log.NewNopLogger()).
		WithAccountCache(auth.NewAccountCache(accountStoreCache))
	return ctx, handler, issueHandler, accountKeeper, tokenMapper
}

func TestHandleTransferTokenOwner(t *testing.T) {
	ctx, handler, issueHandler, accountKeeper, tokenMapper := setup()
	_, originOwner := testutils.NewAccount(ctx, accountKeeper, 100e8)
	_, newOwner := testutils.NewAccount(ctx, accountKeeper, 100e8)
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	issueMsg := issue.NewIssueMsg(originOwner.GetAddress(), "New BNB", "NNB", 10000e8, false)
	sdkResult := issueHandler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err := tokenMapper.GetToken(ctx, "NNB-000")
	require.Nil(t, err)

	// test wrong symbol
	ctx = ctx.WithValue(baseapp.TxHashKey, "001")
	msg := NewTransferOwnershipMsg(originOwner.GetAddress(), "NNB-001", newOwner.GetAddress())
	sdkResult = handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	token, err = tokenMapper.GetToken(ctx, "NNB-000")
	require.Nil(t, err)
	require.Equal(t, originOwner.GetAddress(), token.GetOwner())

	// test wrong owner
	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	msg = NewTransferOwnershipMsg(acc.GetAddress(), token.GetSymbol(), newOwner.GetAddress())
	sdkResult = handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	token, err = tokenMapper.GetToken(ctx, "NNB-000")
	require.Nil(t, err)
	require.Equal(t, originOwner.GetAddress(), token.GetOwner())

	ctx = ctx.WithValue(baseapp.TxHashKey, "003")
	msg = NewTransferOwnershipMsg(originOwner.GetAddress(), token.GetSymbol(), newOwner.GetAddress())
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err = tokenMapper.GetToken(ctx, "NNB-000")
	require.Nil(t, err)
	require.Equal(t, newOwner.GetAddress(), token.GetOwner())
}

func TestHandleTransferMiniTokenOwner(t *testing.T) {
	ctx, handler, issueHandler, accountKeeper, tokenMapper := setup()
	_, originOwner := testutils.NewAccount(ctx, accountKeeper, 100e8)
	_, newOwner := testutils.NewAccount(ctx, accountKeeper, 100e8)
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	issueMsg := issue.NewIssueMiniMsg(originOwner.GetAddress(), "New BNB", "NNB", 10000e8, false, "http://www.xyz.com/nnb.json")
	sdkResult := issueHandler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.Nil(t, err)

	// test wrong symbol
	ctx = ctx.WithValue(baseapp.TxHashKey, "001")
	msg := NewTransferOwnershipMsg(originOwner.GetAddress(), "NNB-001", newOwner.GetAddress())
	sdkResult = handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	token, err = tokenMapper.GetToken(ctx, "NNB-000M")
	require.Nil(t, err)
	require.Equal(t, originOwner.GetAddress(), token.GetOwner())

	// test wrong owner
	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	msg = NewTransferOwnershipMsg(acc.GetAddress(), token.GetSymbol(), newOwner.GetAddress())
	sdkResult = handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	token, err = tokenMapper.GetToken(ctx, "NNB-000M")
	require.Nil(t, err)
	require.Equal(t, originOwner.GetAddress(), token.GetOwner())

	ctx = ctx.WithValue(baseapp.TxHashKey, "003")
	msg = NewTransferOwnershipMsg(originOwner.GetAddress(), token.GetSymbol(), newOwner.GetAddress())
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err = tokenMapper.GetToken(ctx, "NNB-000M")
	require.Nil(t, err)
	require.Equal(t, newOwner.GetAddress(), token.GetOwner())
}
