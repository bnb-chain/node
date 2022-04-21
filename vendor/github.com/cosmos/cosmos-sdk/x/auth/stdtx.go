package auth

import (
	"encoding/json"

	"github.com/tendermint/tendermint/crypto"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultSource = 0
)

var _ sdk.Tx = (*StdTx)(nil)

// StdTx is a standard way to wrap a Msg with Signatures.
type StdTx struct {
	Msgs       []sdk.Msg      `json:"msg"`
	Signatures []StdSignature `json:"signatures"`
	Memo       string         `json:"memo"`
	Source     int64          `json:"source"`
	Data       []byte         `json:"data"`
}

func NewStdTx(msgs []sdk.Msg, sigs []StdSignature, memo string, source int64, data []byte) StdTx {
	return StdTx{
		Msgs:       msgs,
		Signatures: sigs,
		Memo:       memo,
		Source:     source,
		Data:       data,
	}
}

//nolint
func (tx StdTx) GetMsgs() []sdk.Msg { return tx.Msgs }

// GetSigners returns the addresses that must sign the transaction.
// Addresses are returned in a determistic order.
// They are accumulated from the GetSigners method for each Msg
// in the order they appear in tx.GetMsgs().
// Duplicate addresses will be omitted.
func (tx StdTx) GetSigners() []sdk.AccAddress {
	seen := map[string]bool{}
	var signers []sdk.AccAddress
	for _, msg := range tx.GetMsgs() {
		for _, addr := range msg.GetSigners() {
			addrStr := string(addr.Bytes())
			if !seen[addrStr] {
				signers = append(signers, addr)
				seen[addrStr] = true
			}
		}
	}
	return signers
}

//nolint
func (tx StdTx) GetMemo() string { return tx.Memo }

//nolint
func (tx StdTx) GetSource() int64 { return tx.Source }

//nolint
func (tx StdTx) GetData() []byte { return tx.Data }

// Signatures returns the signature of signers who signed the Msg.
// GetSignatures returns the signature of signers who signed the Msg.
// CONTRACT: Length returned is same as length of
// pubkeys returned from MsgKeySigners, and the order
// matches.
// CONTRACT: If the signature is missing (ie the Msg is
// invalid), then the corresponding signature is
// .Empty().
func (tx StdTx) GetSignatures() []StdSignature { return tx.Signatures }

//__________________________________________________________

// StdSignDoc is replay-prevention structure.
// It includes the result of msg.GetSignBytes(),
// as well as the ChainID (prevent cross chain replay)
// and the Sequence numbers for each signature (prevent
// inchain replay and enforce tx ordering per account).
type StdSignDoc struct {
	AccountNumber int64             `json:"account_number"`
	ChainID       string            `json:"chain_id"`
	Memo          string            `json:"memo"`
	Msgs          []json.RawMessage `json:"msgs"`
	Sequence      int64             `json:"sequence"`
	Source        int64             `json:"source"`
	Data          []byte            `json:"data"`
}

// StdSignBytes returns the bytes to sign for a transaction.
func StdSignBytes(chainID string, accnum int64, sequence int64, msgs []sdk.Msg, memo string, source int64, data []byte) []byte {
	var msgsBytes []json.RawMessage
	for _, msg := range msgs {
		msgsBytes = append(msgsBytes, json.RawMessage(msg.GetSignBytes()))
	}
	bz, err := msgCdc.MarshalJSON(StdSignDoc{
		AccountNumber: accnum,
		ChainID:       chainID,
		Memo:          memo,
		Msgs:          msgsBytes,
		Sequence:      sequence,
		Source:        source,
		Data:          data,
	})
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

// Standard Signature
type StdSignature struct {
	crypto.PubKey `json:"pub_key"` // optional
	Signature     []byte           `json:"signature"`
	AccountNumber int64            `json:"account_number"`
	Sequence      int64            `json:"sequence"`
}

// logic for standard transaction decoding
func DefaultTxDecoder(cdc *codec.Codec) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, sdk.Error) {
		var tx = StdTx{}

		if len(txBytes) == 0 {
			return nil, sdk.ErrTxDecode("txBytes are empty")
		}

		// StdTx.Msg is an interface. The concrete types
		// are registered by MakeTxCodec
		err := cdc.UnmarshalBinaryLengthPrefixed(txBytes, &tx)
		if err != nil {
			return nil, sdk.ErrTxDecode("").TraceSDK(err.Error())
		}
		return tx, nil
	}
}
