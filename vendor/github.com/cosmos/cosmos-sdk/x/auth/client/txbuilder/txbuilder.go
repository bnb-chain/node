package context

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// TxBuilder implements a transaction context created in SDK modules.
type TxBuilder struct {
	Codec         *codec.Codec
	AccountNumber int64
	Sequence      int64
	ChainID       string
	Memo          string
	Source        int64
}

// NewTxBuilderFromCLI returns a new initialized TxBuilder with parameters from
// the command line using Viper.
func NewTxBuilderFromCLI() TxBuilder {
	return TxBuilder{
		ChainID:       viper.GetString(client.FlagChainID),
		AccountNumber: viper.GetInt64(client.FlagAccountNumber),
		Sequence:      viper.GetInt64(client.FlagSequence),
		Memo:          viper.GetString(client.FlagMemo),
		Source:        viper.GetInt64(client.FlagSource),
	}
}

// WithCodec returns a copy of the context with an updated codec.
func (bldr TxBuilder) WithCodec(cdc *codec.Codec) TxBuilder {
	bldr.Codec = cdc
	return bldr
}

// WithChainID returns a copy of the context with an updated chainID.
func (bldr TxBuilder) WithChainID(chainID string) TxBuilder {
	bldr.ChainID = chainID
	return bldr
}

// WithSequence returns a copy of the context with an updated sequence number.
func (bldr TxBuilder) WithSequence(sequence int64) TxBuilder {
	bldr.Sequence = sequence
	return bldr
}

// WithMemo returns a copy of the context with an updated memo.
func (bldr TxBuilder) WithMemo(memo string) TxBuilder {
	bldr.Memo = memo
	return bldr
}

// WithAccountNumber returns a copy of the context with an account number.
func (bldr TxBuilder) WithAccountNumber(accnum int64) TxBuilder {
	bldr.AccountNumber = accnum
	return bldr
}

// WithSource returns a copy of the context with an updated source.
func (bldr TxBuilder) WithSource(source int64) TxBuilder {
	bldr.Source = source
	return bldr
}

// Build builds a single message to be signed from a TxBuilder given a set of
// messages.
func (bldr TxBuilder) Build(msgs []sdk.Msg) (StdSignMsg, error) {
	chainID := bldr.ChainID
	if chainID == "" {
		return StdSignMsg{}, errors.Errorf("chain ID required but not specified")
	}

	return StdSignMsg{
		ChainID:       bldr.ChainID,
		AccountNumber: bldr.AccountNumber,
		Sequence:      bldr.Sequence,
		Memo:          bldr.Memo,
		Msgs:          msgs,
		Source:        bldr.Source,
	}, nil
}

// Sign signs a transaction given a name, passphrase, and a single message to
// signed. An error is returned if signing fails.
func (bldr TxBuilder) Sign(name, passphrase string, msg StdSignMsg) ([]byte, error) {
	sig, err := MakeSignature(name, passphrase, msg)
	if err != nil {
		return nil, err
	}
	return bldr.Codec.MarshalBinaryLengthPrefixed(auth.NewStdTx(msg.Msgs, []auth.StdSignature{sig}, msg.Memo, msg.Source, msg.Data))
}

// BuildAndSign builds a single message to be signed, and signs a transaction
// with the built message given a name, passphrase, and a set of
// messages.
func (bldr TxBuilder) BuildAndSign(name, passphrase string, msgs []sdk.Msg) ([]byte, error) {
	msg, err := bldr.Build(msgs)
	if err != nil {
		return nil, err
	}

	return bldr.Sign(name, passphrase, msg)
}

// BuildWithPubKey builds a single message to be signed from a TxBuilder given a set of
// messages and attach the public key associated to the given name.
func (bldr TxBuilder) BuildWithPubKey(name string, msgs []sdk.Msg) ([]byte, error) {
	msg, err := bldr.Build(msgs)
	if err != nil {
		return nil, err
	}

	keybase, err := keys.GetKeyBase()
	if err != nil {
		return nil, err
	}

	info, err := keybase.Get(name)
	if err != nil {
		return nil, err
	}

	sigs := []auth.StdSignature{{
		AccountNumber: msg.AccountNumber,
		Sequence:      msg.Sequence,
		PubKey:        info.GetPubKey(),
	}}

	return bldr.Codec.MarshalBinaryLengthPrefixed(auth.NewStdTx(msg.Msgs, sigs, msg.Memo, msg.Source, msg.Data))
}

// SignStdTx appends a signature to a StdTx and returns a copy of a it. If append
// is false, it replaces the signatures already attached with the new signature.
func (bldr TxBuilder) SignStdTx(name, passphrase string, stdTx auth.StdTx, appendSig bool) (signedStdTx auth.StdTx, err error) {
	stdSignature, err := MakeSignature(name, passphrase, StdSignMsg{
		ChainID:       bldr.ChainID,
		AccountNumber: bldr.AccountNumber,
		Sequence:      bldr.Sequence,
		Msgs:          stdTx.GetMsgs(),
		Memo:          stdTx.GetMemo(),
		Source:        stdTx.GetSource(),
		Data:          stdTx.GetData(),
	})
	if err != nil {
		return
	}

	sigs := stdTx.GetSignatures()
	if len(sigs) == 0 || !appendSig {
		sigs = []auth.StdSignature{stdSignature}
	} else {
		sigs = append(sigs, stdSignature)
	}
	signedStdTx = auth.NewStdTx(stdTx.GetMsgs(), sigs, stdTx.GetMemo(), stdTx.GetSource(), stdTx.GetData())
	return
}

// MakeSignature builds a StdSignature given key name, passphrase, and a StdSignMsg.
func MakeSignature(name, passphrase string, msg StdSignMsg) (sig auth.StdSignature, err error) {
	keybase, err := keys.GetKeyBase()
	if err != nil {
		return
	}
	sigBytes, pubkey, err := keybase.Sign(name, passphrase, msg.Bytes())
	if err != nil {
		return
	}
	return auth.StdSignature{
		AccountNumber: msg.AccountNumber,
		Sequence:      msg.Sequence,
		PubKey:        pubkey,
		Signature:     sigBytes,
	}, nil
}
