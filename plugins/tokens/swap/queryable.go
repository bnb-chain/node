package swap

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	cmm "github.com/tendermint/tendermint/libs/common"
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
			return querySwapByID(ctx, req, keeper)
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
	SwapID cmm.HexBytes
}

// nolint: unparam
func querySwapByID(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QuerySwapByID
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(fmt.Sprintf("incorrectly formatted request data: %s", err.Error()))
	}

	if len(params.SwapID) != SwapIDLength {
		return nil, ErrInvalidSwapID(fmt.Sprintf("length of swapID should be %d", SwapIDLength))
	}

	swap := keeper.GetSwap(ctx, params.SwapID)
	if swap == nil {
		return nil, nil
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
	if params.Limit <= 0 || params.Limit > 100 {
		return nil, ErrInvalidPaginationParameters("limit should be in (0, 100]")
	}
	if params.Offset < 0 {
		return nil, ErrInvalidPaginationParameters("offset must be positive")
	}

	iterator := keeper.GetSwapCreatorIterator(ctx, params.Creator)
	defer iterator.Close()

	count := int64(0)
	swapIDList := make([]cmm.HexBytes, 0, params.Limit)
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
	if params.Limit <= 0 || params.Limit > 100 {
		return nil, ErrInvalidPaginationParameters("limit should be in (0, 100]")
	}
	if params.Offset < 0 {
		return nil, ErrInvalidPaginationParameters("offset must be positive")
	}

	iterator := keeper.GetSwapRecipientIterator(ctx, params.Recipient)
	defer iterator.Close()

	count := int64(0)
	swapIDList := make([]cmm.HexBytes, 0, params.Limit)
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
