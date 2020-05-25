package issue

import (
	"fmt"
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
	"github.com/binance-chain/node/plugins/tokens/store"
	"github.com/binance-chain/node/wire"
)

func setupMini() (sdk.Context, sdk.Handler, auth.AccountKeeper, store.Mapper) {
	ms, capKey1, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	cdc.RegisterInterface((*types.IToken)(nil), nil)
	cdc.RegisterConcrete(&types.Token{}, "bnbchain/Token", nil)
	cdc.RegisterConcrete(&types.MiniToken{}, "bnbchain/MiniToken", nil)
	tokenMapper := store.NewMapper(cdc, capKey1)
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

func TestHandleIssueMiniToken(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, handler, accountKeeper, tokenMapper := setupMini()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg := NewIssueTinyMsg(acc.GetAddress(), "New BNB", "NNB", 10000e8+100, false, "http://www.xyz.com/nnb.json")
	sdkResult := handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "total supply is too large, the max total supply ")

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	msg = NewIssueTinyMsg(acc.GetAddress(), "New BNB", "NNB", 10000e8, false, "http://www.xyz.com/nnb.json")
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err := tokenMapper.GetToken(ctx, "NNB-000M")
	require.NoError(t, err)
	expectedToken, err := types.NewMiniToken("New BNB", "NNB-000M", 1, 10000e8, acc.GetAddress(), false, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, *(token.(*types.MiniToken)))

	sdkResult = handler(ctx, msg)
	require.Contains(t, sdkResult.Log, "symbol(NNB) already exists")

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	msgMini := NewIssueMiniMsg(acc.GetAddress(), "New BB", "NBB", 100000e8+100, false, "http://www.xyz.com/nnb.json")
	sdkResult = handler(ctx, msgMini)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "total supply is too large, the max total supply ")

	ctx = ctx.WithValue(baseapp.TxHashKey, "002")
	msgMini = NewIssueMiniMsg(acc.GetAddress(), "New BB", "NBB", 10000e8+100, false, "http://www.xyz.com/nnb.json")
	sdkResult = handler(ctx, msgMini)
	require.Equal(t, true, sdkResult.Code.IsOK())

	token, err = tokenMapper.GetToken(ctx, "NBB-002M")
	require.NoError(t, err)
	expectedToken, err = types.NewMiniToken("New BB", "NBB-002M", 2, 10000e8+100, acc.GetAddress(), false, "http://www.xyz.com/nnb.json")
	require.Equal(t, *expectedToken, *(token.(*types.MiniToken)))
}

func TestHandleMintMiniToken(t *testing.T) {
	setChainVersion()
	defer resetChainVersion()
	ctx, handler, miniTokenHandler, accountKeeper, tokenMapper := setup()
	_, acc := testutils.NewAccount(ctx, accountKeeper, 100e8)
	mintMsg := NewMintMsg(acc.GetAddress(), "NNB-000M", 1001e8)
	sdkResult := handler(ctx, mintMsg)
	require.Contains(t, sdkResult.Log, "symbol(NNB-000M) does not exist")

	issueMsg := NewIssueTinyMsg(acc.GetAddress(), "New BNB", "NNB", 9000e8, true, "http://www.xyz.com/nnb.json")
	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	sdkResult = miniTokenHandler(ctx, issueMsg)
	require.Equal(t, true, sdkResult.Code.IsOK())

	sdkResult = handler(ctx, mintMsg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, fmt.Sprintf("mint amount is too large, the max total supply is %d", types.TinyRangeType.UpperBound()))

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
	issueMsg = NewIssueTinyMsg(acc.GetAddress(), "New BNB2", "NNB2", 9000e8, false, "http://www.xyz.com/nnb.json")
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
