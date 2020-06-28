package app

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"runtime/debug"
	"sort"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/fees"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/oracle"
	"github.com/cosmos/cosmos-sdk/x/paramHub"
	param "github.com/cosmos/cosmos-sdk/x/paramHub/keeper"
	paramTypes "github.com/cosmos/cosmos-sdk/x/paramHub/types"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/stake"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmstore "github.com/tendermint/tendermint/store"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/binance-chain/node/admin"
	"github.com/binance-chain/node/app/config"
	"github.com/binance-chain/node/app/pub"
	"github.com/binance-chain/node/common"
	"github.com/binance-chain/node/common/runtime"
	"github.com/binance-chain/node/common/tx"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/account"
	"github.com/binance-chain/node/plugins/bridge"
	"github.com/binance-chain/node/plugins/dex"
	"github.com/binance-chain/node/plugins/dex/list"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/plugins/tokens/issue"
	"github.com/binance-chain/node/plugins/tokens/seturi"
	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/plugins/tokens/timelock"
	"github.com/binance-chain/node/wire"
)

const (
	appName = "BNBChain"
)

// default home directories for expected binaries
var (
	DefaultCLIHome      = os.ExpandEnv("$HOME/.bnbcli")
	DefaultNodeHome     = os.ExpandEnv("$HOME/.bnbchaind")
	Bech32PrefixAccAddr string
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
	CoinKeeper     bank.Keeper
	DexKeeper      *dex.DexKeeper
	AccountKeeper  auth.AccountKeeper
	TokenMapper    tokens.Mapper
	ValAddrCache   *ValAddrCache
	stakeKeeper    stake.Keeper
	slashKeeper    slashing.Keeper
	govKeeper      gov.Keeper
	timeLockKeeper timelock.Keeper
	swapKeeper     swap.Keeper
	oracleKeeper   oracle.Keeper
	bridgeKeeper   bridge.Keeper
	ibcKeeper      ibc.Keeper
	scKeeper       sidechain.Keeper
	// keeper to process param store and update
	ParamHub *param.Keeper

	baseConfig         *config.BaseConfig
	upgradeConfig      *config.UpgradeConfig
	crossChainConfig   *config.CrossChainConfig
	abciQueryBlackList map[string]bool
	publicationConfig  *config.PublicationConfig
	publisher          pub.MarketDataPublisher

	dexConfig *config.DexConfig

	// Unlike tendermint, we don't need implement a no-op metrics, usage of this field should
	// check nil-ness to know whether metrics collection is turn on
	// TODO(#246): make it an aggregated wrapper of all component metrics (i.e. DexKeeper, StakeKeeper)
	metrics *pub.Metrics

	takeSnapshotHeight int64 // whether to take snapshot of current height, set at endblock(), reset at commit()
}

