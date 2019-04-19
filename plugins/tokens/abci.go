package tokens

import (
	"fmt"
	"strconv"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/binance-chain/node/common/types"
)

func createAbciQueryHandler(mapper Mapper) app.AbciQueryHandler {
	return func(app app.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		// expects at least two query path segments.
		if path[0] != abciQueryPrefix || len(path) < 2 {
			return nil
		}
		switch path[1] {
		case "info": // args: ["tokens", "info", <symbol>]
			if len(path) < 3 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log: fmt.Sprintf(
						"%s %s query requires a symbol path arg",
						abciQueryPrefix, path[1]),
				}
			}
			ctx := app.GetContextForCheckState()
			symbol := path[2]
			if len(symbol) == 0 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "empty symbol not permitted",
				}
			}
			token, err := mapper.GetToken(ctx, symbol)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  err.Error(),
				}
			}
			bz, err := app.GetCodec().MarshalBinaryLengthPrefixed(token)
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
		case "list": // args: ["tokens", "list", <offset>, <limit>]
			if len(path) < 4 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log: fmt.Sprintf(
						"%s %s query requires offset and limit path segments",
						abciQueryPrefix, path[1]),
				}
			}
			ctx := app.GetContextForCheckState()
			tokens := mapper.GetTokenList(ctx)
			offset, err := strconv.ParseInt(path[2], 10, 64)
			if err != nil || offset < 0 || offset >= int64(len(tokens)) {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "unable to parse offset",
				}
			}
			limit, err := strconv.ParseInt(path[3], 10, 64)
			if err != nil || limit <= 0 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "unable to parse limit",
				}
			}
			end := offset + limit
			if end > int64(len(tokens)) {
				end = int64(len(tokens))
			}
			if end <= 0 || end <= offset {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "malformed range",
				}
			}
			bz, err := app.GetCodec().MarshalBinaryLengthPrefixed(
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
				Info: fmt.Sprintf(
					"Unknown `%s` query path: %v",
					abciQueryPrefix, path),
			}
		}
	}
}
