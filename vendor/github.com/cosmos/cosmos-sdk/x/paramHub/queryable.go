package paramHub

import (
	"fmt"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

func NewQuerier(hub *ParamHub, cdc *codec.Codec) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case "fees":
			fp := hub.GetFeeParams(ctx)
			res, err := cdc.MarshalBinaryLengthPrefixed(fp)
			if err != nil {
				return nil, sdk.ErrInternal(err.Error())
			}
			return res, nil
		case "sideParams":
			if len(req.Data) == 0 {
				return nil, types.ErrMissSideChainId(types.DefaultCodespace)
			}
			var sideChainId string
			err := cdc.UnmarshalJSON(req.Data, &sideChainId)
			if err != nil {
				return nil, types.ErrInvalidSideChainId(types.DefaultCodespace, err.Error())
			}
			params, sdkErr := hub.GetSCParams(ctx, sideChainId)
			if err != nil {
				return nil, sdkErr
			}
			res, err := cdc.MarshalJSON(params)
			if err != nil {
				return nil, sdk.ErrInternal(err.Error())
			}
			return res, nil

		default:
			return res, sdk.ErrUnknownRequest(req.Path)
		}
	}
}

// tolerate the previous RPC api.
func CreateAbciQueryHandler(paramHub *ParamHub) func(sdk.Context, abci.RequestQuery, []string) *abci.ResponseQuery {
	return func(ctx sdk.Context, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		// expects at least two query path segments.
		if path[0] != AbciQueryPrefix || len(path) < 2 {
			return nil
		}
		switch path[1] {
		case "fees":
			fp := paramHub.GetFeeParams(ctx)
			bz, err := paramHub.GetCodeC().MarshalBinaryLengthPrefixed(fp)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  err.Error(),
				}
			}
			return &abci.ResponseQuery{
				Code:  uint32(sdk.ABCICodeOK),
				Value: bz,
			}
		case "sideParams":
			if len(req.Data) == 0 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "missing side chain id",
				}
			}
			var sideChainId string
			err := paramHub.GetCodeC().UnmarshalJSON(req.Data, &sideChainId)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  fmt.Sprintf("invalid data %v", err),
				}
			}
			params, sdkErr := paramHub.GetSCParams(ctx, sideChainId)
			if sdkErr != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdkErr.ABCICode()),
					Log:  sdkErr.ABCILog(),
				}
			}
			bz, err := paramHub.GetCodeC().MarshalJSON(params)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  err.Error(),
				}
			}
			return &abci.ResponseQuery{
				Code:  uint32(sdk.ABCICodeOK),
				Value: bz,
			}

		default:
			return &abci.ResponseQuery{
				Code: uint32(sdk.CodeOK),
				Info: fmt.Sprintf(
					"Unknown `%s` query path: %v",
					AbciQueryPrefix, path),
			}
		}
	}
}
