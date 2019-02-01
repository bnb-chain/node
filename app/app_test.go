package app

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/fees"
	common "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex"
	"github.com/binance-chain/node/plugins/param"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/wire"
)

type TestClient struct {
	cl  abcicli.Client
	cdc *wire.Codec
}

func NewMockAnteHandler(cdc *wire.Codec) sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, runTxMode sdk.RunTxMode) (newCtx sdk.Context, result sdk.Result, abort bool) {
		msg := tx.GetMsgs()[0]
		fee := fees.GetCalculator(msg.Type())(msg)

		if ctx.IsDeliverTx() {
			// add fee to pool, even it's free
			stdTx := tx.(auth.StdTx)
			txHash := cmn.HexBytes(tmhash.Sum(cdc.MustMarshalBinaryLengthPrefixed(stdTx))).String()
			fees.Pool.AddFee(txHash, fee)
		}

		return newCtx, sdk.Result{}, false
	}
}

func (tc *TestClient) DeliverTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)
	return tc.cl.DeliverTxAsync(tx)
}

func (tc *TestClient) CheckTxAsync(msg sdk.Msg, cdc *wire.Codec) *abcicli.ReqRes {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)
	return tc.cl.CheckTxAsync(tx)
}

func (tc *TestClient) DeliverTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseDeliverTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)
	return tc.cl.DeliverTxSync(tx)
}

func (tc *TestClient) CheckTxSync(msg sdk.Msg, cdc *wire.Codec) (*types.ResponseCheckTx, error) {
	stdtx := auth.NewStdTx([]sdk.Msg{msg}, nil, "test", 0, nil)
	tx, _ := tc.cdc.MarshalBinaryLengthPrefixed(stdtx)
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
	a.SetAnteHandler(NewMockAnteHandler(a.Codec)) // clear AnteHandler to skip the signature verification step
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
		Tokens:       tokens,
		Accounts:     genaccs,
		DexGenesis:   dex.DefaultGenesis,
		ParamGenesis: param.DefaultGenesisState,
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
	tokens := []tokens.GenesisToken{{"BNB", "BNB", 100000, addr, false}}
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

func defaultLogger() log.Logger {
	return log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "sdk/app")
}

func newBinanceChainApp(options ...func(baseApp *baseapp.BaseApp)) *BinanceChain {
	logger := defaultLogger()
	db := dbm.NewMemDB()
	return NewBinanceChain(logger, db, os.Stdout, options...)
}

// msg type for testing
type TestMsg struct {
	Signers []sdk.AccAddress
}

func NewTestMsg(addrs ...sdk.AccAddress) *TestMsg {
	return &TestMsg{
		Signers: addrs,
	}
}

//nolint
func (msg *TestMsg) Route() string { return "TestMsg" }
func (msg *TestMsg) Type() string  { return "Test message" }
func (msg *TestMsg) GetSignBytes() []byte {
	bz, err := json.Marshal(msg.Signers)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}
func (msg *TestMsg) ValidateBasic() sdk.Error { return nil }
func (msg *TestMsg) GetSigners() []sdk.AccAddress {
	return msg.Signers
}
func (msg *TestMsg) GetInvolvedAddresses() []sdk.AccAddress {
	return msg.Signers
}

func handleTestMsg() sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		return sdk.Result{}
	}
}

func newTestMsg(addrs ...sdk.AccAddress) *TestMsg {
	testMsg := NewTestMsg(addrs...)
	return testMsg
}

func newTestTx(ctx sdk.Context, msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64, data []byte, memo string) auth.StdTx {
	sigs := make([]auth.StdSignature, len(privs))
	for i, priv := range privs {
		chainId := ctx.ChainID()
		signBytes := auth.StdSignBytes(chainId, accNums[i], seqs[i], msgs, memo, 0, data)
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = auth.StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := auth.NewStdTx(msgs, sigs, memo, 0, data)
	return tx
}

func TestPreCheckTxWithRightPubKey(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	priv1, addr1 := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")

	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.PreCheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ABCICodeOK))
	res = app.CheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ABCICodeOK))
}

