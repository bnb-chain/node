package param

import (
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"

	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/param/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func createAbciQueryHandler(paramHub *ParamHub) app.AbciQueryHandler {
	return func(app app.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		// expects at least two query path segments.
		if path[0] != AbciQueryPrefix || len(path) < 2 {
			return nil
		}
		switch path[1] {
		case "fees":
			ctx := app.GetContextForCheckState()
			fp := paramHub.GetFeeParams(ctx)
			bz, err := app.GetCodec().MarshalBinaryLengthPrefixed(fp)
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
			err := app.GetCodec().UnmarshalJSON(req.Data, &sideChainId)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  fmt.Sprintf("invalid data %v", err),
				}
			}
			ctx := app.GetContextForCheckState()
			storePrefix := paramHub.ScKeeper.GetSideChainStorePrefix(ctx, sideChainId)
			if len(storePrefix) == 0 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "the side chain id is not registered",
				}
			}
			newCtx := ctx.WithSideChainKeyPrefix(storePrefix)
			params := make([]types.SCParam, 0)
			for _, subSpace := range paramHub.GetSubscriberParamSpace() {
				param := subSpace.Proto()
				if _, native := types.ToSCParam(param).GetParamAttribute(); native {
					subSpace.ParamSpace.GetParamSet(ctx, param)
				} else {
					subSpace.ParamSpace.GetParamSet(newCtx, param)
				}
				params = append(params, types.ToSCParam(param))
			}
			bz, err := app.GetCodec().MarshalJSON(params)
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
