package auth

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
)

func newTestMsg(addrs ...sdk.AccAddress) *sdk.TestMsg {
	return sdk.NewTestMsg(addrs...)
}

func newCoins() sdk.Coins {
	return sdk.Coins{
		sdk.NewCoin("atom", 10000000),
	}
}

// generate a priv key and return it with its address
func privAndAddr() (crypto.PrivKey, sdk.AccAddress) {
	priv := ed25519.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())
	return priv, addr
}

// run the tx through the anteHandler and ensure its valid
func checkValidTx(t *testing.T, anteHandler sdk.AnteHandler, ctx sdk.Context, tx sdk.Tx, mode sdk.RunTxMode) {
	_, result, abort := anteHandler(ctx, tx, mode)
	require.False(t, abort)
	require.Equal(t, sdk.ABCICodeOK, result.Code)
	require.True(t, result.IsOK())
}

// run the tx through the anteHandler and ensure it fails with the given code
func checkInvalidTx(t *testing.T, anteHandler sdk.AnteHandler, ctx sdk.Context, tx sdk.Tx, mode sdk.RunTxMode, code sdk.CodeType) {
	_, result, abort := anteHandler(ctx, tx, mode)
	require.True(t, abort)
	require.Equal(t, sdk.ToABCICode(sdk.CodespaceRoot, code), result.Code,
		fmt.Sprintf("Expected %v, got %v", sdk.ToABCICode(sdk.CodespaceRoot, code), result))
}

func newTestTx(ctx sdk.Context, msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64) sdk.Tx {
	sigs := make([]StdSignature, len(privs))
	for i, priv := range privs {
		signBytes := StdSignBytes(ctx.ChainID(), accNums[i], seqs[i], msgs, "", 0, nil)
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := NewStdTx(msgs, sigs, "", 0, nil)
	return tx
}

func newTestTxWithMemo(ctx sdk.Context, msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64, memo string) sdk.Tx {
	sigs := make([]StdSignature, len(privs))
	for i, priv := range privs {
		signBytes := StdSignBytes(ctx.ChainID(), accNums[i], seqs[i], msgs, memo, 0, nil)
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := NewStdTx(msgs, sigs, memo, 0, nil)
	return tx
}

// All signers sign over the same StdSignDoc. Should always create invalid signatures
func newTestTxWithSignBytes(msgs []sdk.Msg, privs []crypto.PrivKey, accNums []int64, seqs []int64, signBytes []byte, memo string) sdk.Tx {
	sigs := make([]StdSignature, len(privs))
	for i, priv := range privs {
		sig, err := priv.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i] = StdSignature{PubKey: priv.PubKey(), Signature: sig, AccountNumber: accNums[i], Sequence: seqs[i]}
	}
	tx := NewStdTx(msgs, sigs, memo, 0, nil)
	return tx
}

// Test various error cases in the AnteHandler control flow.
func TestAnteHandlerSigErrors(t *testing.T) {
	// setup
	ms, capKey, _ := setupMultiStore()
	cdc := codec.New()
	RegisterBaseAccount(cdc)
	mapper := NewAccountKeeper(cdc, capKey, ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	anteHandler := NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)

	// keys and addresses
	priv1, addr1 := privAndAddr()
	priv2, addr2 := privAndAddr()
	priv3, addr3 := privAndAddr()

	// msg and signatures
	var tx sdk.Tx
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr1, addr3)

	msgs := []sdk.Msg{msg1, msg2}

	// test no signatures
	privs, accNums, seqs := []crypto.PrivKey{}, []int64{}, []int64{}
	tx = newTestTx(ctx, msgs, privs, accNums, seqs)

	// tx.GetSigners returns addresses in correct order: addr1, addr2, addr3
	expectedSigners := []sdk.AccAddress{addr1, addr2, addr3}
	stdTx := tx.(StdTx)
	require.Equal(t, expectedSigners, stdTx.GetSigners())

	// Check no signatures fails
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeUnauthorized)

	// test num sigs dont match GetSigners
	privs, accNums, seqs = []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accNums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeUnauthorized)

	// test an unrecognized account
	privs, accNums, seqs = []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{0, 0, 0}
	tx = newTestTx(ctx, msgs, privs, accNums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeUnknownAddress)

	// save the first account, but second is still unrecognized
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	mapper.SetAccount(ctx, acc1)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeUnknownAddress)
}

