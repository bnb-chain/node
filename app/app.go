package app

import (
	"encoding/json"
	"io"
	"os"

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

// default home directories for expected binaries
var (
	DefaultCLIHome  = os.ExpandEnv("$HOME/.bnbcli")
	DefaultNodeHome = os.ExpandEnv("$HOME/.bnbchaind")
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
	tokens.RegisterWire(cdc)
	types.RegisterWire(cdc)

	return cdc
}

// initChainerFn performs custom logic for chain initialization.
func (app *BinanceChain) initChainerFn() sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		stateJSON := req.AppStateBytes

		genesisState := new(GenesisState)
		err := app.Codec.UnmarshalJSON(stateJSON, genesisState)
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			// return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		for _, gacc := range genesisState.Accounts {
			acc := gacc.ToAppAccount()
			acc.AccountNumber = app.AccountMapper.GetNextAccountNumber(ctx)
			app.AccountMapper.SetAccount(ctx, acc)
		}

		for _, token := range genesisState.Tokens {
			// TODO: replace by Issue and move to token.genesis
			err = app.TokenMapper.NewToken(ctx, token)
			if err != nil {
				panic(err)
			}

			_, _, sdkErr := app.CoinKeeper.AddCoins(ctx, token.Owner, append((sdk.Coins)(nil),
				sdk.Coin{Denom: token.Symbol, Amount: sdk.NewInt(token.TotalSupply)}))
			if sdkErr != nil {
				panic(sdkErr)
			}
		}

		// Application specific genesis handling
		app.OrderKeeper.InitGenesis(ctx, genesisState.DexGenesis.TradingGenesis)
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
	accounts := []GenesisAccount{}
	appendAccount := func(acc auth.Account) (stop bool) {
		account := GenesisAccount{
			Address: acc.GetAddress(),
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
	switch path[1] {
	case "orderbook":
		//TODO: sync lock, validate pair, level number
		if len(path) < 3 {
			return abci.ResponseQuery{
				Code: uint32(sdk.CodeUnknownRequest),
				Log:  "OrderBook Query Requires Pair Name",
			}
		}
		pair := path[2]
		orderbook := make([][]int64, 10)
		for l := range orderbook {
			orderbook[l] = make([]int64, 4)
		}
		i, j := 0, 0
		app.OrderKeeper.GetOrderBookUnSafe(pair, 20,
			func(price, qty int64) {
				orderbook[i][2] = price
				orderbook[i][3] = qty
				i++
			},
			func(price, qty int64) {
				orderbook[j][1] = price
				orderbook[j][0] = qty
				j++
			})

		resValue, err := app.Codec.MarshalBinary(orderbook)
		if err != nil {
			return abci.ResponseQuery{
				Code: uint32(sdk.CodeInternal),
				Log:  err.Error(),
			}
		}

		return abci.ResponseQuery{
			Code:  uint32(sdk.ABCICodeOK),
			Value: resValue,
		}
	default:
		return abci.ResponseQuery{
			Code: uint32(sdk.ABCICodeOK),
			Info: "Unknown 'app' Query Path",
		}
	}
}
