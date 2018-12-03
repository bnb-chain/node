package common

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	MainStoreName    = "main"
	AccountStoreName = "acc"
	ValAddrStoreName = "val"
	TokenStoreName   = "tokens"
	DexStoreName     = "dex"
	PairStoreName    = "pairs"
	StakeStoreName   = "stake"
	ParamsStoreName  = "params"
	GovStoreName     = "gov"

	StakeTransientStoreName  = "transient_stake"
	ParamsTransientStoreName = "transient_params"
)

var (
	// keys to access the substores
	MainStoreKey    = sdk.NewKVStoreKey(MainStoreName)
	AccountStoreKey = sdk.NewKVStoreKey(AccountStoreName)
	ValAddrStoreKey = sdk.NewKVStoreKey(ValAddrStoreName)
	TokenStoreKey   = sdk.NewKVStoreKey(TokenStoreName)
	DexStoreKey     = sdk.NewKVStoreKey(DexStoreName)
	PairStoreKey    = sdk.NewKVStoreKey(PairStoreName)
	StakeStoreKey   = sdk.NewKVStoreKey(StakeStoreName)
	ParamsStoreKey  = sdk.NewKVStoreKey(ParamsStoreName)
	GovStoreKey     = sdk.NewKVStoreKey(GovStoreName)

	TStakeStoreKey  = sdk.NewTransientStoreKey(StakeTransientStoreName)
	TParamsStoreKey = sdk.NewTransientStoreKey(ParamsTransientStoreName)
)
