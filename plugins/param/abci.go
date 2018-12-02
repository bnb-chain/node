package param

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	app "github.com/BiJie/BinanceChain/common/types"
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
			bz, err := app.GetCodec().MarshalBinary(fp)
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
