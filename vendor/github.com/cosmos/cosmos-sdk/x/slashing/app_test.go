package slashing

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

var (
	priv1 = ed25519.GenPrivKey()
	addr1 = sdk.AccAddress(priv1.PubKey().Address())
	coins = sdk.Coins{sdk.NewCoin("foocoin", 10)}
)

// initialize the mock application for this module
func getMockApp(t *testing.T) (*mock.App, stake.Keeper, Keeper) {
	mapp := mock.NewApp()

	RegisterCodec(mapp.Cdc)
	keyStake := sdk.NewKVStoreKey("stake")
	keyStakeReward := sdk.NewKVStoreKey("stake_reward")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keySlashing := sdk.NewKVStoreKey("slashing")
	keyIbc := sdk.NewKVStoreKey("ibc")
	keySideChain := sdk.NewKVStoreKey("sc")

	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")
	bankKeeper := bank.NewBaseKeeper(mapp.AccountKeeper)

	paramsKeeper := params.NewKeeper(mapp.Cdc, keyParams, tkeyParams)
	scKeeper := sidechain.NewKeeper(keySideChain, paramsKeeper.Subspace(sidechain.DefaultParamspace), mapp.Cdc)
	ibcKeeper := ibc.NewKeeper(keyIbc, paramsKeeper.Subspace(ibc.DefaultParamspace), ibc.DefaultCodespace, scKeeper)
	stakeKeeper := stake.NewKeeper(mapp.Cdc, keyStake, keyStakeReward, tkeyStake, bankKeeper, nil, paramsKeeper.Subspace(stake.DefaultParamspace), mapp.RegisterCodespace(stake.DefaultCodespace))
	stakeKeeper.SetupForSideChain(&scKeeper, &ibcKeeper)
	keeper := NewKeeper(mapp.Cdc, keySlashing, stakeKeeper, paramsKeeper.Subspace(DefaultParamspace), mapp.RegisterCodespace(DefaultCodespace), bankKeeper)
	mapp.Router().AddRoute("stake", stake.NewStakeHandler(stakeKeeper))
	mapp.Router().AddRoute("slashing", NewSlashingHandler(keeper))

	mapp.SetEndBlocker(getEndBlocker(stakeKeeper))
	mapp.SetInitChainer(getInitChainer(mapp, stakeKeeper))

	require.NoError(t, mapp.CompleteSetup(keyStake, tkeyStake, keySlashing, keyParams, tkeyParams))

	return mapp, stakeKeeper, keeper
}

// stake endblocker
func getEndBlocker(keeper stake.Keeper) sdk.EndBlocker {
	return func(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		validatorUpdates, _ := stake.EndBlocker(ctx, keeper)
		return abci.ResponseEndBlock{
			ValidatorUpdates: validatorUpdates,
		}
	}
}

// overwrite the mock init chainer
func getInitChainer(mapp *mock.App, keeper stake.Keeper) sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		mapp.InitChainer(ctx, req)
		stakeGenesis := stake.DefaultGenesisState()
		stakeGenesis.Pool.LooseTokens = sdk.NewDecWithoutFra(100000)
		validators, err := stake.InitGenesis(ctx, keeper, stakeGenesis)
		if err != nil {
			panic(err)
		}

		return abci.ResponseInitChain{
			Validators: validators,
		}
	}
}

func checkValidator(t *testing.T, mapp *mock.App, keeper stake.Keeper,
	addr sdk.AccAddress, expFound bool) stake.Validator {
	ctxCheck := mapp.BaseApp.NewContext(sdk.RunTxModeCheck, abci.Header{})
	validator, found := keeper.GetValidator(ctxCheck, sdk.ValAddress(addr1))
	require.Equal(t, expFound, found)
	return validator
}

func checkValidatorSigningInfo(t *testing.T, mapp *mock.App, keeper Keeper,
	addr sdk.ConsAddress, expFound bool) ValidatorSigningInfo {
	ctxCheck := mapp.BaseApp.NewContext(sdk.RunTxModeCheck, abci.Header{})
	signingInfo, found := keeper.getValidatorSigningInfo(ctxCheck, addr)
	require.Equal(t, expFound, found)
	return signingInfo
}

func TestSlashingMsgs(t *testing.T) {
	mapp, stakeKeeper, keeper := getMockApp(t)

	genCoin := sdk.NewCoin("steak", sdk.NewDecWithoutFra(42000).RawInt())
	bondCoin := sdk.NewCoin("steak", sdk.NewDecWithoutFra(10000).RawInt())

	acc1 := &auth.BaseAccount{
		Address: addr1,
		Coins:   sdk.Coins{genCoin},
	}
	accs := []sdk.Account{acc1}
	mock.SetGenesis(mapp, accs)

	description := stake.NewDescription("foo_moniker", "", "", "")
	commission := stake.NewCommissionMsg(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec())

	createValidatorMsg := stake.NewMsgCreateValidator(
		sdk.ValAddress(addr1), priv1.PubKey(), bondCoin, description, commission,
	)
	mock.SignCheckDeliver(t, mapp.BaseApp, []sdk.Msg{createValidatorMsg}, []int64{0}, []int64{0}, true, true, priv1)
	mock.CheckBalance(t, mapp, addr1, sdk.Coins{genCoin.Minus(bondCoin)})
	mapp.BeginBlock(abci.RequestBeginBlock{})

	validator := checkValidator(t, mapp, stakeKeeper, addr1, true)
	require.Equal(t, sdk.ValAddress(addr1), validator.OperatorAddr)
	require.Equal(t, sdk.Bonded, validator.Status)
	require.True(sdk.DecEq(t, sdk.NewDecWithoutFra(10000), validator.BondedTokens()))
	unjailMsg := MsgUnjail{ValidatorAddr: sdk.ValAddress(validator.ConsPubKey.Address())}

	// no signing info yet
	checkValidatorSigningInfo(t, mapp, keeper, sdk.ConsAddress(addr1), false)

	// unjail should fail with unknown validator
	res := mock.SignCheckDeliver(t, mapp.BaseApp, []sdk.Msg{unjailMsg}, []int64{0}, []int64{1}, false, false, priv1)
	require.Equal(t, sdk.ToABCICode(DefaultCodespace, CodeValidatorNotJailed), res.Code)
}