// NewBinanceChain creates a new instance of the BinanceChain.
func NewBinanceChain(logger log.Logger, db dbm.DB, traceStore io.Writer, baseAppOptions ...func(*baseapp.BaseApp)) *BinanceChain {
	// create app-level codec for txs and accounts
	var cdc = Codec
	// create composed tx decoder
	decoders := wire.ComposeTxDecoders(cdc, defaultTxDecoder)

	// create the applicationsimulate object
	var app = &BinanceChain{
		BaseApp:            baseapp.NewBaseApp(appName /*, cdc*/, logger, db, decoders, sdk.CollectConfig{ServerContext.PublishAccountBalance, ServerContext.PublishTransfer || ServerContext.PublishBlock}, baseAppOptions...),
		Codec:              cdc,
		queryHandlers:      make(map[string]types.AbciQueryHandler),
		baseConfig:         ServerContext.BaseConfig,
		upgradeConfig:      ServerContext.UpgradeConfig,
		crossChainConfig:   ServerContext.CrossChainConfig,
		abciQueryBlackList: getABCIQueryBlackList(ServerContext.QueryConfig),
		publicationConfig:  ServerContext.PublicationConfig,
		dexConfig:          ServerContext.DexConfig,
	}
	// set upgrade config
	SetUpgradeConfig(app.upgradeConfig)
	app.initRunningMode()
	app.SetCommitMultiStoreTracer(traceStore)

	// mappers
	app.AccountKeeper = auth.NewAccountKeeper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	app.TokenMapper = tokens.NewMapper(cdc, common.TokenStoreKey)
	app.CoinKeeper = bank.NewBaseKeeper(app.AccountKeeper)
	app.ParamHub = param.NewKeeper(cdc, common.ParamsStoreKey, common.TParamsStoreKey)
	app.scKeeper = sidechain.NewKeeper(common.SideChainStoreKey, app.ParamHub.Subspace(sidechain.DefaultParamspace))
	app.ibcKeeper = ibc.NewKeeper(common.IbcStoreKey, app.ParamHub.Subspace(ibc.DefaultParamspace), app.RegisterCodespace(ibc.DefaultCodespace), app.scKeeper)

	app.slashKeeper = slashing.NewKeeper(
		cdc,
		common.SlashingStoreKey, &app.stakeKeeper,
		app.ParamHub.Subspace(slashing.DefaultParamspace),
		app.RegisterCodespace(slashing.DefaultCodespace),
		app.CoinKeeper,
	)

	app.stakeKeeper = stake.NewKeeper(
		cdc,
		common.StakeStoreKey, common.TStakeStoreKey,
		app.CoinKeeper, app.Pool, app.ParamHub.Subspace(stake.DefaultParamspace),
		app.RegisterCodespace(stake.DefaultCodespace),
	)

	app.ValAddrCache = NewValAddrCache(app.stakeKeeper)

	app.govKeeper = gov.NewKeeper(
		cdc,
		common.GovStoreKey,
		app.ParamHub.Keeper, app.ParamHub.Subspace(gov.DefaultParamSpace), app.CoinKeeper, app.stakeKeeper,
		app.RegisterCodespace(gov.DefaultCodespace),
		app.Pool,
	)

	app.timeLockKeeper = timelock.NewKeeper(cdc, common.TimeLockStoreKey, app.CoinKeeper, app.AccountKeeper,
		timelock.DefaultCodespace)

	app.swapKeeper = swap.NewKeeper(cdc, common.AtomicSwapStoreKey, app.CoinKeeper, app.Pool, swap.DefaultCodespace)
	app.oracleKeeper = oracle.NewKeeper(cdc, common.OracleStoreKey, app.ParamHub.Subspace(oracle.DefaultParamSpace),
		app.stakeKeeper, app.scKeeper, app.ibcKeeper, app.CoinKeeper, app.Pool)
	app.bridgeKeeper = bridge.NewKeeper(cdc, common.BridgeStoreKey, app.AccountKeeper, app.TokenMapper, app.scKeeper, app.CoinKeeper,
		app.ibcKeeper, app.Pool, sdk.IbcChainID(app.crossChainConfig.BscIbcChainId))

	if ServerContext.Config.Instrumentation.Prometheus {
		app.metrics = pub.PrometheusMetrics() // TODO(#246): make it an aggregated wrapper of all component metrics (i.e. DexKeeper, StakeKeeper)
	}

	if app.publicationConfig.ShouldPublishAny() {
		pub.Logger = logger.With("module", "pub")
		pub.Cfg = app.publicationConfig
		pub.ToPublishCh = make(chan pub.BlockInfoToPublish, app.publicationConfig.PublicationChannelSize)

		publishers := make([]pub.MarketDataPublisher, 0, 1)
		if app.publicationConfig.PublishKafka {
			publishers = append(publishers, pub.NewKafkaMarketDataPublisher(app.Logger, ServerContext.Config.DBDir(), app.publicationConfig.StopOnKafkaFail))
		}
		if app.publicationConfig.PublishLocal {
			publishers = append(publishers, pub.NewLocalMarketDataPublisher(ServerContext.Config.RootDir, app.Logger, app.publicationConfig))
		}

		if len(publishers) == 0 {
			panic(fmt.Errorf("Cannot find any publisher in config, there might be some wrong configuration"))
		} else {
			if len(publishers) == 1 {
				app.publisher = publishers[0]
			} else {
				app.publisher = pub.NewAggregatedMarketDataPublisher(publishers...)
			}

			go pub.Publish(app.publisher, app.metrics, logger, app.publicationConfig, pub.ToPublishCh)
			pub.IsLive = true
		}
	}

	// finish app initialization
	app.SetInitChainer(app.initChainerFn())
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.MountStoresIAVL(
		common.MainStoreKey,
		common.AccountStoreKey,
		common.ValAddrStoreKey,
		common.TokenStoreKey,
		common.DexStoreKey,
		common.PairStoreKey,
		common.ParamsStoreKey,
		common.StakeStoreKey,
		common.SlashingStoreKey,
		common.GovStoreKey,
		common.TimeLockStoreKey,
		common.AtomicSwapStoreKey,
		common.SideChainStoreKey,
		common.BridgeStoreKey,
		common.OracleStoreKey,
		common.IbcStoreKey,
	)
	app.SetAnteHandler(tx.NewAnteHandler(app.AccountKeeper))
	app.SetPreChecker(tx.NewTxPreChecker())
	app.MountStoresTransient(common.TParamsStoreKey, common.TStakeStoreKey)

	// block store required to hydrate dex OB
	err := app.LoadCMSLatestVersion()
	if err != nil {
		cmn.Exit(err.Error())
	}

	// init app cache
	accountStore := app.BaseApp.GetCommitMultiStore().GetKVStore(common.AccountStoreKey)
	app.SetAccountStoreCache(cdc, accountStore, app.baseConfig.AccountCacheSize)

	tx.InitSigCache(app.baseConfig.SignatureCacheSize)

	err = app.InitFromStore(common.MainStoreKey)
	if err != nil {
		cmn.Exit(err.Error())
	}

	// remaining plugin init
	app.initPlugins()

	if ServerContext.Config.StateSyncReactor {
		lastBreatheBlockHeight := app.getLastBreatheBlockHeight()
		app.StateSyncHelper = store.NewStateSyncHelper(app.Logger.With("module", "statesync"), db, app.GetCommitMultiStore(), app.Codec)
		app.StateSyncHelper.Init(lastBreatheBlockHeight)
	}

	return app
}

