package slashing

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	QueryConsAddrSlashRecords     = "consAddrSlashHistories"
	QueryConsAddrTypeSlashRecords = "consAddrTypeSlashHistories"
)

// creates a querier for staking REST endpoints
func NewQuerier(k Keeper, cdc *codec.Codec) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QueryConsAddrSlashRecords:
			param := new(QueryConsAddrParams)
			ctx, err = RequestPrepare(ctx, k, req, param)
			if err != nil {
				return res, err
			}
			return queryConsAddrSlashRecords(ctx, k, param)
		case QueryConsAddrTypeSlashRecords:
			param := new(QueryConsAddrTypeParams)
			ctx, err = RequestPrepare(ctx, k, req, param)
			if err != nil {
				return res, err
			}
			return queryConsAddrTypeSlashRecords(ctx, k, param)
		default:
			return nil, sdk.ErrUnknownRequest("unknown slashing query endpoint")
		}
	}
}

// BaseParams
type BaseParams struct {
	SideChainId string
}

func NewBaseParams(sideChainId string) BaseParams {
	return BaseParams{
		SideChainId: sideChainId,
	}
}

func (p BaseParams) GetSideChainId() string {
	return p.SideChainId
}

type QueryConsAddrParams struct {
	BaseParams
	ConsAddr []byte
}

type QueryConsAddrTypeParams struct {
	BaseParams
	ConsAddr       []byte
	InfractionType byte
}

func RequestPrepare(ctx sdk.Context, k Keeper, req abci.RequestQuery, p types.SideChainIder) (newCtx sdk.Context, err sdk.Error) {
	if req.Data == nil || len(req.Data) == 0 {
		return ctx, nil
	}

	newCtx = ctx
	errRes := json.Unmarshal(req.Data, p)
	if errRes != nil {
		return newCtx, sdk.ErrInternal("can not unmarshal request")
	}
	if len(p.GetSideChainId()) != 0 {
		newCtx, err = prepareSideChainCtx(ctx, k, p.GetSideChainId())
		if err != nil {
			return newCtx, err
		}
	}
	return newCtx, nil
}

func prepareSideChainCtx(ctx sdk.Context, k Keeper, sideChainId string) (sdk.Context, sdk.Error) {
	scCtx, err := k.ScKeeper.PrepareCtxForSideChain(ctx, sideChainId)
	if err != nil {
		return sdk.Context{}, types.ErrInvalidSideChainId(k.Codespace)
	}
	return scCtx, nil
}

func queryConsAddrSlashRecords(ctx sdk.Context, k Keeper, params *QueryConsAddrParams) (res []byte, err sdk.Error) {
	slashRecords := k.getSlashRecordsByConsAddr(ctx, params.ConsAddr)
	if len(slashRecords) == 0 {
		return
	}

	res, resErr := codec.MarshalJSONIndent(k.cdc, slashRecords)
	if resErr != nil {
		return res, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", resErr.Error()))
	}

	return res, nil
}

func queryConsAddrTypeSlashRecords(ctx sdk.Context, k Keeper, params *QueryConsAddrTypeParams) (res []byte, err sdk.Error) {
	slashRecords := k.getSlashRecordsByConsAddrAndType(ctx, params.ConsAddr, params.InfractionType)
	if len(slashRecords) == 0 {
		return
	}

	res, resErr := codec.MarshalJSONIndent(k.cdc, slashRecords)
	if resErr != nil {
		return res, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", resErr.Error()))
	}

	return res, nil
}
