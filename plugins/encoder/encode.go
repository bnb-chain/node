package encoder

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	abci "github.com/tendermint/tendermint/abci/types"
)

func NewQuerier(codec *codec.Codec) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		fmt.Println(path)
		switch path[0] {
		case encodeTx:
			return txEncoder(codec, req)
		default:
			return nil, sdk.ErrUnknownRequest(fmt.Sprintf("unknown encoder query endpoint %s", path[0]))
		}
	}
}

func txEncoder(codec *codec.Codec, req abci.RequestQuery) ([]byte, sdk.Error) {
	var stdTx auth.StdTx

	err := codec.UnmarshalJSON(req.Data, &stdTx)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(err.Error())
	}

	txBytes, err := codec.MarshalBinaryLengthPrefixed(stdTx)
	if err != nil {
		return nil, sdk.ErrInternal(err.Error())
	}

	return txBytes, nil
}