// setUpgradeConfig will overwrite default upgrade config
func SetUpgradeConfig(upgradeConfig *config.UpgradeConfig) {
	// register upgrade height
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP6, upgradeConfig.BEP6Height)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP9, upgradeConfig.BEP9Height)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP10, upgradeConfig.BEP10Height)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP12, upgradeConfig.BEP12Height)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP19, upgradeConfig.BEP19Height)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP3, upgradeConfig.BEP3Height)
	upgrade.Mgr.AddUpgradeHeight(upgrade.FixSignBytesOverflow, upgradeConfig.FixSignBytesOverflowHeight)
	upgrade.Mgr.AddUpgradeHeight(upgrade.LotSizeOptimization, upgradeConfig.LotSizeUpgradeHeight)
	upgrade.Mgr.AddUpgradeHeight(upgrade.ListingRuleUpgrade, upgradeConfig.ListingRuleUpgradeHeight)
	upgrade.Mgr.AddUpgradeHeight(upgrade.FixZeroBalance, upgradeConfig.FixZeroBalanceHeight)
	upgrade.Mgr.AddUpgradeHeight(upgrade.LaunchBscUpgrade, upgradeConfig.LaunchBscUpgradeHeight)

	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP8, upgradeConfig.BEP8Height)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP67, upgradeConfig.BEP67Height)
	upgrade.Mgr.AddUpgradeHeight(upgrade.BEP70, upgradeConfig.BEP70Height)

	// register store keys of upgrade
	upgrade.Mgr.RegisterStoreKeys(upgrade.BEP9, common.TimeLockStoreKey.Name())
	upgrade.Mgr.RegisterStoreKeys(upgrade.BEP3, common.AtomicSwapStoreKey.Name())
	upgrade.Mgr.RegisterStoreKeys(upgrade.LaunchBscUpgrade, common.IbcStoreKey.Name(), common.SideChainStoreKey.Name(),
		common.SlashingStoreKey.Name(), common.BridgeStoreKey.Name(), common.OracleStoreKey.Name())

	// register msg types of upgrade
	upgrade.Mgr.RegisterMsgTypes(upgrade.BEP9,
		timelock.TimeLockMsg{}.Type(),
		timelock.TimeRelockMsg{}.Type(),
		timelock.TimeUnlockMsg{}.Type(),
	)
	upgrade.Mgr.RegisterMsgTypes(upgrade.BEP12, account.SetAccountFlagsMsg{}.Type())
	upgrade.Mgr.RegisterMsgTypes(upgrade.BEP3,
		swap.HTLTMsg{}.Type(),
		swap.DepositHTLTMsg{}.Type(),
		swap.ClaimHTLTMsg{}.Type(),
		swap.RefundHTLTMsg{}.Type(),
	)
	upgrade.Mgr.RegisterMsgTypes(upgrade.LaunchBscUpgrade,
		stake.MsgCreateSideChainValidator{}.Type(),
		stake.MsgEditSideChainValidator{}.Type(),
		stake.MsgSideChainDelegate{}.Type(),
		stake.MsgSideChainRedelegate{}.Type(),
		stake.MsgSideChainUndelegate{}.Type(),
		slashing.MsgBscSubmitEvidence{}.Type(),
		slashing.MsgSideChainUnjail{}.Type(),
		gov.MsgSideChainSubmitProposal{}.Type(),
		gov.MsgSideChainDeposit{}.Type(),
		gov.MsgSideChainVote{}.Type(),
		bridge.BindMsg{}.Type(),
		bridge.UnbindMsg{}.Type(),
		bridge.TransferOutMsg{}.Type(),
		oracle.ClaimMsg{}.Type(),
	)
	// register msg types of upgrade
	upgrade.Mgr.RegisterMsgTypes(upgrade.BEP8,
		issue.IssueMiniMsg{}.Type(),
		issue.IssueTinyMsg{}.Type(),
		seturi.SetURIMsg{}.Type(),
		list.ListMiniMsg{}.Type(),
	)
}

func getABCIQueryBlackList(queryConfig *config.QueryConfig) map[string]bool {
	cfg := make(map[string]bool)
	for _, path := range queryConfig.ABCIQueryBlackList {
		cfg[path] = true
	}
	return cfg
}

func (app *BinanceChain) initRunningMode() {
	err := runtime.RecoverFromFile(ServerContext.Config.RootDir, runtime.Mode(ServerContext.StartMode))
	if err != nil {
		cmn.Exit(err.Error())
	}
}

func (app *BinanceChain) initDex() {
	pairMapper := dex.NewTradingPairMapper(app.Codec, common.PairStoreKey)
	app.DexKeeper = dex.NewDexKeeper(common.DexStoreKey, app.AccountKeeper, pairMapper,
		app.RegisterCodespace(dex.DefaultCodespace), app.baseConfig.OrderKeeperConcurrency, app.Codec,
		app.publicationConfig.ShouldPublishAny())
	app.DexKeeper.SubscribeParamChange(app.ParamHub)
	app.DexKeeper.SetBUSDSymbol(app.dexConfig.BUSDSymbol)

	// do not proceed if we are in a unit test and `CheckState` is unset.
	if app.CheckState == nil {
		return
	}
	// count back to days in config.
	blockDB := baseapp.LoadBlockDB()
	defer blockDB.Close()
	blockStore := tmstore.NewBlockStore(blockDB)
	stateDB := baseapp.LoadStateDB()
	defer stateDB.Close()

	app.DexKeeper.Init(
		app.CheckState.Ctx,
		app.baseConfig.BreatheBlockInterval,
		app.baseConfig.BreatheBlockDaysCountBack,
		blockStore,
		stateDB,
		app.LastBlockHeight(),
		app.TxDecoder)

}

