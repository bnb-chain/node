package tx

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common/fees"
	"github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/common/types"
)

const (
	maxMemoCharacters = 100

	maxCacheNumber = 30000
)

type sigLRUCache struct {
	*lru.Cache
}

func newSigLRUCache(cap int) *sigLRUCache {
	cache, err := lru.New(cap)
	if err != nil {
		panic(err)
	}

	return &sigLRUCache{
		cache,
	}
}

func (cache *sigLRUCache) getSig(txHash string) (ok bool) {
	_, ok = cache.Get(txHash)
	return ok
}

func (cache *sigLRUCache) addSig(txHash string) {
	if txHash != "" {
		cache.Add(txHash, true)
	}
}

// signature-key: txHash
// based on the assumption that tx hash will never collide.
var sigCache = newSigLRUCache(maxCacheNumber)

// NewAnteHandler returns an AnteHandler that checks
// and increments sequence numbers, checks signatures & account numbers,
// and deducts fees from the first signer.
// NOTE: Receiving the `NewOrder` dependency here avoids an import cycle.
// nolint: gocyclo
func NewAnteHandler(am auth.AccountKeeper) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, mode sdk.RunTxMode,
	) (newCtx sdk.Context, res sdk.Result, abort bool) {
		newCtx = ctx
		// This AnteHandler requires Txs to be StdTxs
		stdTx, ok := tx.(auth.StdTx)
		if !ok {
			return newCtx, sdk.ErrInternal("tx must be StdTx").Result(), true
		}

		if mode != sdk.RunTxModeReCheck {
			err := validateBasic(stdTx)
			if err != nil {
				return newCtx, err.Result(), true
			}
		}

		sigs := stdTx.GetSignatures()
		signerAddrs := stdTx.GetSigners()
		msgs := tx.GetMsgs()

		// get the sign bytes (requires all account & sequence numbers and the fee)
		sequences := make([]int64, len(sigs))
		accNums := make([]int64, len(sigs))
		for i := 0; i < len(sigs); i++ {
			sequences[i] = sigs[i].Sequence
			accNums[i] = sigs[i].AccountNumber
		}

		// collect signer accounts
		// TODO: abort if there is more than one signer?
		var signerAccs = make([]auth.Account, len(signerAddrs))
		// check sigs and nonce
		for i := 0; i < len(sigs); i++ {
			signerAddr, sig := signerAddrs[i], sigs[i]
			signerAcc, err := processAccount(newCtx, am, signerAddr, sig)
			if err != nil {
				return newCtx, err.Result(), true
			}

			if mode != sdk.RunTxModeReCheck {
				// check signature, return account with incremented nonce
				signBytes := auth.StdSignBytes(ctx.ChainID(), accNums[i], sequences[i], msgs, stdTx.GetMemo())
				txHash, _ := ctx.Value(baseapp.TxHashKey).(string)

				res := processSig(txHash, sig, signerAcc, signBytes)
				if !res.IsOK() {
					return newCtx, res, true
				}
			}

			// Save the account.
			am.SetAccount(newCtx, signerAcc)
			signerAccs[i] = signerAcc
		}

		// for blockHeight == 0, we do not collect fees since we have some StdTx(s) in InitChain.
		if newCtx.BlockHeight() != 0 {
			newCtx, res = calcAndCollectFees(newCtx, am, signerAccs[0], msgs[0])
			if !res.IsOK() {
				return newCtx, res, true
			}
		}

		// cache the signer accounts in the context
		newCtx = auth.WithSigners(newCtx, signerAccs)

		// TODO: tx tags (?)
		return newCtx, sdk.Result{}, false // continue...
	}
}

