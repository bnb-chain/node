package wire

import (
	"bytes"
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	amino "github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
)

// amino codec to marshal/unmarshal
type Codec = amino.Codec

type txDecoderFn func(cdc *Codec) sdk.TxDecoder

func NewCodec() *Codec {
	cdc := amino.NewCodec()
	return cdc
}

// Register the go-crypto to the codec
func RegisterCrypto(cdc *Codec) {
	cryptoAmino.RegisterAmino(cdc)
}

// attempt to make some pretty json
func MarshalJSONIndent(cdc *Codec, obj interface{}) ([]byte, error) {
	bz, err := cdc.MarshalJSON(obj)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	err = json.Indent(&out, bz, "", "  ")
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// ComposeTxDecoders composes tx decoders together and tries each one until a decoded tx is returned.
func ComposeTxDecoders(cdc *Codec, decoders ...txDecoderFn) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, sdk.Error) {
		var tx sdk.Tx
		var err sdk.Error
		for i := range decoders {
			tx, err = decoders[i](cdc)(txBytes)
			if err != nil || tx == nil {
				continue
			}
			return tx, nil
		}
		return nil, err
	}
}

//__________________________________________________________________

// generic sealed codec to be used throughout sdk
var Cdc *Codec

func init() {
	cdc := NewCodec()
	RegisterCrypto(cdc)
	Cdc = cdc.Seal()
}