// Test logic around account number checking with one signer and many signers.
func TestAnteHandlerAccountNumbers(t *testing.T) {
	// setup
	ms, capKey, _ := setupMultiStore()
	cdc := codec.New()
	RegisterBaseAccount(cdc)
	mapper := NewAccountKeeper(cdc, capKey, ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	anteHandler := NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	ctx = ctx.WithBlockHeight(1)

	// keys and addresses
	priv1, addr1 := privAndAddr()
	priv2, addr2 := privAndAddr()

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
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// new tx from wrong account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidSequence)

	// from correct account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{0}, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// new tx with another signer and incorrect account numbers
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr2, addr1)
	msgs = []sdk.Msg{msg1, msg2}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{1, 0}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidSequence)

	// correct account numbers
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)
}

// Test logic around account number checking with many signers when BlockHeight is 0.
func TestAnteHandlerAccountNumbersAtBlockHeightZero(t *testing.T) {
	// setup
	ms, capKey, _ := setupMultiStore()
	cdc := codec.New()
	RegisterBaseAccount(cdc)
	mapper := NewAccountKeeper(cdc, capKey, ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	anteHandler := NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	ctx = ctx.WithBlockHeight(0)

	// keys and addresses
	priv1, addr1 := privAndAddr()
	priv2, addr2 := privAndAddr()

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
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// new tx from wrong account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidSequence)

	// from correct account number
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, []int64{0}, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// new tx with another signer and incorrect account numbers
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr2, addr1)
	msgs = []sdk.Msg{msg1, msg2}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{1, 0}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidSequence)

	// correct account numbers
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 0}, []int64{2, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)
}

// Test logic around sequence checking with one signer and many signers.
func TestAnteHandlerSequences(t *testing.T) {
	// setup
	ms, capKey, _ := setupMultiStore()
	cdc := codec.New()
	RegisterBaseAccount(cdc)
	mapper := NewAccountKeeper(cdc, capKey, ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	anteHandler := NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	ctx = ctx.WithBlockHeight(1)

	// keys and addresses
	priv1, addr1 := privAndAddr()
	priv2, addr2 := privAndAddr()
	priv3, addr3 := privAndAddr()

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
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// test sending it again fails (replay protection)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidSequence)

	// fix sequence, should pass
	seqs = []int64{1}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// new tx with another signer and correct sequences
	msg1 := newTestMsg(addr1, addr2)
	msg2 := newTestMsg(addr3, addr1)
	msgs = []sdk.Msg{msg1, msg2}

	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2, priv3}, []int64{0, 1, 2}, []int64{2, 0, 0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// replay fails
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidSequence)

	// tx from just second signer with incorrect sequence fails
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{1}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidSequence)

	// fix the sequence and it passes
	tx = newTestTx(ctx, msgs, []crypto.PrivKey{priv2}, []int64{1}, []int64{1})
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// another tx from both of them that passes
	msg = newTestMsg(addr1, addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{3, 2}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)
}

func TestAnteHandlerMultiSigner(t *testing.T) {
	// setup
	ms, capKey, _ := setupMultiStore()
	cdc := codec.New()
	RegisterBaseAccount(cdc)
	mapper := NewAccountKeeper(cdc, capKey, ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	anteHandler := NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	ctx = ctx.WithBlockHeight(1)

	// keys and addresses
	priv1, addr1 := privAndAddr()
	priv2, addr2 := privAndAddr()
	priv3, addr3 := privAndAddr()

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

	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// change sequence numbers
	tx = newTestTx(ctx, []sdk.Msg{msg1}, []crypto.PrivKey{priv1, priv2}, []int64{0, 1}, []int64{1, 1})
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)
	tx = newTestTx(ctx, []sdk.Msg{msg2}, []crypto.PrivKey{priv3, priv1}, []int64{2, 0}, []int64{1, 2})
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	// expected seqs = [3, 2, 2]
	tx = newTestTxWithMemo(ctx, msgs, privs, accnums, []int64{3, 2, 2}, "Check signers are in expected order and different account numbers and sequence numbers works")
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)
}

