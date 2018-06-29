package common

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	MainStoreName    = "main"
	AccountStoreName = "acc"
	TokenStoreName   = "tokens"
	DexStoreName     = "dex"
	// StakeStoreName   = "stake"
)

var (
	// keys to access the substores
	MainStoreKey    = sdk.NewKVStoreKey(MainStoreName)
	AccountStoreKey = sdk.NewKVStoreKey(AccountStoreName)
	// StakingStoreKey = sdk.NewKVStoreKey(StakeStoreName)
	TokenStoreKey = sdk.NewKVStoreKey(TokenStoreName)
	DexStoreKey   = sdk.NewKVStoreKey(DexStoreName)
)
