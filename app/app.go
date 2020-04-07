package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/ibc"
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
	"github.com/binance-chain/node/common/fees"
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
	"github.com/binance-chain/node/plugins/oracle"
	"github.com/binance-chain/node/plugins/param"
	"github.com/binance-chain/node/plugins/param/paramhub"
	"github.com/binance-chain/node/plugins/tokens"
	tkstore "github.com/binance-chain/node/plugins/tokens/store"
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
	TokenMapper    tkstore.Mapper
	ValAddrCache   *ValAddrCache
	stakeKeeper    stake.Keeper
	govKeeper      gov.Keeper
	timeLockKeeper timelock.Keeper
	swapKeeper     swap.Keeper
	oracleKeeper   oracle.Keeper
	bridgeKeeper   bridge.Keeper
	ibcKeeper      ibc.Keeper
	// keeper to process param store and update
	ParamHub *param.ParamHub

	baseConfig         *config.BaseConfig
	upgradeConfig      *config.UpgradeConfig
	crossChainConfig   *config.CrossChainConfig
	abciQueryBlackList map[string]bool
	publicationConfig  *config.PublicationConfig
	publisher          pub.MarketDataPublisher

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
	}
	// set upgrade config
	SetUpgradeConfig(app.upgradeConfig)
	app.initRunningMode()
	app.SetCommitMultiStoreTracer(traceStore)

	// mappers
	app.AccountKeeper = auth.NewAccountKeeper(cdc, common.AccountStoreKey, types.ProtoAppAccount)
	app.TokenMapper = tkstore.NewMapper(cdc, common.TokenStoreKey)
	app.CoinKeeper = bank.NewBaseKeeper(app.AccountKeeper)
	app.ParamHub = paramhub.NewKeeper(cdc, common.ParamsStoreKey, common.TParamsStoreKey)
	app.ibcKeeper = ibc.NewKeeper(common.IbcStoreKey, app.RegisterCodespace(ibc.DefaultCodespace))

	app.stakeKeeper = stake.NewKeeper(
		cdc,
		common.StakeStoreKey, common.TStakeStoreKey,
		app.CoinKeeper, app.ibcKeeper, app.Pool, app.ParamHub.Subspace(stake.DefaultParamspace),
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
	app.ParamHub.SetGovKeeper(app.govKeeper)

	app.timeLockKeeper = timelock.NewKeeper(cdc, common.TimeLockStoreKey, app.CoinKeeper, app.AccountKeeper,
		timelock.DefaultCodespace)

	app.swapKeeper = swap.NewKeeper(cdc, common.AtomicSwapStoreKey, app.CoinKeeper, app.Pool, swap.DefaultCodespace)

	app.oracleKeeper = oracle.NewKeeper(cdc, common.OracleStoreKey, app.ParamHub.Subspace(oracle.DefaultParamSpace), app.stakeKeeper)
	app.ibcKeeper = ibc.NewKeeper(common.IbcStoreKey, ibc.DefaultCodespace)
	app.bridgeKeeper = bridge.NewKeeper(cdc, common.BridgeStoreKey, app.TokenMapper, app.oracleKeeper, app.CoinKeeper,
		app.ibcKeeper, app.Pool, app.crossChainConfig.BscChainId)

	// legacy bank route (others moved to plugin init funcs)
	app.Router().
		AddRoute("bank", bank.NewHandler(app.CoinKeeper)).
		AddRoute("stake", stake.NewHandler(app.stakeKeeper, app.govKeeper)).
		AddRoute("gov", gov.NewHandler(app.govKeeper))

	app.QueryRouter().AddRoute("gov", gov.NewQuerier(app.govKeeper))
	app.QueryRouter().AddRoute("stake", stake.NewQuerier(app.stakeKeeper, cdc))
	app.QueryRouter().AddRoute("timelock", timelock.NewQuerier(app.timeLockKeeper))
	app.QueryRouter().AddRoute(swap.AtomicSwapRoute, swap.NewQuerier(app.swapKeeper))

	app.RegisterQueryHandler("account", app.AccountHandler)
	app.RegisterQueryHandler("admin", admin.GetHandler(ServerContext.Config))

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
		common.GovStoreKey,
		common.TimeLockStoreKey,
		common.AtomicSwapStoreKey,
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
	app.registerInitFuncForUpgrade()

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

	// register store keys of upgrade
	upgrade.Mgr.RegisterStoreKeys(upgrade.BEP9, common.TimeLockStoreKey.Name())
	upgrade.Mgr.RegisterStoreKeys(upgrade.BEP3, common.AtomicSwapStoreKey.Name())
	upgrade.Mgr.RegisterStoreKeys(upgrade.LaunchBscUpgrade, common.BridgeStoreKey.Name())
	upgrade.Mgr.RegisterStoreKeys(upgrade.LaunchBscUpgrade, common.OracleStoreKey.Name())
	upgrade.Mgr.RegisterStoreKeys(upgrade.LaunchBscUpgrade, common.IbcStoreKey.Name())

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
		bridge.UpdateTransferOutMsg{}.Type(),
		bridge.TransferInMsg{}.Type(),
		bridge.BindMsg{}.Type(),
		bridge.TransferOutMsg{}.Type(),
		bridge.UpdateBindMsg{}.Type(),
	)
}

