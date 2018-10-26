package app

import (
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"io"
	"os"

	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/app/pub"
	"github.com/BiJie/BinanceChain/common"
	bnclog "github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/ico"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	tkstore "github.com/BiJie/BinanceChain/plugins/tokens/store"
	"github.com/BiJie/BinanceChain/wire"
)

const (
	appName = "BNBChain"
)

// default home directories for expected binaries
var (
	DefaultCLIHome  = os.ExpandEnv("$HOME/.bnbcli")
	DefaultNodeHome = os.ExpandEnv("$HOME/.bnbchaind")
)

// BinanceChain implements ChainApp
var _ types.ChainApp = (*BinanceChain)(nil)

var (
	Codec         = MakeCodec()
	ServerContext = config.NewDefaultContext()
)

// BinanceChain is the BNBChain ABCI application
type BinanceChain struct {
	*baseapp.BaseApp
	Codec *wire.Codec

	// the abci query handler mapping is `prefix -> handler`
	queryHandlers map[string]types.AbciQueryHandler

	// keepers
	FeeCollectionKeeper tx.FeeCollectionKeeper
	CoinKeeper          bank.Keeper
	DexKeeper           *dex.DexKeeper
	AccountKeeper       auth.AccountKeeper
	TokenMapper         tkstore.Mapper

	publicationConfig *config.PublicationConfig
	publisher         pub.MarketDataPublisher
}

// NewBinanceChain creates a new instance of the BinanceChain.
func NewBinanceChain(logger log.Logger, db dbm.DB, traceStore io.Writer, baseAppOptions ...func(*BaseApp)) *BinanceChain {

	// create app-level codec for txs and accounts
	var cdc = Codec

	// create composed tx decoder
	decoders := wire.ComposeTxDecoders(cdc, defaultTxDecoder)

	// create the application object
	var app = &BinanceChain{
		BaseApp:           baseapp.NewBaseApp(appName /*, cdc*/, logger, db, decoders /*, ServerContext.PublishAccountBalance*/, baseAppOptions...),
		Codec:             cdc,
		queryHandlers:     make(map[string]types.AbciQueryHandler),
		publicationConfig: ServerContext.PublicationConfig,
	}

	app.SetCommitMultiStoreTracer(traceStore)

	// mappers
	app.AccountKeeper = auth.NewAccountKeeper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	app.TokenMapper = tkstore.NewMapper(cdc, common.TokenStoreKey)

	// handlers
	app.CoinKeeper = bank.NewBaseKeeper(app.AccountKeeper)
	// TODO: make the concurrency configurable

	// Currently we do not need the ibc and staking part
	// app.ibcMapper = ibc.NewMapper(app.cdc, app.capKeyIBCStore, app.RegisterCodespace(ibc.DefaultCodespace))
	// app.stakeKeeper = simplestake.NewKeeper(app.capKeyStakingStore, app.coinKeeper, app.RegisterCodespace(simplestake.DefaultCodespace))

	// legacy bank route (others moved to plugin init funcs)
	sdkBankHandler := bank.NewHandler(app.CoinKeeper)
	bankHandler := func(ctx sdk.Context, msg sdk.Msg, simulate bool) sdk.Result {
		return sdkBankHandler(ctx, msg)
	}
	app.Router().AddRoute("bank", bankHandler)

	if app.publicationConfig.ShouldPublishAny() {
		app.publisher = pub.NewKafkaMarketDataPublisher(app.publicationConfig)
	}

	// finish app initialization
	app.SetInitChainer(app.initChainerFn())
	app.SetEndBlocker(app.EndBlocker)
	app.MountStoresIAVL(common.MainStoreKey, common.AccountStoreKey, common.TokenStoreKey, common.DexStoreKey, common.PairStoreKey)
	app.SetAnteHandler(tx.NewAnteHandler(app.AccountKeeper, app.FeeCollectionKeeper))

	// block store required to hydrate dex OB
	err := app.LoadLatestVersion(common.MainStoreKey)
	if err != nil {
		cmn.Exit(err.Error())
	}

	// remaining plugin init
	app.initDex()
	app.initPlugins()

	return app
}

