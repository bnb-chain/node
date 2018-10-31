package tx

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/common/types"
)

const (
	deductFeesCost    sdk.Gas = 10
	memoCostPerByte   sdk.Gas = 1
	verifyCost                = 100
	maxMemoCharacters         = 100
)

// NewAnteHandler returns an AnteHandler that checks
// and increments sequence numbers, checks signatures & account numbers,
// and deducts fees from the first signer.
// NOTE: Receiving the `NewOrder` dependency here avoids an import cycle.
// nolint: gocyclo
// TODO: remove gas
func NewAnteHandler(am auth.AccountMapper, orderMsgType string) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx,
	) (newCtx sdk.Context, res sdk.Result, abort bool) {

		// This AnteHandler requires Txs to be StdTxs
		stdTx, ok := tx.(StdTx)
		if !ok {
			return ctx, sdk.ErrInternal("tx must be StdTx").Result(), true
		}

		// set the gas meter
		newCtx = ctx.WithGasMeter(sdk.NewGasMeter(stdTx.Fee.Gas))

		// AnteHandlers must have their own defer/recover in order
		// for the BaseApp to know how much gas was used!
		// This is because the GasMeter is created in the AnteHandler,
		// but if it panics the context won't be set properly in runTx's recover ...
		defer func() {
			if r := recover(); r != nil {
				switch rType := r.(type) {
				case sdk.ErrorOutOfGas:
					log := fmt.Sprintf("out of gas in location: %v", rType.Descriptor)
					res = sdk.ErrOutOfGas(log).Result()
					res.GasWanted = stdTx.Fee.Gas
					res.GasUsed = newCtx.GasMeter().GasConsumed()
					abort = true
				default:
					panic(r)
				}
			}
		}()

		err := validateBasic(stdTx)
		if err != nil {
			return newCtx, err.Result(), true
		}

		sigs := stdTx.GetSignatures()
		signerAddrs := stdTx.GetSigners()
		msgs := tx.GetMsgs()

		// charge gas for the memo
		newCtx.GasMeter().ConsumeGas(memoCostPerByte*sdk.Gas(len(stdTx.GetMemo())), "memo")

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
		for i := 0; i < len(sigs); i++ {
			signerAddr := signerAddrs[i]
			signerAcc := am.GetAccount(ctx, signerAddr)
			signerAccs[i] = signerAcc
		}

		// TODO: order ID validation hoisted here temporarily for now - ensures sequence is always in sync with msg
		// avoid importing NewOrderMsg, which causes an import cycle :(
		if msgs[0].Type() == orderMsgType {
			expectedOrderID := fmt.Sprintf("\"%X-%d\"", signerAccs[0].GetAddress(), sequences[0])
			if !strings.Contains(string(msgs[0].GetSignBytes()), expectedOrderID) {
				log.Info("Invalid order ID encountered", "signbytes", string(msgs[0].GetSignBytes()))
				return newCtx, sdk.Result{
					Code: sdk.ToABCICode(sdk.CodespaceRoot, sdk.CodeUnknownRequest),
					Log:  fmt.Sprintf("Invalid order ID provided, expected `%s`.", expectedOrderID),
					Data: []byte("Unexpected order ID encountered"),
				}, true
			}
		}

		// check sigs and nonce
		for i := 0; i < len(sigs); i++ {
			signerAddr, sig := signerAddrs[i], sigs[i]

			// check signature, return account with incremented nonce
			signBytes := StdSignBytes(newCtx.ChainID(), accNums[i], sequences[i], stdTx.Fee, msgs, stdTx.GetMemo())
			signerAcc, res := processSig(newCtx, am, signerAddr, sig, signBytes)
			if !res.IsOK() {
				return newCtx, res, true
			}

			// Save the account.
			am.SetAccount(newCtx, signerAcc)
			signerAccs[i] = signerAcc
		}

		newCtx, res = calcAndCollectFees(newCtx, am, signerAccs[0], msgs[0])
		if !res.IsOK() {
			return newCtx, res, true
		}

		// cache the signer accounts in the context
		newCtx = auth.WithSigners(newCtx, signerAccs)

		// TODO: tx tags (?)
		return newCtx, sdk.Result{GasWanted: stdTx.Fee.Gas}, false // continue...
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

// verify the signature and increment the sequence.
// if the account doesn't have a pubkey, set it.
func processSig(
	ctx sdk.Context, am auth.AccountMapper,
	addr sdk.AccAddress, sig StdSignature, signBytes []byte) (
	acc auth.Account, res sdk.Result) {

	// Get the account.
	acc = am.GetAccount(ctx, addr)
	if acc == nil {
		return nil, sdk.ErrUnknownAddress(addr.String()).Result()
	}

	// Check account number.
	accnum := acc.GetAccountNumber()
	if accnum != sig.AccountNumber {
		return nil, sdk.ErrInvalidSequence(
			fmt.Sprintf("Invalid account number. Got %d, expected %d", sig.AccountNumber, accnum)).Result()
	}

	// Check and increment sequence number.
	seq := acc.GetSequence()
	if seq != sig.Sequence {
		return nil, sdk.ErrInvalidSequence(
			fmt.Sprintf("Invalid sequence. Got %d, expected %d", sig.Sequence, seq)).Result()
	}
	err := acc.SetSequence(seq + 1)
	if err != nil {
		// Handle w/ #870
		panic(err)
	}
	// If pubkey is not known for account,
	// set it from the StdSignature.
	pubKey := acc.GetPubKey()
	if pubKey == nil {
		pubKey = sig.PubKey
		if pubKey == nil {
			return nil, sdk.ErrInvalidPubKey("PubKey not found").Result()
		}
		if !bytes.Equal(pubKey.Address(), addr) {
			return nil, sdk.ErrInvalidPubKey(
				fmt.Sprintf("PubKey does not match Signer address %v", addr)).Result()
		}
		err = acc.SetPubKey(pubKey)
		if err != nil {
			return nil, sdk.ErrInternal("setting PubKey on signer's account").Result()
		}
	}

	// Check sig.
	ctx.GasMeter().ConsumeGas(verifyCost, "ante verify")
	if !pubKey.VerifyBytes(signBytes, sig.Signature) {
		return nil, sdk.ErrUnauthorized("signature verification failed").Result()
	}

	return
}

func calcAndCollectFees(ctx sdk.Context, am auth.AccountMapper, acc auth.Account, msg sdk.Msg) (sdk.Context, sdk.Result) {
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

	// record fees in ctx.
	totalFee := Fee(ctx)
	totalFee.AddFee(fee)
	ctx = WithFee(ctx, totalFee)
	return ctx, sdk.Result{}
}

func calculateFees(msg sdk.Msg) (types.Fee, error) {
	calculator := GetCalculator(msg.Type())
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

func deductFees(ctx sdk.Context, acc auth.Account, fee types.Fee, am auth.AccountMapper) sdk.Result {
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

// BurnFeeHandler burns all fees (decreasing total supply)
func BurnFeeHandler(_ sdk.Context, _ sdk.Tx, _ sdk.Coins) {}
