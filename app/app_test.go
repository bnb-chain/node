package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/libs/db"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/mock"

	common "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
)

type TestClient struct {
	cl  abcicli.Client
	cdc *wire.Codec
}

func (tc *TestClient) DeliverTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.DeliverTxAsync(tx)
}

func (tc *TestClient) CheckTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.CheckTxAsync(tx)
}

func (tc *TestClient) DeliverTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseDeliverTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.DeliverTxSync(tx)
}

func (tc *TestClient) CheckTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseCheckTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test")
	tx, _ := tc.cdc.MarshalBinary(stdtx)
	return tc.cl.CheckTxSync(tx)
}

// util objects
var (
	memDB                             = db.NewMemDB()
	logger                            = log.NewTMLogger(os.Stdout)
	testApp                           = NewBinanceChain(logger, memDB, os.Stdout)
	genAccs, addrs, pubKeys, privKeys = mock.CreateGenAccounts(4,
		sdk.Coins{sdk.NewCoin("BNB", 500e8), sdk.NewCoin("BTC-000", 200e8)})
	testClient = NewTestClient(testApp)
)

func TearDown() {
	// remove block db
	os.RemoveAll(cfg.DefaultConfig().DBDir())
}

func InitAccounts(ctx sdk.Context, app *BinanceChain) *[]sdk.Account {
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

func ResetAccounts(ctx sdk.Context, app *BinanceChain, ccy1 int64, ccy2 int64, ccy3 int64) {
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

func NewTestClient(a *BinanceChain) *TestClient {
	a.SetCheckState(types.Header{})
	a.SetAnteHandler(nil) // clear AnteHandler to skip the signature verification step
	return &TestClient{abcicli.NewLocalClient(nil, a), MakeCodec()}
}

func GetAvail(ctx sdk.Context, add sdk.AccAddress, ccy string) int64 {
	return testApp.CoinKeeper.GetCoins(ctx, add).AmountOf(ccy)
}

func GetLocked(ctx sdk.Context, add sdk.AccAddress, ccy string) int64 {
	return testApp.AccountKeeper.GetAccount(ctx, add).(common.NamedAccount).GetLockedCoins().AmountOf(ccy)
}

func setGenesis(bapp *BinanceChain, tokens []tokens.GenesisToken, accs ...*common.AppAccount) error {
	genaccs := make([]GenesisAccount, len(accs))
	for i, acc := range accs {
		pk := ed25519.GenPrivKey().PubKey()
		valAddr := pk.Address()
		genaccs[i] = NewGenesisAccount(acc, valAddr)
	}

	genesisState := GenesisState{
		Tokens:     tokens,
		Accounts:   genaccs,
		DexGenesis: dex.DefaultGenesis,
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
	bapp := NewBinanceChain(logger, db, os.Stdout)

	// Construct some genesis bytes to reflect democoin/types/AppAccount
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	baseAcc := auth.BaseAccount{
		Address: addr,
	}
	tokens := []tokens.GenesisToken{{"BNB", "BNB", 100000, addr}}
	acc := &common.AppAccount{baseAcc, "blah", sdk.Coins(nil), sdk.Coins(nil)}

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
