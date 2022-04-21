package auth

import (
	"bytes"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

const (
	ed25519VerifyCost   = 59
	secp256k1VerifyCost = 100
	maxMemoCharacters   = 100
)

// NewAnteHandler returns an AnteHandler that checks
// and increments sequence numbers, checks signatures & account numbers
func NewAnteHandler(am AccountKeeper) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, mode sdk.RunTxMode,
	) (newCtx sdk.Context, res sdk.Result, abort bool) {
		newCtx = ctx
		// This AnteHandler requires Txs to be StdTxs
		stdTx, ok := tx.(StdTx)
		if !ok {
			return ctx, sdk.ErrInternal("tx must be StdTx").Result(), true
		}

		// AnteHandlers must have their own defer/recover in order
		defer func() {
			if r := recover(); r != nil {
				panic(r)
			}
		}()
		if mode != sdk.RunTxModeReCheck {
			err := validateBasic(stdTx)
			if err != nil {
				return newCtx, err.Result(), true
			}
		}

		// stdSigs contains the sequence number, account number, and signatures
		stdSigs := stdTx.GetSignatures() // When simulating, this would just be a 0-length slice.
		signerAddrs := stdTx.GetSigners()

		signerAccs, res := getSignerAccs(newCtx, am, signerAddrs)
		if !res.IsOK() {
			return newCtx, res, true
		}
		res = validateAccNumAndSequence(ctx, signerAccs, stdSigs)
		if !res.IsOK() {
			return newCtx, res, true
		}

		var signBytesList [][]byte

		if mode != sdk.RunTxModeReCheck {
			// create the list of all sign bytes
			signBytesList = getSignBytesList(newCtx.ChainID(), stdTx, stdSigs)
		}

		for i := 0; i < len(stdSigs); i++ {
			// check signature, return account with incremented nonce
			var signBytes []byte
			if mode != sdk.RunTxModeReCheck {
				signBytes = signBytesList[i]
			} else {
				signBytes = nil
			}
			signerAccs[i], res = processSig(newCtx, signerAccs[i],
				stdSigs[i], signBytes, mode)
			if !res.IsOK() {
				return newCtx, res, true
			}

			// Save the account.
			am.SetAccount(newCtx, signerAccs[i])
		}

		// cache the signer accounts in the context
		newCtx = WithSigners(newCtx, signerAccs)

		// TODO: tx tags (?)
		return newCtx, sdk.Result{}, false // continue...
	}
}

// Validate the transaction based on things that don't depend on the context
func validateBasic(tx StdTx) (err sdk.Error) {
	// Assert that there are signatures.
	sigs := tx.GetSignatures()
	if len(sigs) == 0 {
		return sdk.ErrUnauthorized("no signers")
	}

	// Assert that number of signatures is correct.
	var signerAddrs = tx.GetSigners()
	if len(sigs) != len(signerAddrs) {
		return sdk.ErrUnauthorized("wrong number of signers")
	}

	memo := tx.GetMemo()
	if len(memo) > maxMemoCharacters {
		return sdk.ErrMemoTooLarge(
			fmt.Sprintf("maximum number of characters is %d but received %d characters",
				maxMemoCharacters, len(memo)))
	}
	return nil
}

func getSignerAccs(ctx sdk.Context, am AccountKeeper, addrs []sdk.AccAddress) (accs []sdk.Account, res sdk.Result) {
	accs = make([]sdk.Account, len(addrs))
	for i := 0; i < len(accs); i++ {
		accs[i] = am.GetAccount(ctx, addrs[i])
		if accs[i] == nil {
			return nil, sdk.ErrUnknownAddress(addrs[i].String()).Result()
		}
	}
	return
}

func validateAccNumAndSequence(ctx sdk.Context, accs []sdk.Account, sigs []StdSignature) sdk.Result {
	for i := 0; i < len(accs); i++ {
		// On InitChain, make sure account number == 0
		if ctx.BlockHeight() == 0 && sigs[i].AccountNumber != 0 {
			return sdk.ErrInvalidSequence(
				fmt.Sprintf("Invalid account number for BlockHeight == 0. Got %d, expected 0", sigs[i].AccountNumber)).Result()
		}

		// Check account number.
		accnum := accs[i].GetAccountNumber()
		if ctx.BlockHeight() != 0 && accnum != sigs[i].AccountNumber {
			return sdk.ErrInvalidSequence(
				fmt.Sprintf("Invalid account number. Got %d, expected %d", sigs[i].AccountNumber, accnum)).Result()
		}

		// Check sequence number.
		seq := accs[i].GetSequence()
		if seq != sigs[i].Sequence {
			return sdk.ErrInvalidSequence(
				fmt.Sprintf("Invalid sequence. Got %d, expected %d", sigs[i].Sequence, seq)).Result()
		}
	}
	return sdk.Result{}
}

// verify the signature and increment the sequence.
// if the account doesn't have a pubkey, set it.
func processSig(ctx sdk.Context,
	acc sdk.Account, sig StdSignature, signBytes []byte, mode sdk.RunTxMode) (updatedAcc sdk.Account, res sdk.Result) {
	pubKey, res := processPubKey(acc, sig, mode == sdk.RunTxModeSimulate)
	if !res.IsOK() {
		return nil, res
	}

	err := acc.SetPubKey(pubKey)
	if err != nil {
		return nil, sdk.ErrInternal("setting PubKey on signer's account").Result()
	}
	if (mode == sdk.RunTxModeCheck || mode == sdk.RunTxModeDeliver) && !pubKey.VerifyBytes(signBytes, sig.Signature) {
		return nil, sdk.ErrUnauthorized("signature verification failed").Result()
	}
	// increment the sequence number
	err = acc.SetSequence(acc.GetSequence() + 1)
	if err != nil {
		// Handle w/ #870
		panic(err)
	}

	return acc, res
}

var dummySecp256k1Pubkey secp256k1.PubKeySecp256k1

func init() {
	bz, _ := hex.DecodeString("035AD6810A47F073553FF30D2FCC7E0D3B1C0B74B61A1AAA2582344037151E143A")
	copy(dummySecp256k1Pubkey[:], bz)
}

func processPubKey(acc sdk.Account, sig StdSignature, simulate bool) (crypto.PubKey, sdk.Result) {
	// If pubkey is not known for account,
	// set it from the StdSignature.
	pubKey := acc.GetPubKey()
	if simulate {
		// In simulate mode the transaction comes with no signatures, thus
		// if the account's pubkey is nil, signature verification.
		if pubKey == nil {
			return dummySecp256k1Pubkey, sdk.Result{}
		}
		return pubKey, sdk.Result{}
	}
	if pubKey == nil {
		pubKey = sig.PubKey
		if pubKey == nil {
			return nil, sdk.ErrInvalidPubKey("PubKey not found").Result()
		}
		if !bytes.Equal(pubKey.Address(), acc.GetAddress()) {
			return nil, sdk.ErrInvalidPubKey(
				fmt.Sprintf("PubKey does not match Signer address %v", acc.GetAddress())).Result()
		}
	}
	return pubKey, sdk.Result{}
}

func getSignBytesList(chainID string, stdTx StdTx, stdSigs []StdSignature) (signatureBytesList [][]byte) {
	signatureBytesList = make([][]byte, len(stdSigs))
	for i := 0; i < len(stdSigs); i++ {
		signatureBytesList[i] = StdSignBytes(chainID,
			stdSigs[i].AccountNumber, stdSigs[i].Sequence,
			stdTx.Msgs, stdTx.Memo, stdTx.Source, stdTx.Data)
	}
	return
}
