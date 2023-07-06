package apptest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkfees "github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/cosmos/cosmos-sdk/x/paramHub"

	abcicli "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	. "github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/bnb-chain/node/app"
	common "github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/plugins/dex"
	"github.com/bnb-chain/node/plugins/tokens"
	"github.com/bnb-chain/node/wire"
)

type TestClient struct {
	cl  abcicli.Client
	cdc *wire.Codec
}

func NewMockAnteHandler(cdc *wire.Codec) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, runTxMode sdk.RunTxMode) (newCtx sdk.Context, result sdk.Result, abort bool) {
		msg := tx.GetMsgs()[0]
		fee := sdkfees.GetCalculator(msg.Type())(msg)

		if ctx.IsDeliverTx() {
			// add fee to pool, even it's free
			stdTx := tx.(auth.StdTx)
			txHash := cmn.HexBytes(tmhash.Sum(cdc.MustMarshalBinaryLengthPrefixed(stdTx))).String()
			sdkfees.Pool.AddFee(txHash, fee)
		}

		return newCtx, sdk.Result{}, false
	}
}

func (tc *TestClient) DeliverTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)
	return tc.cl.DeliverTxAsync(abci.RequestDeliverTx{Tx: tx})
}

func (tc *TestClient) CheckTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)
	return tc.cl.CheckTxAsync(abci.RequestCheckTx{Tx: tx})
}

func (tc *TestClient) DeliverTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseDeliverTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)
	return tc.cl.DeliverTxSync(abci.RequestDeliverTx{Tx: tx})
}

func (tc *TestClient) CheckTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseCheckTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)
	return tc.cl.CheckTxSync(abci.RequestCheckTx{Tx: tx})
}

// util objects
var (
	memDB                             = db.NewMemDB()
	logger                            = log.NewTMLogger(os.Stdout)
	testApp                           = app.NewBNBBeaconChain(logger, memDB, os.Stdout)
	genAccs, addrs, pubKeys, privKeys = mock.CreateGenAccounts(4,
		sdk.Coins{sdk.NewCoin("BNB", 500e8), sdk.NewCoin("BTC-000", 200e8)})
	testClient = NewTestClient(testApp)
)

func TearDown() {
	// remove block db
	os.RemoveAll(cfg.DefaultConfig().DBDir())
}

func InitAccounts(ctx sdk.Context, app *app.BNBBeaconChain) *[]sdk.Account {
	for _, acc := range genAccs {
		aacc := &common.AppAccount{
			BaseAccount: auth.BaseAccount{
				Address: acc.GetAddress(),
				Coins:   acc.GetCoins(),
			}}
		if app.AccountKeeper.GetAccount(ctx, acc.GetAddress()) == nil {
			aacc.BaseAccount.AccountNumber = app.AccountKeeper.GetNextAccountNumber(ctx)
		}
		app.AccountKeeper.SetAccount(ctx, aacc)
	}
	return &genAccs
}

func ResetAccounts(ctx sdk.Context, app *app.BNBBeaconChain, ccy1 int64, ccy2 int64, ccy3 int64) {
	for _, acc := range genAccs {
		a := app.AccountKeeper.GetAccount(ctx, acc.GetAddress())
		a.SetCoins(sdk.Coins{sdk.NewCoin("BNB", ccy1), sdk.NewCoin("BTC-000", ccy2), sdk.NewCoin("ETH-000", ccy3)})
		app.AccountKeeper.SetAccount(ctx, a)
	}
}

func Account(i int) sdk.Account {
	return genAccs[i]
}

func Address(i int) sdk.AccAddress {
	return addrs[i]
}

func NewTestClient(a *app.BNBBeaconChain) *TestClient {
	a.SetCheckState(types.Header{})
	a.SetAnteHandler(NewMockAnteHandler(a.Codec)) // clear AnteHandler to skip the signature verification step
	return &TestClient{abcicli.NewLocalClient(nil, a), app.MakeCodec()}
}

func GetAvail(ctx sdk.Context, add sdk.AccAddress, ccy string) int64 {
	return testApp.CoinKeeper.GetCoins(ctx, add).AmountOf(ccy)
}

func GetLocked(ctx sdk.Context, add sdk.AccAddress, ccy string) int64 {
	return testApp.AccountKeeper.GetAccount(ctx, add).(common.NamedAccount).GetLockedCoins().AmountOf(ccy)
}

func setGenesis(bapp *app.BNBBeaconChain, tokens []tokens.GenesisToken, accs ...*common.AppAccount) error {
	genaccs := make([]app.GenesisAccount, len(accs))
	for i, acc := range accs {
		pk := GenPrivKey().PubKey()
		valAddr := pk.Address()
		genaccs[i] = app.NewGenesisAccount(acc, valAddr)
	}

	genesisState := app.GenesisState{
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: paramHub.DefaultGenesisState,
	}

	stateBytes, err := wire.MarshalJSONIndent(bapp.Codec, genesisState)
	if err != nil {
		return err
	}

	// Initialize the chain
	vals := []abci.ValidatorUpdate{}
	bapp.InitChain(abci.RequestInitChain{Validators: vals, AppStateBytes: stateBytes})
	bapp.Commit()

	return nil
}

func TestGenesis(t *testing.T) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "sdk/app")
	db := dbm.NewMemDB()
	bapp := app.NewBNBBeaconChain(logger, db, os.Stdout)

	// Construct some genesis bytes to reflect democoin/types/AppAccount
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{
		Address: addr,
	}
	tokens := []tokens.GenesisToken{{"BNB", "BNB", 100000, addr, false}}
	acc := &common.AppAccount{baseAcc, "blah", sdk.Coins(nil), sdk.Coins(nil), 0}

	err := setGenesis(bapp, tokens, acc)
	require.Nil(t, err)
	// A checkTx context
	ctx := bapp.BaseApp.NewContext(sdk.RunTxModeCheck, abci.Header{})
	if err := acc.SetCoins(sdk.Coins{sdk.Coin{"BNB", 100000}}); err != nil {
		t.Fatalf("SetCoins error: " + err.Error())
	}
	res1 := bapp.AccountKeeper.GetAccount(ctx, baseAcc.Address).(common.NamedAccount)
	require.Equal(t, acc, res1)
}
