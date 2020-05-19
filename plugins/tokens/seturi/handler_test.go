package seturi

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
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/plugins/tokens/issue"
	"github.com/binance-chain/node/plugins/tokens/store"
	"github.com/binance-chain/node/wire"
)

func setup() (sdk.Context, sdk.Handler, sdk.Handler, auth.AccountKeeper, store.Mapper) {
	ms, capKey1, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	cdc.RegisterInterface((*types.IToken)(nil), nil)
	cdc.RegisterConcrete(&types.Token{}, "bnbchain/Token", nil)
	cdc.RegisterConcrete(&types.MiniToken{}, "bnbchain/MiniToken", nil)
	tokenMapper := store.NewMapper(cdc, capKey1)
	accountKeeper := auth.NewAccountKeeper(cdc, capKey2, auth.ProtoBaseAccount)
	handler := NewHandler(tokenMapper)

	bankKeeper := bank.NewBaseKeeper(accountKeeper)
	miniTokenHandler := issue.NewMiniHandler(tokenMapper, bankKeeper)

	accountStore := ms.GetKVStore(capKey2)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1},
		sdk.RunTxModeDeliver, log.NewNopLogger()).
		WithAccountCache(auth.NewAccountCache(accountStoreCache))
	return ctx, handler, miniTokenHandler, accountKeeper, tokenMapper
}

func setChainVersion() {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP8, -1)
}

func resetChainVersion() {
	upgrade.Mgr.Config.HeightMap = nil
}

func TestHandleSetURI(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, handler, miniIssueHandler, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg := issue.NewIssueMiniMsg(acc.GetAddress(), "New BNB", "NNB", 1, 10000e8, false, "http://www.xyz.com/nnb.json")
	sdkResult := miniIssueHandler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err := types.NewMiniToken("New BNB", "NNB-000M", 1, 10000e8, acc.GetAddress(), false, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, *(token.(*types.MiniToken)))

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	setUriMsg := NewSetUriMsg(acc.GetAddress(), "NBB", "http://www.123.com/nnb_new.json")
	sdkResult = handler(ctx, setUriMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "symbol(NBB) does not exist")

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	setUriMsg = NewSetUriMsg(acc.GetAddress(), "NNB-000M", "http://www.123.com/nnb_new.json")
	sdkResult = handler(ctx, setUriMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err = tokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err = types.NewMiniToken("New BNB", "NNB-000M", 1, 10000e8, acc.GetAddress(), false, "http://www.123.com/nnb_new.json")
	require.Equal(t, *expectedToken, *(token.(*types.MiniToken)))

	_, acc2 := testutils.NewAccount(ctx, accountKeeper, 100e8)
	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	setUriMsg = NewSetUriMsg(acc2.GetAddress(), "NNB-000M", "http://www.124.com/nnb_new.json")
	sdkResult = handler(ctx, setUriMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "only the owner can mint token")
}
