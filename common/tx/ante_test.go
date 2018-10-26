package tx_test

import (
	"fmt"
	"github.com/BiJie/BinanceChain/wire"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/common/testutils"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
)

func newTestMsg(addrs ...sdk.AccAddress) *sdk.TestMsg {
	tx.UnsetAllCalculators()
	testMsg := sdk.NewTestMsg(addrs...)
	tx.RegisterCalculator(testMsg.Type(), tx.FreeFeeCalculator())
	return testMsg
}

func newTestMsgWithFeeCalculator(calculator tx.FeeCalculator, addrs ...sdk.AccAddress) *sdk.TestMsg {
	tx.UnsetAllCalculators()
	testMsg := sdk.NewTestMsg(addrs...)
	tx.RegisterCalculator(testMsg.Type(), calculator)
	return testMsg
}

func newStdFee() tx.StdFee {
	return tx.NewStdFee(5000,
		sdk.NewInt64Coin("atom", 150),
	)
}

// coins to more than cover the fee
func newCoins() sdk.Coins {
	return testutils.NewNativeTokens(100)
}

// run the tx through the anteHandler and ensure its valid
func checkValidTx(t *testing.T, anteHandler sdk.AnteHandler, ctx sdk.Context, tx sdk.Tx) {
	_, result, abort := anteHandler(ctx, tx)
	require.False(t, abort)
	require.Equal(t, sdk.ABCICodeOK, result.Code)
	require.True(t, result.IsOK())
}

// run the tx through the anteHandler and ensure it fails with the given code
func checkInvalidTx(t *testing.T, anteHandler sdk.AnteHandler, ctx sdk.Context, tx sdk.Tx, code sdk.CodeType) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case sdk.ErrorOutOfGas:
				require.Equal(t, sdk.ToABCICode(sdk.CodespaceRoot, code), sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeOutOfGas),
					fmt.Sprintf("Expected ErrorOutOfGas, got %v", r))
			default:
				panic(r)
			}
		}
	}()
	_, result, abort := anteHandler(ctx, tx)
	require.True(t, abort)
	require.Equal(t, sdk.ToABCICode(sdk.CodespaceRoot, code), result.Code,
		fmt.Sprintf("Expected %v, got %v", sdk.ToABCICode(sdk.CodespaceRoot, code), result))
}

func newTestTx(ctx sdk.Context, msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64, fee tx.StdFee) sdk.Tx {
	sigs := make([]tx.StdSignature, len(privs))
	for i, priv := range privs {
		signBytes := tx.StdSignBytes(ctx.ChainID(), accNums[i], seqs[i], fee, msgs, "")
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = tx.StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := tx.NewStdTx(msgs, fee, sigs, "")
	return tx
}

func newTestTxWithMemo(ctx sdk.Context, msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64, fee tx.StdFee, memo string) sdk.Tx {
	sigs := make([]tx.StdSignature, len(privs))
	for i, priv := range privs {
		signBytes := tx.StdSignBytes(ctx.ChainID(), accNums[i], seqs[i], fee, msgs, memo)
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = tx.StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := tx.NewStdTx(msgs, fee, sigs, memo)
	return tx
}

// All signers sign over the same StdSignDoc. Should always create invalid signatures
func newTestTxWithSignBytes(msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64, fee tx.StdFee, signBytes []byte, memo string) sdk.Tx {
	sigs := make([]tx.StdSignature, len(privs))
	for i, priv := range privs {
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = tx.StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := tx.NewStdTx(msgs, fee, sigs, memo)
	return tx
}

// Test various error cases in the AnteHandler control flow.
func TestAnteHandlerSigErrors(t *testing.T) {
	// setup
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

	// keys and addresses
	priv1, addr1 := testutils.PrivAndAddr()
	priv2, addr2 := testutils.PrivAndAddr()
	priv3, addr3 := testutils.PrivAndAddr()

	// msg and signatures
	var txn sdk.Tx
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr1, addr3)
	fee := newStdFee()

	msgs := []sdk.Msg{msg1, msg2}

	// test no signatures
	privs, accNums, seqs := []crypto.PrivKey{}, []int64{}, []int64{}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs, fee)

	// tx.GetSigners returns addresses in correct order: addr1, addr2, addr3
	expectedSigners := []sdk.AccAddress{addr1, addr2, addr3}
	stdTx := txn.(tx.StdTx)
	require.Equal(t, expectedSigners, stdTx.GetSigners())

	// Check no signatures fails
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized)

	// test num sigs dont match GetSigners
	privs, accNums, seqs = []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized)

	// test an unrecognized account
	privs, accNums, seqs = []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{0, 0, 0}
	txn = newTestTx(ctx, msgs, privs, accNums, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnknownAddress)

	// save the first account, but second is still unrecognized
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(fee.Amount)
	mapper.SetAccount(ctx, acc1)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnknownAddress)
}

