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
	"github.com/binance-chain/node/common/upgrade"
	miniIssue "github.com/binance-chain/node/plugins/tokens/issue_mini"
	"github.com/binance-chain/node/plugins/tokens/store"
	"github.com/binance-chain/node/wire"
)

func setup() (sdk.Context, sdk.Handler, sdk.Handler, auth.AccountKeeper, store.Mapper) {
	ms, capKey1, capKey2, _ := testutils.SetupThreeMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	cdc.RegisterInterface((*types.IToken)(nil), nil)
	cdc.RegisterConcrete(&types.Token{}, "bnbchain/Token", nil)
	cdc.RegisterConcrete(&types.MiniToken{}, "bnbchain/MiniToken", nil)
	tokenMapper := store.NewMapper(cdc, capKey1)
	accountKeeper := auth.NewAccountKeeper(cdc, capKey2, auth.ProtoBaseAccount)
	bankKeeper := bank.NewBaseKeeper(accountKeeper)
	handler := NewHandler(tokenMapper, bankKeeper)
	miniTokenHandler := miniIssue.NewHandler(tokenMapper, bankKeeper)
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

func TestHandleIssueToken(t *testing.T) {
	ctx, handler, _, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg := NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 100000e8, false)
	sdkResult := handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())
	token, err := tokenMapper.GetToken(ctx, "NNB-000")
	require.NoError(t, err)
	expectedToken, err := types.NewToken("New BNB", "NNB-000", 100000e8, acc.GetAddress(), false)
	require.Equal(t, *expectedToken, *token.(*types.Token))

	sdkResult = handler(ctx, msg)
	require.Contains(t, sdkResult.Log, "symbol(NNB) already exists")
}

func TestHandleMintToken(t *testing.T) {
	ctx, handler, _, accountKeeper, tokenMapper := setup()
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
	require.Equal(t, *expectedToken, *token.(*types.Token))

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
	ctx, handler, miniTokenHandler, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)
	mintMsg := NewMintMsg(acc.GetAddress(), "NNB-000M", 1001e8)
	sdkResult := handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "symbol(NNB-000M) does not exist")

	issueMsg := miniIssue.NewIssueMsg(acc.GetAddress(), "New BNB", "NNB", 1, 9000e8, true, "http://www.xyz.com/nnb.json")
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = miniTokenHandler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	sdkResult = handler(ctx, mintMsg)
	token, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err := types.NewMiniToken("New BNB", "NNB-000M", 1, 9000e8, acc.GetAddress(), true, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, *(token.(*types.MiniToken)))

	_, err = tokenMapper.GetToken(ctx, "NNB-000")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "token(NNB-000) not found")

	sdkResult = handler(ctx, mintMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "mint amount is too large")

	validMintMsg := NewMintMsg(acc.GetAddress(), "NNB-000M", 1000e8)
	sdkResult = handler(ctx, validMintMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())
	token, err = tokenMapper.GetToken(ctx, "NNB-000M")
	expectedToken, err = types.NewMiniToken("New BNB", "NNB-000M", 1, 10000e8, acc.GetAddress(), true, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, *(token.(*types.MiniToken)))

	_, acc2 := testutils.NewAccount(ctx, accountKeeper, 100e8)
	invalidMintMsg := NewMintMsg(acc2.GetAddress(), "NNB-000M", 100e8)
	sdkResult = handler(ctx, invalidMintMsg)
	require.Contains(t, sdkResult.Log, "only the owner can mint token NNB")

	// issue a non-mintable token
	issueMsg = miniIssue.NewIssueMsg(acc.GetAddress(), "New BNB2", "NNB2", 1, 9000e8, false, "http://www.xyz.com/nnb.json")
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = miniTokenHandler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	mintMsg = NewMintMsg(acc.GetAddress(), "NNB2-000M", 1000e8)
	sdkResult = handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "token(NNB2-000M) cannot be minted")

	// mint native token
	invalidMintMsg = NewMintMsg(acc.GetAddress(), "BNB", 10000e8)
	require.Contains(t, invalidMintMsg.ValidateBasic().Error(), "cannot mint native token")
}
