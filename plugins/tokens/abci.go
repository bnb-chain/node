package tokens

import (
	"fmt"
	"strconv"
	"strings"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/node/common/types"
)

func createAbciQueryHandler(mapper Mapper, prefix string) types.AbciQueryHandler {
	queryPrefix := prefix
	var isMini bool
	switch queryPrefix {
	case abciQueryPrefix:
		isMini = false
	case miniAbciQueryPrefix:
		isMini = true
	default:
		isMini = false
	}
	return func(app types.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		// expects at least two query path segments.
		if path[0] != queryPrefix || len(path) < 2 {
			return nil
		}
		switch path[1] {
		case "info": // args: ["tokens", "info", <symbol>]
			if len(path) < 3 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log: fmt.Sprintf(
						"%s %s query requires a symbol path arg",
						queryPrefix, path[1]),
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
			return queryAndMarshallToken(app, mapper, ctx, symbol)
		case "list": // args: ["tokens", "list", <offset>, <limit>, <showZeroSupplyTokens>]
			if len(path) < 4 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log: fmt.Sprintf(
						"%s %s query requires offset and limit path segments",
						queryPrefix, path[1]),
				}
			}
			showZeroSupplyTokens := false
			if len(path) == 5 && strings.ToLower(path[4]) == "true" {
				showZeroSupplyTokens = true
			}
			ctx := app.GetContextForCheckState()

			tokens := mapper.GetTokenList(ctx, showZeroSupplyTokens, isMini)

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
			var bz []byte
			if isMini {
				miniTokens := make([]*types.MiniToken, end-offset)
				for i, token := range tokens[offset:end] {
					miniTokens[i] = token.(*types.MiniToken)
				}
				bz, err = app.GetCodec().MarshalBinaryLengthPrefixed(
					miniTokens,
				)
			} else {
				bep2Tokens := make([]*types.Token, end-offset)
				for i, token := range tokens[offset:end] {
					bep2Tokens[i] = token.(*types.Token)
				}
				bz, err = app.GetCodec().MarshalBinaryLengthPrefixed(
					bep2Tokens,
				)
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
		default:
			return &abci.ResponseQuery{
				Code: uint32(sdk.ABCICodeOK),
				Info: fmt.Sprintf(
					"Unknown `%s` query path: %v",
					queryPrefix, path),
			}
		}
	}
}

func queryAndMarshallToken(app types.ChainApp, mapper Mapper, ctx sdk.Context, symbol string) *abci.ResponseQuery {
	var bz []byte
	var err error
	var token types.IToken

	token, err = mapper.GetToken(ctx, symbol)
	if err != nil {
		return &abci.ResponseQuery{
			Code: uint32(sdk.CodeInternal),
			Log:  err.Error(),
		}
	}
	bz, err = app.GetCodec().MarshalBinaryLengthPrefixed(token)
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
