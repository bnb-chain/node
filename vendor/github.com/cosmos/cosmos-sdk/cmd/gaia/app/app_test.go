package app

import (
	"errors"
	"io"
	"os"
	"testing"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/require"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	abci "github.com/tendermint/tendermint/abci/types"
)

type MockGaiaApp struct {
	*GaiaApp
}

// NewGaiaApp returns a reference to an initialized GaiaApp.
func NewMockGaiaApp(logger log.Logger, db dbm.DB, traceStore io.Writer, baseAppOptions ...func(*bam.BaseApp)) *MockGaiaApp {
	cdc := MakeCodec()

	bApp := bam.NewBaseApp(appName, logger, db, auth.DefaultTxDecoder(cdc), sdk.CollectConfig{}, baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)

	gApp := &GaiaApp{
		BaseApp:        bApp,
		cdc:            cdc,
		keyMain:        sdk.NewKVStoreKey("main"),
		keyAccount:     sdk.NewKVStoreKey("acc"),
		keyStake:       sdk.NewKVStoreKey("stake"),
		keyStakeReward: sdk.NewKVStoreKey("stake_reward"),
		tkeyStake:      sdk.NewTransientStoreKey("transient_stake"),
		keyMint:        sdk.NewKVStoreKey("mint"),
		keyDistr:       sdk.NewKVStoreKey("distr"),
		tkeyDistr:      sdk.NewTransientStoreKey("transient_distr"),
		keySlashing:    sdk.NewKVStoreKey("slashing"),
		keyGov:         sdk.NewKVStoreKey("gov"),
		keyParams:      sdk.NewKVStoreKey("params"),
		tkeyParams:     sdk.NewTransientStoreKey("transient_params"),
		keyIbc:         sdk.NewKVStoreKey("ibc"),
		keySide:        sdk.NewKVStoreKey("side"),
	}

	var app = &MockGaiaApp{gApp}

	// define the accountKeeper
	app.accountKeeper = auth.NewAccountKeeper(
		app.cdc,
		app.keyAccount,        // target store
		auth.ProtoBaseAccount, // prototype
	)

	// add handlers
	app.bankKeeper = bank.NewBaseKeeper(app.accountKeeper)
	app.paramsKeeper = params.NewKeeper(
		app.cdc,
		app.keyParams, app.tkeyParams,
	)
	app.ibcKeeper = ibc.NewKeeper(app.keyIbc, app.paramsKeeper.Subspace(ibc.DefaultParamspace), ibc.DefaultCodespace,
		sidechain.NewKeeper(app.keySide, app.paramsKeeper.Subspace(sidechain.DefaultParamspace), app.cdc))

	app.stakeKeeper = stake.NewKeeper(
		app.cdc,
		app.keyStake, app.keyStakeReward, app.tkeyStake,
		app.bankKeeper, nil, app.paramsKeeper.Subspace(stake.DefaultParamspace),
		app.RegisterCodespace(stake.DefaultCodespace),
	)
	app.mintKeeper = mint.NewKeeper(app.cdc, app.keyMint,
		app.paramsKeeper.Subspace(mint.DefaultParamspace),
		app.stakeKeeper,
	)
	app.distrKeeper = distr.NewKeeper(
		app.cdc,
		app.keyDistr,
		app.paramsKeeper.Subspace(distr.DefaultParamspace),
		app.bankKeeper, app.stakeKeeper, nil,
		app.RegisterCodespace(stake.DefaultCodespace),
	)
	app.slashingKeeper = slashing.NewKeeper(
		app.cdc,
		app.keySlashing,
		app.stakeKeeper, app.paramsKeeper.Subspace(slashing.DefaultParamspace),
		app.RegisterCodespace(slashing.DefaultCodespace),
		app.bankKeeper,
	)
	app.govKeeper = gov.NewKeeper(
		app.cdc,
		app.keyGov,
		app.paramsKeeper, app.paramsKeeper.Subspace(gov.DefaultParamSpace), app.bankKeeper, app.stakeKeeper,
		app.RegisterCodespace(gov.DefaultCodespace),
		app.Pool,
	)

	// register the staking hooks
	app.stakeKeeper = app.stakeKeeper.WithHooks(
		NewHooks(app.distrKeeper.Hooks(), app.slashingKeeper.Hooks()))

	// register message routes
	app.Router().
		AddRoute("bank", bank.NewHandler(app.bankKeeper)).
		AddRoute("stake", stake.NewHandler(app.stakeKeeper, app.govKeeper)).
		AddRoute("distr", distr.NewHandler(app.distrKeeper)).
		AddRoute("slashing", slashing.NewSlashingHandler(app.slashingKeeper)).
		AddRoute("gov", gov.NewHandler(app.govKeeper))

	app.QueryRouter().
		AddRoute("gov", gov.NewQuerier(app.govKeeper)).
		AddRoute("stake", stake.NewQuerier(app.stakeKeeper, app.cdc))

	// initialize BaseApp
	app.MountStoresIAVL(app.keyMain, app.keyAccount, app.keyStake, app.keyMint, app.keyDistr,
		app.keySlashing, app.keyGov, app.keyParams)
	app.SetInitChainer(app.initChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountKeeper))
	app.MountStoresTransient(app.tkeyParams, app.tkeyStake, app.tkeyDistr)
	app.SetEndBlocker(app.EndBlocker)

	err := app.LoadLatestVersion(app.keyMain)
	if err != nil {
		cmn.Exit(err.Error())
	}

	return app
}

func (app *MockGaiaApp) LoadLatestVersion(mainKey sdk.StoreKey) error {
	err := app.GetCommitMultiStore().LoadLatestVersion()
	if err != nil {
		return err
	}
	return app.initFromStore(mainKey)
}

func (app *MockGaiaApp) initFromStore(mainKey sdk.StoreKey) error {
	// main store should exist.
	// TODO: we don't actually need the main store here
	main := app.GetCommitMultiStore().GetKVStore(mainKey)
	if main == nil {
		return errors.New("baseapp expects MultiStore with 'main' KVStore")
	}

	app.SetCheckState(abci.Header{})

	return nil
}

func setMockGenesis(gapp *MockGaiaApp, accs ...*auth.BaseAccount) error {
	genaccs := make([]GenesisAccount, len(accs))
	for i, acc := range accs {
		genaccs[i] = NewGenesisAccount(acc)
	}

	genesisState := GenesisState{
		Accounts:     genaccs,
		StakeData:    stake.DefaultGenesisState(),
		DistrData:    distr.DefaultGenesisState(),
		SlashingData: slashing.DefaultGenesisState(),
	}

	stateBytes, err := codec.MarshalJSONIndent(gapp.cdc, genesisState)
	if err != nil {
		return err
	}

	// Initialize the chain
	vals := []abci.ValidatorUpdate{}
	gapp.InitChain(abci.RequestInitChain{Validators: vals, AppStateBytes: stateBytes})
	gapp.Commit()

	return nil
}

func TestGaiadExport(t *testing.T) {
	db := db.NewMemDB()
	gapp := NewMockGaiaApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil)
	setMockGenesis(gapp)

	// Making a new app object with the db, so that initchain hasn't been called
	newGapp := NewMockGaiaApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil)
	_, _, err := newGapp.ExportAppStateAndValidators()
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}
