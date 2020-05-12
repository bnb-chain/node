package tokens

import (
	"fmt"
	"strconv"
	"strings"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
)

func createAbciQueryHandler(mapper Mapper, prefix string) types.AbciQueryHandler {
	abciQueryPrefix := prefix
	var isMini bool
	switch abciQueryPrefix {
	case abciQueryPrefix:
		isMini = false
	case miniAbciQueryPrefix:
		isMini = true
	default:
		isMini = false
	}
	return func(app types.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
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
			return queryAndMarshallToken(app, mapper, ctx, symbol, isMini)
		case "list": // args: ["tokens", "list", <offset>, <limit>, <showZeroSupplyTokens>]
			if len(path) < 4 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log: fmt.Sprintf(
						"%s %s query requires offset and limit path segments",
						abciQueryPrefix, path[1]),
				}
			}
			showZeroSupplyTokens := false
			if len(path) == 5 && strings.ToLower(path[4]) == "true" {
				showZeroSupplyTokens = true
			}
			ctx := app.GetContextForCheckState()
			tokens := mapper.GetTokenList(ctx, showZeroSupplyTokens)
			offset, err := strconv.Atoi(path[2])
			if err != nil || offset < 0 || offset >= len(tokens) {
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

func queryAndMarshallToken(app types.ChainApp, mapper Mapper, ctx sdk.Context, symbol string, isMini bool) *abci.ResponseQuery {
	var bz []byte
	var err error
	var token interface{}

	token, err = getToken(mapper, ctx, symbol, isMini)
	if err != nil {
		return &abci.ResponseQuery{
			Code: uint32(sdk.CodeInternal),
			Log:  err.Error(),
		}
	}
	switch token.(type) {
	case types.MiniToken:
		bz, err = app.GetCodec().MarshalBinaryLengthPrefixed(token.(types.MiniToken))
	case types.Token:
		bz, err = app.GetCodec().MarshalBinaryLengthPrefixed(token.(types.Token))
	default:
		return &abci.ResponseQuery{
			Code: uint32(sdk.CodeInternal),
			Log:  err.Error(),
		}
	}

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
}

func getToken(mapper Mapper, ctx sdk.Context, symbol string, isMini bool) (interface{}, error) {
	if isMini {
		return mapper.GetMiniToken(ctx, symbol)
	} else {
		return mapper.GetToken(ctx, symbol)
	}
}
