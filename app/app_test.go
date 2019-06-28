package app

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/testutils"
)

func TearDown() {
	// remove block db
	os.RemoveAll(cfg.DefaultConfig().DBDir())
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
	require.Nil(t, err)

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
	require.Nil(t, err)

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
	require.Nil(t, err)

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
	require.Nil(t, err)

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
	require.Nil(t, err)

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
	require.Nil(t, err)

	res := app.PreCheckTx(txBytes)
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
	require.Nil(t, err)

	res := app.PreCheckTx(txBytes)
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
	require.Nil(t, err)

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
	require.Nil(t, err)

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
	require.Nil(t, err)

	res := app.CheckTx(txBytes)
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
	require.Nil(t, err)

	res := app.CheckTx(txBytes)
	require.Equal(t, res.Code, uint32(sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeInvalidSequence)))
	require.Contains(t, res.Log, "Invalid account number")
}
