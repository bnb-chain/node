package tx_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app"
	"github.com/BiJie/BinanceChain/common/fees"
	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

func newTestMsg(addrs ...sdk.AccAddress) *sdk.TestMsg {
	fees.UnsetAllCalculators()
	testMsg := sdk.NewTestMsg(addrs...)
	fees.RegisterCalculator(testMsg.Type(), fees.FreeFeeCalculator())
	return testMsg
}

func newTestMsgWithFeeCalculator(calculator fees.FeeCalculator, addrs ...sdk.AccAddress) *sdk.TestMsg {
	fees.UnsetAllCalculators()
	testMsg := sdk.NewTestMsg(addrs...)
	fees.RegisterCalculator(testMsg.Type(), calculator)
	return testMsg
}

// coins to more than cover the fee
func newCoins() sdk.Coins {
	return testutils.NewNativeTokens(100)
}

// run the tx through the anteHandler and ensure its valid
func checkValidTx(t *testing.T, anteHandler sdk.AnteHandler, ctx sdk.Context, tx sdk.Tx) {
	_, result, abort := anteHandler(ctx, tx, sdk.RunTxModeCheck)
	require.False(t, abort)
	require.Equal(t, sdk.ABCICodeOK, result.Code)
	require.True(t, result.IsOK())
}

// run the tx through the anteHandler and ensure it fails with the given code
func checkInvalidTx(t *testing.T, anteHandler sdk.AnteHandler, ctx sdk.Context, tx sdk.Tx, code sdk.CodeType) {
	_, result, abort := anteHandler(ctx, tx, sdk.RunTxModeCheck)
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
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized)

	// test num sigs dont match GetSigners
	privs, accNums, seqs = []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized)

	// test an unrecognized account
	privs, accNums, seqs = []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{0, 0, 0}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnknownAddress)

	// save the first account, but second is still unrecognized
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	mapper.SetAccount(ctx, acc1)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnknownAddress)
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
	checkValidTx(t, anteHandler, ctx, tx)

	// new tx from wrong account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// from correct account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{0}, seqs)
	checkValidTx(t, anteHandler, ctx, tx)

	// new tx with another signer and incorrect account numbers
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr2, addr1)
	msgs = []sdk.Msg{msg1, msg2}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{1, 0}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// correct account numbers
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx)
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
	checkValidTx(t, anteHandler, ctx, tx)

	// test sending it again fails (replay protection)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// fix sequence, should pass
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx)

	// new tx with another signer and correct sequences
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr3, addr1)
	msgs = []sdk.Msg{msg1, msg2}

	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{2, 0, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx)

	// replay fails
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// tx from just second signer with incorrect sequence fails
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{1}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// fix the sequence and it passes
	tx = newTestTx(ctx, msgs, []crypto.PrivKey{priv2}, []int64{1}, []int64{1})
	checkValidTx(t, anteHandler, ctx, tx)

	// another tx from both of them that passes
	msg = newTestMsg(addr1, addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{3, 2}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx)
}