func TestAnteHandlerBadSignBytes(t *testing.T) {
	// setup
	ms, capKey, _ := setupMultiStore()
	cdc := codec.New()
	RegisterBaseAccount(cdc)
	mapper := NewAccountKeeper(cdc, capKey, ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	anteHandler := NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	ctx = ctx.WithBlockHeight(1)

	// keys and addresses
	priv1, addr1 := privAndAddr()
	priv2, addr2 := privAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)
	acc2 := mapper.NewAccountWithAddress(ctx, addr2)
	acc2.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc2)

	var tx sdk.Tx
	msg := newTestMsg(addr1)
	msgs := []sdk.Msg{msg}

	// test good tx and signBytes
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

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
		tx := newTestTxWithSignBytes(

			msgs, privs, accnums, seqs,
			StdSignBytes(cs.chainID, cs.accnum, cs.seq, cs.msgs, "", 0, nil),
			"",
		)
		checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, cs.code)
	}

	// test wrong signer if public key exist
	privs, accnums, seqs = []crypto.PrivKey{priv2}, []int64{0}, []int64{1}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeUnauthorized)

	// test wrong signer if public doesn't exist
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	privs, accnums, seqs = []crypto.PrivKey{priv1}, []int64{1}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidPubKey)

}

func TestAnteHandlerSetPubKey(t *testing.T) {
	// setup
	ms, capKey, _ := setupMultiStore()
	cdc := codec.New()
	RegisterBaseAccount(cdc)
	mapper := NewAccountKeeper(cdc, capKey, ProtoBaseAccount)
	accountCache := getAccountCache(cdc, ms, capKey)
	anteHandler := NewAnteHandler(mapper)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	ctx = ctx.WithBlockHeight(1)

	// keys and addresses
	priv1, addr1 := privAndAddr()
	_, addr2 := privAndAddr()

	// set the accounts
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	acc1.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc1)
	acc2 := mapper.NewAccountWithAddress(ctx, addr2)
	acc2.SetCoins(newCoins())
	mapper.SetAccount(ctx, acc2)

	var tx sdk.Tx

	// test good tx and set public key
	msg := newTestMsg(addr1)
	msgs := []sdk.Msg{msg}
	privs, accnums, seqs := []crypto.PrivKey{priv1}, []int64{0}, []int64{0}
	tx = newTestTx(ctx, msgs, privs, accnums, seqs)
	checkValidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver)

	acc1 = mapper.GetAccount(ctx, addr1)
	require.Equal(t, acc1.GetPubKey(), priv1.PubKey())

	// test public key not found
	msg = newTestMsg(addr2)
	msgs = []sdk.Msg{msg}
	tx = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	sigs := tx.(StdTx).GetSignatures()
	sigs[0].PubKey = nil
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidPubKey)

	acc2 = mapper.GetAccount(ctx, addr2)
	require.Nil(t, acc2.GetPubKey())

	// test invalid signature and public key
	tx = newTestTx(ctx, msgs, privs, []int64{1}, seqs)
	checkInvalidTx(t, anteHandler, ctx, tx, sdk.RunTxModeDeliver, sdk.CodeInvalidPubKey)

	acc2 = mapper.GetAccount(ctx, addr2)
	require.Nil(t, acc2.GetPubKey())
}

func TestProcessPubKey(t *testing.T) {
	ms, capKey, _ := setupMultiStore()
	cdc := codec.New()
	RegisterBaseAccount(cdc)
	accountCache := getAccountCache(cdc, ms, capKey)
	mapper := NewAccountKeeper(cdc, capKey, ProtoBaseAccount)
	ctx := sdk.NewContext(ms, abci.Header{ChainID: "mychainid"}, sdk.RunTxModeDeliver, log.NewNopLogger()).WithAccountCache(accountCache)
	// keys
	_, addr1 := privAndAddr()
	priv2, _ := privAndAddr()
	acc1 := mapper.NewAccountWithAddress(ctx, addr1)
	type args struct {
		acc      sdk.Account
		sig      StdSignature
		simulate bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"no sigs, simulate off", args{acc1, StdSignature{}, false}, true},
		{"no sigs, simulate on", args{acc1, StdSignature{}, true}, false},
		{"pubkey doesn't match addr, simulate off", args{acc1, StdSignature{PubKey: priv2.PubKey()}, false}, true},
		{"pubkey doesn't match addr, simulate on", args{acc1, StdSignature{PubKey: priv2.PubKey()}, true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := processPubKey(tt.args.acc, tt.args.sig, tt.args.simulate)
			require.Equal(t, tt.wantErr, !err.IsOK())
		})
	}
}
