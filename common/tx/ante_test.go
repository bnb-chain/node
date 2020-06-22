package tx_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkfees "github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/auth"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/app"
	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/common/tx"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/wire"
)

func newTestMsg(addrs ...sdk.AccAddress) *sdk.TestMsg {
	sdkfees.UnsetAllCalculators()
	testMsg := sdk.NewTestMsg(addrs...)
	sdkfees.RegisterCalculator(testMsg.Type(), sdkfees.FreeFeeCalculator())
	return testMsg
}

func newTestMsgWithFeeCalculator(calculator sdkfees.FeeCalculator, addrs ...sdk.AccAddress) *sdk.TestMsg {
	sdkfees.UnsetAllCalculators()
	testMsg := sdk.NewTestMsg(addrs...)
	sdkfees.RegisterCalculator(testMsg.Type(), calculator)
	return testMsg
}

// coins to more than cover the fee
func newCoins() sdk.Coins {
	return testutils.NewNativeTokens(100)
}

// run the tx through the anteHandler and ensure its valid
func checkValidTx(t *testing.T, anteHandler sdk.AnteHandler, ctx sdk.Context, tx sdk.Tx, mode sdk.RunTxMode) {
	_, result, abort := anteHandler(ctx, tx, mode)
	require.False(t, abort)
	require.Equal(t, sdk.ABCICodeOK, result.Code)
	require.True(t, result.IsOK())
}

// run the tx through the anteHandler and ensure it fails with the given code
func checkInvalidTx(t *testing.T, anteHandler sdk.AnteHandler, ctx sdk.Context, tx sdk.Tx, code sdk.CodeType, mode sdk.RunTxMode) {
	_, result, abort := anteHandler(ctx, tx, mode)
	require.True(t, abort)
	require.Equal(t, sdk.ToABCICode(sdk.CodespaceRoot, code), result.Code,
		fmt.Sprintf("Expected %v, got %v", sdk.ToABCICode(sdk.CodespaceRoot, code), result))
}

func newTestTx(ctx sdk.Context, msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64) auth.StdTx {
	sigs := make([]auth.StdSignature, len(privs))
	for i, priv := range privs {
		signBytes := auth.StdSignBytes(ctx.ChainID(), accNums[i], seqs[i], msgs, "", 0, nil)
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = auth.StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := auth.NewStdTx(msgs, sigs, "", 0, nil)
	return tx
}

func newTestTxWithMemo(ctx sdk.Context, msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64, memo string) sdk.Tx {
	sigs := make([]auth.StdSignature, len(privs))
	for i, priv := range privs {
		signBytes := auth.StdSignBytes(ctx.ChainID(), accNums[i], seqs[i], msgs, memo, 0, nil)
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = auth.StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := auth.NewStdTx(msgs, sigs, memo, 0, nil)
	return tx
}

// All signers sign over the same StdSignDoc. Should always create invalid signatures
func newTestTxWithSignBytes(msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64, signBytes []byte, memo string) sdk.Tx {
	sigs := make([]auth.StdSignature, len(privs))
	for i, priv := range privs {
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = auth.StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := auth.NewStdTx(msgs, sigs, memo, 0, nil)
	return tx
}

func getAccountCache(cdc *codec.Codec, ms sdk.MultiStore, accountKey *sdk.KVStoreKey) sdk.AccountCache {
	accountStore := ms.GetKVStore(accountKey)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	return auth.NewAccountCache(accountStoreCache)
}

// Test various error cases in the AnteHandler control flow.
func TestAnteHandlerSigErrors(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	priv2, addr2 := testutils.PrivAndAddr()
	priv3, addr3 := testutils.PrivAndAddr()

	// msg and signatures
	var txn sdk.Tx
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr1, addr3)

	msgs := []sdk.Msg{msg1, msg2}

	// test no signatures
	privs, accNums, seqs := []crypto.PrivKey{}, []int64{}, []int64{}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs)

	// tx.GetSigners returns addresses in correct order: addr1, addr2, addr3
	expectedSigners := []sdk.AccAddress{addr1, addr2, addr3}
	stdTx := txn.(auth.StdTx)
	require.Equal(t, expectedSigners, stdTx.GetSigners())

	// Check no signatures fails
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized, sdk.RunTxModeCheck)

	// test num sigs dont match GetSigners
	privs, accNums, seqs = []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized, sdk.RunTxModeCheck)

	// test an unrecognized account
	privs, accNums, seqs = []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{0, 0, 0}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnknownAddress, sdk.RunTxModeCheck)

	// save the first account, but second is still unrecognized
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	mapper.SetAccount(ctx, acc1)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnknownAddress, sdk.RunTxModeCheck)
}