// Test logic around memo gas consumption.
func TestAnteHandlerMemoGas(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountMapper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper, "")
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	mapper.SetAccount(ctx, acc1)

	// msg and signatures
	var txn sdk.Tx
	msg := newTestMsg(addr1)
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	fee := tx.NewStdFee(0, sdk.NewCoin("atom", 0))

	// tx does not have enough gas
	txn = newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeOutOfGas)

	// tx with memo doesn't have enough gas
	fee = tx.NewStdFee(801, sdk.NewCoin("atom", 0))
	txn = newTestTxWithMemo(ctx, []sdk.Msg{msg}, privs, accnums, seqs, fee, "abcininasidniandsinasindiansdiansdinaisndiasndiadninsd")
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeOutOfGas)

	// memo too large
	fee = tx.NewStdFee(2001, sdk.NewCoin("atom", 0))
	txn = newTestTxWithMemo(ctx, []sdk.Msg{msg}, privs, accnums, seqs, fee, "abcininasidniandsinasindiansdiansdinaisndiasndiadninsdabcininasidniandsinasindiansdiansdinaisndiasndiadninsdabcininasidniandsinasindiansdiansdinaisndiasndiadninsd")
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeMemoTooLarge)

	// tx with memo has enough gas
	fee = tx.NewStdFee(1100, sdk.NewCoin("atom", 0))
	txn = newTestTxWithMemo(ctx, []sdk.Msg{msg}, privs, accnums, seqs, fee, "abcininasidniandsinasindiansdiansdinaisndiasndiadninsd")
	checkValidTx(t, anteHandler, ctx, txn)
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

	checkValidTx(t, anteHandler, ctx, tx)

	// change sequence numbers
	tx = newTestTx(ctx, []sdk.Msg{msg1}, []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{1, 1})
	checkValidTx(t, anteHandler, ctx, tx)
	tx = newTestTx(ctx, []sdk.Msg{msg2}, []crypto.PrivKey{priv3, priv1}, []int64{2, 0}, []int64{1, 2})
	checkValidTx(t, anteHandler, ctx, tx)

	// expected seqs = [3, 2, 2]
	tx = newTestTxWithMemo(ctx, msgs, privs, accnums, []int64{3, 2, 2}, "Check signers are in expected order and different account numbers and sequence numbers works")
	checkValidTx(t, anteHandler, ctx, tx)
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
	checkValidTx(t, anteHandler, ctx, txn)

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
		checkInvalidTx(t, anteHandler, ctx, txn, cs.code)
	}

	// test wrong signer if public key exist
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{0}, []int64{1}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized)

	// test wrong signer if public doesn't exist
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv1}, []int64{1}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey)
}

func TestAnteHandlerGoodOrderID(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountMapper(cdc, capKey, auth.ProtoBaseAccount)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()

	// set the accounts
	sequence := int64(50)
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	acc1.SetSequence(sequence)
	mapper.SetAccount(ctx, acc1)

	orderId := fmt.Sprintf("%X-%d", acc1.GetAddress(), sequence)
	orderMsg := order.NewNewOrderMsg(acc1.GetAddress(), orderId, 1, "XXX_XXX", 0, 0)

	// bogus fees calculator
	tx.UnsetAllCalculators()
	tx.RegisterCalculator(orderMsg.Type(), tx.FreeFeeCalculator())

	anteHandler := tx.NewAnteHandler(mapper, orderMsg.Type())
	msgs := []sdk.Msg{orderMsg}
	fee := newStdFee()

	// test good tx and signBytes
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{sequence}
	txn := newTestTx(ctx, msgs, privs, accnums, seqs, fee)

	checkValidTx(t, anteHandler, ctx, txn)
}

func TestAnteHandlerBadOrderID(t *testing.T) {
	// setup
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountMapper(cdc, capKey, auth.ProtoBaseAccount)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)

	orderId := "INVALID"
	orderMsg := order.NewNewOrderMsg(acc1.GetAddress(), orderId, 2, "XXX_XXX", 0, 0)

	anteHandler := tx.NewAnteHandler(mapper, orderMsg.Type())
	msgs := []sdk.Msg{orderMsg}
	fee := newStdFee()

	// test good tx and signBytes
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn := newTestTx(ctx, msgs, privs, accnums, seqs, fee)

	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnknownRequest)
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
	checkValidTx(t, anteHandler, ctx, txn)

	acc1 = mapper.GetAccount(ctx, addr1)
	require.Equal(t, acc1.GetPubKey(), priv1.PubKey())

	// test public key not found
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	txn = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	sigs := txn.(auth.StdTx).GetSignatures()
	sigs[0].PubKey = nil
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey)

	acc2 = mapper.GetAccount(ctx, addr2)
	require.Nil(t, acc2.GetPubKey())

	// test invalid signature and public key
	txn = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey)

	acc2 = mapper.GetAccount(ctx, addr2)
	require.Nil(t, acc2.GetPubKey())
}

