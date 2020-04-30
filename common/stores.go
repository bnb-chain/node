package common

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	MainStoreName          = "main"
	AccountStoreName       = "acc"
	ValAddrStoreName       = "val"
	TokenStoreName         = "tokens"
	MiniTokenStoreName     = "minitokens"
	DexStoreName           = "dex"
	DexMiniStoreName       = "dex_mini"
	PairStoreName          = "pairs"
	MiniTokenPairStoreName = "mini_pairs"
	StakeStoreName         = "stake"
	ParamsStoreName        = "params"
	GovStoreName           = "gov"
	TimeLockStoreName      = "time_lock"
	AtomicSwapStoreName    = "atomic_swap"

	StakeTransientStoreName  = "transient_stake"
	ParamsTransientStoreName = "transient_params"
)

var (
	// keys to access the substores
	MainStoreKey       = sdk.NewKVStoreKey(MainStoreName)
	AccountStoreKey    = sdk.NewKVStoreKey(AccountStoreName)
	ValAddrStoreKey    = sdk.NewKVStoreKey(ValAddrStoreName)
	TokenStoreKey      = sdk.NewKVStoreKey(TokenStoreName)
	DexStoreKey        = sdk.NewKVStoreKey(DexStoreName)
	DexMiniStoreKey    = sdk.NewKVStoreKey(DexMiniStoreName)
	PairStoreKey       = sdk.NewKVStoreKey(PairStoreName)
	StakeStoreKey      = sdk.NewKVStoreKey(StakeStoreName)
	ParamsStoreKey     = sdk.NewKVStoreKey(ParamsStoreName)
	GovStoreKey        = sdk.NewKVStoreKey(GovStoreName)
	TimeLockStoreKey   = sdk.NewKVStoreKey(TimeLockStoreName)
	AtomicSwapStoreKey = sdk.NewKVStoreKey(AtomicSwapStoreName)

	TStakeStoreKey  = sdk.NewTransientStoreKey(StakeTransientStoreName)
	TParamsStoreKey = sdk.NewTransientStoreKey(ParamsTransientStoreName)

	MiniTokenStoreKey     = sdk.NewKVStoreKey(MiniTokenStoreName)
	MiniTokenPairStoreKey = sdk.NewKVStoreKey(MiniTokenPairStoreName)

	StoreKeyNameMap = map[string]sdk.StoreKey{
		MainStoreName:            MainStoreKey,
		AccountStoreName:         AccountStoreKey,
		ValAddrStoreName:         ValAddrStoreKey,
		TokenStoreName:           TokenStoreKey,
		DexStoreName:             DexStoreKey,
		DexMiniStoreName:         DexMiniStoreKey,
		PairStoreName:            PairStoreKey,
		StakeStoreName:           StakeStoreKey,
		ParamsStoreName:          ParamsStoreKey,
		GovStoreName:             GovStoreKey,
		TimeLockStoreName:        TimeLockStoreKey,
		AtomicSwapStoreName:      AtomicSwapStoreKey,
		StakeTransientStoreName:  TStakeStoreKey,
		ParamsTransientStoreName: TParamsStoreKey,
		MiniTokenStoreName:       MiniTokenStoreKey,
		MiniTokenPairStoreName:   MiniTokenPairStoreKey,
	}

	NonTransientStoreKeyNames = []string{
		MainStoreName,
		AccountStoreName,
		ValAddrStoreName,
		TokenStoreName,
		DexStoreName,
		DexMiniStoreName,
		PairStoreName,
		StakeStoreName,
		ParamsStoreName,
		GovStoreName,
		TimeLockStoreName,
		AtomicSwapStoreName,
		MiniTokenStoreName,
		MiniTokenPairStoreName,
	}
)

func GetNonTransientStoreKeys() []sdk.StoreKey {
	storeKeys := make([]sdk.StoreKey, 0, len(NonTransientStoreKeyNames))
	for _, name := range NonTransientStoreKeyNames {
		storeKeys = append(storeKeys, StoreKeyNameMap[name])
	}
	return storeKeys
}
