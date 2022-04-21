package gov_test

import (
	"bytes"
	"log"
	"sort"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

// initialize the mock application for this module
func getMockApp(t *testing.T, numGenAccs int) (*mock.App, bank.BaseKeeper, gov.Keeper, stake.Keeper, []sdk.AccAddress, []crypto.PubKey, []crypto.PrivKey) {
	mapp := mock.NewApp()

	stake.RegisterCodec(mapp.Cdc)
	gov.RegisterCodec(mapp.Cdc)

	keyGlobalParams := sdk.NewKVStoreKey("params")
	tkeyGlobalParams := sdk.NewTransientStoreKey("transient_params")
	keyStake := sdk.NewKVStoreKey("stake")
	keyStakeReward := sdk.NewKVStoreKey("stake_reward")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keyGov := sdk.NewKVStoreKey("gov")
	keyIbc := sdk.NewKVStoreKey("ibc")
	keySideChain := sdk.NewKVStoreKey("sc")

	pk := params.NewKeeper(mapp.Cdc, keyGlobalParams, tkeyGlobalParams)
	ck := bank.NewBaseKeeper(mapp.AccountKeeper)
	scKeeper := sidechain.NewKeeper(keySideChain, pk.Subspace(sidechain.DefaultParamspace), mapp.Cdc)
	ibcKeeper := ibc.NewKeeper(keyIbc, pk.Subspace(ibc.DefaultParamspace), ibc.DefaultCodespace, scKeeper)

	sk := stake.NewKeeper(mapp.Cdc, keyStake, keyStakeReward, tkeyStake, ck, nil, pk.Subspace(stake.DefaultParamspace), mapp.RegisterCodespace(stake.DefaultCodespace))
	sk.SetupForSideChain(&scKeeper, &ibcKeeper)
	keeper := gov.NewKeeper(mapp.Cdc, keyGov, pk, pk.Subspace("testgov"), ck, sk, gov.DefaultCodespace, new(sdk.Pool))

	mapp.Router().AddRoute("gov", gov.NewHandler(keeper))

	mapp.SetEndBlocker(getEndBlocker(keeper))
	mapp.SetInitChainer(getInitChainer(mapp, keeper, sk))

	require.NoError(t, mapp.CompleteSetup(keyStake, tkeyStake, keyGov, keyGlobalParams, tkeyGlobalParams))

	genAccs, addrs, pubKeys, privKeys := mock.CreateGenAccounts(numGenAccs, sdk.Coins{sdk.NewCoin(gov.DefaultDepositDenom, 5000e8)})

	mock.SetGenesis(mapp, genAccs)

	return mapp, ck, keeper, sk, addrs, pubKeys, privKeys
}

// gov and stake endblocker
func getEndBlocker(keeper gov.Keeper) sdk.EndBlocker {
	return func(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
		gov.EndBlocker(ctx, keeper)
		return abci.ResponseEndBlock{
			Events: ctx.EventManager().ABCIEvents(),
		}
	}
}

// gov and stake initchainer
func getInitChainer(mapp *mock.App, keeper gov.Keeper, stakeKeeper stake.Keeper) sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		mapp.InitChainer(ctx, req)

		stakeGenesis := stake.DefaultGenesisState()
		stakeGenesis.Pool.LooseTokens = sdk.NewDecWithoutFra(100000)

		validators, err := stake.InitGenesis(ctx, stakeKeeper, stakeGenesis)
		if err != nil {
			panic(err)
		}
		gov.InitGenesis(ctx, keeper, gov.DefaultGenesisState())
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