// Test logic around account number checking with one signer and many signers.
func TestAnteHandlerAccountNumbers(t *testing.T) {
	// setup
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

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
	fee := newStdFee()

	msgs := []sdk.Msg{msg}

	// test good tx from one signer
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkValidTx(t, anteHandler, ctx, tx)

	// new tx from wrong account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{1}, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// from correct account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{0}, seqs, fee)
	checkValidTx(t, anteHandler, ctx, tx)

	// new tx with another signer and incorrect account numbers
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr2, addr1)
	msgs = []sdk.Msg{msg1, msg2}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{1, 0}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// correct account numbers
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkValidTx(t, anteHandler, ctx, tx)
}

// Test logic around sequence checking with one signer and many signers.
func TestAnteHandlerSequences(t *testing.T) {
	// setup
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

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
	fee := newStdFee()

	msgs := []sdk.Msg{msg}

	// test good tx from one signer
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkValidTx(t, anteHandler, ctx, tx)

	// test sending it again fails (replay protection)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// fix sequence, should pass
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkValidTx(t, anteHandler, ctx, tx)

	// new tx with another signer and correct sequences
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr3, addr1)
	msgs = []sdk.Msg{msg1, msg2}

	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{2, 0, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkValidTx(t, anteHandler, ctx, tx)

	// replay fails
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// tx from just second signer with incorrect sequence fails
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{1}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.CodeInvalidSequence)

	// fix the sequence and it passes
	tx = newTestTx(ctx, msgs, []crypto.PrivKey{priv2}, []int64{1}, []int64{1}, fee)
	checkValidTx(t, anteHandler, ctx, tx)

	// another tx from both of them that passes
	msg = newTestMsg(addr1, addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{3, 2}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkValidTx(t, anteHandler, ctx, tx)
}

// Test logic around memo gas consumption.
func TestAnteHandlerMemoGas(t *testing.T) {
	// setup
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
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
	fee := tx.NewStdFee(0, sdk.NewInt64Coin("atom", 0))

	// tx does not have enough gas
	txn = newTestTx(ctx, []sdk.Msg{msg}, privs, accnums, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeOutOfGas)

	// tx with memo doesn't have enough gas
	fee = tx.NewStdFee(801, sdk.NewInt64Coin("atom", 0))
	txn = newTestTxWithMemo(ctx, []sdk.Msg{msg}, privs, accnums, seqs, fee, "abcininasidniandsinasindiansdiansdinaisndiasndiadninsd")
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeOutOfGas)

	// memo too large
	fee = tx.NewStdFee(2001, sdk.NewInt64Coin("atom", 0))
	txn = newTestTxWithMemo(ctx, []sdk.Msg{msg}, privs, accnums, seqs, fee, "abcininasidniandsinasindiansdiansdinaisndiasndiadninsdabcininasidniandsinasindiansdiansdinaisndiasndiadninsdabcininasidniandsinasindiansdiansdinaisndiasndiadninsd")
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeMemoTooLarge)

	// tx with memo has enough gas
	fee = tx.NewStdFee(1100, sdk.NewInt64Coin("atom", 0))
	txn = newTestTxWithMemo(ctx, []sdk.Msg{msg}, privs, accnums, seqs, fee, "abcininasidniandsinasindiansdiansdinaisndiasndiadninsd")
	checkValidTx(t, anteHandler, ctx, txn)
}