func (app *BinanceChain) initPlugins() {
	app.initSideChain()
	app.initIbc()
	app.initDex()
	app.initGov()
	app.initGovHooks()
	app.initStaking()
	app.initSlashing()
	app.initOracle()
	app.initParamHub()
	tokens.InitPlugin(app, app.TokenMapper, app.AccountKeeper, app.CoinKeeper, app.timeLockKeeper, app.swapKeeper)
	dex.InitPlugin(app, app.DexKeeper, app.TokenMapper, app.govKeeper)
	account.InitPlugin(app, app.AccountKeeper)
	bridge.InitPlugin(app, app.bridgeKeeper)
	app.initParams()

	// add handlers from bnc-cosmos-sdk (others moved to plugin init funcs)
	// we need to add handlers after all keepers initialized
	app.Router().
		AddRoute("bank", bank.NewHandler(app.CoinKeeper)).
		AddRoute("stake", stake.NewHandler(app.stakeKeeper, app.govKeeper)).
		AddRoute("slashing", slashing.NewHandler(app.slashKeeper)).
		AddRoute("gov", gov.NewHandler(app.govKeeper)).
		AddRoute(oracle.RouteOracle, oracle.NewHandler(app.oracleKeeper))

	app.QueryRouter().AddRoute("gov", gov.NewQuerier(app.govKeeper))
	app.QueryRouter().AddRoute("stake", stake.NewQuerier(app.stakeKeeper, app.Codec))
	app.QueryRouter().AddRoute("slashing", slashing.NewQuerier(app.slashKeeper, app.Codec))
	app.QueryRouter().AddRoute("timelock", timelock.NewQuerier(app.timeLockKeeper))
	app.QueryRouter().AddRoute(swap.AtomicSwapRoute, swap.NewQuerier(app.swapKeeper))
	app.QueryRouter().AddRoute("param", paramHub.NewQuerier(app.ParamHub, app.Codec))

	app.RegisterQueryHandler("account", app.AccountHandler)
	app.RegisterQueryHandler("admin", admin.GetHandler(ServerContext.Config))

}

func (app *BinanceChain) initSideChain() {
	upgrade.Mgr.RegisterBeginBlocker(sdk.LaunchBscUpgrade, func(ctx sdk.Context) {
		bscStorePrefix := []byte{0x99}
		app.scKeeper.SetSideChainIdAndStorePrefix(ctx, ServerContext.BscChainId, bscStorePrefix)
		app.scKeeper.SetParams(ctx, sidechain.Params{
			BscSideChainId: ServerContext.BscChainId,
		})
	})
}

func (app *BinanceChain) initOracle() {
	if ServerContext.Config.Instrumentation.Prometheus {
		app.oracleKeeper.EnablePrometheusMetrics()
	}
	app.oracleKeeper.SubscribeParamChange(app.ParamHub)
	oracle.RegisterUpgradeBeginBlocker(app.oracleKeeper)
}

func (app *BinanceChain) initParamHub() {
	paramHub.RegisterUpgradeBeginBlocker(app.ParamHub)
	handler := paramHub.CreateAbciQueryHandler(app.ParamHub)
	// paramHub used to be a plugin of node, we still keep the old api here.
	app.RegisterQueryHandler(paramHub.AbciQueryPrefix, func(app types.ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery) {
		return handler(app.GetContextForCheckState(), req, path)
	})
}

func (app *BinanceChain) initStaking() {
	app.stakeKeeper.SetupForSideChain(&app.scKeeper, &app.ibcKeeper)
	upgrade.Mgr.RegisterBeginBlocker(sdk.LaunchBscUpgrade, func(ctx sdk.Context) {
		stake.MigratePowerRankKey(ctx, app.stakeKeeper)
		storePrefix := app.scKeeper.GetSideChainStorePrefix(ctx, ServerContext.BscChainId)
		newCtx := ctx.WithSideChainKeyPrefix(storePrefix)
		app.stakeKeeper.SetParams(newCtx, stake.Params{
			UnbondingTime:       60 * 60 * 24 * 7 * time.Second, // 7 days
			MaxValidators:       11,
			BondDenom:           types.NativeTokenSymbol,
			MinSelfDelegation:   50000e8,
			MinDelegationChange: 1e8,
		})
		app.stakeKeeper.SetPool(newCtx, stake.Pool{
			// TODO: optimize these parameters
			LooseTokens: sdk.NewDec(5e15),
		})
	})
	app.stakeKeeper.SubscribeParamChange(app.ParamHub)
	app.stakeKeeper = app.stakeKeeper.WithHooks(app.slashKeeper.Hooks())
}

func (app *BinanceChain) initGov() {
	app.govKeeper.SetupForSideChain(&app.scKeeper)
	upgrade.Mgr.RegisterBeginBlocker(sdk.LaunchBscUpgrade, func(ctx sdk.Context) {
		storePrefix := app.scKeeper.GetSideChainStorePrefix(ctx, ServerContext.BscChainId)
		newCtx := ctx.WithSideChainKeyPrefix(storePrefix)
		err := app.govKeeper.SetInitialProposalID(newCtx, 1)
		if err != nil {
			panic(err)
		}
		app.govKeeper.SetDepositParams(newCtx, gov.DepositParams{
			MinDeposit:       sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, 2000e8)},
			MaxDepositPeriod: time.Duration(2*24) * time.Hour, // 2 days
		})
		app.govKeeper.SetTallyParams(newCtx, gov.TallyParams{
			Quorum:    sdk.NewDecWithPrec(5, 1),
			Threshold: sdk.NewDecWithPrec(5, 1),
			Veto:      sdk.NewDecWithPrec(334, 3),
		})
	})
}