func (app *BinanceChain) initDex() {
	tradingPairMapper := dex.NewTradingPairMapper(app.cdc, common.PairStoreKey)
	app.DexKeeper = dex.NewOrderKeeper(common.DexStoreKey, app.CoinKeeper, tradingPairMapper,
		app.RegisterCodespace(dex.DefaultCodespace), 2, app.cdc, app.publicationConfig.PublishOrderUpdates)
	// do not proceed if we are in a unit test and `checkState` is unset.
	if app.checkState == nil {
		return
	}
	// configure dex keeper
	app.DexKeeper.FeeConfig.Init(app.checkState.ctx)
	// count back to 7 days.
	app.DexKeeper.InitOrderBook(app.checkState.ctx, 7, loadBlockDB(), app.LastBlockHeight(), app.txDecoder)
}

func (app *BinanceChain) initPlugins() {
	tokens.InitPlugin(app, app.TokenMapper, app.AccountKeeper, app.CoinKeeper)
	dex.InitPlugin(app, app.DexKeeper, app.TokenMapper, app.AccountKeeper)
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
			acc.AccountNumber = app.AccountKeeper.GetNextAccountNumber(ctx)
			app.AccountKeeper.SetAccount(ctx, acc)
		}

		for _, token := range genesisState.Tokens {
			// TODO: replace by Issue and move to token.genesis
			err = app.tokenMapper.NewToken(ctx, token)
			if err != nil {
				panic(err)
			}

			_, _, sdkErr := app.CoinKeeper.AddCoins(ctx, token.Owner, append((sdk.Coins)(nil),
				sdk.Coin{
					Denom:  token.Symbol,
					Amount: sdk.NewInt(token.TotalSupply.ToInt64()),
				}))
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
	// lastBlockTime would be 0 if this is the first block.
	lastBlockTime := app.checkState.ctx.BlockHeader().Time
	blockTime := ctx.BlockHeader().Time
	// we shouldn't use ctx.BlockHeight() here because for the first block, it would be 0 and 2 for the second block
	height := ctx.BlockHeader().Height

	var tradesToPublish []pub.Trade

	//match may end with transaction faliure, which is better to save into
	//the EndBlock response. However, current cosmos doesn't support this.
	//future TODO: add failure info.
	response := abci.ResponseEndBlock{}

	if utils.SameDayInUTC(lastBlockTime, blockTime) || height == 1 {
		// only match in the normal block
		app.Logger.Debug("normal block", "height", height)
		if app.publicationConfig.PublishOrderUpdates && pub.IsLive {
			tradesToPublish = pub.MatchAndAllocateAllForPublish(app.DexKeeper, app.AccountKeeper, ctx)
		} else {
			ctx, _, _ = app.DexKeeper.MatchAndAllocateAll(ctx, app.AccountKeeper, nil)
		}

	} else {
		// breathe block
		bnclog.Info("Start Breathe Block Handling",
			"height", height, "lastBlockTime", lastBlockTime, "newBlockTime", blockTime)
		icoDone := ico.EndBlockAsync(ctx)
		dex.EndBreatheBlock(ctx, app.AccountKeeper, app.DexKeeper, height, blockTime)

		// other end blockers
		<-icoDone
	}

	// distribute fees TODO: enable it after upgraded to tm 0.24.0
	// distributeFee(ctx, app.AccountKeeper)
	// TODO: update validators

	if app.publicationConfig.ShouldPublishAny() && pub.IsLive {
		app.publish(tradesToPublish, ctx, height, blockTime.Unix())
	}

	return response
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
	app.AccountKeeper.IterateAccounts(ctx, appendAccount)

	genState := GenesisState{
		Accounts: accounts,
	}
	appState, err = wire.MarshalJSONIndent(app.cdc, genState)
	if err != nil {
		return nil, nil, err
	}
	return appState, validators, nil
}