func TestAnteHandlerMultiSigner(t *testing.T) {
	// setup
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

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

	// set up msgs and fee
	var tx sdk.Tx
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr3, addr1)
	msg3 := newTestMsg(addr2, addr3)
	msgs := []sdk.Msg{msg1, msg2, msg3}
	fee := newStdFee()

	// signers in order
	privs, accnums, seqs := []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{0, 0, 0}
	tx = newTestTxWithMemo(ctx, msgs, privs, accnums, seqs, fee, "Check signers are in expected order and different account numbers works")

	checkValidTx(t, anteHandler, ctx, tx)

	// change sequence numbers
	tx = newTestTx(ctx, []sdk.Msg{msg1}, []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{1, 1}, fee)
	checkValidTx(t, anteHandler, ctx, tx)
	tx = newTestTx(ctx, []sdk.Msg{msg2}, []crypto.PrivKey{priv3, priv1}, []int64{2, 0}, []int64{1, 2}, fee)
	checkValidTx(t, anteHandler, ctx, tx)

	// expected seqs = [3, 2, 2]
	tx = newTestTxWithMemo(ctx, msgs, privs, accnums, []int64{3, 2, 2}, fee, "Check signers are in expected order and different account numbers and sequence numbers works")
	checkValidTx(t, anteHandler, ctx, tx)
}

func TestAnteHandlerBadSignBytes(t *testing.T) {
	// setup
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

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
	fee := newStdFee()
	fee2 := newStdFee()
	fee2.Gas += 100
	fee3 := newStdFee()
	fee3.Amount[0].Amount = fee3.Amount[0].Amount.AddRaw(100)

	// test good tx and signBytes
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkValidTx(t, anteHandler, ctx, txn)

	chainID := ctx.ChainID()
	chainID2 := chainID + "somemorestuff"
	codeUnauth := sdk.CodeUnauthorized

	cases := []struct {
		chainID string
		accnum  int64
		seq     int64
		fee     tx.StdFee
		msgs    []sdk.Msg
		code    sdk.CodeType
	}{
		{chainID2, 0, 1, fee, msgs, codeUnauth},                        // test wrong chain_id
		{chainID, 0, 2, fee, msgs, codeUnauth},                         // test wrong seqs
		{chainID, 1, 1, fee, msgs, codeUnauth},                         // test wrong accnum
		{chainID, 0, 1, fee, []sdk.Msg{newTestMsg(addr2)}, codeUnauth}, // test wrong msg
		{chainID, 0, 1, fee2, msgs, codeUnauth},                        // test wrong fee
		{chainID, 0, 1, fee3, msgs, codeUnauth},                        // test wrong fee
	}

	privs, seqs = []crypto.PrivKey{priv1}, []int64{1}
	for _, cs := range cases {
		txn := newTestTxWithSignBytes(

			msgs, privs, accnums, seqs, fee,
			tx.StdSignBytes(cs.chainID, cs.accnum, cs.seq, cs.fee, cs.msgs, ""),
			"",
		)
		checkInvalidTx(t, anteHandler, ctx, txn, cs.code)
	}

	// test wrong signer if public key exist
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{0}, []int64{1}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeUnauthorized)

	// test wrong signer if public doesn't exist
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv1}, []int64{1}, []int64{0}
	txn = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey)

}

func TestAnteHandlerSetPubKey(t *testing.T) {
	// setup
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper := auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler := tx.NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())

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
	fee := newStdFee()
	txn = newTestTx(ctx, msgs, privs, accnums, seqs, fee)
	checkValidTx(t, anteHandler, ctx, txn)

	acc1 = mapper.GetAccount(ctx, addr1)
	require.Equal(t, acc1.GetPubKey(), priv1.PubKey())

	// test public key not found
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	txn = newTestTx(ctx, msgs, privs, []int64{1}, seqs, fee)
	sigs := txn.(tx.StdTx).GetSignatures()
	sigs[0].PubKey = nil
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey)

	acc2 = mapper.GetAccount(ctx, addr2)
	require.Nil(t, acc2.GetPubKey())

	// test invalid signature and public key
	txn = newTestTx(ctx, msgs, privs, []int64{1}, seqs, fee)
	checkInvalidTx(t, anteHandler, ctx, txn, sdk.CodeInvalidPubKey)

	acc2 = mapper.GetAccount(ctx, addr2)
	require.Nil(t, acc2.GetPubKey())
}

