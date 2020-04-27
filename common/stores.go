package common

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	MainStoreName       = "main"
	AccountStoreName    = "acc"
	ValAddrStoreName    = "val"
	TokenStoreName      = "tokens"
	DexStoreName        = "dex"
	PairStoreName       = "pairs"
	StakeStoreName      = "stake"
	SlashingStoreName   = "slashing"
	ParamsStoreName     = "params"
	GovStoreName        = "gov"
	TimeLockStoreName   = "time_lock"
	AtomicSwapStoreName = "atomic_swap"
	IbcStoreName        = "ibc"
	SideChainStoreName  = "sc"

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
	PairStoreKey       = sdk.NewKVStoreKey(PairStoreName)
	StakeStoreKey      = sdk.NewKVStoreKey(StakeStoreName)
	SlashingStoreKey   = sdk.NewKVStoreKey(SlashingStoreName)
	ParamsStoreKey     = sdk.NewKVStoreKey(ParamsStoreName)
	GovStoreKey        = sdk.NewKVStoreKey(GovStoreName)
	TimeLockStoreKey   = sdk.NewKVStoreKey(TimeLockStoreName)
	AtomicSwapStoreKey = sdk.NewKVStoreKey(AtomicSwapStoreName)
	IbcStoreKey        = sdk.NewKVStoreKey(IbcStoreName)
	SideChainStoreKey  = sdk.NewKVStoreKey(SideChainStoreName)

	TStakeStoreKey  = sdk.NewTransientStoreKey(StakeTransientStoreName)
	TParamsStoreKey = sdk.NewTransientStoreKey(ParamsTransientStoreName)

	StoreKeyNameMap = map[string]sdk.StoreKey{
		MainStoreName:            MainStoreKey,
		AccountStoreName:         AccountStoreKey,
		ValAddrStoreName:         ValAddrStoreKey,
		TokenStoreName:           TokenStoreKey,
		DexStoreName:             DexStoreKey,
		PairStoreName:            PairStoreKey,
		StakeStoreName:           StakeStoreKey,
		SlashingStoreName:        SlashingStoreKey,
		ParamsStoreName:          ParamsStoreKey,
		GovStoreName:             GovStoreKey,
		TimeLockStoreName:        TimeLockStoreKey,
		AtomicSwapStoreName:      AtomicSwapStoreKey,
		IbcStoreName:             IbcStoreKey,
		SideChainStoreName:       SideChainStoreKey,
		StakeTransientStoreName:  TStakeStoreKey,
		ParamsTransientStoreName: TParamsStoreKey,
	}

	NonTransientStoreKeyNames = []string{
		MainStoreName,
		AccountStoreName,
		ValAddrStoreName,
		TokenStoreName,
		DexStoreName,
		PairStoreName,
		StakeStoreName,
		SlashingStoreName,
		ParamsStoreName,
		GovStoreName,
		TimeLockStoreName,
		AtomicSwapStoreName,
		IbcStoreName,
		SideChainStoreName,
	}
)

func GetNonTransientStoreKeys() []sdk.StoreKey {
	storeKeys := make([]sdk.StoreKey, 0, len(NonTransientStoreKeyNames))
	for _, name := range NonTransientStoreKeyNames {
		storeKeys = append(storeKeys, StoreKeyNameMap[name])
	}
	return storeKeys
}