// Test logic around account number checking with one signer and many signers.
func TestAnteHandlerAccountNumbers(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	accountCache := getAccountCache(cdc, ms, capKey)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	priv2, addr2 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)
	acc2 := mapper.NewAccountWithAddress(ctx, addr2)
	acc2.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc2)

	// msg and signatures
	var tx sdk.Tx
	msg := newTestMsg(addr1)

	msgs := []sdk.Msg{msg}

	// test good tx from one signer
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)

	// new tx from wrong account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence, sdk.RunTxModeCheck)

	// from correct account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{0}, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)

	// new tx with another signer and incorrect account numbers
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr2, addr1)
	msgs = []sdk.Msg{msg1, msg2}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{1, 0}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence, sdk.RunTxModeCheck)

	// correct account numbers
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)
}

// Test logic around sequence checking with one signer and many signers.
func TestAnteHandlerSequences(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	priv2, addr2 := testutils.PrivAndAddr()
	priv3, addr3 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)
	acc2 := mapper.NewAccountWithAddress(ctx, addr2)
	acc2.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc2)
	acc3 := mapper.NewAccountWithAddress(ctx, addr3)
	acc3.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc3)

	// msg and signatures
	var tx sdk.Tx
	msg := newTestMsg(addr1)

	msgs := []sdk.Msg{msg}

	// test good tx from one signer
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)

	// test sending it again fails (replay protection)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence, sdk.RunTxModeCheck)

	// fix sequence, should pass
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)

	// new tx with another signer and correct sequences
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr3, addr1)
	msgs = []sdk.Msg{msg1, msg2}

	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{2, 0, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)

	// replay fails
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence, sdk.RunTxModeCheck)

	// tx from just second signer with incorrect sequence fails
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{1}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence, sdk.RunTxModeCheck)

	// fix the sequence and it passes
	tx = newTestTx(ctx, msgs, []crypto.PrivKey{priv2}, []int64{1}, []int64{1})
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)

	// another tx from both of them that passes
	msg = newTestMsg(addr1, addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{3, 2}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)
}

func TestAnteHandlerMultiSigner(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	priv2, addr2 := testutils.PrivAndAddr()
	priv3, addr3 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)
	acc2 := mapper.NewAccountWithAddress(ctx, addr2)
	acc2.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc2)
	acc3 := mapper.NewAccountWithAddress(ctx, addr3)
	acc3.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc3)

	// set up msgs
	var tx sdk.Tx
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr3, addr1)
	msg3 := newTestMsg(addr2, addr3)
	msgs := []sdk.Msg{msg1, msg2, msg3}

	// signers in order
	privs, accnums, seqs := []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{0, 0, 0}
	tx = newTestTxWithMemo(ctx, msgs, privs, accnums, seqs, "Check signers are in expected order and different account numbers works")

	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)

	// change sequence numbers
	tx = newTestTx(ctx, []sdk.Msg{msg1}, []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{1, 1})
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)
	tx = newTestTx(ctx, []sdk.Msg{msg2}, []crypto.PrivKey{priv3, priv1}, []int64{2, 0}, []int64{1, 2})
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)

	// expected seqs = [3, 2, 2]
	tx = newTestTxWithMemo(ctx, msgs, privs, accnums, []int64{3, 2, 2}, "Check signers are in expected order and different account numbers and sequence numbers works")
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeCheck)
}