func (app *BinanceChain) initSlashing() {
	app.slashKeeper.SetSideChain(&app.scKeeper)
	app.slashKeeper.SubscribeParamChange(app.ParamHub)
	upgrade.Mgr.RegisterBeginBlocker(sdk.LaunchBscUpgrade, func(ctx sdk.Context) {
		storePrefix := app.scKeeper.GetSideChainStorePrefix(ctx, ServerContext.BscChainId)
		newCtx := ctx.WithSideChainKeyPrefix(storePrefix)
		app.slashKeeper.SetParams(newCtx, slashing.Params{
			MaxEvidenceAge:           60 * 60 * 24 * 3 * time.Second, // 3 days
			DoubleSignUnbondDuration: math.MaxInt64,                  // forever
			DowntimeUnbondDuration:   60 * 60 * 24 * 2 * time.Second, // 2 days
			TooLowDelUnbondDuration:  60 * 60 * 24 * time.Second,     // 1 day
			DoubleSignSlashAmount:    10000e8,
			SubmitterReward:          1000e8,
			DowntimeSlashAmount:      50e8,
			DowntimeSlashFee:         10e8,
		})
	})
}

func (app *BinanceChain) initIbc() {
	// set up IBC chainID for BBC
	app.scKeeper.SetSrcIbcChainID(sdk.IbcChainID(ServerContext.IbcChainId))
	// set up IBC chainID for BSC
	err := app.scKeeper.RegisterDestChain(ServerContext.BscChainId, sdk.IbcChainID(ServerContext.BscIbcChainId))
	if err != nil {
		panic(fmt.Sprintf("register IBC chainID error: chainID=%s, err=%s", ServerContext.BscChainId, err.Error()))
	}
	app.ibcKeeper.SubscribeParamChange(app.ParamHub)
	upgrade.Mgr.RegisterBeginBlocker(sdk.LaunchBscUpgrade, func(ctx sdk.Context) {
		storePrefix := app.scKeeper.GetSideChainStorePrefix(ctx, ServerContext.BscChainId)
		newCtx := ctx.WithSideChainKeyPrefix(storePrefix)
		app.ibcKeeper.SetParams(newCtx, ibc.Params{
			RelayerFee: ibc.DefaultRelayerFeeParam,
		})
	})
}

func (app *BinanceChain) initGovHooks() {
	listHooks := list.NewListHooks(app.DexKeeper, app.TokenMapper)
	feeChangeHooks := paramHub.NewFeeChangeHooks(app.Codec)
	cscParamChangeHooks := paramHub.NewCSCParamsChangeHook(app.Codec)
	scParamChangeHooks := paramHub.NewSCParamsChangeHook(app.Codec)
	delistHooks := list.NewDelistHooks(app.DexKeeper)
	app.govKeeper.AddHooks(gov.ProposalTypeListTradingPair, listHooks)
	app.govKeeper.AddHooks(gov.ProposalTypeFeeChange, feeChangeHooks)
	app.govKeeper.AddHooks(gov.ProposalTypeCSCParamsChange, cscParamChangeHooks)
	app.govKeeper.AddHooks(gov.ProposalTypeSCParamsChange, scParamChangeHooks)
	app.govKeeper.AddHooks(gov.ProposalTypeDelistTradingPair, delistHooks)
}

func (app *BinanceChain) initParams() {
	app.ParamHub.SetGovKeeper(&app.govKeeper)
	app.ParamHub.SetupForSideChain(&app.scKeeper, &app.ibcKeeper)
	upgrade.Mgr.RegisterBeginBlocker(sdk.LaunchBscUpgrade, func(ctx sdk.Context) {
		storePrefix := app.scKeeper.GetSideChainStorePrefix(ctx, ServerContext.BscChainId)
		newCtx := ctx.WithSideChainKeyPrefix(storePrefix)
		app.ParamHub.SetLastSCParamChangeProposalId(newCtx, paramTypes.LastProposalID{ProposalID: 0})
	})
	if app.CheckState != nil && app.CheckState.Ctx.BlockHeight() != 0 {
		app.ParamHub.Load(app.CheckState.Ctx)
	}
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

		selfDelegationAddrs := make([]sdk.AccAddress, 0, len(genesisState.Accounts))
		for _, gacc := range genesisState.Accounts {
			acc := gacc.ToAppAccount()
			acc.AccountNumber = app.AccountKeeper.GetNextAccountNumber(ctx)
			app.AccountKeeper.SetAccount(ctx, acc)
			// this relies on that the non-operator addresses are all used for self-delegation
			if len(gacc.ConsensusAddr) == 0 {
				selfDelegationAddrs = append(selfDelegationAddrs, acc.Address)
			}
		}
		tokens.InitGenesis(ctx, app.TokenMapper, app.CoinKeeper, genesisState.Tokens,
			selfDelegationAddrs, DefaultSelfDelegationToken.Amount)

		app.ParamHub.InitGenesis(ctx, genesisState.ParamGenesis)
		validators, err := stake.InitGenesis(ctx, app.stakeKeeper, genesisState.StakeData)
		gov.InitGenesis(ctx, app.govKeeper, genesisState.GovData)

		if err != nil {
			panic(err) // TODO find a way to do this w/o panics
		}

		// before we deliver the genTxs, we have transferred some delegation tokens to the validator candidates.
		if len(genesisState.GenTxs) > 0 {
			for _, genTx := range genesisState.GenTxs {
				var tx auth.StdTx
				err = app.Codec.UnmarshalJSON(genTx, &tx)
				if err != nil {
					panic(err)
				}
				bz := app.Codec.MustMarshalBinaryLengthPrefixed(tx)
				res := app.BaseApp.DeliverTx(abci.RequestDeliverTx{Tx: bz})
				if !res.IsOK() {
					panic(res.Log)
				}
			}
			_, validators = app.stakeKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
		}

		// sanity check
		if len(req.Validators) > 0 {
			if len(req.Validators) != len(validators) {
				panic(fmt.Errorf("genesis validator numbers are not matched, staked=%d, req=%d",
					len(validators), len(req.Validators)))
			}
			sort.Sort(abci.ValidatorUpdates(req.Validators))
			sort.Sort(abci.ValidatorUpdates(validators))
			for i, val := range validators {
				if !val.Equal(req.Validators[i]) {
					panic(fmt.Errorf("invalid genesis validator, index=%d", i))
				}
			}
		}

		return abci.ResponseInitChain{
			Validators: validators,
		}
	}
}

