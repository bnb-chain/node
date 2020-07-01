package dex

import (
	"fmt"
	"strconv"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/plugins/dex/utils"
)

// TODO: improve, should be configurable
const MaxDepthLevels = 1000    // matches UI requirement
const DefaultDepthLevels = 100 // matches UI requirement

func createAbciQueryHandler(keeper *DexKeeper, abciQueryPrefix string) app.AbciQueryHandler {
	queryPrefix := abciQueryPrefix
	return func(app app.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		// expects at least two query path segments.
		if path[0] != queryPrefix || len(path) < 2 {
			return nil
		}
		switch path[1] {
		case "pairs": // args: ["dex" or "dex-mini", "pairs", <offset>, <limit>]
			if len(path) < 4 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log: fmt.Sprintf(
						"%s %s query requires offset and limit in the path",
						queryPrefix, path[1]),
				}
			}
			ctx := app.GetContextForCheckState()
			pairs := listPairs(keeper, ctx, queryPrefix)
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
			//TODO: sync lock, validate pair
			if len(path) < 3 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log:  "OrderBook query requires the pair symbol and levels",
				}
			}
			pair := path[2]
			height := app.GetContextForCheckState().BlockHeight()
			levelLimit := DefaultDepthLevels
			if len(path) == 4 {
				if l, err := strconv.Atoi(path[3]); err != nil {
					return &abci.ResponseQuery{
						Code: uint32(sdk.CodeUnknownRequest),
						Log:  fmt.Sprintf("OrderBook query requires valid int levels parameter: %v", err),
					}
				} else if l <= 0 || l > MaxDepthLevels {
					return &abci.ResponseQuery{
						Code: uint32(sdk.CodeUnknownRequest),
						Log:  "OrderBook query requires valid levels (>0 && <1000)",
					}
				} else {
					levelLimit = l
				}
			}
			levels := keeper.GetOrderBookLevels(pair, levelLimit)
			book := store.OrderBook{
				Height:       height,
				Levels:       levels,
				PendingMatch: pendingMatch,
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
			if queryPrefix == DexMiniAbciQueryPrefix {
				return &abci.ResponseQuery{
					Code: uint32(sdk.ABCICodeOK),
					Info: fmt.Sprintf(
						"Unknown `%s` query path: %v",
						queryPrefix, path),
				}
			}
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
			if !keeper.PairMapper.Exists(ctx, baseAsset, quoteAsset) {
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
					queryPrefix, path),
			}
		}
	}
}

func listPairs(keeper *DexKeeper, ctx sdk.Context, abciPrefix string) []types.TradingPair {
	pairs := keeper.PairMapper.ListAllTradingPairs(ctx)
	rs := make([]types.TradingPair, 0, len(pairs))
	for _, pair := range pairs {
		if keeper.GetPairType(pair.GetSymbol()) == order.PairType.MINI {
			if abciPrefix == DexMiniAbciQueryPrefix {
				rs = append(rs, pair)
			}
		} else {
			if abciPrefix == DexAbciQueryPrefix {
				rs = append(rs, pair)
			}
		}
	}
	return rs
}
