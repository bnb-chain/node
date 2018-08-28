package app

import (
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/BiJie/BinanceChain/app/pub"
	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/ico"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	tokenStore "github.com/BiJie/BinanceChain/plugins/tokens/store"
	"github.com/BiJie/BinanceChain/wire"
)

const (
	appName = "BNBChain"
	publishMarketDataFlag = "publishMarketData"
)

// default home directories for expected binaries
var (
	DefaultCLIHome  = os.ExpandEnv("$HOME/.bnbcli")
	DefaultNodeHome = os.ExpandEnv("$HOME/.bnbchaind")
)

var (
	Codec = makeCodec()
	ServerContext = server.NewDefaultContext()
	RootCmd = makeRootCmd()
)

// BinanceChain is the BNBChain ABCI application
type BinanceChain struct {
	*BaseApp
	Codec *wire.Codec

	FeeCollectionKeeper tx.FeeCollectionKeeper
	CoinKeeper          bank.Keeper
	DexKeeper           *dex.DexKeeper
	AccountMapper       auth.AccountMapper
	TokenMapper         tokenStore.Mapper

	isPublishMarketData bool
	publisher           pub.MarketDataPublisher
}

// NewBinanceChain creates a new instance of the BinanceChain.
func NewBinanceChain(logger log.Logger, db dbm.DB, traceStore io.Writer) *BinanceChain {

	// create app-level codec for txs and accounts
	var cdc = Codec

	// create composed tx decoder
	decoders := wire.ComposeTxDecoders(cdc, defaultTxDecoder)

	// create your application object
	var app = &BinanceChain{
		BaseApp: NewBaseApp(appName, cdc, logger, db, decoders),
		Codec:   cdc,
		isPublishMarketData: RootCmd.Flag(publishMarketDataFlag).Value.String() == "true",
	}

	app.SetCommitMultiStoreTracer(traceStore)
	// mappers
	app.AccountMapper = auth.NewAccountMapper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	app.TokenMapper = tokenStore.NewMapper(cdc, common.TokenStoreKey)

	// Add handlers.
	app.CoinKeeper = bank.NewKeeper(app.AccountMapper)
	// TODO: make the concurrency configurable

	tradingPairMapper := dex.NewTradingPairMapper(cdc, common.PairStoreKey)
	var err error
	app.DexKeeper, err = dex.NewOrderKeeper(common.DexStoreKey, app.CoinKeeper, tradingPairMapper,
		app.RegisterCodespace(dex.DefaultCodespace), 2, app.cdc, app.isPublishMarketData)
	if err != nil {
		logger.Error("Failed to create an order keep", "error", err)
		panic(err)
	}
	// Currently we do not need the ibc and staking part
	// app.ibcMapper = ibc.NewMapper(app.cdc, app.capKeyIBCStore, app.RegisterCodespace(ibc.DefaultCodespace))
	// app.stakeKeeper = simplestake.NewKeeper(app.capKeyStakingStore, app.coinKeeper, app.RegisterCodespace(simplestake.DefaultCodespace))

	app.registerHandlers(cdc)

	if app.isPublishMarketData {
		app.publisher = pub.MarketDataPublisher{Logger: app.Logger, ToPublishChannel: make(chan pub.BlockInfoToPublish, pub.PublicationBufferSize)}
		if err := app.publisher.Init(); err != nil {
			app.publisher.Stop()
			app.Logger.Error("Cannot start up market data kafka publisher", "err", err)
			/**
			  TODO(#66): we should return nil here, but cosmos start-up logic doesn't process nil newapp vendor/github.com/cosmos/cosmos-sdk/server/constructors.go:34
			  app := appFn(logger, db, traceStoreWriter)
			  return app, nil
			 */
		}
	}

	// Initialize BaseApp.
	app.SetInitChainer(app.initChainerFn())
	app.SetEndBlocker(app.EndBlocker)
	app.MountStoresIAVL(common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.DexStoreKey, common.PairStoreKey)
	app.SetAnteHandler(tx.NewAnteHandler(app.AccountMapper, app.FeeCollectionKeeper))
	err = app.LoadLatestVersion(common.MainStoreKey)
	if err != nil {
		cmn.Exit(err.Error())
	}

	app.InitDexKeeperBook()

	return app
}

func (app *BinanceChain) InitDexKeeperBook() {
	if app.checkState == nil {
		return
	}

	//count back to 7 days.
	app.DexKeeper.InitOrderBook(app.checkState.ctx, 7, app.db, app.LastBlockHeight(), app.txDecoder)
}

