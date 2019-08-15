package swap

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	QuerySwapCreator   = "swapcreator"
	QuerySwapRecipient = "swaprecipient"
)

func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QuerySwapCreator:
			return querySwapByCreator(ctx, req, keeper)
		case QuerySwapRecipient:
			return querySwapByRecipient(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest(fmt.Sprintf("unknown atomic swap query endpoint %s", path[0]))
		}
	}
}

// Params for query 'custom/atomicswap/swapcreator'
type QuerySwapByCreatorParams struct {
	Creator sdk.AccAddress
	Status  SwapStatus
	Limit   int64
	Offset  int64
}

// nolint: unparam
func querySwapByCreator(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapByCreatorParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	if len(params.Creator) != sdk.AddrLen {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of address should be %d", sdk.AddrLen))
	}
	if params.Limit > 1000 {
		return nil, ErrTooLargeQueryLimit("limit should not be greater 1000")
	}
	// Assign default limit value
	if params.Limit == 0 {
		params.Limit = 100
	}

	iterator := keeper.GetSwapCreatorIterator(ctx, params.Creator)
	defer iterator.Close()

	count := int64(0)
	atomicSwaps := make([]AtomicSwap, 0, params.Limit)
	for ; iterator.Valid(); iterator.Next() {
		swap := keeper.GetSwap(ctx, iterator.Value())
		if swap == nil {
			continue
		}
		if params.Status != NULL && swap.Status != params.Status {
			continue
		}
		count++
		if count <= params.Offset {
			continue
		}
		if count-params.Offset > params.Limit {
			break
		}
		atomicSwaps = append(atomicSwaps, *swap)
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, atomicSwaps)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}

// Params for query 'custom/atomicswap/swaprecipient'
type QuerySwapByRecipientParams struct {
	Recipient sdk.AccAddress
	Status    SwapStatus
	Limit     int64
	Offset    int64
}

// nolint: unparam
func querySwapByRecipient(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapByRecipientParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	if len(params.Recipient) != sdk.AddrLen {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of address should be %d", sdk.AddrLen))
	}
	if params.Limit > 1000 {
		return nil, ErrTooLargeQueryLimit("limit should not be greater 1000")
	}
	// Assign default limit value
	if params.Limit == 0 {
		params.Limit = 100
	}

	iterator := keeper.GetSwapRecipientIterator(ctx, params.Recipient)
	defer iterator.Close()

	count := int64(0)
	atomicSwaps := make([]AtomicSwap, 0, params.Limit)
	for ; iterator.Valid(); iterator.Next() {
		swap := keeper.GetSwap(ctx, iterator.Value())
		if swap == nil {
			continue
		}
		if params.Status != NULL && swap.Status != params.Status {
			continue
		}
		count++
		if count <= params.Offset {
			continue
		}
		if count-params.Offset > params.Limit {
			break
		}
		atomicSwaps = append(atomicSwaps, *swap)
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, atomicSwaps)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}
