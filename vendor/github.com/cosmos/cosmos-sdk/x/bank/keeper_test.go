package bank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

func setupMultiStore() (sdk.MultiStore, *sdk.KVStoreKey) {
	db := dbm.NewMemDB()
	authKey := sdk.NewKVStoreKey("authkey")
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(authKey, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	return ms, authKey
}

func getAccountCache(cdc *codec.Codec, ms sdk.MultiStore, accountKey *sdk.KVStoreKey) sdk.AccountCache {
	accountStore := ms.GetKVStore(accountKey)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	return auth.NewAccountCache(accountStoreCache)
}

func TestKeeper(t *testing.T) {
	ms, authKey := setupMultiStore()

	cdc := codec.New()
	auth.RegisterBaseAccount(cdc)
	accountCache := getAccountCache(cdc, ms, authKey)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	accountKeeper := auth.NewAccountKeeper(cdc, authKey, auth.ProtoBaseAccount)
	bankKeeper := NewBaseKeeper(accountKeeper)

	addr := sdk.AccAddress([]byte("addr1"))
	addr2 := sdk.AccAddress([]byte("addr2"))
	addr3 := sdk.AccAddress([]byte("addr3"))
	acc := accountKeeper.NewAccountWithAddress(ctx, addr)

	// Test GetCoins/SetCoins
	accountKeeper.SetAccount(ctx, acc)
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{}))

	bankKeeper.SetCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 10)})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 10)}))

	// Test HasCoins
	require.True(t, bankKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 10)}))
	require.True(t, bankKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 5)}))
	require.False(t, bankKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 15)}))
	require.False(t, bankKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 5)}))

	// Test AddCoins
	bankKeeper.AddCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 15)})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 25)}))

	bankKeeper.AddCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 15)})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 15), sdk.NewCoin("foocoin", 25)}))

	// Test SubtractCoins
	bankKeeper.SubtractCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 10)})
	bankKeeper.SubtractCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 5)})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 10), sdk.NewCoin("foocoin", 15)}))

	bankKeeper.SubtractCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 11)})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 10), sdk.NewCoin("foocoin", 15)}))

	bankKeeper.SubtractCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 10)})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 15)}))
	require.False(t, bankKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 1)}))

	// Test SendCoins
	bankKeeper.SendCoins(ctx, addr, addr2, sdk.Coins{sdk.NewCoin("foocoin", 5)})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 10)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 5)}))

	_, err2 := bankKeeper.SendCoins(ctx, addr, addr2, sdk.Coins{sdk.NewCoin("foocoin", 50)})
	assert.Implements(t, (*sdk.Error)(nil), err2)
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 10)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 5)}))

	bankKeeper.AddCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 30)})
	bankKeeper.SendCoins(ctx, addr, addr2, sdk.Coins{sdk.NewCoin("barcoin", 10), sdk.NewCoin("foocoin", 5)})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 20), sdk.NewCoin("foocoin", 5)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 10), sdk.NewCoin("foocoin", 10)}))

	// Test InputOutputCoins
	input1 := NewInput(addr2, sdk.Coins{sdk.NewCoin("foocoin", 2)})
	output1 := NewOutput(addr, sdk.Coins{sdk.NewCoin("foocoin", 2)})
	bankKeeper.InputOutputCoins(ctx, []Input{input1}, []Output{output1})
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 20), sdk.NewCoin("foocoin", 7)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 10), sdk.NewCoin("foocoin", 8)}))

	inputs := []Input{
		NewInput(addr, sdk.Coins{sdk.NewCoin("foocoin", 3)}),
		NewInput(addr2, sdk.Coins{sdk.NewCoin("barcoin", 3), sdk.NewCoin("foocoin", 2)}),
	}

	outputs := []Output{
		NewOutput(addr, sdk.Coins{sdk.NewCoin("barcoin", 1)}),
		NewOutput(addr3, sdk.Coins{sdk.NewCoin("barcoin", 2), sdk.NewCoin("foocoin", 5)}),
	}
	bankKeeper.InputOutputCoins(ctx, inputs, outputs)
	require.True(t, bankKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 21), sdk.NewCoin("foocoin", 4)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 7), sdk.NewCoin("foocoin", 6)}))
	require.True(t, bankKeeper.GetCoins(ctx, addr3).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 2), sdk.NewCoin("foocoin", 5)}))

}

