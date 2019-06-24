package timelock

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	QueryTimeLocks = "timelocks"
	QueryTimeLock  = "timelock"
)

func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QueryTimeLocks:
			return queryTimeLocks(ctx, req, keeper)
		case QueryTimeLock:
			return queryTimeLock(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest(fmt.Sprintf("unknown time lock query endpoint %s", path[0]))
		}
	}
}

// Params for query 'custom/timelock/timelocks'
type QueryTimeLocksParams struct {
	Account sdk.AccAddress
}

// nolint: unparam
func queryTimeLocks(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QueryTimeLocksParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	if len(params.Account) != sdk.AddrLen {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of address should be %d", sdk.AddrLen))
	}

	timeLocks := keeper.GetTimeLockRecords(ctx, params.Account)
	bz, err := codec.MarshalJSONIndent(keeper.cdc, timeLocks)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}

// Params for query 'custom/timelock/timelock'
type QueryTimeLockParams struct {
	Account sdk.AccAddress
	Id      int64
}

// nolint: unparam
func queryTimeLock(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QueryTimeLockParams
	err := keeper.cdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdk.ErrUnknownRequest(sdk.AppendMsgToErr("incorrectly formatted request data", err.Error()))
	}

	if len(params.Account) != sdk.AddrLen {
		return nil, sdk.ErrInvalidAddress(fmt.Sprintf("length of address should be %d", sdk.AddrLen))
	}

	if params.Id < InitialRecordId {
		return nil, ErrInvalidTimeLockId(DefaultCodespace,
			fmt.Sprintf("time lock id(%d) should not be less than %d", params.Id, InitialRecordId))
	}

	timeLock, found := keeper.GetTimeLockRecord(ctx, params.Account, params.Id)
	if !found {
		return nil, ErrUnknownTimeLock(DefaultCodespace, params.Account, params.Id)
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, timeLock)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", err.Error()))
	}

	return bz, nil
}
