package common

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	MainStoreName        = "main"
	AccountStoreName     = "acc"
	ValAddrStoreName     = "val"
	TokenStoreName       = "tokens"
	DexStoreName         = "dex"
	PairStoreName        = "pairs"
	StakeStoreName       = "stake"
	StakeRewardStoreName = "stake_reward"
	SlashingStoreName    = "slashing"
	ParamsStoreName      = "params"
	GovStoreName         = "gov"
	TimeLockStoreName    = "time_lock"
	AtomicSwapStoreName  = "atomic_swap"
	BridgeStoreName      = "bridge"
	OracleStoreName      = "oracle"
	IbcStoreName         = "ibc"
	SideChainStoreName   = "sc"
	ReconStoreName       = "recon"

	StakeTransientStoreName  = "transient_stake"
	ParamsTransientStoreName = "transient_params"
)

var (
	// keys to access the substores
	MainStoreKey        = sdk.NewKVStoreKey(MainStoreName)
	AccountStoreKey     = sdk.NewKVStoreKey(AccountStoreName)
	ValAddrStoreKey     = sdk.NewKVStoreKey(ValAddrStoreName)
	TokenStoreKey       = sdk.NewKVStoreKey(TokenStoreName)
	DexStoreKey         = sdk.NewKVStoreKey(DexStoreName)
	PairStoreKey        = sdk.NewKVStoreKey(PairStoreName)
	StakeStoreKey       = sdk.NewKVStoreKey(StakeStoreName)
	StakeRewardStoreKey = sdk.NewKVStoreKey(StakeRewardStoreName)
	SlashingStoreKey    = sdk.NewKVStoreKey(SlashingStoreName)
	ParamsStoreKey      = sdk.NewKVStoreKey(ParamsStoreName)
	GovStoreKey         = sdk.NewKVStoreKey(GovStoreName)
	TimeLockStoreKey    = sdk.NewKVStoreKey(TimeLockStoreName)
	AtomicSwapStoreKey  = sdk.NewKVStoreKey(AtomicSwapStoreName)
	BridgeStoreKey      = sdk.NewKVStoreKey(BridgeStoreName)
	OracleStoreKey      = sdk.NewKVStoreKey(OracleStoreName)
	IbcStoreKey         = sdk.NewKVStoreKey(IbcStoreName)
	SideChainStoreKey   = sdk.NewKVStoreKey(SideChainStoreName)
	ReconStoreKey       = sdk.NewKVStoreKey(ReconStoreName)

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
		StakeRewardStoreName:     StakeRewardStoreKey,
		SlashingStoreName:        SlashingStoreKey,
		ParamsStoreName:          ParamsStoreKey,
		GovStoreName:             GovStoreKey,
		TimeLockStoreName:        TimeLockStoreKey,
		AtomicSwapStoreName:      AtomicSwapStoreKey,
		IbcStoreName:             IbcStoreKey,
		SideChainStoreName:       SideChainStoreKey,
		BridgeStoreName:          BridgeStoreKey,
		OracleStoreName:          OracleStoreKey,
		ReconStoreName:           ReconStoreKey,
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
		StakeRewardStoreName,
		SlashingStoreName,
		ParamsStoreName,
		GovStoreName,
		TimeLockStoreName,
		AtomicSwapStoreName,
		IbcStoreName,
		SideChainStoreName,
		BridgeStoreName,
		OracleStoreName,
		ReconStoreName,
	}
)

func GetNonTransientStoreKeys() []sdk.StoreKey {
	storeKeys := make([]sdk.StoreKey, 0, len(NonTransientStoreKeyNames))
	for _, name := range NonTransientStoreKeyNames {
		storeKeys = append(storeKeys, StoreKeyNameMap[name])
	}
	return storeKeys
}