// Query performs an abci query.
func (app *BinanceChain) Query(req abci.RequestQuery) (res abci.ResponseQuery) {
	path := splitPath(req.Path)
	if len(path) == 0 {
		msg := "no query path provided"
		return sdk.ErrUnknownRequest(msg).QueryResult()
	}
	prefix := path[0]
	if handler, ok := app.queryHandlers[prefix]; ok {
		res := handler(app, req, path)
		if res == nil {
			return app.BaseApp.Query(req)
		}
		return *res
	}
	return app.BaseApp.Query(req)
}

// RegisterQueryHandler registers an abci query handler, implements ChainApp.RegisterQueryHandler.
func (app *BinanceChain) RegisterQueryHandler(prefix string, handler types.AbciQueryHandler) {
	if _, ok := app.queryHandlers[prefix]; ok {
		panic(fmt.Errorf("registerQueryHandler: prefix `%s` is already registered", prefix))
	} else {
		app.queryHandlers[prefix] = handler
	}
}

// GetCodec returns the app's Codec.
func (app *BinanceChain) GetCodec() *wire.Codec {
	return app.Codec
}

// GetRouter returns the app's Router.
func (app *BinanceChain) GetRouter() router.Router {
	return app.Router()
}

// GetContextForCheckState gets the context for the check state.
func (app *BinanceChain) GetContextForCheckState() sdk.Context {
	return app.checkState.ctx
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

// MakeCodec creates a custom tx codec.
func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	wire.RegisterCrypto(cdc) // Register crypto.
	bank.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc) // Register Msgs
	dex.RegisterWire(cdc)
	tokens.RegisterWire(cdc)
	types.RegisterWire(cdc)
	tx.RegisterWire(cdc)

	return cdc
}

func (app *BinanceChain) publish(tradesToPublish []pub.Trade, ctx sdk.Context, height, blockTime int64) {
	pub.Logger.Info("start to collect publish information", "height", height)

	var accountsToPublish map[string]pub.Account
	if app.publicationConfig.PublishAccountBalance {
		txRelatedAccounts, _ := ctx.Value(InvolvedAddressKey).([]string)
		tradeRelatedAccounts := app.DexKeeper.GetTradeAndOrdersRelatedAccounts(app.DexKeeper.OrderChanges)
		accountsToPublish = pub.GetAccountBalances(app.AccountKeeper, ctx, txRelatedAccounts, tradeRelatedAccounts)
		defer func() {
			app.deliverState.ctx = ctx.WithValue(InvolvedAddressKey, make([]string, 0))
		}() // clean up
	}

	var latestPriceLevels order.ChangedPriceLevelsMap
	if app.publicationConfig.PublishOrderBook {
		latestPriceLevels = app.DexKeeper.GetOrderBooks(pub.MaxOrderBookLevel)
	}

	pub.Logger.Info("start to publish", "height", height,
		"blockTime", blockTime, "numOfTrades", len(tradesToPublish),
		"numOfOrders", // the order num we collected here doesn't include trade related orders
		len(app.DexKeeper.OrderChanges),
		"numOfAccounts",
		len(accountsToPublish))
	pub.ToRemoveOrderIdCh = make(chan string, pub.ToRemoveOrderIdChannelSize)
	pub.ToPublishCh <- pub.NewBlockInfoToPublish(
		height,
		blockTime,
		tradesToPublish,
		app.DexKeeper.OrderChanges,    // thread-safety is guarded by the signal from RemoveDoneCh
		app.DexKeeper.OrderChangesMap, // ditto
		accountsToPublish,
		latestPriceLevels)

	// remove item from OrderInfoForPublish when we published removed order (cancel, iocnofill, fullyfilled, expired)
	for id := range pub.ToRemoveOrderIdCh {
		pub.Logger.Debug("delete order from order changes map", "orderId", id)
		delete(app.DexKeeper.OrderChangesMap, id)
	}

	// clean up intermediate cached data
	app.DexKeeper.ClearOrderChanges()
	pub.Logger.Debug("finish publish", "height", height)
}
