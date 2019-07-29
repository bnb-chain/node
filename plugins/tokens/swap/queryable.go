package swap

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	QuerySwapFrom = "swapfrom"
	QuerySwapTo   = "swapto"
)

func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QuerySwapFrom:
			return querySwapFrom(ctx, req, keeper)
		case QuerySwapTo:
			return querySwapTo(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest(fmt.Sprintf("unknown atomic swap query endpoint %s", path[0]))
		}
	}
}

// Params for query 'custom/atomicswap/swapfrom'
type QuerySwapFromParams struct {
	From   sdk.AccAddress
	Status SwapStatus
	Limit  int64
	Offset int64
}

// nolint: unparam
func querySwapFrom(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapFromParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	if len(params.From) != sdk.AddrLen {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of address should be %d", sdk.AddrLen))
	}
	if params.Limit > 1000 {
		return nil, ErrTooLargeQueryLimit("limit should not be greater 1000")
	}
	// Assign default page size
	if params.Limit == 0 {
		params.Limit = 100
	}

	iterator := keeper.GetSwapFromIterator(ctx, params.From)
	defer iterator.Close()

	count := int64(0)
	atomicSwaps := make([]AtomicSwap, 0, params.Limit)
	for ; iterator.Valid(); iterator.Next() {
		swap := keeper.QuerySwap(ctx, iterator.Value())
		count++
		if count <= params.Offset {
			continue
		}
		if count-params.Offset > params.Limit {
			break
		}
		if params.Status != NULL && swap.Status != params.Status {
			continue
		}
		if swap != nil {
			atomicSwaps = append(atomicSwaps, *swap)
		}
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, atomicSwaps)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}

// Params for query 'custom/atomicswap/swapto'
type QuerySwapToParams struct {
	To     sdk.AccAddress
	Status SwapStatus
	Limit  int64
	Offset int64
}

// nolint: unparam
func querySwapTo(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapToParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	if len(params.To) != sdk.AddrLen {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of address should be %d", sdk.AddrLen))
	}
	if params.Limit > 1000 {
		return nil, ErrTooLargeQueryLimit("limit should not be greater 1000")
	}
	// Assign default page size
	if params.Limit == 0 {
		params.Limit = 100
	}

	iterator := keeper.GetSwapToIterator(ctx, params.To)
	defer iterator.Close()

	count := int64(0)
	atomicSwaps := make([]AtomicSwap, 0, params.Limit)
	for ; iterator.Valid(); iterator.Next() {
		swap := keeper.QuerySwap(ctx, iterator.Value())
		count++
		if count <= params.Offset {
			continue
		}
		if count-params.Offset > params.Limit {
			break
		}
		if params.Status != NULL && swap.Status != params.Status {
			continue
		}
		if swap != nil {
			atomicSwaps = append(atomicSwaps, *swap)
		}
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, atomicSwaps)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}
