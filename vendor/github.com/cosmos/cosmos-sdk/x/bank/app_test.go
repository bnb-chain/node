package bank_test

import (
	"github.com/cosmos/cosmos-sdk/x/bank"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

type (
	expectedBalance struct {
		addr  sdk.AccAddress
		coins sdk.Coins
	}

	appTestCase struct {
		expSimPass       bool
		expPass          bool
		msgs             []sdk.Msg
		accNums          []int64
		accSeqs          []int64
		privKeys         []crypto.PrivKey
		expectedBalances []expectedBalance
	}
)

var (
	priv1 = ed25519.GenPrivKey()
	addr1 = sdk.AccAddress(priv1.PubKey().Address())
	priv2 = ed25519.GenPrivKey()
	addr2 = sdk.AccAddress(priv2.PubKey().Address())
	addr3 = sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	priv4 = ed25519.GenPrivKey()
	addr4 = sdk.AccAddress(priv4.PubKey().Address())

	coins     = sdk.Coins{sdk.NewCoin("foocoin", 10)}
	halfCoins = sdk.Coins{sdk.NewCoin("foocoin", 5)}
	manyCoins = sdk.Coins{sdk.NewCoin("foocoin", 1), sdk.NewCoin("barcoin", 1)}

	sendMsg1 = bank.MsgSend{
		Inputs:  []bank.Input{bank.NewInput(addr1, coins)},
		Outputs: []bank.Output{bank.NewOutput(addr2, coins)},
	}
	sendMsg2 = bank.MsgSend{
		Inputs: []bank.Input{bank.NewInput(addr1, coins)},
		Outputs: []bank.Output{
			bank.NewOutput(addr2, halfCoins),
			bank.NewOutput(addr3, halfCoins),
		},
	}
	sendMsg3 = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(addr1, coins),
			bank.NewInput(addr4, coins),
		},
		Outputs: []bank.Output{
			bank.NewOutput(addr2, coins),
			bank.NewOutput(addr3, coins),
		},
	}
	sendMsg4 = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(addr2, coins),
		},
		Outputs: []bank.Output{
			bank.NewOutput(addr1, coins),
		},
	}
	sendMsg5 = bank.MsgSend{
		Inputs: []bank.Input{
			bank.NewInput(addr1, manyCoins),
		},
		Outputs: []bank.Output{
			bank.NewOutput(addr2, manyCoins),
		},
	}
)

// initialize the mock application for this module
func getMockApp(t *testing.T) *mock.App {
	mapp, err := getBenchmarkMockApp()
	require.NoError(t, err)
	return mapp
}

func TestMsgSendWithAccounts(t *testing.T) {
	mapp := getMockApp(t)
	acc := &auth.BaseAccount{
		Address: addr1,
		Coins:   sdk.Coins{sdk.NewCoin("foocoin", 67)},
	}

	mock.SetGenesis(mapp, []sdk.Account{acc})

	ctxCheck := mapp.BaseApp.NewContext(sdk.RunTxModeCheck, abci.Header{})

	res1 := mapp.AccountKeeper.GetAccount(ctxCheck, addr1)
	require.NotNil(t, res1)
	require.Equal(t, acc, res1.(*auth.BaseAccount))

	testCases := []appTestCase{
		{
			msgs:       []sdk.Msg{sendMsg1},
			accNums:    []int64{0},
			accSeqs:    []int64{0},
			expSimPass: true,
			expPass:    true,
			privKeys:   []crypto.PrivKey{priv1},
			expectedBalances: []expectedBalance{
				{addr1, sdk.Coins{sdk.NewCoin("foocoin", 57)}},
				{addr2, sdk.Coins{sdk.NewCoin("foocoin", 10)}},
			},
		},
		{
			msgs:       []sdk.Msg{sendMsg1, sendMsg2},
			accNums:    []int64{0},
			accSeqs:    []int64{0},
			expSimPass: false,
			expPass:    false,
			privKeys:   []crypto.PrivKey{priv1},
		},
	}

	for _, tc := range testCases {
		mock.SignCheckDeliver(t, mapp.BaseApp, tc.msgs, tc.accNums, tc.accSeqs, tc.expSimPass, tc.expPass, tc.privKeys...)

		for _, eb := range tc.expectedBalances {
			mock.CheckBalance(t, mapp, eb.addr, eb.coins)
		}
	}

	// bumping the tx nonce number without resigning should be an auth error
	mapp.BeginBlock(abci.RequestBeginBlock{})

	tx := mock.GenTx([]sdk.Msg{sendMsg1}, []int64{0}, []int64{0}, priv1)
	tx.Signatures[0].Sequence = 1

	res := mapp.Deliver(tx)
	require.Equal(t, sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeUnauthorized), res.Code, res.Log)

	// resigning the tx with the bumped sequence should work
	mock.SignCheckDeliver(t, mapp.BaseApp, []sdk.Msg{sendMsg1}, []int64{0}, []int64{1}, true, true, priv1)
}

