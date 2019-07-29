package swap

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	QuerySwapOut = "swapout"
	QuerySwapIn  = "swapin"
)

func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QuerySwapOut:
			return querySwapOut(ctx, req, keeper)
		case QuerySwapIn:
			return querySwapIn(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest(fmt.Sprintf("unknown atomic swap query endpoint %s", path[0]))
		}
	}
}

// Params for query 'custom/atomicswap/swapout'
type QuerySwapOutParams struct {
	SwapCreator sdk.AccAddress
	PageSize    int64
	PageNum     int64
}

// nolint: unparam
func querySwapOut(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapOutParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	if len(params.SwapCreator) != sdk.AddrLen {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of address should be %d", sdk.AddrLen))
	}
	if params.PageSize > 1000 {
		return nil, ErrTooLargePageSize("Page size should be no greater than 1000")
	}
	// Assign default page size
	if params.PageSize == 0 {
		params.PageSize = 100
	}

	iterator := keeper.GetSwapOutIterator(ctx, params.SwapCreator)
	defer iterator.Close()

	skipQuantity :=  params.PageSize * params.PageNum
	count := int64(0)
	atomicSwaps := make([]AtomicSwap, 0, params.PageSize)
	for ; iterator.Valid(); iterator.Next() {
		count++
		if count <= skipQuantity {
			continue
		}
		if count - skipQuantity > params.PageSize {
			break
		}
		swap := keeper.QuerySwap(ctx, iterator.Value())
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

// Params for query 'custom/atomicswap/swapin'
type QuerySwapInParams struct {
	SwapReceiver sdk.AccAddress
	PageSize     int64
	PageNum      int64
}

// nolint: unparam
func querySwapIn(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapInParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	if len(params.SwapReceiver) != sdk.AddrLen {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of address should be %d", sdk.AddrLen))
	}
	if params.PageSize > 1000 {
		return nil, ErrTooLargePageSize("Page size should be no greater than 1000")
	}
	// Assign default page size
	if params.PageSize == 0 {
		params.PageSize = 100
	}

	iterator := keeper.GetSwapOutIterator(ctx, params.SwapReceiver)
	defer iterator.Close()

	skipQuantity :=  params.PageSize * params.PageNum
	count := int64(0)
	atomicSwaps := make([]AtomicSwap, 0, params.PageSize)
	for ; iterator.Valid(); iterator.Next() {
		count++
		if count <= skipQuantity {
			continue
		}
		if count - skipQuantity > params.PageSize {
			break
		}
		swap := keeper.QuerySwap(ctx, iterator.Value())
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
