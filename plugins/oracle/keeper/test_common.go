package keeper

import (
	"bytes"
	"log"
	"sort"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/ibc"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

// initialize the mock application for this module
func getMockApp(t *testing.T, numGenAccs int) (*mock.App, bank.BaseKeeper, Keeper, stake.Keeper, []sdk.AccAddress, []crypto.PubKey, []crypto.PrivKey) {
	mapp := mock.NewApp()

	stake.RegisterCodec(mapp.Cdc)

	keyGlobalParams := sdk.NewKVStoreKey("params")
	tkeyGlobalParams := sdk.NewTransientStoreKey("transient_params")
	keyStake := sdk.NewKVStoreKey("stake")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keyOracle := sdk.NewKVStoreKey("oracle")

	pk := params.NewKeeper(mapp.Cdc, keyGlobalParams, tkeyGlobalParams)
	ck := bank.NewBaseKeeper(mapp.AccountKeeper)
	sk := stake.NewKeeper(mapp.Cdc, keyStake, tkeyStake, ck, ibc.Keeper{}, nil, pk.Subspace(stake.DefaultParamspace), mapp.RegisterCodespace(stake.DefaultCodespace))

	mapp.SetInitChainer(getInitChainer(mapp, sk))

	require.NoError(t, mapp.CompleteSetup(keyStake, tkeyStake, keyOracle, keyGlobalParams, tkeyGlobalParams))
	genAccs, addrs, pubKeys, privKeys := mock.CreateGenAccounts(numGenAccs, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5000e8)})

	mock.SetGenesis(mapp, genAccs)

	oracleKeeper := NewKeeper(mapp.Cdc, keyOracle, pk.Subspace("testoracle"), sk)

	//ctx := mapp.BaseApp.NewContext(sdk.RunTxModeCheck, abci.Header{})
	//valAddrs := make([]sdk.ValAddress, numGenAccs)
	//pool := stake.InitialPool()
	//for i, _ := range pubKeys {
	//	valPubKey := pubKeys[i]
	//	valAddr := sdk.ValAddress(valPubKey.Address().Bytes())
	//	valAddrs[i] = valAddr
	//	// test how the validator is set from a purely unbonbed pool
	//	validator := stake.NewValidator(valAddr, valPubKey, stake.Description{})
	//	validator, _, _ = validator.AddTokensFromDel(pool, 5000e8)
	//	sk.SetValidator(ctx, validator)
	//	sk.SetValidatorByPowerIndex(ctx, validator, pool)
	//	sk.ApplyAndReturnValidatorSetUpdates(ctx)
	//}

	return mapp, ck, oracleKeeper, sk, addrs, pubKeys, privKeys
}

func getInitChainer(mapp *mock.App, stakeKeeper stake.Keeper) sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		mapp.InitChainer(ctx, req)

		stakeGenesis := stake.DefaultGenesisState()
		stakeGenesis.Pool.LooseTokens = sdk.NewDecWithoutFra(100000)

		validators, err := stake.InitGenesis(ctx, stakeKeeper, stakeGenesis)
		if err != nil {
			panic(err)
		}
		return abci.ResponseInitChain{
			Validators: validators,
		}
	}
}

// TODO: Remove once address interface has been implemented (ref: #2186)
func SortValAddresses(addrs []sdk.ValAddress) {
	var byteAddrs [][]byte
	for _, addr := range addrs {
		byteAddrs = append(byteAddrs, addr.Bytes())
	}

	SortByteArrays(byteAddrs)

	for i, byteAddr := range byteAddrs {
		addrs[i] = byteAddr
	}
}

// Sorts Addresses
func SortAddresses(addrs []sdk.AccAddress) {
	var byteAddrs [][]byte
	for _, addr := range addrs {
		byteAddrs = append(byteAddrs, addr.Bytes())
	}
	SortByteArrays(byteAddrs)
	for i, byteAddr := range byteAddrs {
		addrs[i] = byteAddr
	}
}

// implement `Interface` in sort package.
type sortByteArrays [][]byte

func (b sortByteArrays) Len() int {
	return len(b)
}

func (b sortByteArrays) Less(i, j int) bool {
	// bytes package already implements Comparable for []byte.
	switch bytes.Compare(b[i], b[j]) {
	case -1:
		return true
	case 0, 1:
		return false
	default:
		log.Panic("not fail-able with `bytes.Comparable` bounded [-1, 1].")
		return false
	}
}

func (b sortByteArrays) Swap(i, j int) {
	b[j], b[i] = b[i], b[j]
}

// Public
func SortByteArrays(src [][]byte) [][]byte {
	sorted := sortByteArrays(src)
	sort.Sort(sorted)
	return sorted
}
