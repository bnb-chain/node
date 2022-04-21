package bank

import (
	"github.com/stretchr/testify/require"
	"testing"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

func setup() (sdk.Context, sdk.Handler, BaseKeeper, auth.AccountKeeper) {
	ms, authKey := setupMultiStore()

	cdc := codec.New()
	auth.RegisterBaseAccount(cdc)
	accountCache := getAccountCache(cdc, ms, authKey)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	accountKeeper := auth.NewAccountKeeper(cdc, authKey, auth.ProtoBaseAccount)
	bankKeeper := NewBaseKeeper(accountKeeper)
	handler := NewHandler(bankKeeper)
	sdk.UpgradeMgr.AddUpgradeHeight(sdk.BEP8, int64(-1))
	return ctx, handler, bankKeeper, accountKeeper
}

func TestHandleSendToken(t *testing.T) {
	ctx, handler, bankKeeper, accountKeeper := setup()

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	addr := sdk.AccAddress([]byte("addr1"))
	addr2 := sdk.AccAddress([]byte("addr2"))
	acc := accountKeeper.NewAccountWithAddress(ctx, addr)
	acc2 := accountKeeper.NewAccountWithAddress(ctx, addr2)

	accountKeeper.SetAccount(ctx, acc)
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{}))

	//transfer BEP2 token Successfully
	bankKeeper.SetCoins(ctx, addr, sdk.Coins{sdk.NewCoin("NNB-000", 100)})

	msg := createSendMsg(acc.GetAddress(), acc2.GetAddress(), sdk.Coins{sdk.NewCoin("NNB-000", 60)})
	sdkResult := handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("NNB-000", 40)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("NNB-000", 60)}))
}

func TestHandleSendMiniToken(t *testing.T) {
	ctx, handler, bankKeeper, accountKeeper := setup()

	ctx = ctx.WithValue(baseapp.TxHashKey, "000")
	addr := sdk.AccAddress([]byte("addr1"))
	addr2 := sdk.AccAddress([]byte("addr2"))
	addr3 := sdk.AccAddress([]byte("addr3"))
	acc := accountKeeper.NewAccountWithAddress(ctx, addr)
	acc2 := accountKeeper.NewAccountWithAddress(ctx, addr2)

	accountKeeper.SetAccount(ctx, acc)
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{}))

	//Transfer Mini token
	//Fail to Transfer with value < 1e8
	MiniTokenFoo := "foocoin-000M"
	bankKeeper.SetCoins(ctx, addr, sdk.Coins{sdk.NewCoin(MiniTokenFoo, 10e8)})
	msg := createSendMsg(acc.GetAddress(), acc2.GetAddress(), sdk.Coins{sdk.NewCoin(MiniTokenFoo, 2)})
	sdkResult := handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "transfer amount is too small")

	//Success with amount >= 1e8
	msg = createSendMsg(acc.GetAddress(), acc2.GetAddress(), sdk.Coins{sdk.NewCoin(MiniTokenFoo, 1e8)})
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin(MiniTokenFoo, 9e8)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin(MiniTokenFoo, 1e8)}))

	//Fail to Multisend
	MiniTokenBar := "barcoin-000M"
	bankKeeper.SetCoins(ctx, addr2, sdk.Coins{sdk.NewCoin(MiniTokenBar, 10), sdk.NewCoin(MiniTokenFoo, 1e8)})

	inputs := []Input{
		NewInput(addr, sdk.Coins{sdk.NewCoin(MiniTokenFoo, 3e8)}),
		NewInput(addr2, sdk.Coins{sdk.NewCoin(MiniTokenBar, 3), sdk.NewCoin(MiniTokenFoo, 1e8)}),
	}

	outputs := []Output{
		NewOutput(addr, sdk.Coins{sdk.NewCoin(MiniTokenBar, 1)}),
		NewOutput(addr3, sdk.Coins{sdk.NewCoin(MiniTokenBar, 2), sdk.NewCoin(MiniTokenFoo, 4e8)}),
	}
	msg = NewMsgSend(inputs, outputs)
	sdkResult = handler(ctx, msg)
	require.Equal(t, false, sdkResult.Code.IsOK())
	require.Contains(t, sdkResult.Log, "transfer amount is too small")

	//
	//Success with all balance
	inputs = []Input{
		NewInput(addr, sdk.Coins{sdk.NewCoin(MiniTokenFoo, 3e8)}),
		NewInput(addr2, sdk.Coins{sdk.NewCoin(MiniTokenBar, 10), sdk.NewCoin(MiniTokenFoo, 1e8)}),
	}

	outputs = []Output{
		NewOutput(addr, sdk.Coins{sdk.NewCoin(MiniTokenBar, 4)}),
		NewOutput(addr3, sdk.Coins{sdk.NewCoin(MiniTokenBar, 6), sdk.NewCoin(MiniTokenFoo, 4e8)}),
	}
	msg = NewMsgSend(inputs, outputs)
	sdkResult = handler(ctx, msg)
	require.Equal(t, true, sdkResult.Code.IsOK())
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin(MiniTokenBar, 4), sdk.NewCoin(MiniTokenFoo, 6e8)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{}))
	require.True(t, bankKeeper.GetCoins(ctx, addr3).IsEqual(sdk.Coins{sdk.NewCoin(MiniTokenBar, 6), sdk.NewCoin(MiniTokenFoo, 4e8)}))
}

func createSendMsg(from sdk.AccAddress, to sdk.AccAddress, coins sdk.Coins) sdk.Msg {
	input := NewInput(from, coins)
	output := NewOutput(to, coins)
	msg := NewMsgSend([]Input{input}, []Output{output})
	return msg
}
