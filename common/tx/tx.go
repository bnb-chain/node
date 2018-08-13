package tx

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/wire"
)

var msgCdc = wire.NewCodec()

var _ Tx = (*StdTx)(nil)

// StdTx is a standard way to wrap a Msg with Fee and Signatures.
// NOTE: the first signature is the FeePayer (Signatures must not be nil).
type StdTx struct {
	Msg        Msg            `json:"msg"`
	Fee        StdFee         `json:"fee"`
	Signatures []StdSignature `json:"signatures"`
	Memo       string         `json:"memo"`
}

func NewStdTx(msg Msg, fee StdFee, sigs []StdSignature, memo string) StdTx {
	return StdTx{
		Msg:        msg,
		Fee:        fee,
		Signatures: sigs,
		Memo:       memo,
	}
}

//nolint
func (tx StdTx) GetMsg() Msg { return tx.Msg }

// GetSigners returns the addresses that must sign the transaction.
// Addresses are returned in a determistic order.
// They are accumulated from the GetSigners method for each Msg
// in the order they appear in tx.GetMsg().
// Duplicate addresses will be omitted.
func (tx StdTx) GetSigners() []sdk.AccAddress {
	seen := map[string]bool{}
	var signers []sdk.AccAddress
	for _, addr := range tx.Msg.GetSigners() {
		if !seen[addr.String()] {
			signers = append(signers, addr)
			seen[addr.String()] = true
		}
	}
	return signers
}

//nolint
func (tx StdTx) GetMemo() string { return tx.Memo }

// Signatures returns the signature of signers who signed the Msg.
// GetSignatures returns the signature of signers who signed the Msg.
// CONTRACT: Length returned is same as length of
// pubkeys returned from MsgKeySigners, and the order
// matches.
// CONTRACT: If the signature is missing (ie the Msg is
// invalid), then the corresponding signature is
// .Empty().
func (tx StdTx) GetSignatures() []StdSignature { return tx.Signatures }

// FeePayer returns the address responsible for paying the fees
// for the transactions. It's the first address returned by msg.GetSigners().
// If GetSigners() is empty, this panics.
func FeePayer(tx Tx) sdk.AccAddress {
	return tx.GetMsg().GetSigners()[0]
}

//__________________________________________________________

// StdFee includes the amount of coins paid in fees and the maximum
// gas to be used by the transaction. The ratio yields an effective "gasprice",
// which must be above some miminum to be accepted into the mempool.
type StdFee = auth.StdFee

// NewStdFee constructs a new std fee
func NewStdFee(gas int64, amount ...sdk.Coin) StdFee {
	return StdFee{
		Amount: amount,
		Gas:    gas,
	}
}

//__________________________________________________________

// StdSignDoc is replay-prevention structure.
// It includes the result of msg.GetSignBytes(),
// as well as the ChainID (prevent cross chain replay)
// and the Sequence numbers for each signature (prevent
// inchain replay and enforce tx ordering per account).
type StdSignDoc struct {
	AccountNumber int64           `json:"account_number"`
	ChainID       string          `json:"chain_id"`
	Fee           json.RawMessage `json:"fee"`
	Memo          string          `json:"memo"`
	Msg           json.RawMessage `json:"msg"`
	Sequence      int64           `json:"sequence"`
}

// StdSignBytes returns the bytes to sign for a transaction.
func StdSignBytes(chainID string, accnum int64, sequence int64, fee StdFee, msg Msg, memo string) []byte {
	bz, err := msgCdc.MarshalJSON(StdSignDoc{
		AccountNumber: accnum,
		ChainID:       chainID,
		Fee:           json.RawMessage(fee.Bytes()),
		Memo:          memo,
		Msg:           msg.GetSignBytes(),
		Sequence:      sequence,
	})
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

// StdSignMsg is a convenience structure for passing along
// a Msg with the other requirements for a StdSignDoc before
// it is signed. For use in the CLI.
type StdSignMsg struct {
	ChainID       string
	AccountNumber int64
	Sequence      int64
	Fee           StdFee
	Msg           Msg
	Memo          string
}

// Bytes gets message bytes
func (msg StdSignMsg) Bytes() []byte {
	return StdSignBytes(msg.ChainID, msg.AccountNumber, msg.Sequence, msg.Fee, msg.Msg, msg.Memo)
}

// StdSignature is a standard signature
type StdSignature = auth.StdSignature

// DefaultTxDecoder houses the logic for standard transaction decoding
func DefaultTxDecoder(cdc *wire.Codec) TxDecoder {
	return func(txBytes []byte) (Tx, sdk.Error) {
		var tx = StdTx{}

		if len(txBytes) == 0 {
			return nil, sdk.ErrTxDecode("txBytes are empty")
		}

		// StdTx.Msg is an interface. The concrete types
		// are registered by MakeTxCodec
		err := cdc.UnmarshalBinary(txBytes, &tx)
		if err != nil {
			return nil, sdk.ErrTxDecode("").TraceSDK(err.Error())
		}
		return tx, nil
	}
}