func (app *BinanceChain) registerInitFuncForUpgrade() {
	upgrade.Mgr.RegisterBeginBlocker(upgrade.LaunchBscUpgrade, func(ctx sdk.Context) {
		err := app.TokenMapper.UpdateBind(ctx, types.NativeTokenSymbol, "0x0000000000000000000000000000000000000000", 18)
		if err != nil {
			panic(err)
		}
	})
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
	app.DexKeeper = dex.NewOrderKeeper(common.DexStoreKey, app.AccountKeeper, pairMapper,
		app.RegisterCodespace(dex.DefaultCodespace), app.baseConfig.OrderKeeperConcurrency, app.Codec,
		app.publicationConfig.ShouldPublishAny())
	app.DexKeeper.SubscribeParamChange(app.ParamHub)

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
	app.initIbc()
	app.initDex()
	app.initGovHooks()
	app.initStaking()
	tokens.InitPlugin(app, app.TokenMapper, app.AccountKeeper, app.CoinKeeper, app.timeLockKeeper, app.swapKeeper)
	dex.InitPlugin(app, app.DexKeeper, app.TokenMapper, app.AccountKeeper, app.govKeeper)
	param.InitPlugin(app, app.ParamHub)
	account.InitPlugin(app, app.AccountKeeper)
	app.initParams()
	oracle.InitPlugin(app.oracleKeeper)
	bridge.InitPlugin(app, app.bridgeKeeper)
}

func (app *BinanceChain) initStaking() {
	upgrade.Mgr.RegisterBeginBlocker(sdk.LaunchBscUpgrade, func(ctx sdk.Context) {
		bscStorePrefix := []byte{0x99}
		app.stakeKeeper.SetSideChainIdAndStorePrefix(ctx, ServerContext.BscChainId, bscStorePrefix)
		newCtx := ctx.WithSideChainKeyPrefix(bscStorePrefix)
		app.stakeKeeper.SetParams(newCtx, stake.Params{
			UnbondingTime: 60 * 60 * 24 * time.Second, // 1 day
			MaxValidators: 21,
			BondDenom:     types.NativeTokenSymbol,
		})
		app.stakeKeeper.SetPool(newCtx, stake.Pool{
			// TODO: optimize these parameters
			LooseTokens: sdk.NewDec(5e15),
		})
	})
}

func (app *BinanceChain) initIbc() {
	// set up IBC chainID for BBC
	app.ibcKeeper.SetSrcIbcChainID(sdk.IbcChainID(ServerContext.IbcChainId))
	// set up IBC chainID for BSC
	err := app.ibcKeeper.RegisterDestChain(ServerContext.BscChainId, sdk.IbcChainID(ServerContext.BscIbcChainId))
	if err != nil {
		panic(fmt.Sprintf("register IBC chainID error: chainID=%s, err=%s", ServerContext.BscChainId, err.Error()))
	}
}

func (app *BinanceChain) initGovHooks() {
	listHooks := list.NewListHooks(app.DexKeeper, app.TokenMapper)
	feeChangeHooks := param.NewFeeChangeHooks(app.Codec)
	delistHooks := list.NewDelistHooks(app.DexKeeper)
	app.govKeeper.AddHooks(gov.ProposalTypeListTradingPair, listHooks)
	app.govKeeper.AddHooks(gov.ProposalTypeFeeChange, feeChangeHooks)
	app.govKeeper.AddHooks(gov.ProposalTypeDelistTradingPair, delistHooks)
}