func TestMsgSendMultipleOut(t *testing.T) {
	mapp := getMockApp(t)

	acc1 := &auth.BaseAccount{
		Address: addr1,
		Coins:   sdk.Coins{sdk.NewCoin("foocoin", 42)},
	}
	acc2 := &auth.BaseAccount{
		Address: addr2,
		Coins:   sdk.Coins{sdk.NewCoin("foocoin", 42)},
	}

	mock.SetGenesis(mapp, []sdk.Account{acc1, acc2})

	testCases := []appTestCase{
		{
			msgs:       []sdk.Msg{sendMsg2},
			accNums:    []int64{0},
			accSeqs:    []int64{0},
			expSimPass: true,
			expPass:    true,
			privKeys:   []crypto.PrivKey{priv1},
			expectedBalances: []expectedBalance{
				{addr1, sdk.Coins{sdk.NewCoin("foocoin", 32)}},
				{addr2, sdk.Coins{sdk.NewCoin("foocoin", 47)}},
				{addr3, sdk.Coins{sdk.NewCoin("foocoin", 5)}},
			},
		},
	}

	for _, tc := range testCases {
		mock.SignCheckDeliver(t, mapp.BaseApp, tc.msgs, tc.accNums, tc.accSeqs, tc.expSimPass, tc.expPass, tc.privKeys...)

		for _, eb := range tc.expectedBalances {
			mock.CheckBalance(t, mapp, eb.addr, eb.coins)
		}
	}
}

func TestSengMsgMultipleInOut(t *testing.T) {
	mapp := getMockApp(t)

	acc1 := &auth.BaseAccount{
		Address: addr1,
		Coins:   sdk.Coins{sdk.NewCoin("foocoin", 42)},
	}
	acc2 := &auth.BaseAccount{
		Address: addr2,
		Coins:   sdk.Coins{sdk.NewCoin("foocoin", 42)},
	}
	acc4 := &auth.BaseAccount{
		Address: addr4,
		Coins:   sdk.Coins{sdk.NewCoin("foocoin", 42)},
	}

	mock.SetGenesis(mapp, []sdk.Account{acc1, acc2, acc4})

	testCases := []appTestCase{
		{
			msgs:       []sdk.Msg{sendMsg3},
			accNums:    []int64{0, 0},
			accSeqs:    []int64{0, 0},
			expSimPass: true,
			expPass:    true,
			privKeys:   []crypto.PrivKey{priv1, priv4},
			expectedBalances: []expectedBalance{
				{addr1, sdk.Coins{sdk.NewCoin("foocoin", 32)}},
				{addr4, sdk.Coins{sdk.NewCoin("foocoin", 32)}},
				{addr2, sdk.Coins{sdk.NewCoin("foocoin", 52)}},
				{addr3, sdk.Coins{sdk.NewCoin("foocoin", 10)}},
			},
		},
	}

	for _, tc := range testCases {
		mock.SignCheckDeliver(t, mapp.BaseApp, tc.msgs, tc.accNums, tc.accSeqs, tc.expSimPass, tc.expPass, tc.privKeys...)

		for _, eb := range tc.expectedBalances {
			mock.CheckBalance(t, mapp, eb.addr, eb.coins)
		}
	}
}

func TestMsgSendDependent(t *testing.T) {
	mapp := getMockApp(t)

	acc1 := &auth.BaseAccount{
		Address: addr1,
		Coins:   sdk.Coins{sdk.NewCoin("foocoin", 42)},
	}

	mock.SetGenesis(mapp, []sdk.Account{acc1})

	testCases := []appTestCase{
		{
			msgs:       []sdk.Msg{sendMsg1},
			accNums:    []int64{0},
			accSeqs:    []int64{0},
			expSimPass: true,
			expPass:    true,
			privKeys:   []crypto.PrivKey{priv1},
			expectedBalances: []expectedBalance{
				{addr1, sdk.Coins{sdk.NewCoin("foocoin", 32)}},
				{addr2, sdk.Coins{sdk.NewCoin("foocoin", 10)}},
			},
		},
		{
			msgs:       []sdk.Msg{sendMsg4},
			accNums:    []int64{0},
			accSeqs:    []int64{0},
			expSimPass: true,
			expPass:    true,
			privKeys:   []crypto.PrivKey{priv2},
			expectedBalances: []expectedBalance{
				{addr1, sdk.Coins{sdk.NewCoin("foocoin", 42)}},
			},
		},
	}

	for _, tc := range testCases {
		mock.SignCheckDeliver(t, mapp.BaseApp, tc.msgs, tc.accNums, tc.accSeqs, tc.expSimPass, tc.expPass, tc.privKeys...)

		for _, eb := range tc.expectedBalances {
			mock.CheckBalance(t, mapp, eb.addr, eb.coins)
		}
	}
}