func TestAnteHandlerBadSignBytes(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	priv2, addr2 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)
	acc2 := mapper.NewAccountWithAddress(ctx, addr2)
	acc2.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc2)

	var txn sdk.Tx
	msg := newTestMsg(addr1)
	msgs := []sdk.Msg{msg}

	// test good tx and signBytes
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, txn, sdk.RunTxModeCheck)

	chainID := ctx.ChainID()
	chainID2 := chainID + "somemorestuff"
	codeUnauth := sdk.CodeUnauthorized

	cases := []struct {
		chainID string
		accnum  int64
		seq     int64
		msgs    []sdk.Msg
		code    sdk.CodeType
	}{
		{chainID2, 0, 1, msgs, codeUnauth},                        // test wrong chain_id
		{chainID, 0, 2, msgs, codeUnauth},                         // test wrong seqs
		{chainID, 1, 1, msgs, codeUnauth},                         // test wrong accnum
		{chainID, 0, 1, []sdk.Msg{newTestMsg(addr2)}, codeUnauth}, // test wrong msg
	}

	privs, seqs = []crypto.PrivKey{priv1}, []int64{1}
	for _, cs := range cases {
		txn := newTestTxWithSignBytes(

			msgs, privs, accnums, seqs,
			auth.StdSignBytes(cs.chainID, cs.accnum, cs.seq, cs.msgs, "", 0, nil),
			"",
		)
		checkInvalidTx(t, anteHandler, ctx, txn, cs.code, sdk.RunTxModeCheck)
	}

	// test wrong signer if public key exist
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{0}, []int64{1}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized, sdk.RunTxModeCheck)

	// test wrong signer if public doesn't exist
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv1}, []int64{1}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey, sdk.RunTxModeCheck)
}

func TestAnteHandlerSetPubKey(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	accountCache := getAccountCache(cdc, ms, capKey)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	_, addr2 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)
	acc2 := mapper.NewAccountWithAddress(ctx, addr2)
	acc2.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc2)

	var txn sdk.Tx

	// test good tx and set public key
	msg := newTestMsg(addr1)
	msgs := []sdk.Msg{msg}
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, txn, sdk.RunTxModeCheck)

	acc1 = mapper.GetAccount(ctx, addr1)
	require.Equal(t, acc1.GetPubKey(), priv1.PubKey())

	// test public key not found
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	txn = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	sigs := txn.(auth.StdTx).GetSignatures()
	sigs[0].PubKey = nil
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey, sdk.RunTxModeCheck)

	acc2 = mapper.GetAccount(ctx, addr2)
	require.Nil(t, acc2.GetPubKey())

	// test invalid signature and public key
	txn = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey, sdk.RunTxModeCheck)

	acc2 = mapper.GetAccount(ctx, addr2)
	require.Nil(t, acc2.GetPubKey())
}

func setup() (mapper auth.AccountKeeper, ctx sdk.Context, anteHandler sdk.AnteHandler) {
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper = auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler = tx.NewAnteHandler(mapper)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx = sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	return
}

func runAnteHandlerWithMultiTxFees(ctx sdk.Context, anteHandler sdk.AnteHandler, priv crypto.PrivKey, addr sdk.AccAddress, feeCalculators ...sdkfees.FeeCalculator) sdk.Context {
	for i := 0; i < len(feeCalculators); i++ {
		msg := newTestMsgWithFeeCalculator(feeCalculators[i], addr)
		txn := newTestTx(ctx, []sdk.Msg{msg}, []crypto.PrivKey{priv}, []int64{0}, []int64{int64(i)})
		txBytes, _ := app.Codec.MarshalBinaryLengthPrefixed(txn)
		txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
		ctx, _, _ = anteHandler(ctx.WithValue(baseapp.TxHashKey, txHash), txn, sdk.RunTxModeCheck)
		if ctx.IsDeliverTx() {
			sdkfees.Pool.CommitFee(txHash)
		}
	}

	return ctx
}