//TODO???: where to init checkState in reboot
func (app *BinanceChain) SetCheckState(header abci.Header) {
	ms := app.cms.CacheMultiStore()
	app.checkState = &state{
		ms:  ms,
		ctx: sdk.NewContext(ms, header, true, app.Logger),
	}
}

func (app *BinanceChain) registerHandlers(cdc *wire.Codec) {
	app.Router().AddRoute("bank", bank.NewHandler(app.CoinKeeper))
	// AddRoute("ibc", ibc.NewHandler(ibcMapper, coinKeeper)).
	// AddRoute("simplestake", simplestake.NewHandler(stakeKeeper))
	for route, handler := range tokens.Routes(app.TokenMapper, app.AccountMapper, app.CoinKeeper) {
		app.Router().AddRoute(route, handler)
	}

	for route, handler := range dex.Routes(cdc, app.DexKeeper, app.TokenMapper, app.AccountMapper) {
		app.Router().AddRoute(route, handler)
	}
}

// MakeCodec creates a custom tx codec.
func makeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	wire.RegisterCrypto(cdc) // Register crypto.
	bank.RegisterWire(cdc)
	sdk.RegisterWire(cdc) // Register Msgs
	dex.RegisterWire(cdc)
	tokens.RegisterWire(cdc)
	types.RegisterWire(cdc)
	tx.RegisterWire(cdc)

	return cdc
}

// construct a cobra Command and configure BinanceChain ABCI command line options
// the options are correctly assigned with command line parameter during/after this cobra.Command
// executed.
// To access Flags: var haha = RootCmd.Flag("isPublishMarketData").Value.String()
// To set Flags: --mode validator (for non-boolean) --isPublishMarketData (for boolean)
func makeRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "bnbchaind",
		Short:             "BNBChain Daemon (server)",
		PersistentPreRunE: server.PersistentPreRunEFn(ServerContext),
	}
	rootCmd.PersistentFlags().Bool(publishMarketDataFlag, false, "whether this node should publish market data")
	return rootCmd
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
		app.DexKeeper.InitGenesis(ctx, genesisState.DexGenesis.TradingGenesis)
		return abci.ResponseInitChain{}
	}
}

func (app *BinanceChain) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	lastBlockTime := app.checkState.ctx.BlockHeader().Time
	blockTime := ctx.BlockHeader().Time
	height := ctx.BlockHeight()

	if utils.SameDayInUTC(lastBlockTime, blockTime) {
		// only match in the normal block
		app.DexKeeper.MatchAndAllocateAll(ctx, app.AccountMapper)
	} else {
		// breathe block

		icoDone := ico.EndBlockAsync(ctx)

		dex.EndBreatheBlock(ctx, app.AccountMapper, app.DexKeeper, height, blockTime)

		// other end blockers
		<-icoDone
	}

	if app.isPublishMarketData && app.publisher.IsLive {
		app.Logger.Info("start to publish market data")
		// TODO(#66): confirm the performance is acceptable when there are a lot of orders and books here (orders might get accmulated for 3 days - the time limit of GTC order to expire)
		orders, ordersMap := app.DexKeeper.GetLastOrdersCopy()
		latestPriceLevels := app.DexKeeper.GetOrderBookForPublish(20)

		app.publisher.ToPublishChannel <- pub.NewBlockInfoToPublish(ctx.BlockHeader().Height, ctx.BlockHeader().Time, app.DexKeeper.GetLastTradesCopy(), orders, ordersMap, latestPriceLevels)
		app.DexKeeper.ClearOrderChanges()
	}
	// distribute fees
	distributeFee(ctx, app.AccountMapper)
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
		orderbook := app.DexKeeper.GetOrderBook(pair, 20)
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

// default custom logic for transaction decoding
func defaultTxDecoder(cdc *wire.Codec) sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, sdk.Error) {
		var tx = auth.StdTx{}

		if len(txBytes) == 0 {
			return nil, sdk.ErrTxDecode("txBytes are empty")
		}

		// StdTx.Msg is an interface. The concrete types
		// are registered by MakeTxCodec
		err := cdc.UnmarshalBinary(txBytes, &tx)
		if err != nil {
			return nil, sdk.ErrTxDecode("").TraceSDK(err.Error())
		}
		return tx, nil
	}
}
