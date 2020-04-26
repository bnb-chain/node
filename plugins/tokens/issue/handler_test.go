package issue

import (
	"github.com/binance-chain/node/common/upgrade"
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
	miniIssue "github.com/binance-chain/node/plugins/minitokens/issue"
	miniTkstore "github.com/binance-chain/node/plugins/minitokens/store"
	"github.com/binance-chain/node/plugins/tokens/store"
	"github.com/binance-chain/node/wire"
)

func setup() (sdk.Context, sdk.Handler, sdk.Handler, auth.AccountKeeper, store.Mapper, miniTkstore.MiniTokenMapper) {
	ms, capKey1, capKey2, capKey3 := testutils.SetupThreeMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	tokenMapper := store.NewMapper(cdc, capKey1)
	accountKeeper := auth.NewAccountKeeper(cdc, capKey2, auth.ProtoBaseAccount)
	miniTokenMapper := miniTkstore.NewMiniTokenMapper(cdc, capKey3)
	bankKeeper := bank.NewBaseKeeper(accountKeeper)
	handler := NewHandler(tokenMapper, miniTokenMapper, bankKeeper)
	miniTokenHandler := miniIssue.NewHandler(miniTokenMapper, bankKeeper)
	accountStore := ms.GetKVStore(capKey2)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1},
		sdk.RunTxModeDeliver, log.NewNopLogger()).
		WithAccountCache(auth.NewAccountCache(accountStoreCache))
	return ctx, handler, miniTokenHandler, accountKeeper, tokenMapper, miniTokenMapper
}

func setChainVersion() {
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP8, -1)
}

func resetChainVersion() {
	upgrade.Mgr.Config.HeightMap = nil
}

func TestHandleIssueToken(t *testing.T) {
	ctx, handler, _, accountKeeper, tokenMapper, _ := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg := NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 100000e8, false)
	sdkResult := handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())
	token, err := tokenMapper.GetToken(ctx, "NNB-000")
	require.NoError(t, err)
	expectedToken, err := types.NewToken("New BNB", "NNB-000", 100000e8, acc.GetAddress(), false)
	require.Equal(t, *expectedToken, token)

	sdkResult = handler(ctx, msg)
	require.Contains(t, sdkResult.Log, "symbol(NNB) already exists")
}

func TestHandleMintToken(t *testing.T) {
	ctx, handler, _, accountKeeper, tokenMapper, _ := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)
	mintMsg := NewMintMsg(acc.GetAddress(), "NNB-000", 10000e8)
	sdkResult := handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "symbol(NNB-000) does not exist")

	issueMsg := NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 100000e8, true)
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = handler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	sdkResult = handler(ctx, mintMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err := tokenMapper.GetToken(ctx, "NNB-000")
	require.NoError(t, err)
	expectedToken, err := types.NewToken("New BNB", "NNB-000", 110000e8, acc.GetAddress(), true)
	require.Equal(t, *expectedToken, token)

	invalidMintMsg := NewMintMsg(acc.GetAddress(), "NNB-000", types.TokenMaxTotalSupply)
	sdkResult = handler(ctx, invalidMintMsg)
	require.Contains(t, sdkResult.Log, "mint amount is too large")

	_, acc2 := testutils.NewAccount(ctx, accountKeeper, 100e8)
	invalidMintMsg = NewMintMsg(acc2.GetAddress(), "NNB-000", types.TokenMaxTotalSupply)
	sdkResult = handler(ctx, invalidMintMsg)
	require.Contains(t, sdkResult.Log, "only the owner can mint token NNB")

	// issue a non-mintable token
	issueMsg = NewIssueMsg(acc.GetAddress(), "New BNB2", "NNB2", 100000e8, false)
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = handler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	mintMsg = NewMintMsg(acc.GetAddress(), "NNB2-000", 10000e8)
	sdkResult = handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "token(NNB2-000) cannot be minted")

	// mint native token
	invalidMintMsg = NewMintMsg(acc.GetAddress(), "BNB", 10000e8)
	require.Contains(t, invalidMintMsg.ValidateBasic().Error(), "cannot mint native token")
}


func TestHandleMintMiniToken(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, handler, miniTokenHandler, accountKeeper, tokenMapper, miniTokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)
	mintMsg := NewMintMsg(acc.GetAddress(), "NNB-000M", 10000e8)
	sdkResult := handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "symbol(NNB-000M) does not exist")

	issueMsg := miniIssue.NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 100000e8, 90000e8, true, "http://www.xyz.com/nnb.json")
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = miniTokenHandler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	sdkResult = handler(ctx, mintMsg)
	token, err := miniTokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err := types.NewMiniToken("New BNB", "NNB-000M", 100000e8, 100000e8, acc.GetAddress(), true, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, token)

	_, err = tokenMapper.GetToken(ctx, "NNB-000M")
	require.NotNil(t,err)
	require.Contains(t, err.Error(), "token(NNB-000M) not found")

	sdkResult = handler(ctx, mintMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "mint amount is too large")

	invalidMintMsg := NewMintMsg(acc.GetAddress(), "NNB-000M", types.MiniTokenSupplyUpperBound)
	sdkResult = handler(ctx, invalidMintMsg)
	require.Contains(t, sdkResult.Log, "mint amount is too large")

	_, acc2 := testutils.NewAccount(ctx, accountKeeper, 100e8)
	invalidMintMsg = NewMintMsg(acc2.GetAddress(), "NNB-000M", types.MiniTokenSupplyUpperBound)
	sdkResult = handler(ctx, invalidMintMsg)
	require.Contains(t, sdkResult.Log, "only the owner can mint token NNB")

	// issue a non-mintable token
	issueMsg = miniIssue.NewIssueMsg(acc.GetAddress(), "New BNB2", "NNB2", 100000e8, 100000e8, false, "http://www.xyz.com/nnb.json")
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = miniTokenHandler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	mintMsg = NewMintMsg(acc.GetAddress(), "NNB2-000M", 10000e8)
	sdkResult = handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "token(NNB2-000M) cannot be minted")

	// mint native token
	invalidMintMsg = NewMintMsg(acc.GetAddress(), "BNB", 10000e8)
	require.Contains(t, invalidMintMsg.ValidateBasic().Error(), "cannot mint native token")
}