func checkBalance(t *testing.T, am auth.AccountKeeper, ctx sdk.Context, addr sdk.AccAddress, accNewBalance sdk.Coins) {
	newBalance := am.GetAccount(ctx, addr).GetCoins()
	require.Equal(t, accNewBalance, newBalance)
}

func checkFee(t *testing.T, expectFee sdk.Fee) {
	fee := sdkfees.Pool.BlockFees()
	require.Equal(t, expectFee, fee)
	sdkfees.Pool.Clear()
}

// Test logic around fee deduction.
func TestAnteHandlerFeesInCheckTx(t *testing.T) {
	am, ctx, anteHandler := setup()
	// set the accounts
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)

	ctx = ctx.WithRunTxMode(sdk.RunTxModeCheck)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), sdkfees.FixedFeeCalculator(10, sdk.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, sdk.Fee{})
}

func TestAnteHandlerOneTxFee(t *testing.T) {
	// one tx, FeeFree
	am, ctx, anteHandler := setup()
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), sdkfees.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 100)})
	checkFee(t, sdk.Fee{})

	// one tx, FeeForProposer
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), sdkfees.FixedFeeCalculator(10, sdk.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, sdk.FeeForProposer))

	// one tx, FeeForAll
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), sdkfees.FixedFeeCalculator(10, sdk.FeeForAll))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, sdk.FeeForAll))
}

func TestAnteHandlerMultiTxFees(t *testing.T) {
	// two txs, 1. FeeFree 2. FeeProposer
	am, ctx, anteHandler := setup()
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		sdkfees.FreeFeeCalculator(),
		sdkfees.FixedFeeCalculator(10, sdk.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, sdk.FeeForProposer))

	// two txs, 1. FeeProposer 2. FeeFree
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		sdkfees.FixedFeeCalculator(10, sdk.FeeForProposer),
		sdkfees.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, sdk.FeeForProposer))

	// two txs, 1. FeeProposer 2. FeeForAll
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		sdkfees.FixedFeeCalculator(10, sdk.FeeForProposer),
		sdkfees.FixedFeeCalculator(10, sdk.FeeForAll))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 80)})
	checkFee(t, sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 20)}, sdk.FeeForAll))

	// two txs, 1. FeeForAll 2. FeeProposer
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		sdkfees.FixedFeeCalculator(10, sdk.FeeForAll),
		sdkfees.FixedFeeCalculator(10, sdk.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 80)})
	checkFee(t, sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 20)}, sdk.FeeForAll))

	// three txs, 1. FeeForAll 2. FeeProposer 3. FeeFree
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		sdkfees.FixedFeeCalculator(10, sdk.FeeForAll),
		sdkfees.FixedFeeCalculator(10, sdk.FeeForProposer),
		sdkfees.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 80)})
	checkFee(t, sdk.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 20)}, sdk.FeeForAll))
}

func TestNewTxPreCheckerEmptySigner(t *testing.T) {
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	sdk.RegisterCodec(cdc)
	cdc.RegisterConcrete(sdk.TestMsg{}, "antetest/TestMsg", nil)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	_, addr2 := testutils.PrivAndAddr()
	_, addr3 := testutils.PrivAndAddr()

	// msg and signatures
	var txn sdk.Tx
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr1, addr3)

	msgs := []sdk.Msg{msg1, msg2}

	// test no signatures
	privs, accNums, seqs := []crypto.PrivKey{}, []int64{}, []int64{}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs)

	// tx.GetSigners returns addresses in correct order: addr1, addr2, addr3
	expectedSigners := []sdk.AccAddress{addr1, addr2, addr3}
	stdTx := txn.(auth.StdTx)
	require.Equal(t, expectedSigners, stdTx.GetSigners())

	prechecker := tx.NewTxPreChecker()
	res := prechecker(ctx, cdc.MustMarshalBinaryLengthPrefixed(txn), txn)
	require.NotEqual(t, sdk.ABCICodeOK, res.Code, "Failed prechecker")
	require.Contains(t, res.Log, "no signers")

	privs, accNums, seqs = []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs)
	res = prechecker(ctx, cdc.MustMarshalBinaryLengthPrefixed(txn), txn)
	require.NotEqual(t, sdk.ABCICodeOK, res.Code, "Failed prechecker2")
	require.Contains(t, res.Log, "wrong number of signers")
}

