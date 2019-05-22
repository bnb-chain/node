package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cosmossrv "github.com/cosmos/cosmos-sdk/server"
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
	"github.com/tendermint/tmlibs/cli"

	"github.com/binance-chain/node/app/config"
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

// binance-chain implementation of PruningStrategy
type KeepRecentAndBreatheBlock struct {
	breatheBlockInterval int64

	// Keep recent number blocks in case of rollback
	numRecent int64

	blockStore *blockchain.BlockStore

	blockStoreInitializer sync.Once
}

func NewKeepRecentAndBreatheBlock(breatheBlockInterval, numRecent int64, config *tmcfg.Config) *KeepRecentAndBreatheBlock {
	return &KeepRecentAndBreatheBlock{
		breatheBlockInterval: breatheBlockInterval,
		numRecent:            numRecent,
	}
}

// TODO: must enhance performance!
func (strategy KeepRecentAndBreatheBlock) ShouldPrune(version, latestVersion int64) bool {
	// we are replay the possible 1 block diff between state and blockstore db
	// save this block anyway and don't init strategy's blockStore
	if cosmossrv.BlockStore == nil {
		return false
	}

	// only at this time block store is initialized!
	// block store has been opened after the start of tendermint node, we have to share same instance of block store
	strategy.blockStoreInitializer.Do(func() {
		strategy.blockStore = cosmossrv.BlockStore
	})

	if version == 1 {
		return false
	} else if latestVersion-version < strategy.numRecent {
		return false
	} else {
		if strategy.breatheBlockInterval > 0 {
			return version%strategy.breatheBlockInterval != 0
		} else {
			lastBlock := strategy.blockStore.LoadBlock(version - 1)
			block := strategy.blockStore.LoadBlock(version)

			if lastBlock == nil {
				// this node is a state_synced node, previously block is not synced
				// so we cannot tell whether this (first) block is breathe block or not
				return false
			}
			return utils.SameDayInUTC(lastBlock.Time, block.Time)
		}
	}
}
