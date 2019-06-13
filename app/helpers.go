package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/tendermint/tendermint/blockchain"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tendermint/libs/cli"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/snapshot"

	"github.com/binance-chain/node/app/config"
	"github.com/binance-chain/node/common"
	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/utils"
	"github.com/binance-chain/node/plugins/dex/order"
)

// If a new config is created, change some of the default tendermint settings
func interceptLoadConfigInPlace(context *config.BinanceChainContext) (err error) {
	tmpConf := tmcfg.DefaultConfig()
	err = viper.Unmarshal(tmpConf)
	if err != nil {
		return err
	}
	rootDir := tmpConf.RootDir
	configFilePath := filepath.Join(rootDir, "config/config.toml")
	// Intercept only if the file doesn't already exist

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// the following parse config is needed to create directories
		conf, _ = tcmd.ParseConfig()
		conf.ProfListenAddress = "localhost:6060"
		conf.P2P.RecvRate = 5120000
		conf.P2P.SendRate = 5120000
		conf.Consensus.TimeoutCommit = 0
		cfg.WriteConfigFile(configFilePath, conf)
		// Fall through, just so that its parsed into memory.
	}

	appConfigFilePath := filepath.Join(rootDir, "config/", config.AppConfigFileName+".toml")
	if _, err := os.Stat(appConfigFilePath); os.IsNotExist(err) {
		config.WriteConfigFile(appConfigFilePath, ServerContext.BinanceChainConfig)
	} else {
		err = context.ParseAppConfigInPlace()
		if err != nil {
			return err
		}
	}

	return nil
}

func newLogger(ctx *config.BinanceChainContext) log.Logger {
	if ctx.LogConfig.LogToConsole {
		return bnclog.NewConsoleLogger()
	} else {
		logFilePath := ""
		if ctx.LogConfig.LogFileRoot == "" {
			logFilePath = path.Join(ctx.Config.RootDir, ctx.LogConfig.LogFilePath)
		} else {
			logFilePath = path.Join(ctx.LogConfig.LogFileRoot, ctx.LogConfig.LogFilePath)
		}
		err := cmn.EnsureDir(path.Dir(logFilePath), 0755)
		if err != nil {
			panic(fmt.Sprintf("create log dir failed, err=%s", err.Error()))
		}
		return bnclog.NewAsyncFileLogger(logFilePath, ctx.LogConfig.LogBuffSize)
	}
}

// PersistentPreRunEFn returns a PersistentPreRunE function for cobra
// that initailizes the passed in context with a properly configured
// logger and config object
func PersistentPreRunEFn(context *config.BinanceChainContext) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == version.VersionCmd.Name() {
			return nil
		}
		err := interceptLoadConfigInPlace(context)
		if err != nil {
			return err
		}

		config := sdk.GetConfig()
		config.SetBech32PrefixForAccount(context.Bech32PrefixAccAddr, context.Bech32PrefixAccPub)
		config.SetBech32PrefixForValidator(context.Bech32PrefixValAddr, context.Bech32PrefixValPub)
		config.SetBech32PrefixForConsensusNode(context.Bech32PrefixConsAddr, context.Bech32PrefixConsPub)
		config.Seal()

		// TODO: add config for logging to stdout for debug sake
		logger := newLogger(context)
		logger, err = tmflags.ParseLogLevel(context.Config.LogLevel, logger, tmcfg.DefaultLogLevel())
		if err != nil {
			return err
		}
		if viper.GetBool(cli.TraceFlag) {
			logger = log.NewTracingLogger(logger)
		}
		logger = logger.With("module", "main")
		bnclog.InitLogger(logger)

		context.Logger = logger
		return nil
	}
}

func (app *BinanceChain) processErrAbciResponseForPub(txBytes []byte) {
	defer func() {
		if r := recover(); r != nil {
			stackTrace := fmt.Sprintf("recovered: %v\nstack:\n%v", r, string(debug.Stack()))
			app.Logger.Error(stackTrace)
		}

	}()
	tx, err := app.TxDecoder(txBytes)
	txHash := cmn.HexBytes(tmhash.Sum(txBytes)).String()
	if err != nil {
		app.Logger.Info("failed to process invalid tx", "tx", txHash)
	} else {
		if msgs := tx.GetMsgs(); len(msgs) != 1 {
			// The error message here should be consistent with vendor/github.com/cosmos/cosmos-sdk/baseapp/baseapp.go:537
			app.Logger.Error("Tx.GetMsgs() must return exactly one message")
		} else {
			switch msg := msgs[0].(type) {
			case order.NewOrderMsg:
				app.Logger.Info("failed to process NewOrderMsg", "oid", msg.Id)
				// The error on deliver should be rare and only impact witness publisher's performance
				app.DexKeeper.OrderChangesMtx.Lock()
				app.DexKeeper.OrderChanges = append(app.DexKeeper.OrderChanges, order.OrderChange{msg.Id, order.FailedBlocking, msg})
				app.DexKeeper.OrderChangesMtx.Unlock()
			case order.CancelOrderMsg:
				app.Logger.Info("failed to process CancelOrderMsg", "oid", msg.RefId)
				// The error on deliver should be rare and only impact witness publisher's performance
				app.DexKeeper.OrderChangesMtx.Lock()
				// OrderInfo must has been in keeper.OrderInfosForPub
				app.DexKeeper.OrderChanges = append(app.DexKeeper.OrderChanges, order.OrderChange{msg.RefId, order.FailedBlocking, msg})
				app.DexKeeper.OrderChangesMtx.Unlock()
			default:
				// deliberately do nothing for message other than NewOrderMsg
				// in future, we may publish fail status of send msg
			}
		}
	}
}

func (app *BinanceChain) getLastBreatheBlockHeight() int64 {
	// we should only sync to breathe block height
	latestBlockHeight := app.LastBlockHeight()
	var timeOfLatestBlock time.Time
	if latestBlockHeight == 0 {
		timeOfLatestBlock = utils.Now()
	} else {
		blockDB := baseapp.LoadBlockDB()
		defer blockDB.Close()
		blockStore := blockchain.NewBlockStore(blockDB)
		block := blockStore.LoadBlock(latestBlockHeight)
		timeOfLatestBlock = block.Time
	}

	height := app.DexKeeper.GetLastBreatheBlockHeight(
		app.CheckState.Ctx,
		latestBlockHeight,
		timeOfLatestBlock,
		app.baseConfig.BreatheBlockInterval,
		app.baseConfig.BreatheBlockDaysCountBack)
	app.Logger.Info("get last breathe block height", "height", height)
	return height
}

func (app *BinanceChain) reInitChain() error {
	app.DexKeeper.Init(
		app.CheckState.Ctx,
		app.baseConfig.BreatheBlockInterval,
		app.baseConfig.BreatheBlockDaysCountBack,
		snapshot.Manager().GetBlockStore(),
		snapshot.Manager().GetTxDB(),
		app.LastBlockHeight(),
		app.TxDecoder)
	app.initParams()

	// init app cache
	stores := app.GetCommitMultiStore()
	accountStore := stores.GetKVStore(common.AccountStoreKey)
	app.SetAccountStoreCache(app.Codec, accountStore, app.baseConfig.AccountCacheSize)

	return nil
}
