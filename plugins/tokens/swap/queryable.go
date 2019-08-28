package swap

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	QuerySwapID        = "swapid"
	QuerySwapCreator   = "swapcreator"
	QuerySwapRecipient = "swaprecipient"
)

func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QuerySwapID:
			return querySwapByHash(ctx, req, keeper)
		case QuerySwapCreator:
			return querySwapByCreator(ctx, req, keeper)
		case QuerySwapRecipient:
			return querySwapByRecipient(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest(fmt.Sprintf("unknown atomic swap query endpoint %s", path[0]))
		}
	}
}

// Params for query 'custom/atomicswap/swapid'
type QuerySwapByID struct {
	SwapID HexData
}

// nolint: unparam
func querySwapByHash(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapByID
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(fmt.Sprintf("incorrectly formatted request data: %s", err.Error()))
	}

	if len(params.SwapID) != SwapIDLength {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of swapID should be %d", SwapIDLength))
	}

	swap := keeper.GetSwap(ctx, params.SwapID)
	if swap == nil {
		return nil, ErrNonExistSwapID(fmt.Sprintf("No matched swap with swapID %v", params.SwapID))
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, *swap)
	if err != nil {
		return nil, sdk.ErrInternal(fmt.Sprintf("could not marshal result to JSON: %s", err.Error()))
	}

	return bz, nil
}

// Params for query 'custom/atomicswap/swapcreator'
type QuerySwapByCreatorParams struct {
	Creator sdk.AccAddress
	Limit   int64
	Offset  int64
}

// nolint: unparam
func querySwapByCreator(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapByCreatorParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(fmt.Sprintf("incorrectly formatted request data: %s", err.Error()))
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
	swapIDList := make([]HexData, 0, params.Limit)
	for ; iterator.Valid(); iterator.Next() {
		count++
		if count <= params.Offset {
			continue
		}
		if count-params.Offset > params.Limit {
			break
		}
		swapIDList = append(swapIDList, iterator.Value())
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, swapIDList)
	if err != nil {
		return nil, sdk.ErrInternal(fmt.Sprintf("could not marshal result to JSON: %s", err.Error()))
	}

	return bz, nil
}

// Params for query 'custom/atomicswap/swaprecipient'
type QuerySwapByRecipientParams struct {
	Recipient sdk.AccAddress
	Limit     int64
	Offset    int64
}

// nolint: unparam
func querySwapByRecipient(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapByRecipientParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(fmt.Sprintf("incorrectly formatted request data: %s", err.Error()))
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
	swapIDList := make([]HexData, 0, params.Limit)
	for ; iterator.Valid(); iterator.Next() {
		count++
		if count <= params.Offset {
			continue
		}
		if count-params.Offset > params.Limit {
			break
		}
		swapIDList = append(swapIDList, iterator.Value())
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, swapIDList)
	if err != nil {
		return nil, sdk.ErrInternal(fmt.Sprintf("could not marshal result to JSON: %s", err.Error()))
	}

	return bz, nil
}
