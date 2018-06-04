package common

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	MainStoreName    = "main"
	AccountStoreName = "acc"
	TokenStoreName   = "tokens"
	IBCStoreName     = "ibc"
	StakeStoreName   = "stake"
)

var (
	// keys to access the substores
	MainStoreKey    = sdk.NewKVStoreKey(MainStoreName)
	AccountStoreKey = sdk.NewKVStoreKey(AccountStoreName)
	IBCStoreKey     = sdk.NewKVStoreKey(IBCStoreName)
	StakingStoreKey = sdk.NewKVStoreKey(StakeStoreName)
	TokenStoreKey   = sdk.NewKVStoreKey(TokenStoreName)
)
