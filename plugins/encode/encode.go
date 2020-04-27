package encode

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
		case "tx":
			return encodeTx(ctx, codec, req)
		default:
			return nil, sdk.ErrUnknownRequest(fmt.Sprintf("unknown atomic swap query endpoint %s", path[0]))
		}
	}
}

func encodeTx(ctx sdk.Context, codec *codec.Codec, req abci.RequestQuery) ([]byte, sdk.Error) {
	var stdTx auth.StdTx

	err := codec.UnmarshalJSON(req.Data, &stdTx)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(err.Error())
	}

	txBytes, err := codec.MarshalBinaryLengthPrefixed(req)
	if err != nil {
		return nil, sdk.ErrInternal(err.Error())
	}

	return txBytes, nil
}
