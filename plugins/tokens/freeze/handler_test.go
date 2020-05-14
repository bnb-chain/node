package freeze

import (
	"github.com/binance-chain/node/plugins/tokens/issue"
	"github.com/binance-chain/node/plugins/tokens/issue_mini"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"testing"

	"github.com/binance-chain/node/common/upgrade"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/tokens/store"
	"github.com/binance-chain/node/wire"
)

func setup() (sdk.Context, sdk.Handler, sdk.Handler, sdk.Handler, auth.AccountKeeper, store.Mapper) {
	ms, capKey1, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	cdc.RegisterInterface((*types.IToken)(nil), nil)
	cdc.RegisterConcrete(&types.Token{}, "bnbchain/Token", nil)
	cdc.RegisterConcrete(&types.MiniToken{}, "bnbchain/MiniToken", nil)
	tokenMapper := store.NewMapper(cdc, capKey1)
	accountKeeper := auth.NewAccountKeeper(cdc, capKey2, auth.ProtoBaseAccount)
	bankKeeper := bank.NewBaseKeeper(accountKeeper)
	handler := NewHandler(tokenMapper, accountKeeper, bankKeeper)
	tokenHandler := issue.NewHandler(tokenMapper, bankKeeper)
	miniTokenHandler := issue_mini.NewHandler(tokenMapper, bankKeeper)

	accountStore := ms.GetKVStore(capKey2)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1},
		sdk.RunTxModeDeliver, log.NewNopLogger()).
		WithAccountCache(auth.NewAccountCache(accountStoreCache))
	return ctx, handler, tokenHandler, miniTokenHandler, accountKeeper, tokenMapper
}

func setChainVersion() {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP8, -1)
}

func resetChainVersion() {
	upgrade.Mgr.Config.HeightMap = nil
}

func TestHandleFreezeMini(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, handler, _, miniIssueHandler, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg := issue_mini.NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 1, 10000e8, false, "http://www.xyz.com/nnb.json")
	sdkResult := miniIssueHandler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err := types.NewMiniToken("New BNB", "NNB-000M", 1, 10000e8, acc.GetAddress(), false, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, *(token.(*types.MiniToken)))

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	freezeMsg := NewFreezeMsg(acc.GetAddress(), "NNB-000M", 10001e8+1)
	sdkResult = handler(ctx, freezeMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "do not have enough token to freeze")

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	freezeMsg = NewFreezeMsg(acc.GetAddress(), "NNB-000M", 9999e8+1)
	sdkResult = handler(ctx, freezeMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "do not have enough token to freeze")

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	freezeMsg = NewFreezeMsg(acc.GetAddress(), "NNB-000M", 9001e8)
	sdkResult = handler(ctx, freezeMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	//token, err = tokenMapper.GetToken(ctx, "NNB-000M")
	//require.NoError(t, err)
	//expectedToken, err = types.NewMiniToken("New BNB", "NNB-000M", 1, 999e8, acc.GetAddress(), false, "http://www.xyz.com/nnb.json")
	//require.Equal(t, *expectedToken, *(token.(*types.MiniToken)))
	//
	//_, acc2 := testutils.NewAccount(ctx, accountKeeper, 100e8)
	//ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	//freezeMsg = NewMsg(acc2.GetAddress(), "NNB-000M", 1e8)
	//sdkResult = handler(ctx, freezeMsg)
	//require.Equal(t, false, sdkResult.Code.IsOK())
	//require.Contains(t, sdkResult.Log, "only the owner of the token can burn the token")
}

func TestHandleFreeze(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, handler, issueHandler, _, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg := issue.NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 10000e8, false)
	sdkResult := issueHandler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())


	_, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "token(NNB-000M) not found")

	_, err = tokenMapper.GetToken(ctx, "NNB-000")
	require.Equal(t, true, sdkResult.Code.IsOK())

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	freezeMsg := NewFreezeMsg(acc.GetAddress(), "NNB-000", 10001e8)
	sdkResult = handler(ctx, freezeMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "do not have enough token to freeze")

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	freezeMsg = NewFreezeMsg(acc.GetAddress(), "NNB-000", 9999e8+1)
	sdkResult = handler(ctx, freezeMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	//token, err = tokenMapper.GetToken(ctx, "NNB-000")
	//require.NoError(t, err)
	//expectedToken, err := types.NewToken("New BNB", "NNB-000", 1e8-1, acc.GetAddress(), false)
	//require.Equal(t, *expectedToken, *(token.(*types.Token)))
	//
	//_, acc2 := testutils.NewAccount(ctx, accountKeeper, 100e8)
	//ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	//freezeMsg = NewFreezeMsg(acc2.GetAddress(), "NNB-000", 1e8)
	//sdkResult = handler(ctx, freezeMsg)
	//require.Equal(t, false, sdkResult.Code.IsOK())
	//require.Contains(t, sdkResult.Log, "only the owner of the token can burn the token")
}