func (app *BinanceChain) CheckTx(req abci.RequestCheckTx) (res abci.ResponseCheckTx) {
	var result sdk.Result
	var tx sdk.Tx
	txBytes := req.Tx
	// try to get the Tx first from cache, if succeed, it means it is PreChecked.
	tx, ok := app.GetTxFromCache(txBytes)
	if ok {
		if admin.IsTxAllowed(tx) {
			txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
			app.Logger.Debug("Handle CheckTx", "Tx", txHash)
			result = app.RunTx(sdk.RunTxModeCheckAfterPre, tx, txHash)
			if !result.IsOK() {
				app.RemoveTxFromCache(txBytes)
			}
		} else {
			result = admin.TxNotAllowedError().Result()
		}
	} else {
		tx, err := app.TxDecoder(txBytes)
		if err != nil {
			result = err.Result()
		} else {
			if admin.IsTxAllowed(tx) {
				txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
				app.Logger.Debug("Handle CheckTx", "Tx", txHash)
				result = app.RunTx(sdk.RunTxModeCheck, tx, txHash)
				if result.IsOK() {
					app.AddTxToCache(txBytes, tx)
				}
			} else {
				result = admin.TxNotAllowedError().Result()
			}
		}
	}

	return abci.ResponseCheckTx{
		Code:   uint32(result.Code),
		Data:   result.Data,
		Log:    result.Log,
		Events: result.GetEvents(),
	}
}

// Implements ABCI
func (app *BinanceChain) DeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {
	res = app.BaseApp.DeliverTx(req)
	txHash := cmn.HexBytes(tmhash.Sum(req.Tx)).String()
	if res.IsOK() {
		// commit or panic
		fees.Pool.CommitFee(txHash)
	} else {
		if app.publicationConfig.PublishOrderUpdates {
			app.processErrAbciResponseForPub(req.Tx)
		}
	}
	if app.publicationConfig.PublishBlock {
		pub.Pool.AddTxRes(txHash, res)
	}
	return res
}

// PreDeliverTx implements extended ABCI for concurrency
// PreCheckTx would perform decoding, signture and other basic verification
func (app *BinanceChain) PreDeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {
	res = app.BaseApp.PreDeliverTx(req)
	if res.IsErr() {
		txHash := cmn.HexBytes(tmhash.Sum(req.Tx)).String()
		app.Logger.Error("failed to process invalid tx during pre-deliver", "tx", txHash, "res", res.String())
		// TODO(#446): comment out temporally for thread safety
		//if app.publicationConfig.PublishOrderUpdates {
		//	app.processErrAbciResponseForPub(txBytes)
		//}
	}
	return res
}

func (app *BinanceChain) isBreatheBlock(height int64, lastBlockTime time.Time, blockTime time.Time) bool {
	// lastBlockTime is zero if this blockTime is for the first block (first block doesn't mean height = 1, because after
	// state sync from breathe block, the height is breathe block + 1)
	if app.baseConfig.BreatheBlockInterval > 0 {
		return height%int64(app.baseConfig.BreatheBlockInterval) == 0
	} else {
		return !lastBlockTime.IsZero() && !utils.SameDayInUTC(lastBlockTime, blockTime)
	}
}

func (app *BinanceChain) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {
	upgrade.Mgr.BeginBlocker(ctx)
	return
}

