package dex

import (
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/BiJie/BinanceChain/common/types"
)

func createAbciQueryHandler(keeper *DexKeeper) app.AbciQueryHandler {
	return func(app app.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		// expects at least two query path segments.
		if path[0] != abciQueryPrefix || len(path) < 2 {
			return nil
		}
		switch path[1] {
		case "pairs":
			ctx := app.GetContextForCheckState()
			pairs := keeper.PairMapper.ListAllTradingPairs(ctx)
			pairss := make([]string, len(pairs))
			for _, pair := range pairs {
				pairss = append(pairss, pair.GetSymbol())
			}
			resValue, err := app.GetCodec().MarshalBinary(pairss)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  err.Error(),
				}
			}
			return &abci.ResponseQuery{
				Code:  uint32(sdk.ABCICodeOK),
				Value: resValue,
			}
		case "orderbook":
			//TODO: sync lock, validate pair, level number
			if len(path) < 3 {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeUnknownRequest),
					Log:  "OrderBook query requires the pair symbol",
				}
			}
			pair := path[2]
			orderbook := keeper.GetOrderBook(pair, 20)
			resValue, err := app.GetCodec().MarshalBinary(orderbook)
			if err != nil {
				return &abci.ResponseQuery{
					Code: uint32(sdk.CodeInternal),
					Log:  err.Error(),
				}
			}
			return &abci.ResponseQuery{
				Code:  uint32(sdk.ABCICodeOK),
				Value: resValue,
			}
		default:
			return &abci.ResponseQuery{
				Code: uint32(sdk.ABCICodeOK),
				Info: fmt.Sprintf("Unknown `%s` query path", abciQueryPrefix),
			}
		}
	}
}