func Test_NewTxPreCheckerSignature(t *testing.T) {
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	sdk.RegisterCodec(cdc)
	cdc.RegisterConcrete(sdk.TestMsg{}, "antetest/TestMsg", nil)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	anteHandler := tx.NewAnteHandler(mapper)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	priv2, addr2 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)
	acc2 := mapper.NewAccountWithAddress(ctx, addr2)
	acc1.SetPubKey(priv2.PubKey())
	acc2.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc2)

	var txn sdk.Tx
	msg := newTestMsg(addr1)
	msgs := []sdk.Msg{msg}

	// test good tx and signBytes
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	prechecker := tx.NewTxPreChecker()
	res := prechecker(ctx, cdc.MustMarshalBinaryLengthPrefixed(txn), txn)
	require.Equal(t, sdk.ABCICodeOK, res.Code, "Failed prechecker")

	chainID := ctx.ChainID()
	chainID2 := chainID + "somemorestuff"
	codeUnauth := sdk.CodeUnauthorized

	cases := []struct {
		chainID string
		accnum  int64
		seq     int64
		msgs    []sdk.Msg
		code    sdk.CodeType
	}{
		{chainID2, 0, 1, msgs, codeUnauth},                        // test wrong chain_id
		{chainID, 0, 2, msgs, codeUnauth},                         // test wrong seqs
		{chainID, 1, 1, msgs, codeUnauth},                         // test wrong accnum
		{chainID, 0, 1, []sdk.Msg{newTestMsg(addr2)}, codeUnauth}, // test wrong msg
	}

	privs, seqs = []crypto.PrivKey{priv1}, []int64{1}
	for _, cs := range cases {
		txn := newTestTxWithSignBytes(

			msgs, privs, accnums, seqs,
			auth.StdSignBytes(cs.chainID, cs.accnum, cs.seq, cs.msgs, "", 0, nil),
			"",
		)
		res := prechecker(ctx, cdc.MustMarshalBinaryLengthPrefixed(txn), txn)
		require.NotEqual(t, sdk.ABCICodeOK, res.Code)
	}

	// test wrong signer if public key exist
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	res = prechecker(ctx, cdc.MustMarshalBinaryLengthPrefixed(txn), txn)
	require.Equal(t, sdk.ABCICodeOK, res.Code)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey, sdk.RunTxModeCheckAfterPre)

	// test empty pubkey
	privs, accnums, seqs = []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	stdtx := txn.(auth.StdTx)
	stdtx.Signatures[0].PubKey = nil
	res = prechecker(ctx, cdc.MustMarshalBinaryLengthPrefixed(txn), txn)
	require.Equal(t, sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeInvalidPubKey), res.Code)
}

func Test_NewTxPreCheckerNilMsg(t *testing.T) {
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	sdk.RegisterCodec(cdc)
	cdc.RegisterConcrete(sdk.TestMsg{}, "antetest/TestMsg", nil)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetPubKey(priv1.PubKey())
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)

	var txn auth.StdTx
	msg := newTestMsg(addr1)
	msgs := []sdk.Msg{msg}

	// test good tx and signBytes
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	txn.Msgs[0] = nil
	prechecker := tx.NewTxPreChecker()
	res := prechecker(ctx, cdc.MustMarshalBinaryLengthPrefixed(txn), txn)
	require.Equal(t, sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeUnknownRequest), res.Code)
}
