package querier

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keep "github.com/cosmos/cosmos-sdk/x/stake/keeper"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	addrAcc1, addrAcc2 = keep.Addrs[0], keep.Addrs[1]
	addrVal1, addrVal2 = sdk.ValAddress(keep.Addrs[0]), sdk.ValAddress(keep.Addrs[1])
	pk1, pk2           = keep.PKs[0], keep.PKs[1]
)

func newTestDelegatorQuery(delegatorAddr sdk.AccAddress) QueryDelegatorParams {
	return QueryDelegatorParams{
		DelegatorAddr: delegatorAddr,
	}
}

func newTestValidatorQuery(validatorAddr sdk.ValAddress) QueryValidatorParams {
	return QueryValidatorParams{
		ValidatorAddr: validatorAddr,
	}
}

func newTestBondQuery(delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress) QueryBondsParams {
	return QueryBondsParams{
		DelegatorAddr: delegatorAddr,
		ValidatorAddr: validatorAddr,
	}
}

func TestNewQuerier(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, 1000)
	pool := keeper.GetPool(ctx)
	// Create Validators
	amts := []int64{9, 8}
	var validators [2]types.Validator
	for i, amt := range amts {
		validators[i] = types.NewValidator(sdk.ValAddress(keep.Addrs[i]), keep.PKs[i], types.Description{})
		validators[i], pool, _ = validators[i].AddTokensFromDel(pool, sdk.NewDecWithoutFra(amt).RawInt())
		validators[i].BondIntraTxCounter = int16(i)
		keeper.SetValidator(ctx, validators[i])
		keeper.SetValidatorByPowerIndex(ctx, validators[i])
	}
	keeper.SetPool(ctx, pool)

	query := abci.RequestQuery{
		Path: "",
		Data: []byte{},
	}

	querier := NewQuerier(keeper, cdc)

	bz, err := querier(ctx, []string{"other"}, query)
	require.NotNil(t, err)
	require.Nil(t, bz)

	_, err = querier(ctx, []string{"validators"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"pool"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"parameters"}, query)
	require.Nil(t, err)

	queryValParams := newTestValidatorQuery(addrVal1)
	bz, errRes := json.Marshal(queryValParams)
	require.Nil(t, errRes)

	query.Path = "/custom/stake/validator"
	query.Data = bz

	_, err = querier(ctx, []string{"validator"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"validatorUnbondingDelegations"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"validatorRedelegations"}, query)
	require.Nil(t, err)

	queryDelParams := newTestDelegatorQuery(addrAcc2)
	bz, errRes = json.Marshal(queryDelParams)
	require.Nil(t, errRes)

	query.Path = "/custom/stake/validator"
	query.Data = bz

	_, err = querier(ctx, []string{"delegatorDelegations"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"delegatorUnbondingDelegations"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"delegatorRedelegations"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"delegatorValidators"}, query)
	require.Nil(t, err)
}

func TestQueryParametersPool(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, 1000)

	res, err := queryParameters(ctx, cdc, keeper)
	require.Nil(t, err)

	var params types.Params
	errRes := cdc.UnmarshalJSON(res, &params)
	require.Nil(t, errRes)
	require.Equal(t, keeper.GetParams(ctx), params)

	res, err = queryPool(ctx, cdc, keeper)
	require.Nil(t, err)

	var pool types.Pool
	errRes = cdc.UnmarshalJSON(res, &pool)
	require.Nil(t, errRes)
	require.Equal(t, keeper.GetPool(ctx), pool)
}

func TestQueryValidators(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, 10000)
	pool := keeper.GetPool(ctx)
	params := keeper.GetParams(ctx)

	// Create Validators
	amts := []int64{9, 8}
	var validators [2]types.Validator
	for i, amt := range amts {
		validators[i] = types.NewValidator(sdk.ValAddress(keep.Addrs[i]), keep.PKs[i], types.Description{})
		validators[i], pool, _ = validators[i].AddTokensFromDel(pool, sdk.NewDecWithoutFra(amt).RawInt())
	}
	keeper.SetPool(ctx, pool)
	keeper.SetValidator(ctx, validators[0])
	keeper.SetValidator(ctx, validators[1])

	// Query Validators
	queriedValidators := keeper.GetValidators(ctx, params.MaxValidators)

	res, err := queryValidators(ctx, cdc, keeper)
	require.Nil(t, err)

	var validatorsResp []types.Validator
	errRes := cdc.UnmarshalJSON(res, &validatorsResp)
	require.Nil(t, errRes)

	require.Equal(t, len(queriedValidators), len(validatorsResp))
	require.ElementsMatch(t, queriedValidators, validatorsResp)

	// Query each validator
	queryParams := newTestValidatorQuery(addrVal1)
	bz, errRes := json.Marshal(queryParams)
	require.Nil(t, errRes)

	querier := NewQuerier(keeper, cdc)

	query := abci.RequestQuery{
		Path: "/custom/stake/validator",
		Data: bz,
	}
	//res, err = queryValidator(ctx, cdc, query, keeper)
	res, err = querier(ctx, []string{"validator"}, query)
	require.Nil(t, err)

	var validator types.Validator
	errRes = cdc.UnmarshalJSON(res, &validator)
	require.Nil(t, errRes)

	require.Equal(t, queriedValidators[0], validator)
}

