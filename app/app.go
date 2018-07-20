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
	cdc *wire.Codec

	// keepers
	feeCollectionKeeper auth.FeeCollectionKeeper
	coinKeeper          bank.Keeper
	orderKeeper         dex.OrderKeeper
	accountMapper       auth.AccountMapper
	tokenMapper         tokenStore.Mapper
	tradingPairMapper   dex.TradingPairMapper
}

// NewBinanceChain creates a new instance of the BinanceChain.
func NewBinanceChain(logger log.Logger, db dbm.DB, traceStore io.Writer) *BinanceChain {

	// Create app-level codec for txs and accounts.
	var cdc = MakeCodec()

	// Create your application object.
	var app = &BinanceChain{
		BaseApp: NewBaseApp(appName, cdc, logger, db),
		cdc:     cdc,
	}

	app.SetCommitMultiStoreTracer(traceStore)
	// mappers
	app.accountMapper = auth.NewAccountMapper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	app.tokenMapper = tokenStore.NewMapper(cdc, common.TokenStoreKey)
	app.tradingPairMapper = dex.NewTradingPairMapper(cdc, common.PairStoreKey)

	// Add handlers.
	app.coinKeeper = bank.NewKeeper(app.accountMapper)
	app.orderKeeper = dex.NewOrderKeeper(common.DexStoreKey, app.coinKeeper, app.RegisterCodespace(dex.DefaultCodespace))
	// Currently we do not need the ibc and staking part
	// app.ibcMapper = ibc.NewMapper(app.cdc, app.capKeyIBCStore, app.RegisterCodespace(ibc.DefaultCodespace))
	// app.stakeKeeper = simplestake.NewKeeper(app.capKeyStakingStore, app.coinKeeper, app.RegisterCodespace(simplestake.DefaultCodespace))

	app.registerHandlers()

	// Initialize BaseApp.
	app.SetInitChainer(app.initChainerFn())
	app.SetEndBlocker(app.EndBlocker)
	app.MountStoresIAVL(common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.DexStoreKey, common.PairStoreKey)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountMapper, app.feeCollectionKeeper))
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

//Getter for testing
func (app *BinanceChain) GetCodec() *wire.Codec {
	return app.cdc
}

//Getter for testing
func (app *BinanceChain) GetOrderKeeper() *dex.OrderKeeper {
	return &app.orderKeeper
}

//Getter for testing
func (app *BinanceChain) GetCoinKeeper() *bank.Keeper {
	return &app.coinKeeper
}

//Getter for testing
func (app *BinanceChain) GetTradingPairMapper() *dex.TradingPairMapper {
	return &app.tradingPairMapper
}

//Getter for testing
func (app *BinanceChain) GetTokenMapper() *tokens.Mapper {
	return &app.tokenMapper
}

//Getter for testing
func (app *BinanceChain) GetAccountMapper() *auth.AccountMapper {
	return &app.accountMapper
}

func (app *BinanceChain) registerHandlers() {
	app.Router().
		AddRoute("bank", bank.NewHandler(app.coinKeeper)).
		AddRoute("dex", dex.NewHandler(app.dexKeeper, app.accountMapper))
	// AddRoute("ibc", ibc.NewHandler(ibcMapper, coinKeeper)).
	// AddRoute("simplestake", simplestake.NewHandler(stakeKeeper))
	for route, handler := range tokens.Routes(app.tokenMapper, app.accountMapper, app.coinKeeper) {
		app.Router().AddRoute(route, handler)
	}

	for route, handler := range dex.Routes(app.tradingPairMapper, app.orderKeeper, app.tokenMapper, app.accountMapper, app.coinKeeper) {
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
			app.accountMapper.SetAccount(ctx, acc)
		}

		// Application specific genesis handling
		// err = app.dexKeeper.InitGenesis(ctx, genesisState.DexGenesis)
		// if err != nil {
		// 	panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
		// 	//	return sdk.ErrGenesisParse("").TraceCause(err, "")
		// }

		return abci.ResponseInitChain{}
	}
}

func (app *BinanceChain) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	lastBlockTime := app.checkState.ctx.BlockHeader().Time
	blockTime := ctx.BlockHeader().Time

	if utils.SameDayInUTC(lastBlockTime, blockTime) {
		//only match in the normal block
		app.dexKeeper.MatchAndAllocateAll(ctx, app.accountMapper)
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
	app.accountMapper.IterateAccounts(ctx, appendAccount)

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
