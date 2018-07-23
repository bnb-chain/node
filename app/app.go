package app

import (
	"encoding/json"
	"io"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/ico"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	tokenStore "github.com/BiJie/BinanceChain/plugins/tokens/store"
)

const (
	appName = "BNBChain"
)

// BinanceChain is the BNBChain ABCI application
type BinanceChain struct {
	*BaseApp
	Codec *wire.Codec

	FeeCollectionKeeper auth.FeeCollectionKeeper
	CoinKeeper          bank.Keeper
	OrderKeeper         dex.OrderKeeper
	AccountMapper       auth.AccountMapper
	TokenMapper         tokenStore.Mapper
	TradingPairMapper   dex.TradingPairMapper
}

// NewBinanceChain creates a new instance of the BinanceChain.
func NewBinanceChain(logger log.Logger, db dbm.DB, traceStore io.Writer) *BinanceChain {

	// Create app-level codec for txs and accounts.
	var cdc = MakeCodec()

	// Create your application object.
	var app = &BinanceChain{
		BaseApp: NewBaseApp(appName, cdc, logger, db),
		Codec:   cdc,
	}

	app.SetCommitMultiStoreTracer(traceStore)
	// mappers
	app.AccountMapper = auth.NewAccountMapper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	app.TokenMapper = tokenStore.NewMapper(cdc, common.TokenStoreKey)
	app.TradingPairMapper = dex.NewTradingPairMapper(cdc, common.PairStoreKey)

	// Add handlers.
	app.CoinKeeper = bank.NewKeeper(app.AccountMapper)
	// TODO: make the concurrency configurable
	app.OrderKeeper = dex.NewOrderKeeper(common.DexStoreKey, app.CoinKeeper, app.RegisterCodespace(dex.DefaultCodespace), 2)
	// Currently we do not need the ibc and staking part
	// app.ibcMapper = ibc.NewMapper(app.cdc, app.capKeyIBCStore, app.RegisterCodespace(ibc.DefaultCodespace))
	// app.stakeKeeper = simplestake.NewKeeper(app.capKeyStakingStore, app.coinKeeper, app.RegisterCodespace(simplestake.DefaultCodespace))

	app.registerHandlers()

	// Initialize BaseApp.
	app.SetInitChainer(app.initChainerFn())
	app.SetEndBlocker(app.EndBlocker)
	app.MountStoresIAVL(common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.DexStoreKey, common.PairStoreKey)
	app.SetAnteHandler(auth.NewAnteHandler(app.AccountMapper, app.FeeCollectionKeeper))
	err := app.LoadLatestVersion(common.MainStoreKey)
	if err != nil {
		cmn.Exit(err.Error())
	}
	return app
}

//TODO???: where to init checkState in reboot
func (app *BinanceChain) SetCheckState(header abci.Header) {
	ms := app.cms.CacheMultiStore()
	app.checkState = &state{
		ms:  ms,
		ctx: sdk.NewContext(ms, header, true, app.Logger),
	}
}

func (app *BinanceChain) registerHandlers() {
	app.Router().AddRoute("bank", bank.NewHandler(app.CoinKeeper))
	// AddRoute("ibc", ibc.NewHandler(ibcMapper, coinKeeper)).
	// AddRoute("simplestake", simplestake.NewHandler(stakeKeeper))
	for route, handler := range tokens.Routes(app.TokenMapper, app.AccountMapper, app.CoinKeeper) {
		app.Router().AddRoute(route, handler)
	}

	for route, handler := range dex.Routes(app.TradingPairMapper, app.OrderKeeper, app.TokenMapper, app.AccountMapper, app.CoinKeeper) {
		app.Router().AddRoute(route, handler)
	}
}

// MakeCodec creates a custom tx codec.
func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	wire.RegisterCrypto(cdc) // Register crypto.
	bank.RegisterWire(cdc)
	sdk.RegisterWire(cdc) // Register Msgs
	dex.RegisterWire(cdc)
	tokens.RegisterTypes(cdc)
	types.RegisterWire(cdc)

	return cdc
}

// initChainerFn performs custom logic for chain initialization.
func (app *BinanceChain) initChainerFn() sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		stateJSON := req.AppStateBytes

		genesisState := new(GenesisState)
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
			app.AccountMapper.SetAccount(ctx, acc)
		}

		// Application specific genesis handling
		err = app.OrderKeeper.InitGenesis(ctx, genesisState.DexGenesis.TradingGenesis)
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			//	return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		return abci.ResponseInitChain{}
	}
}

func (app *BinanceChain) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	lastBlockTime := app.checkState.ctx.BlockHeader().Time
	blockTime := ctx.BlockHeader().Time

	if utils.SameDayInUTC(lastBlockTime, blockTime) {
		// only match in the normal block
		app.OrderKeeper.MatchAndAllocateAll(ctx, app.AccountMapper)
	} else {
		// breathe block
		icoDone := ico.EndBlockAsync(ctx)
		// other end blockers

		<-icoDone
	}

	// TODO: update validators
	return abci.ResponseEndBlock{}
}

// ExportAppStateAndValidators exports blockchain world state to json.
func (app *BinanceChain) ExportAppStateAndValidators() (appState json.RawMessage, validators []tmtypes.GenesisValidator, err error) {
	ctx := app.NewContext(true, abci.Header{})

	// iterate to get the accounts
	accounts := []*GenesisAccount{}
	appendAccount := func(acc auth.Account) (stop bool) {
		account := &GenesisAccount{
			Address: acc.GetAddress(),
			Coins:   acc.GetCoins(),
		}
		accounts = append(accounts, account)
		return false
	}
	app.AccountMapper.IterateAccounts(ctx, appendAccount)

	genState := GenesisState{
		Accounts: accounts,
	}
	appState, err = wire.MarshalJSONIndent(app.cdc, genState)
	if err != nil {
		return nil, nil, err
	}
	return appState, validators, nil
}

func (app *BinanceChain) Query(req abci.RequestQuery) (res abci.ResponseQuery) {
	path := splitPath(req.Path)
	if len(path) == 0 {
		msg := "no query path provided"
		return sdk.ErrUnknownRequest(msg).QueryResult()
	}
	switch path[0] {
	// "/app" prefix for special application queries
	case "app":
		return handleBinanceChainQuery(app, path, req)
	default:
		return app.BaseApp.Query(req)
	}
}

func handleBinanceChainQuery(app *BinanceChain, path []string, req abci.RequestQuery) (res abci.ResponseQuery) {
	return abci.ResponseQuery{
		Code: uint32(sdk.ABCICodeOK),
		Info: "DD",
	}
}