func (app *BinanceChain) initParams() {
	if app.CheckState == nil || app.CheckState.Ctx.BlockHeight() == 0 {
		return
	}
	app.ParamHub.Load(app.CheckState.Ctx)
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
		Events: result.Tags.ToEvents(),
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

	isBreatheBlock := app.isBreatheBlock(height, lastBlockTime, blockTime)
	var tradesToPublish []*pub.Trade

	if sdk.IsUpgrade(upgrade.BEP19) || !isBreatheBlock {
		if app.publicationConfig.ShouldPublishAny() && pub.IsLive {
			tradesToPublish = pub.MatchAndAllocateAllForPublish(app.DexKeeper, ctx)
		} else {
			app.DexKeeper.MatchAndAllocateAll(ctx, nil)
		}
	}

	if isBreatheBlock {
		// breathe block
		app.Logger.Info("Start Breathe Block Handling",
			"height", height, "lastBlockTime", lastBlockTime, "newBlockTime", blockTime)
		app.takeSnapshotHeight = height
		fmt.Println(ctx.BlockHeight())
		dex.EndBreatheBlock(ctx, app.DexKeeper, app.govKeeper, height, blockTime)
		param.EndBreatheBlock(ctx, app.ParamHub)
		tokens.EndBreatheBlock(ctx, app.swapKeeper)

	} else {
		app.Logger.Debug("normal block", "height", height)
	}

	app.DexKeeper.StoreTradePrices(ctx)
	blockFee := distributeFee(ctx, app.AccountKeeper, app.ValAddrCache, app.publicationConfig.PublishBlockFee)

	tags, passed, failed := gov.EndBlocker(ctx, app.govKeeper)
	var proposals pub.Proposals
	if app.publicationConfig.PublishOrderUpdates {
		proposals = pub.CollectProposalsForPublish(passed, failed)
	}

	var completedUbd []stake.UnbondingDelegation
	var validatorUpdates abci.ValidatorUpdates
	var stakingTags sdk.Tags
	if isBreatheBlock {
		validatorUpdates, completedUbd, stakingTags = stake.EndBreatheBlock(ctx, app.stakeKeeper)
	} else if ctx.RouterCallRecord()["stake"] {
		validatorUpdates, completedUbd, stakingTags = stake.EndBlocker(ctx, app.stakeKeeper)
	}
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
	tags = tags.AppendTags(stakingTags)
	return abci.ResponseEndBlock{
		ValidatorUpdates: validatorUpdates,
		Events:           tags.ToEvents(),
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
	dex.RegisterWire(cdc)
	tokens.RegisterWire(cdc)
	account.RegisterWire(cdc)
	types.RegisterWire(cdc)
	tx.RegisterWire(cdc)
	stake.RegisterCodec(cdc)
	gov.RegisterCodec(cdc)
	param.RegisterWire(cdc)
	bridge.RegisterWire(cdc)
	oracle.RegisterWire(cdc)
	return cdc
}

func (app *BinanceChain) publish(tradesToPublish []*pub.Trade, proposalsToPublish *pub.Proposals, stakeUpdates *pub.StakeUpdates, blockFee pub.BlockFee, ctx sdk.Context, height, blockTime int64) {
	pub.Logger.Info("start to collect publish information", "height", height)

	var accountsToPublish map[string]pub.Account
	var transferToPublish *pub.Transfers
	var blockToPublish *pub.Block
	var latestPriceLevels order.ChangedPriceLevelsMap

	duration := pub.Timer(app.Logger, fmt.Sprintf("collect publish information, height=%d", height), func() {
		if app.publicationConfig.PublishAccountBalance {
			txRelatedAccounts := app.Pool.TxRelatedAddrs()
			tradeRelatedAccounts := pub.GetTradeAndOrdersRelatedAccounts(app.DexKeeper, tradesToPublish)
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
		len(app.DexKeeper.OrderChanges),
		"numOfProposals",
		proposalsToPublish.NumOfMsgs,
		"numOfStakeUpdates",
		stakeUpdates.NumOfMsgs,
		"numOfAccounts",
		len(accountsToPublish))
	pub.ToRemoveOrderIdCh = make(chan string, pub.ToRemoveOrderIdChannelSize)
	pub.ToPublishCh <- pub.NewBlockInfoToPublish(
		height,
		blockTime,
		tradesToPublish,
		proposalsToPublish,
		stakeUpdates,
		app.DexKeeper.OrderChanges,     // thread-safety is guarded by the signal from RemoveDoneCh
		app.DexKeeper.OrderInfosForPub, // thread-safety is guarded by the signal from RemoveDoneCh
		accountsToPublish,
		latestPriceLevels,
		blockFee,
		app.DexKeeper.RoundOrderFees,
		transferToPublish,
		blockToPublish)

	// remove item from OrderInfoForPublish when we published removed order (cancel, iocnofill, fullyfilled, expired)
	for id := range pub.ToRemoveOrderIdCh {
		pub.Logger.Debug("delete order from order changes map", "orderId", id)
		delete(app.DexKeeper.OrderInfosForPub, id)
	}

	pub.Logger.Debug("finish publish", "height", height)
}