func setup() (mapper auth.AccountMapper, ctx sdk.Context, anteHandler sdk.AnteHandler) {
	ms, capKey, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper = auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler = tx.NewAnteHandler(mapper)
	accountCache := getAccountCache(cdc, ms, capKey)

	ctx = sdk.NewContext(ms, abci.Header{ChainID: "mychainid", Height: 1}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	return
}

func runAnteHandlerWithMultiTxFees(ctx sdk.Context, anteHandler sdk.AnteHandler, priv crypto.PrivKey, addr sdk.AccAddress, feeCalculators ...fees.FeeCalculator) sdk.Context {
	for i := 0; i < len(feeCalculators); i++ {
		msg := newTestMsgWithFeeCalculator(feeCalculators[i], addr)
		txn := newTestTx(ctx, []sdk.Msg{msg}, []crypto.PrivKey{priv}, []int64{0}, []int64{int64(i)})
		txBytes, _ := app.Codec.MarshalBinary(txn)
		txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
		ctx, _, _ = anteHandler(ctx.WithValue(baseapp.TxHashKey, txHash), txn, sdk.RunTxModeCheck)
		if ctx.IsDeliverTx() {
			fees.Pool.CommitFee(txHash)
		}
	}

	return ctx
}

func checkBalance(t *testing.T, am auth.AccountKeeper, ctx sdk.Context, addr sdk.AccAddress, accNewBalance sdk.Coins) {
	newBalance := am.GetAccount(ctx, addr).GetCoins()
	require.Equal(t, accNewBalance, newBalance)
}

func checkFee(t *testing.T, expectFee types.Fee) {
	fee := fees.Pool.BlockFees()
	require.Equal(t, expectFee, fee)
	fees.Pool.Clear()
}

// Test logic around fee deduction.
func TestAnteHandlerFeesInCheckTx(t *testing.T) {
	am, ctx, anteHandler := setup()
	// set the accounts
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)

	ctx = ctx.WithRunTxMode(sdk.RunTxModeCheck)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), fees.FixedFeeCalculator(10, types.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, types.Fee{})
}

func TestAnteHandlerOneTxFee(t *testing.T) {
	// one tx, FeeFree
	am, ctx, anteHandler := setup()
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), fees.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 100)})
	checkFee(t, types.Fee{})

	// one tx, FeeForProposer
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), fees.FixedFeeCalculator(10, types.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, types.FeeForProposer))

	// one tx, FeeForAll
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), fees.FixedFeeCalculator(10, types.FeeForAll))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, types.FeeForAll))
}

func TestAnteHandlerMultiTxFees(t *testing.T) {
	// two txs, 1. FeeFree 2. FeeProposer
	am, ctx, anteHandler := setup()
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		fees.FreeFeeCalculator(),
		fees.FixedFeeCalculator(10, types.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, types.FeeForProposer))

	// two txs, 1. FeeProposer 2. FeeFree
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		fees.FixedFeeCalculator(10, types.FeeForProposer),
		fees.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 90)})
	checkFee(t, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 10)}, types.FeeForProposer))

	// two txs, 1. FeeProposer 2. FeeForAll
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		fees.FixedFeeCalculator(10, types.FeeForProposer),
		fees.FixedFeeCalculator(10, types.FeeForAll))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 80)})
	checkFee(t, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 20)}, types.FeeForAll))

	// two txs, 1. FeeForAll 2. FeeProposer
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		fees.FixedFeeCalculator(10, types.FeeForAll),
		fees.FixedFeeCalculator(10, types.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 80)})
	checkFee(t, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 20)}, types.FeeForAll))

	// three txs, 1. FeeForAll 2. FeeProposer 3. FeeFree
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		fees.FixedFeeCalculator(10, types.FeeForAll),
		fees.FixedFeeCalculator(10, types.FeeForProposer),
		fees.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 80)})
	checkFee(t, types.NewFee(sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 20)}, types.FeeForAll))
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
	res := prechecker(ctx, cdc.MustMarshalBinary(txn), txn)
	require.NotEqual(t, sdk.ABCICodeOK, res.Code, "Failed prechecker")
	require.Contains(t, res.Log, "no signers")

	privs, accNums, seqs = []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs)
	res = prechecker(ctx, cdc.MustMarshalBinary(txn), txn)
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
	res := prechecker(ctx, cdc.MustMarshalBinary(txn), txn)
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
		res := prechecker(ctx, cdc.MustMarshalBinary(txn), txn)
		require.NotEqual(t, sdk.ABCICodeOK, res.Code)
	}

	// test wrong signer if public key exist
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs)
	res = prechecker(ctx, cdc.MustMarshalBinary(txn), txn)
	require.Equal(t, sdk.ABCICodeOK, res.Code)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized)
}