func (app *BinanceChain) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	// lastBlockTime would be 0 if this is the first block.
	lastBlockTime := app.CheckState.Ctx.BlockHeader().Time
	blockTime := ctx.BlockHeader().Time
	height := ctx.BlockHeader().Height
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	isBreatheBlock := app.isBreatheBlock(height, lastBlockTime, blockTime)
	var tradesToPublish []*pub.Trade
	if sdk.IsUpgrade(upgrade.BEP19) || !isBreatheBlock {
		if app.publicationConfig.ShouldPublishAny() && pub.IsLive {
			tradesToPublish = pub.MatchAndAllocateAllForPublish(app.DexKeeper, ctx, isBreatheBlock)
		} else {
			app.DexKeeper.MatchAndAllocateSymbols(ctx, nil, isBreatheBlock)
		}
	}

	if isBreatheBlock {
		// breathe block
		app.Logger.Info("Start Breathe Block Handling",
			"height", height, "lastBlockTime", lastBlockTime, "newBlockTime", blockTime)
		app.takeSnapshotHeight = height
		fmt.Println(ctx.BlockHeight())
		dex.EndBreatheBlock(ctx, app.DexKeeper, app.govKeeper, height, blockTime)
		paramHub.EndBreatheBlock(ctx, app.ParamHub)
		tokens.EndBreatheBlock(ctx, app.swapKeeper)
	} else {
		app.Logger.Debug("normal block", "height", height)
	}

	app.DexKeeper.StoreTradePrices(ctx)

	blockFee := distributeFee(ctx, app.AccountKeeper, app.ValAddrCache, app.publicationConfig.PublishBlockFee)

	passed, failed := gov.EndBlocker(ctx, app.govKeeper)
	var proposals pub.Proposals
	if app.publicationConfig.PublishOrderUpdates {
		//TODO, "passed" and "failed" contains proposalId and ChainId, please publish chain id as well during
		// the refactor of publisher.
		proposals = pub.CollectProposalsForPublish(passed, failed)
	}
	paramHub.EndBlock(ctx, app.ParamHub)
	var completedUbd []stake.UnbondingDelegation
	var validatorUpdates abci.ValidatorUpdates
	if isBreatheBlock {
		validatorUpdates, completedUbd = stake.EndBreatheBlock(ctx, app.stakeKeeper)
	} else if ctx.RouterCallRecord()["stake"] {
		validatorUpdates, completedUbd = stake.EndBlocker(ctx, app.stakeKeeper)
	}
	ibc.EndBlocker(ctx, app.ibcKeeper)
	if len(validatorUpdates) != 0 {
		app.ValAddrCache.ClearCache()
	}

	if app.publicationConfig.ShouldPublishAny() &&
		pub.IsLive {
		var stakeUpdates pub.StakeUpdates
		stakeUpdates = pub.CollectStakeUpdatesForPublish(completedUbd)
		if height >= app.publicationConfig.FromHeightInclusive {
			app.publish(tradesToPublish, &proposals, &stakeUpdates, blockFee, ctx, height, blockTime.UnixNano())
		}

		// clean up intermediate cached data
		app.DexKeeper.ClearOrderChanges()
		app.DexKeeper.ClearRoundFee()
	}
	fees.Pool.Clear()
	// just clean it, no matter use it or not.
	pub.Pool.Clean()
	//match may end with transaction failure, which is better to save into
	//the EndBlock response. However, current cosmos doesn't support this.
	//future TODO: add failure info.
	return abci.ResponseEndBlock{
		ValidatorUpdates: validatorUpdates,
		Events:           ctx.EventManager().ABCIEvents(),
	}
}

func (app *BinanceChain) Commit() (res abci.ResponseCommit) {
	res = app.BaseApp.Commit()
	if ServerContext.Config.StateSyncReactor && app.takeSnapshotHeight > 0 {
		app.StateSyncHelper.SnapshotHeights <- app.takeSnapshotHeight
		app.takeSnapshotHeight = 0
	}
	return
}

func (app *BinanceChain) WriteRecoveryChunk(hash abci.SHA256Sum, chunk *abci.AppStateChunk, isComplete bool) (err error) {
	err = app.BaseApp.WriteRecoveryChunk(hash, chunk, isComplete)
	if err != nil {
		return err
	}
	if isComplete {
		err = app.reInitChain()
	}
	return err
}

