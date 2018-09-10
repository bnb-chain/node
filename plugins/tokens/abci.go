package tokens

import (
	"fmt"
	"strconv"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/BiJie/BinanceChain/common/types"
)

func createAbciQueryHandler(mapper Mapper) app.AbciQueryHandler {
	return func(app app.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		// expects at least two query path segments.
		if path[0] != abciQueryPrefix || len(path) < 2 {
			return nil
		}
		switch path[1] {
		case "list": // args: ["tokens", "list", <offset>, <limit>]
			if len(path) < 4 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log:  "pairs query requires offset and limit in the path",
				}
			}
			ctx := app.GetContextForCheckState()
			tokens := mapper.GetTokenList(ctx)
			offset, err := strconv.Atoi(path[2])
			if err != nil || offset < 0 || offset > len(tokens)-1 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "unable to parse offset",
				}
			}
			limit, err := strconv.Atoi(path[3])
			if err != nil || limit <= 0 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "unable to parse limit",
				}
			}
			end := offset + limit
			if end > len(tokens) {
				end = len(tokens)
			}
			bz, err := app.GetCodec().MarshalBinary(
				tokens[offset:end],
			)
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
				Code: uint32(sdk.ABCICodeOK),
				Info: fmt.Sprintf("Unknown `%s` query path", abciQueryPrefix),
			}
		}
	}
}