func TestQueryDelegation(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, 10000)
	params := keeper.GetParams(ctx)

	// Create Validators and Delegation
	val1 := types.NewValidator(addrVal1, pk1, types.Description{})
	keeper.SetValidator(ctx, val1)
	keeper.SetValidatorByPowerIndex(ctx, val1)

	keeper.Delegate(ctx, addrAcc2, sdk.NewCoin("steak", sdk.NewDecWithoutFra(20).RawInt()), val1, true)

	// apply TM updates
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)

	// Query Delegator bonded validators
	queryParams := newTestDelegatorQuery(addrAcc2)
	bz, errRes := json.Marshal(queryParams)
	require.Nil(t, errRes)

	querier := NewQuerier(keeper, cdc)

	query := abci.RequestQuery{
		Path: "/custom/stake/delegatorValidators",
		Data: bz,
	}

	delValidators := keeper.GetDelegatorValidators(ctx, addrAcc2, params.MaxValidators)

	res, err := querier(ctx, []string{"delegatorValidators"}, query)
	//res, err := queryDelegatorValidators(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var validatorsResp []types.Validator
	errRes = cdc.UnmarshalJSON(res, &validatorsResp)
	require.Nil(t, errRes)

	require.Equal(t, len(delValidators), len(validatorsResp))
	require.ElementsMatch(t, delValidators, validatorsResp)

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = querier(ctx, []string{"delegatorValidators"}, query)
	//_, err = queryDelegatorValidators(ctx, cdc, query, keeper)
	require.NotNil(t, err)

	// Query bonded validator
	queryBondParams := newTestBondQuery(addrAcc2, addrVal1)
	bz, errRes = json.Marshal(queryBondParams)
	require.Nil(t, errRes)

	query = abci.RequestQuery{
		Path: "/custom/stake/delegatorValidator",
		Data: bz,
	}

	res, err = querier(ctx, []string{"delegatorValidator"}, query)
	//res, err = queryDelegatorValidator(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var validator types.Validator
	errRes = cdc.UnmarshalJSON(res, &validator)
	require.Nil(t, errRes)

	require.Equal(t, delValidators[0], validator)

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = querier(ctx, []string{"delegatorValidator"}, query)
	//_, err = queryDelegatorValidator(ctx, cdc, query, keeper)
	require.NotNil(t, err)

	// Query delegation

	query = abci.RequestQuery{
		Path: "/custom/stake/delegation",
		Data: bz,
	}

	delegation, found := keeper.GetDelegation(ctx, addrAcc2, addrVal1)
	require.True(t, found)

	res, err = querier(ctx, []string{"delegation"}, query)
	//res, err = queryDelegation(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var delegationRes types.DelegationResponse
	errRes = cdc.UnmarshalJSON(res, &delegationRes)
	require.Nil(t, errRes)

	require.EqualValues(t, delegation.Shares, delegationRes.Shares)
	require.EqualValues(t, delegation.DelegatorAddr, delegationRes.DelegatorAddr)
	require.EqualValues(t, delegation.ValidatorAddr, delegationRes.ValidatorAddr)

	// Query Delegator Delegations

	query = abci.RequestQuery{
		Path: "/custom/stake/delegatorDelegations",
		Data: bz,
	}

	res, err = querier(ctx, []string{"delegatorDelegations"}, query)
	//res, err = queryDelegatorDelegations(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var delegatorDelegations []types.DelegationResponse
	errRes = cdc.UnmarshalJSON(res, &delegatorDelegations)
	require.Nil(t, errRes)
	require.Len(t, delegatorDelegations, 1)
	require.EqualValues(t, delegation.Shares, delegatorDelegations[0].Shares)
	require.EqualValues(t, delegation.DelegatorAddr, delegatorDelegations[0].DelegatorAddr)
	require.EqualValues(t, delegation.ValidatorAddr, delegatorDelegations[0].ValidatorAddr)

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = querier(ctx, []string{"delegatorDelegations"}, query)
	//_, err = queryDelegation(ctx, cdc, query, keeper)
	require.NotNil(t, err)

	// Query unbonging delegation
	keeper.BeginUnbonding(ctx, addrAcc2, val1.OperatorAddr, sdk.NewDec(sdk.NewDecWithoutFra(10).RawInt()))

	query = abci.RequestQuery{
		Path: "/custom/stake/unbondingDelegation",
		Data: bz,
	}

	unbond, found := keeper.GetUnbondingDelegation(ctx, addrAcc2, addrVal1)
	require.True(t, found)

	res, err = querier(ctx, []string{"unbondingDelegation"}, query)
	//res, err = queryUnbondingDelegation(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var unbondRes types.UnbondingDelegation
	errRes = cdc.UnmarshalJSON(res, &unbondRes)
	require.Nil(t, errRes)

	require.Equal(t, unbond, unbondRes)

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = querier(ctx, []string{"unbondingDelegation"}, query)
	//_, err = queryUnbondingDelegation(ctx, cdc, query, keeper)
	require.NotNil(t, err)

	// Query Delegator Delegations

	query = abci.RequestQuery{
		Path: "/custom/stake/delegatorUnbondingDelegations",
		Data: bz,
	}

	res, err = querier(ctx, []string{"delegatorUnbondingDelegations"}, query)
	//res, err = queryDelegatorUnbondingDelegations(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var delegatorUbds []types.UnbondingDelegation
	errRes = cdc.UnmarshalJSON(res, &delegatorUbds)
	require.Nil(t, errRes)
	require.Equal(t, unbond, delegatorUbds[0])

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = querier(ctx, []string{"delegatorUnbondingDelegations"}, query)
	//_, err = queryDelegatorUnbondingDelegations(ctx, cdc, query, keeper)
	require.NotNil(t, err)
}