// ExportAppStateAndValidators exports blockchain world state to json.
func (app *BinanceChain) ExportAppStateAndValidators() (appState json.RawMessage, validators []tmtypes.GenesisValidator, err error) {
	ctx := app.NewContext(sdk.RunTxModeCheck, abci.Header{})

	// iterate to get the accounts
	accounts := []GenesisAccount{}
	appendAccount := func(acc sdk.Account) (stop bool) {
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
	appState, err = wire.MarshalJSONIndent(app.Codec, genState)
	if err != nil {
		return nil, nil, err
	}
	return appState, validators, nil
}

// Query performs an abci query.
func (app *BinanceChain) Query(req abci.RequestQuery) (res abci.ResponseQuery) {
	defer func() {
		if r := recover(); r != nil {
			app.Logger.Error("internal error caused by query", "req", req, "stack", debug.Stack())
			res = sdk.ErrInternal("internal error").QueryResult()
		}
	}()

	path := baseapp.SplitPath(req.Path)
	if len(path) == 0 {
		msg := "no query path provided"
		return sdk.ErrUnknownRequest(msg).QueryResult()
	}
	if app.abciQueryBlackList[req.Path] {
		msg := fmt.Sprintf("abci query interface (%s) is in black list", req.Path)
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

func (app *BinanceChain) AccountHandler(chainApp types.ChainApp, req abci.RequestQuery, path []string) *abci.ResponseQuery {
	var res abci.ResponseQuery
	if len(path) == 2 {
		addr := path[1]
		if accAddress, err := sdk.AccAddressFromBech32(addr); err == nil {

			if acc := app.CheckState.AccountCache.GetAccount(accAddress); acc != nil {
				bz, err := Codec.MarshalBinaryBare(acc)
				if err != nil {
					res = sdk.ErrInvalidAddress(addr).QueryResult()
				} else {
					res = abci.ResponseQuery{
						Code:  uint32(sdk.ABCICodeOK),
						Value: bz,
					}
				}
			} else {
				// let api server return 204 No Content
				res = abci.ResponseQuery{
					Code:  uint32(sdk.ABCICodeOK),
					Value: make([]byte, 0, 0),
				}
			}
		} else {
			res = sdk.ErrInvalidAddress(addr).QueryResult()
		}
	} else {
		res = sdk.ErrUnknownRequest("invalid path").QueryResult()
	}
	return &res
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
func (app *BinanceChain) GetRouter() baseapp.Router {
	return app.Router()
}

// GetContextForCheckState gets the context for the check state.
func (app *BinanceChain) GetContextForCheckState() sdk.Context {
	return app.CheckState.Ctx
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
		err := cdc.UnmarshalBinaryLengthPrefixed(txBytes, &tx)
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
	paramHub.RegisterWire(cdc)
	dex.RegisterWire(cdc)
	tokens.RegisterWire(cdc)
	account.RegisterWire(cdc)
	types.RegisterWire(cdc)
	tx.RegisterWire(cdc)
	stake.RegisterCodec(cdc)
	slashing.RegisterCodec(cdc)
	gov.RegisterCodec(cdc)
	bridge.RegisterWire(cdc)
	oracle.RegisterWire(cdc)
	ibc.RegisterWire(cdc)
	return cdc
}

func (app *BinanceChain) publish(tradesToPublish []*pub.Trade, proposalsToPublish *pub.Proposals, stakeUpdates *pub.StakeUpdates, blockFee pub.BlockFee, ctx sdk.Context, height, blockTime int64) {
	pub.Logger.Info("start to collect publish information", "height", height)

	var accountsToPublish map[string]pub.Account
	var transferToPublish *pub.Transfers
	var blockToPublish *pub.Block
	var latestPriceLevels order.ChangedPriceLevelsMap

	orderChanges := app.DexKeeper.GetAllOrderChanges()
	orderInfoForPublish := app.DexKeeper.GetAllOrderInfosForPub()

	duration := pub.Timer(app.Logger, fmt.Sprintf("collect publish information, height=%d", height), func() {
		if app.publicationConfig.PublishAccountBalance {
			txRelatedAccounts := app.Pool.TxRelatedAddrs()
			tradeRelatedAccounts := pub.GetTradeAndOrdersRelatedAccounts(tradesToPublish, orderChanges, orderInfoForPublish)
			accountsToPublish = pub.GetAccountBalances(
				app.AccountKeeper,
				ctx,
				txRelatedAccounts,
				tradeRelatedAccounts,
				blockFee.Validators)
		}
		if app.publicationConfig.PublishTransfer {
			transferToPublish = pub.GetTransferPublished(app.Pool, height, blockTime)
		}

		if app.publicationConfig.PublishBlock {
			header := ctx.BlockHeader()
			blockHash := ctx.BlockHash()
			blockToPublish = pub.GetBlockPublished(app.Pool, header, blockHash)
		}
		if app.publicationConfig.PublishOrderBook {
			latestPriceLevels = app.DexKeeper.GetOrderBooks(pub.MaxOrderBookLevel)
		}
	})

	if app.metrics != nil {
		app.metrics.CollectBlockTimeMs.Set(float64(duration))
	}

	pub.Logger.Info("start to publish", "height", height,
		"blockTime", blockTime, "numOfTrades", len(tradesToPublish),
		"numOfOrders", // the order num we collected here doesn't include trade related orders
		len(orderChanges),
		"numOfProposals",
		proposalsToPublish.NumOfMsgs,
		"numOfStakeUpdates",
		stakeUpdates.NumOfMsgs,
		"numOfAccounts",
		len(accountsToPublish))
	pub.ToRemoveOrderIdCh = make(chan pub.OrderSymbolId, pub.ToRemoveOrderIdChannelSize)

	pub.ToPublishCh <- pub.NewBlockInfoToPublish(
		height,
		blockTime,
		tradesToPublish,
		proposalsToPublish,
		stakeUpdates,
		orderChanges,        // thread-safety is guarded by the signal from RemoveDoneCh
		orderInfoForPublish, // thread-safety is guarded by the signal from RemoveDoneCh
		accountsToPublish,
		latestPriceLevels,
		blockFee,
		app.DexKeeper.RoundOrderFees, //only use DexKeeper RoundOrderFees
		transferToPublish,
		blockToPublish)

	// remove item from OrderInfoForPublish when we published removed order (cancel, iocnofill, fullyfilled, expired)
	for o := range pub.ToRemoveOrderIdCh {
		pub.Logger.Debug("delete order from order changes map", "symbol", o.Symbol, "orderId", o.Id)
		app.DexKeeper.RemoveOrderInfosForPub(o.Symbol, o.Id)
	}

	pub.Logger.Debug("finish publish", "height", height)
}
