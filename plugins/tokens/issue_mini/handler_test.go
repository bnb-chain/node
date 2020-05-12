package issue_mini

import (
	"testing"

	"github.com/binance-chain/node/common/upgrade"

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

func setChainVersion() {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP8, -1)
}

func resetChainVersion() {
	upgrade.Mgr.Config.HeightMap = nil
}

func TestHandleIssueToken(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, handler, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg := NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 1, 10000e8+100, false, "http://www.xyz.com/nnb.json")
	sdkResult := handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "total supply is too large, the max total supply ")

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg = NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 1, 10000e8, false, "http://www.xyz.com/nnb.json")
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err := types.NewMiniToken("New BNB", "NNB-000M", 1, 10000e8, acc.GetAddress(), false, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, token)

	sdkResult = handler(ctx, msg)
	require.Contains(t, sdkResult.Log, "symbol(NNB) already exists")

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	msg = NewIssueMsg(acc.GetAddress(), "New BB", "NBB", 2, 100000e8+100, false, "http://www.xyz.com/nnb.json")
	sdkResult = handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "total supply is too large, the max total supply ")

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	msg = NewIssueMsg(acc.GetAddress(), "New BB", "NBB", 2, 10000e8+100, false, "http://www.xyz.com/nnb.json")
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err = tokenMapper.GetToken(ctx, "NBB-002M")
	require.NoError(t, err)
	expectedToken, err = types.NewMiniToken("New BB", "NBB-002M", 2, 10000e8+100, acc.GetAddress(), false, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, token)
}