func setup() (mapper auth.AccountKeeper, ctx sdk.Context, anteHandler sdk.AnteHandler) {
	ms, capKey, capKey2 := testutils.SetupMultiStoreForUnitTest()
	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)
	mapper = auth.NewAccountKeeper(cdc, capKey, auth.ProtoBaseAccount)
	anteHandler = tx.NewAnteHandler(mapper)
	ctx = sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, false, log.NewNopLogger())
	return
}

func runAnteHandlerWithMultiTxFees(ctx sdk.Context, anteHandler sdk.AnteHandler, priv crypto.PrivKey, addr sdk.AccAddress, fees ...tx.FeeCalculator) sdk.Context {
	for i := 0; i < len(fees); i++ {
		msg := newTestMsgWithFeeCalculator(fees[i], addr)
		txn := newTestTx(ctx, []sdk.Msg{msg}, []crypto.PrivKey{priv}, []int64{0}, []int64{int64(i)}, newStdFee())
		ctx, _, _ = anteHandler(ctx, txn)
	}

	return ctx
}

func checkBalance(t *testing.T, am auth.AccountKeeper, ctx sdk.Context, addr sdk.AccAddress, accNewBalance sdk.Coins) {
	newBalance := am.GetAccount(ctx, addr).GetCoins()
	require.Equal(t, accNewBalance, newBalance)
}

func checkFee(t *testing.T, ctx sdk.Context, expectFee types.Fee) {
	fee := tx.Fee(ctx)
	require.Equal(t, expectFee, fee)
}

// Test logic around fee deduction.
func TestAnteHandlerFeesInCheckTx(t *testing.T) {
	am, ctx, anteHandler := setup()
	// set the accounts
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)

	ctx = ctx.WithIsCheckTx(true)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), tx.FixedFeeCalculator(10, types.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 90)})
	checkFee(t, ctx, types.Fee{})
}

func TestAnteHandlerOneTxFee(t *testing.T) {
	// one tx, FeeFree
	am, ctx, anteHandler := setup()
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), tx.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 100)})
	checkFee(t, ctx, types.Fee{})

	// one tx, FeeForProposer
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), tx.FixedFeeCalculator(10, types.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 90)})
	checkFee(t, ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 10)}, types.FeeForProposer))

	// one tx, FeeForAll
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(), tx.FixedFeeCalculator(10, types.FeeForAll))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 90)})
	checkFee(t, ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 10)}, types.FeeForAll))
}

func TestAnteHandlerMultiTxFees(t *testing.T) {
	// two txs, 1. FeeFree 2. FeeProposer
	am, ctx, anteHandler := setup()
	priv1, acc1 := testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		tx.FreeFeeCalculator(),
		tx.FixedFeeCalculator(10, types.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 90)})
	checkFee(t, ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 10)}, types.FeeForProposer))

	// two txs, 1. FeeProposer 2. FeeFree
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		tx.FixedFeeCalculator(10, types.FeeForProposer),
		tx.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 90)})
	checkFee(t, ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 10)}, types.FeeForProposer))

	// two txs, 1. FeeProposer 2. FeeForAll
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		tx.FixedFeeCalculator(10, types.FeeForProposer),
		tx.FixedFeeCalculator(10, types.FeeForAll))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 80)})
	checkFee(t, ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 20)}, types.FeeForAll))

	// two txs, 1. FeeForAll 2. FeeProposer
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		tx.FixedFeeCalculator(10, types.FeeForAll),
		tx.FixedFeeCalculator(10, types.FeeForProposer))
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 80)})
	checkFee(t, ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 20)}, types.FeeForAll))

	// three txs, 1. FeeForAll 2. FeeProposer 3. FeeFree
	am, ctx, anteHandler = setup()
	priv1, acc1 = testutils.NewAccount(ctx, am, 100)
	ctx = runAnteHandlerWithMultiTxFees(ctx, anteHandler, priv1, acc1.GetAddress(),
		tx.FixedFeeCalculator(10, types.FeeForAll),
		tx.FixedFeeCalculator(10, types.FeeForProposer),
		tx.FreeFeeCalculator())
	checkBalance(t, am, ctx, acc1.GetAddress(), sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 80)})
	checkFee(t, ctx, types.NewFee(sdk.Coins{sdk.NewInt64Coin(types.NativeToken, 20)}, types.FeeForAll))
}
