package app

import (
	"encoding/json"

	tokenStore "github.com/BiJie/BinanceChain/plugins/tokens/store"
	abci "github.com/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/log"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

const (
	appName = "BNBChain"
)

// BasecoinApp is the BNBChain ABCI application
type BasecoinApp struct {
	*bam.BaseApp
	cdc *wire.Codec

	// keys to access the substores
	capKeyMainStore    *sdk.KVStoreKey
	capKeyAccountStore *sdk.KVStoreKey
	capKeyTokensStore  *sdk.KVStoreKey
	capKeyDexStore     *sdk.KVStoreKey
	// capKeyStakingStore *sdk.KVStoreKey

	// keepers
	feeCollectionKeeper auth.FeeCollectionKeeper
	coinKeeper          bank.Keeper
	dexKeeper           dex.Keeper

	// Manage getting and setting accounts
	accountMapper auth.AccountMapper
	tokenMapper   tokenStore.Mapper
}

// NewBasecoinApp creates a new instance of the BasecoinApp.
func NewBasecoinApp(logger log.Logger, db dbm.DB) *BasecoinApp {

	// Create app-level codec for txs and accounts.
	var cdc = MakeCodec()

	// Create your application object.
	var app = &BasecoinApp{
		BaseApp:            bam.NewBaseApp(appName, cdc, logger, db),
		cdc:                cdc,
		capKeyMainStore:    common.MainStoreKey,
		capKeyAccountStore: common.AccountStoreKey,
		capKeyTokensStore:  common.TokenStoreKey,
		capKeyDexStore:     common.DexStoreKey,
		// capKeyStakingStore: sdk.NewKVStoreKey("stake"),
	}

	// Define the accountMapper.
	app.accountMapper = auth.NewAccountMapper(
		cdc,
		app.capKeyAccountStore, // target store
		&types.AppAccount{},    // prototype
	)

	// Add mappers.
	app.tokenMapper = tokenStore.NewMapper(cdc, common.TokenStoreKey)

	// Add handlers.
	app.coinKeeper = bank.NewKeeper(app.accountMapper)
	app.dexKeeper = dex.NewKeeper(common.DexStoreKey, app.coinKeeper, app.RegisterCodespace(dex.DefaultCodespace))
	// Currently we do not need the ibc and staking part
	// app.ibcMapper = ibc.NewMapper(app.cdc, app.capKeyIBCStore, app.RegisterCodespace(ibc.DefaultCodespace))
	// app.stakeKeeper = simplestake.NewKeeper(app.capKeyStakingStore, app.coinKeeper, app.RegisterCodespace(simplestake.DefaultCodespace))

	app.registerHandlers()

	// Initialize BaseApp.
	app.SetInitChainer(app.initChainerFn())
	app.MountStoresIAVL(app.capKeyMainStore, app.capKeyAccountStore, app.capKeyTokensStore, app.capKeyDexStore)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountMapper, app.feeCollectionKeeper))
	err := app.LoadLatestVersion(app.capKeyMainStore)
	if err != nil {
		cmn.Exit(err.Error())
	}
	return app
}

func (app *BasecoinApp) registerHandlers() {
	app.Router().
		AddRoute("bank", bank.NewHandler(app.coinKeeper)).
		AddRoute("dex", dex.NewHandler(app.dexKeeper))
	// AddRoute("ibc", ibc.NewHandler(ibcMapper, coinKeeper)).
	// AddRoute("simplestake", simplestake.NewHandler(stakeKeeper))
	for route, handler := range tokens.Routes(app.tokenMapper, app.accountMapper, app.coinKeeper) {
		app.Router().AddRoute(route, handler)
	}
}

// MakeCodec creates a custom tx codec.
func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	wire.RegisterCrypto(cdc) // Register crypto.
	sdk.RegisterWire(cdc)    // Register Msgs
	dex.RegisterWire(cdc)

	// Register AppAccount
	cdc.RegisterInterface((*auth.Account)(nil), nil)
	cdc.RegisterConcrete(&types.AppAccount{}, "basecoin/Account", nil)
	return cdc
}

// initChainerFn performs custom logic for chain initialization.
func (app *BasecoinApp) initChainerFn() sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		stateJSON := req.AppStateBytes

		genesisState := new(types.GenesisState)
		err := json.Unmarshal(stateJSON, genesisState)
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			// return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		for _, gacc := range genesisState.Accounts {
			acc, err := gacc.ToAppAccount()
			if err != nil {
				panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
				//	return sdk.ErrGenesisParse("").TraceCause(err, "")
			}
			app.accountMapper.SetAccount(ctx, acc)
		}

		// Application specific genesis handling
		err = app.dexKeeper.InitGenesis(ctx, genesisState.DexGenesis)
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			//	return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		return abci.ResponseInitChain{}
	}
}

// ExportAppStateAndValidators exports blockchain world state to json.
func (app *BasecoinApp) ExportAppStateAndValidators() (appState json.RawMessage, validators []tmtypes.GenesisValidator, err error) {
	ctx := app.NewContext(true, abci.Header{})

	// iterate to get the accounts
	accounts := []*types.GenesisAccount{}
	appendAccount := func(acc auth.Account) (stop bool) {
		account := &types.GenesisAccount{
			Address: acc.GetAddress(),
			Coins:   acc.GetCoins(),
		}
		accounts = append(accounts, account)
		return false
	}
	app.accountMapper.IterateAccounts(ctx, appendAccount)

	genState := types.GenesisState{
		Accounts: accounts,
	}
	appState, err = wire.MarshalJSONIndent(app.cdc, genState)
	if err != nil {
		return nil, nil, err
	}
	return appState, validators, nil
}