// Validate the transaction based on things that don't depend on the context
func validateBasic(tx auth.StdTx) (err sdk.Error) {
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

func processAccount(ctx sdk.Context, am auth.AccountKeeper,
	addr sdk.AccAddress, sig auth.StdSignature) (acc auth.Account, err sdk.Error) {
	// Get the account.
	acc = am.GetAccount(ctx, addr)
	if acc == nil {
		return nil, sdk.ErrUnknownAddress(addr.String())
	}

	// On InitChain, make sure account number == 0
	if ctx.BlockHeight() == 0 {
		if sig.AccountNumber != 0 {
			return nil, sdk.ErrInvalidSequence(
				fmt.Sprintf("Invalid account number for BlockHeight == 0. Got %d, expected 0", sig.AccountNumber))
		}
	} else {
		// Check account number.
		accnum := acc.GetAccountNumber()
		if accnum != sig.AccountNumber {
			return nil, sdk.ErrInvalidSequence(
				fmt.Sprintf("Invalid account number. Got %d, expected %d", sig.AccountNumber, accnum))
		}
	}

	// Check and increment sequence number.
	seq := acc.GetSequence()
	if seq != sig.Sequence {
		return nil, sdk.ErrInvalidSequence(
			fmt.Sprintf("Invalid sequence. Got %d, expected %d", sig.Sequence, seq))
	}
	errSeq := acc.SetSequence(seq + 1)
	if errSeq != nil {
		// Handle w/ #870
		panic(err)
	}
	// If pubkey is not known for account,
	// set it from the StdSignature.
	pubKey := acc.GetPubKey()
	if pubKey == nil {
		pubKey = sig.PubKey
		if pubKey == nil {
			return nil, sdk.ErrInvalidPubKey("PubKey not found")
		}
		if !bytes.Equal(pubKey.Address(), addr) {
			return nil, sdk.ErrInvalidPubKey(
				fmt.Sprintf("PubKey does not match Signer address %v", addr))
		}
		errKey := acc.SetPubKey(pubKey)
		if errKey != nil {
			return nil, sdk.ErrInternal("setting PubKey on signer's account")
		}
	}
	return acc, nil
}

// verify the signature and increment the sequence.
// if the account doesn't have a pubkey, set it.
func processSig(txHash string,
	sig auth.StdSignature, acc auth.Account, signBytes []byte) (
	res sdk.Result) {

	if sigCache.getSig(txHash) {
		log.Debug("Tx hits sig cache", "txHash", txHash)
		return
	}

	pubKey := acc.GetPubKey()
	// Check sig.
	if !pubKey.VerifyBytes(signBytes, sig.Signature) {
		return sdk.ErrUnauthorized("signature verification failed").Result()
	}

	sigCache.addSig(txHash)
	return
}

func calcAndCollectFees(ctx sdk.Context, am auth.AccountKeeper, acc auth.Account, msg sdk.Msg) (sdk.Context, sdk.Result) {
	// first sig pays the fees
	// TODO: Add min fees
	// Can this function be moved outside of the loop?

	fee, err := calculateFees(msg)
	if err != nil {
		panic(err)
	}

	if fee.Type == types.FeeFree || fee.Tokens.IsZero() {
		return ctx, sdk.Result{}
	}

	fee.Tokens.Sort()
	res := deductFees(ctx, acc, fee, am)
	if !res.IsOK() {
		return ctx, res
	}

	if ctx.IsCheckTx() {
		return ctx, res
	}

	// add fee to pool
	fees.Pool.AddFee(fee)
	return ctx, sdk.Result{}
}

func calculateFees(msg sdk.Msg) (types.Fee, error) {
	calculator := fees.GetCalculator(msg.Type())
	if calculator == nil {
		return types.Fee{}, errors.New("missing calculator for msgType:" + msg.Type())
	}
	return calculator(msg), nil
}

func checkSufficientFunds(acc auth.Account, fee types.Fee) sdk.Result {
	coins := acc.GetCoins()

	newCoins := coins.Minus(fee.Tokens.Sort())
	if !newCoins.IsNotNegative() {
		errMsg := fmt.Sprintf("%s < %s", coins, fee.Tokens)
		return sdk.ErrInsufficientFunds(errMsg).Result()
	}

	return sdk.Result{}
}

func deductFees(ctx sdk.Context, acc auth.Account, fee types.Fee, am auth.AccountKeeper) sdk.Result {
	if res := checkSufficientFunds(acc, fee); !res.IsOK() {
		return res
	}

	newCoins := acc.GetCoins().Minus(fee.Tokens.Sort())
	err := acc.SetCoins(newCoins)
	if err != nil {
		// Handle w/ #870
		panic(err)
	}

	am.SetAccount(ctx, acc)
	return sdk.Result{}
}
