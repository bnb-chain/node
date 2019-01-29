package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/version"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/tendermint/abci/server"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
	bnclog "github.com/BiJie/BinanceChain/common/log"
)

// RunForever - BasecoinApp execution and cleanup
func RunForever(app abci.Application) {

	// Start the ABCI server
	srv, err := server.NewServer("0.0.0.0:26658", "socket", app)
	if err != nil {
		cmn.Exit(err.Error())
		return
	}
	err = srv.Start()
	if err != nil {
		cmn.Exit(err.Error())
		return
	}

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		err := srv.Stop()
		if err != nil {
			cmn.Exit(err.Error())
		}
	})
}

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

	context.Config, err = tmcmd.ParseConfig()
	if err != nil {
		return err
	}
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// the following parse config is needed to create directories
		context.Config.ProfListenAddress = "localhost:6060"
		context.Config.P2P.RecvRate = 5120000
		context.Config.P2P.SendRate = 5120000
		context.Config.Consensus.TimeoutCommit = 5000
		tmcfg.WriteConfigFile(configFilePath, context.Config)
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
