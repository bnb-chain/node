package dex

import (
	"fmt"
	"strconv"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/dex/utils"
)

// TODO: improve, should be configurable
const MaxDepthLevels = 100 // matches UI requirement

func createAbciQueryHandler(keeper *DexKeeper) app.AbciQueryHandler {
	return func(app app.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		// expects at least two query path segments.
		if path[0] != AbciQueryPrefix || len(path) < 2 {
			return nil
		}
		switch path[1] {
		case "pairs": // args: ["dex", "pairs", <offset>, <limit>]
			if len(path) < 4 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log: fmt.Sprintf(
						"%s %s query requires offset and limit in the path",
						AbciQueryPrefix, path[1]),
				}
			}
			ctx := app.GetContextForCheckState()
			pairs := keeper.PairMapper.ListAllTradingPairs(ctx)
			var offset, limit, end int
			var err error
			if pairs == nil || len(pairs) == 0 {
				pairs = make([]types.TradingPair, 0)
				goto respond
			}
			offset, err = strconv.Atoi(path[2])
			if err != nil || offset < 0 || offset > len(pairs)-1 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "unable to parse offset",
				}
			}
			limit, err = strconv.Atoi(path[3])
			if err != nil || limit <= 0 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "unable to parse limit",
				}
			}
			end = offset + limit
			if end > len(pairs) {
				end = len(pairs)
			}
			if end <= 0 || end <= offset {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "malformed range",
				}
			}
		respond:
			bz, err := app.GetCodec().MarshalBinaryLengthPrefixed(
				pairs[offset:end],
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
		case "orderbook": // args: ["dex", "orderbook"]
			//TODO: sync lock, validate pair, level number
			if len(path) < 3 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log:  "OrderBook query requires the pair symbol",
				}
			}
			pair := path[2]
			height := app.GetContextForCheckState().BlockHeight()
			levels := keeper.GetOrderBookLevels(pair, MaxDepthLevels)
			if levels == nil{
				return &abci.ResponseQuery{
					Code:  uint32(sdk.CodeInternal),
					Log:  "market pair do not exist",
				}
			}
			book := store.OrderBook{
				Height: height,
				Levels: levels,
			}
			bz, err := app.GetCodec().MarshalBinaryLengthPrefixed(book)
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
		case "openorders": // args: ["dex", "openorders", <pair>, <bech32Str>]
			if len(path) < 4 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log:  "OpenOrders query requires the pair symbol and address",
				}
			}

			// verify pair is legal
			pair := path[2]
			baseAsset, quoteAsset, err := utils.TradingPair2Assets(pair)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "pair is not valid",
				}
			}
			ctx := app.GetContextForCheckState()
			existingPair, err := keeper.PairMapper.GetTradingPair(ctx, baseAsset, quoteAsset)
			if pair != existingPair.GetSymbol() || err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "pair is not listed",
				}
			}

			bech32Str := path[3]
			addr, err := sdk.AccAddressFromBech32(bech32Str)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  "address is not valid",
				}
			}
			openOrders := keeper.GetOpenOrders(pair, addr)
			bz, err := app.GetCodec().MarshalBinaryLengthPrefixed(openOrders)
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
					AbciQueryPrefix, path),
			}
		}
	}
}