func TestSendKeeper(t *testing.T) {
	ms, authKey := setupMultiStore()

	cdc := codec.New()
	auth.RegisterBaseAccount(cdc)
	accountCache := getAccountCache(cdc, ms, authKey)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	accountKeeper := auth.NewAccountKeeper(cdc, authKey, auth.ProtoBaseAccount)
	bankKeeper := NewBaseKeeper(accountKeeper)
	sendKeeper := NewBaseSendKeeper(accountKeeper)

	addr := sdk.AccAddress([]byte("addr1"))
	addr2 := sdk.AccAddress([]byte("addr2"))
	addr3 := sdk.AccAddress([]byte("addr3"))
	acc := accountKeeper.NewAccountWithAddress(ctx, addr)

	// Test GetCoins/SetCoins
	accountKeeper.SetAccount(ctx, acc)
	require.True(t, sendKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{}))

	bankKeeper.SetCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 10)})
	require.True(t, sendKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 10)}))

	// Test HasCoins
	require.True(t, sendKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 10)}))
	require.True(t, sendKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 5)}))
	require.False(t, sendKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 15)}))
	require.False(t, sendKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 5)}))

	bankKeeper.SetCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 15)})

	// Test SendCoins
	sendKeeper.SendCoins(ctx, addr, addr2, sdk.Coins{sdk.NewCoin("foocoin", 5)})
	require.True(t, sendKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 10)}))
	require.True(t, sendKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 5)}))

	_, err2 := sendKeeper.SendCoins(ctx, addr, addr2, sdk.Coins{sdk.NewCoin("foocoin", 50)})
	assert.Implements(t, (*sdk.Error)(nil), err2)
	require.True(t, sendKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 10)}))
	require.True(t, sendKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 5)}))

	bankKeeper.AddCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 30)})
	sendKeeper.SendCoins(ctx, addr, addr2, sdk.Coins{sdk.NewCoin("barcoin", 10), sdk.NewCoin("foocoin", 5)})
	require.True(t, sendKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 20), sdk.NewCoin("foocoin", 5)}))
	require.True(t, sendKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 10), sdk.NewCoin("foocoin", 10)}))

	// Test InputOutputCoins
	input1 := NewInput(addr2, sdk.Coins{sdk.NewCoin("foocoin", 2)})
	output1 := NewOutput(addr, sdk.Coins{sdk.NewCoin("foocoin", 2)})
	sendKeeper.InputOutputCoins(ctx, []Input{input1}, []Output{output1})
	require.True(t, sendKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 20), sdk.NewCoin("foocoin", 7)}))
	require.True(t, sendKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 10), sdk.NewCoin("foocoin", 8)}))

	inputs := []Input{
		NewInput(addr, sdk.Coins{sdk.NewCoin("foocoin", 3)}),
		NewInput(addr2, sdk.Coins{sdk.NewCoin("barcoin", 3), sdk.NewCoin("foocoin", 2)}),
	}

	outputs := []Output{
		NewOutput(addr, sdk.Coins{sdk.NewCoin("barcoin", 1)}),
		NewOutput(addr3, sdk.Coins{sdk.NewCoin("barcoin", 2), sdk.NewCoin("foocoin", 5)}),
	}
	sendKeeper.InputOutputCoins(ctx, inputs, outputs)
	require.True(t, sendKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 21), sdk.NewCoin("foocoin", 4)}))
	require.True(t, sendKeeper.GetCoins(ctx, addr2).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 7), sdk.NewCoin("foocoin", 6)}))
	require.True(t, sendKeeper.GetCoins(ctx, addr3).IsEqual(sdk.Coins{sdk.NewCoin("barcoin", 2), sdk.NewCoin("foocoin", 5)}))

}

func TestViewKeeper(t *testing.T) {
	ms, authKey := setupMultiStore()

	cdc := codec.New()
	auth.RegisterBaseAccount(cdc)
	accountCache := getAccountCache(cdc, ms, authKey)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	accountKeeper := auth.NewAccountKeeper(cdc, authKey, auth.ProtoBaseAccount)
	bankKeeper := NewBaseKeeper(accountKeeper)
	viewKeeper := NewBaseViewKeeper(accountKeeper)

	addr := sdk.AccAddress([]byte("addr1"))
	acc := accountKeeper.NewAccountWithAddress(ctx, addr)

	// Test GetCoins/SetCoins
	accountKeeper.SetAccount(ctx, acc)
	require.True(t, viewKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{}))

	bankKeeper.SetCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 10)})
	require.True(t, viewKeeper.GetCoins(ctx, addr).IsEqual(sdk.Coins{sdk.NewCoin("foocoin", 10)}))

	// Test HasCoins
	require.True(t, viewKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 10)}))
	require.True(t, viewKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 5)}))
	require.False(t, viewKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("foocoin", 15)}))
	require.False(t, viewKeeper.HasCoins(ctx, addr, sdk.Coins{sdk.NewCoin("barcoin", 5)}))
}
