package querier

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keep "github.com/cosmos/cosmos-sdk/x/stake/keeper"
	"github.com/cosmos/cosmos-sdk/x/stake/types"

	abci "github.com/tendermint/tendermint/abci/types"
)

// query endpoints supported by the staking Querier
const (
	QueryValidators                    = "validators"
	QueryValidator                     = "validator"
	QueryDelegatorDelegations          = "delegatorDelegations"
	QueryDelegatorUnbondingDelegations = "delegatorUnbondingDelegations"
	QueryDelegatorRedelegations        = "delegatorRedelegations"
	QueryValidatorUnbondingDelegations = "validatorUnbondingDelegations"
	QueryValidatorRedelegations        = "validatorRedelegations"
	QueryDelegator                     = "delegator"
	QueryDelegation                    = "delegation"
	QueryUnbondingDelegation           = "unbondingDelegation"
	QueryDelegatorValidators           = "delegatorValidators"
	QueryDelegatorValidator            = "delegatorValidator"
	QueryPool                          = "pool"
	QueryParameters                    = "parameters"
	QueryTopValidators                 = "topValidators"
	QueryAllValidatorsCount            = "allValidatorsCount"
	QueryAllUnJailValidatorsCount      = "allUnJailValidatorsCount"
)

// creates a querier for staking REST endpoints
func NewQuerier(k keep.Keeper, cdc *codec.Codec) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QueryValidators:
			p := new(BaseParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryValidators(ctx, cdc, k)
		case QueryValidator:
			p := new(QueryValidatorParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryValidator(ctx, cdc, p, k)
		case QueryValidatorUnbondingDelegations:
			p := new(QueryValidatorParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryValidatorUnbondingDelegations(ctx, cdc, p, k)
		case QueryValidatorRedelegations:
			p := new(QueryValidatorParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryValidatorRedelegations(ctx, cdc, p, k)
		case QueryDelegation:
			p := new(QueryBondsParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryDelegation(ctx, cdc, p, k)
		case QueryUnbondingDelegation:
			p := new(QueryBondsParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryUnbondingDelegation(ctx, cdc, p, k)
		case QueryDelegatorDelegations:
			p := new(QueryDelegatorParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryDelegatorDelegations(ctx, cdc, p, k)
		case QueryDelegatorUnbondingDelegations:
			p := new(QueryDelegatorParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryDelegatorUnbondingDelegations(ctx, cdc, p, k)
		case QueryDelegatorRedelegations:
			p := new(QueryDelegatorParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryDelegatorRedelegations(ctx, cdc, p, k)
		case QueryDelegatorValidators:
			p := new(QueryDelegatorParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryDelegatorValidators(ctx, cdc, p, k)
		case QueryDelegatorValidator:
			p := new(QueryBondsParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryDelegatorValidator(ctx, cdc, p, k)
		case QueryPool:
			p := new(BaseParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryPool(ctx, cdc, k)
		case QueryParameters:
			p := new(BaseParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryParameters(ctx, cdc, k)
		case QueryTopValidators:
			p := new(QueryTopValidatorsParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryTopValidators(ctx, cdc, p, k)
		case QueryAllValidatorsCount:
			p := new(BaseParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryAllValidatorsCount(ctx, cdc, k)
		case QueryAllUnJailValidatorsCount:
			p := new(BaseParams)
			ctx, err = RequestPrepare(ctx, k, req, p)
			if err != nil {
				return res, err
			}
			return queryAllUnJailValidatorsCount(ctx, cdc, k)
		default:
			return nil, sdk.ErrUnknownRequest("unknown stake query endpoint")
		}
	}
}

func RequestPrepare(ctx sdk.Context, k keep.Keeper, req abci.RequestQuery, p types.SideChainIder) (newCtx sdk.Context, err sdk.Error) {
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

// defines the params for the following queries:
// - 'custom/stake/delegatorDelegations'
// - 'custom/stake/delegatorUnbondingDelegations'
// - 'custom/stake/delegatorRedelegations'
// - 'custom/stake/delegatorValidators'
type QueryDelegatorParams struct {
	BaseParams
	DelegatorAddr sdk.AccAddress
}

// defines the params for the following queries:
// - 'custom/stake/validator'
// - 'custom/stake/validatorUnbondingDelegations'
// - 'custom/stake/validatorRedelegations'
type QueryValidatorParams struct {
	BaseParams
	ValidatorAddr sdk.ValAddress
}

// defines the params for the following queries:
// - 'custom/stake/delegation'
// - 'custom/stake/unbondingDelegation'
// - 'custom/stake/delegatorValidator'
type QueryBondsParams struct {
	BaseParams
	DelegatorAddr sdk.AccAddress
	ValidatorAddr sdk.ValAddress
}

// defines the params for 'custom/stake/topValidators'
type QueryTopValidatorsParams struct {
	BaseParams
	Top int
}

func queryValidators(ctx sdk.Context, cdc *codec.Codec, k keep.Keeper) (res []byte, err sdk.Error) {
	stakeParams := k.GetParams(ctx)
	validators := k.GetValidators(ctx, stakeParams.MaxValidators)

	res, errRes := codec.MarshalJSONIndent(cdc, validators)
	if err != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryValidator(ctx sdk.Context, cdc *codec.Codec, params *QueryValidatorParams, k keep.Keeper) (res []byte, err sdk.Error) {

	validator, found := k.GetValidator(ctx, params.ValidatorAddr)
	if !found {
		return []byte{}, types.ErrNoValidatorFound(types.DefaultCodespace)
	}

	res, errRes := codec.MarshalJSONIndent(cdc, validator)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryValidatorUnbondingDelegations(ctx sdk.Context, cdc *codec.Codec, params *QueryValidatorParams, k keep.Keeper) (res []byte, err sdk.Error) {

	unbonds := k.GetUnbondingDelegationsFromValidator(ctx, params.ValidatorAddr)

	res, errRes := codec.MarshalJSONIndent(cdc, unbonds)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryValidatorRedelegations(ctx sdk.Context, cdc *codec.Codec, params *QueryValidatorParams, k keep.Keeper) (res []byte, err sdk.Error) {
	redelegations := k.GetRedelegationsFromValidator(ctx, params.ValidatorAddr)

	res, errRes := codec.MarshalJSONIndent(cdc, redelegations)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryDelegatorDelegations(ctx sdk.Context, cdc *codec.Codec, params *QueryDelegatorParams, k keep.Keeper) (res []byte, err sdk.Error) {
	delegations := k.GetAllDelegatorDelegations(ctx, params.DelegatorAddr)
	delResponses, err := delegationsToDelegationResponses(ctx, k, delegations)
	if err != nil {
		return res, err
	}

	res, errRes := codec.MarshalJSONIndent(cdc, delResponses)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryDelegatorUnbondingDelegations(ctx sdk.Context, cdc *codec.Codec, params *QueryDelegatorParams, k keep.Keeper) (res []byte, err sdk.Error) {
	unbondingDelegations := k.GetAllUnbondingDelegations(ctx, params.DelegatorAddr)

	res, errRes := codec.MarshalJSONIndent(cdc, unbondingDelegations)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryDelegatorRedelegations(ctx sdk.Context, cdc *codec.Codec, params *QueryDelegatorParams, k keep.Keeper) (res []byte, err sdk.Error) {
	redelegations := k.GetAllRedelegations(ctx, params.DelegatorAddr)

	res, errRes := codec.MarshalJSONIndent(cdc, redelegations)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryDelegatorValidators(ctx sdk.Context, cdc *codec.Codec, params *QueryDelegatorParams, k keep.Keeper) (res []byte, err sdk.Error) {
	stakeParams := k.GetParams(ctx)
	validators := k.GetDelegatorValidators(ctx, params.DelegatorAddr, stakeParams.MaxValidators)

	res, errRes := codec.MarshalJSONIndent(cdc, validators)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryDelegatorValidator(ctx sdk.Context, cdc *codec.Codec, params *QueryBondsParams, k keep.Keeper) (res []byte, err sdk.Error) {

	validator, err := k.GetDelegatorValidator(ctx, params.DelegatorAddr, params.ValidatorAddr)
	if err != nil {
		return
	}

	res, errRes := codec.MarshalJSONIndent(cdc, validator)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryDelegation(ctx sdk.Context, cdc *codec.Codec, params *QueryBondsParams, k keep.Keeper) (res []byte, err sdk.Error) {
	delegation, found := k.GetDelegation(ctx, params.DelegatorAddr, params.ValidatorAddr)
	if !found {
		return []byte{}, types.ErrNoDelegation(types.DefaultCodespace)
	}

	delResponse, err := delegationToDelegationResponse(ctx, k, delegation)
	if err != nil {
		return res, err
	}

	res, errRes := codec.MarshalJSONIndent(cdc, delResponse)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryUnbondingDelegation(ctx sdk.Context, cdc *codec.Codec, params *QueryBondsParams, k keep.Keeper) (res []byte, err sdk.Error) {

	unbond, found := k.GetUnbondingDelegation(ctx, params.DelegatorAddr, params.ValidatorAddr)
	if !found {
		return []byte{}, types.ErrNoUnbondingDelegation(types.DefaultCodespace)
	}

	res, errRes := codec.MarshalJSONIndent(cdc, unbond)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryPool(ctx sdk.Context, cdc *codec.Codec, k keep.Keeper) (res []byte, err sdk.Error) {
	pool := k.GetPool(ctx)

	res, errRes := codec.MarshalJSONIndent(cdc, pool)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryParameters(ctx sdk.Context, cdc *codec.Codec, k keep.Keeper) (res []byte, err sdk.Error) {

	params := k.GetParams(ctx)

	res, errRes := codec.MarshalJSONIndent(cdc, params)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryTopValidators(ctx sdk.Context, cdc *codec.Codec, params *QueryTopValidatorsParams, k keep.Keeper) (res []byte, err sdk.Error) {

	if params.Top == 0 {
		params.Top = int(k.MaxValidators(ctx))
	}

	if params.Top > 50 || params.Top < 1 {
		return []byte{}, sdk.ErrInternal("top must be between 1 and 50")
	}

	validators := k.GetTopValidatorsByPower(ctx, params.Top)

	res, errRes := codec.MarshalJSONIndent(cdc, validators)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil

}

func queryAllValidatorsCount(ctx sdk.Context, cdc *codec.Codec, k keep.Keeper) ([]byte, sdk.Error) {

	count := k.GetAllValidatorsCount(ctx)
	res, errRes := codec.MarshalJSONIndent(cdc, count)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func queryAllUnJailValidatorsCount(ctx sdk.Context, cdc *codec.Codec, k keep.Keeper) ([]byte, sdk.Error) {

	count := k.GetAllUnJailValidatorsCount(ctx)
	res, errRes := codec.MarshalJSONIndent(cdc, count)
	if errRes != nil {
		return nil, sdk.ErrInternal(sdk.AppendMsgToErr("could not marshal result to JSON", errRes.Error()))
	}
	return res, nil
}

func prepareSideChainCtx(ctx sdk.Context, k keep.Keeper, sideChainId string) (sdk.Context, sdk.Error) {
	scCtx, err := k.ScKeeper.PrepareCtxForSideChain(ctx, sideChainId)
	if err != nil {
		return sdk.Context{}, types.ErrInvalidSideChainId(k.Codespace())
	}
	return scCtx, nil
}

//______________________________________________________
// util

func delegationToDelegationResponse(ctx sdk.Context, k keep.Keeper, del types.Delegation) (types.DelegationResponse, sdk.Error) {
	val, found := k.GetValidator(ctx, del.ValidatorAddr)
	if !found {
		return types.DelegationResponse{}, types.ErrNoValidatorFound(k.Codespace())
	}

	return types.NewDelegationResp(
		del.DelegatorAddr,
		del.ValidatorAddr,
		del.Shares,
		sdk.NewCoin(k.BondDenom(ctx), val.TokensFromShares(del.Shares).RawInt()),
	), nil
}

func delegationsToDelegationResponses(
	ctx sdk.Context, k keep.Keeper, delegations []types.Delegation,
) ([]types.DelegationResponse, sdk.Error) {

	resp := make([]types.DelegationResponse, len(delegations))
	for i, del := range delegations {
		delResp, err := delegationToDelegationResponse(ctx, k, del)
		if err != nil {
			return nil, err
		}

		resp[i] = delResp
	}

	return resp, nil
}
