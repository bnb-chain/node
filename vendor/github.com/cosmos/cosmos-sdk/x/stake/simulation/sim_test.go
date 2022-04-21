package simulation

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/cosmos/cosmos-sdk/x/mock/simulation"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

// TestStakeWithRandomMessages
func TestStakeWithRandomMessages(t *testing.T) {
	mapp := mock.NewApp()

	bank.RegisterCodec(mapp.Cdc)
	mapper := mapp.AccountKeeper
	bankKeeper := bank.NewBaseKeeper(mapper)
	feeKey := sdk.NewKVStoreKey("fee")
	stakeKey := sdk.NewKVStoreKey("stake")
	stakeRewardKey := sdk.NewKVStoreKey("stake_reward")
	stakeTKey := sdk.NewTransientStoreKey("transient_stake")
	paramsKey := sdk.NewKVStoreKey("params")
	paramsTKey := sdk.NewTransientStoreKey("transient_params")
	ibcKey := sdk.NewKVStoreKey("ibc")
	keySideChain := sdk.NewKVStoreKey("sc")
	distrKey := sdk.NewKVStoreKey("distr")

	feeCollectionKeeper := auth.NewFeeCollectionKeeper(mapp.Cdc, feeKey)
	paramstore := params.NewKeeper(mapp.Cdc, paramsKey, paramsTKey)
	scKeeper := sidechain.NewKeeper(keySideChain, paramstore.Subspace(sidechain.DefaultParamspace), mapp.Cdc)
	ibcKeeper := ibc.NewKeeper(ibcKey, paramstore.Subspace(ibc.DefaultParamspace), ibc.DefaultCodespace, scKeeper)
	stakeKeeper := stake.NewKeeper(mapp.Cdc, stakeKey, stakeRewardKey, stakeTKey, bankKeeper, nil, paramstore.Subspace(stake.DefaultParamspace), stake.DefaultCodespace)
	stakeKeeper.SetupForSideChain(&scKeeper, &ibcKeeper)
	distrKeeper := distribution.NewKeeper(mapp.Cdc, distrKey, paramstore.Subspace(distribution.DefaultParamspace), bankKeeper, stakeKeeper, feeCollectionKeeper, distribution.DefaultCodespace)
	mapp.Router().AddRoute("stake", stake.NewStakeHandler(stakeKeeper))
	mapp.SetEndBlocker(func(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
		validatorUpdates, _ := stake.EndBlocker(ctx, stakeKeeper)
		return abci.ResponseEndBlock{
			ValidatorUpdates: validatorUpdates,
		}
	})

	err := mapp.CompleteSetup(stakeKey, stakeTKey, paramsKey, paramsTKey)
	if err != nil {
		panic(err)
	}

	appStateFn := func(r *rand.Rand, accs []simulation.Account) json.RawMessage {
		simulation.RandomSetGenesis(r, mapp, accs, []string{"stake"})
		return json.RawMessage("{}")
	}

	simulation.Simulate(
		t, mapp.BaseApp, appStateFn,
		[]simulation.WeightedOperation{
			{10, SimulateMsgCreateValidator(mapper, stakeKeeper)},
			{5, SimulateMsgEditValidator(stakeKeeper)},
			{15, SimulateMsgDelegate(mapper, stakeKeeper)},
			{10, SimulateMsgBeginUnbonding(mapper, stakeKeeper)},
			{10, SimulateMsgBeginRedelegate(mapper, stakeKeeper)},
		}, []simulation.RandSetup{
			Setup(mapp, stakeKeeper),
		}, []simulation.Invariant{
			AllInvariants(bankKeeper, stakeKeeper, distrKeeper, mapp.AccountKeeper),
		}, 10, 100,
		false,
	)
}