func TestPreCheckTxWithWrongPubKey(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	priv1, addr1 := testutils.PrivAndAddr()
	priv2, _ := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv2}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")

	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.PreCheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ABCICodeOK))
	res = app.CheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeInvalidPubKey)))
	require.Contains(t, res.Log, "PubKey of account does not match PubKey of signature")
}

func TestPreCheckTxWithEmptyPubKey(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	priv1, addr1 := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")
	tx.Signatures[0].PubKey = nil

	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.PreCheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeInvalidPubKey)))
	require.Contains(t, res.Log, "public key of signature should not be nil")
}

func TestPreCheckTxWithEmptySignatures(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	priv1, addr1 := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")
	tx.Signatures = []auth.StdSignature{}

	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.PreCheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeUnauthorized)))
	require.Contains(t, res.Log, "no signers")
}

func TestPreCheckTxWithWrongSignerNum(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	priv1, addr1 := testutils.PrivAndAddr()
	_, addr2 := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1, addr2)
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")

	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.PreCheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeUnauthorized)))
	require.Contains(t, res.Log, "wrong number of signers")
}

func TestPreCheckTxWithData(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	priv1, addr1 := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, []byte("data"), "")

	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.PreCheckTx(txBytes)
	println(res.Log)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeUnauthorized)))
	require.Contains(t, res.Log, "data field is not allowed to use in transaction for now")
}

func TestPreCheckTxWithLargeMemo(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	priv1, addr1 := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, string(make([]byte, 200, 200)))

	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.PreCheckTx(txBytes)
	println(res.Log)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeMemoTooLarge)))
	require.Contains(t, res.Log, "maximum number of characters")
}

func TestPreCheckTxWithWrongPubKeyAndEmptyAccountPubKey(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	_, addr1 := testutils.PrivAndAddr()
	priv2, _ := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv2}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")
	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.PreCheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.CodeOK))
	res = app.CheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeInvalidPubKey)))
	require.Contains(t, res.Log, "PubKey does not match Signer address")
}

func TestCheckTxWithUnrecognizedAccount(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	_, addr1 := testutils.PrivAndAddr()
	priv2, _ := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv2}, []int64{0}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")
	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.CheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeUnknownAddress)))
}

func TestCheckTxWithWrongSequence(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	_, addr1 := testutils.PrivAndAddr()
	priv2, _ := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv2}, []int64{0}, []int64{1}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")
	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.CheckTx(txBytes)
	println(res.Log)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeInvalidSequence)))
	require.Contains(t, res.Log, "Invalid sequence")
}

func TestCheckTxWithWrongAccountNum(t *testing.T) {
	routerOpt := func(bapp *baseapp.BaseApp) {
		bapp.Router().AddRoute("TestMsg", handleTestMsg())
	}

	_, addr1 := testutils.PrivAndAddr()
	priv2, _ := testutils.PrivAndAddr()

	Codec = MakeCodec()
	app := newBinanceChainApp(routerOpt)
	app.Codec.RegisterConcrete(&TestMsg{}, "cosmos-sdk/baseapp/testMsg", nil)

	app.BeginBlock(abci.RequestBeginBlock{})

	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv2}, []int64{1}, []int64{0}
	ctx := app.NewContext(sdk.RunTxModeCheck, types.Header{})
	acc1 := app.AccountKeeper.NewAccountWithAddress(app.CheckState.Ctx, addr1)
	app.AccountKeeper.SetAccount(app.CheckState.Ctx, acc1)

	tx := newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, nil, "")
	txBytes, err := Codec.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		println(err.Error())
	}
	res := app.CheckTx(txBytes)
	println(res.Log)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeInvalidSequence)))
	require.Contains(t, res.Log, "Invalid account number")
}