func TestQueryRedelegations(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, 10000)

	// Create Validators and Delegation
	val1 := types.NewValidator(addrVal1, pk1, types.Description{})
	val2 := types.NewValidator(addrVal2, pk2, types.Description{})
	keeper.SetValidator(ctx, val1)
	keeper.SetValidator(ctx, val2)

	keeper.Delegate(ctx, addrAcc2, sdk.NewCoin("steak", sdk.NewDecWithoutFra(100).RawInt()), val1, true)
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)

	keeper.BeginRedelegation(ctx, addrAcc2, val1.GetOperator(), val2.GetOperator(), sdk.NewDec(sdk.NewDecWithoutFra(20).RawInt()))
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)

	redelegation, found := keeper.GetRedelegation(ctx, addrAcc2, val1.OperatorAddr, val2.OperatorAddr)
	require.True(t, found)

	// delegator redelegations
	queryDelegatorParams := newTestDelegatorQuery(addrAcc2)
	bz, errRes := json.Marshal(queryDelegatorParams)
	require.Nil(t, errRes)

	querier := NewQuerier(keeper, cdc)

	query := abci.RequestQuery{
		Path: "/custom/stake/delegatorRedelegations",
		Data: bz,
	}

	res, err := querier(ctx, []string{"delegatorRedelegations"}, query)
	//res, err := queryDelegatorRedelegations(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var redsRes []types.Redelegation
	errRes = cdc.UnmarshalJSON(res, &redsRes)
	require.Nil(t, errRes)

	require.Equal(t, redelegation, redsRes[0])

	// validator redelegations
	queryValidatorParams := newTestValidatorQuery(val1.GetOperator())
	bz, errRes = json.Marshal(queryValidatorParams)
	require.Nil(t, errRes)

	query = abci.RequestQuery{
		Path: "/custom/stake/validatorRedelegations",
		Data: bz,
	}

	res, err = querier(ctx, []string{"validatorRedelegations"}, query)
	require.Nil(t, err)

	errRes = cdc.UnmarshalJSON(res, &redsRes)
	require.Nil(t, errRes)

	require.Equal(t, redelegation, redsRes[0])
}
