package app

import (
	"os"
	"path/filepath"

	sdk_server "github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/golang/go/src/path"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/abci/server"
	abci "github.com/tendermint/tendermint/abci/types"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	cfg "github.com/tendermint/tendermint/config"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tmlibs/cli"

	blog "github.com/BiJie/BinanceChain/common/log"
)

// nolint - Mostly for testing
func (app *BaseApp) Check(tx sdk.Tx) (result sdk.Result) {
	return app.runTx(runTxModeCheck, nil, tx)
}

// nolint - full tx execution
func (app *BaseApp) Simulate(tx sdk.Tx) (result sdk.Result) {
	return app.runTx(runTxModeSimulate, nil, tx)
}

// nolint
func (app *BaseApp) Deliver(tx sdk.Tx) (result sdk.Result) {
	return app.runTx(runTxModeDeliver, nil, tx)
}

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
func interceptLoadConfig() (conf *cfg.Config, err error) {
	tmpConf := cfg.DefaultConfig()
	err = viper.Unmarshal(tmpConf)
	if err != nil {
		// TODO: Handle with #870
		panic(err)
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
		conf.Consensus.TimeoutCommit = 5000
		cfg.WriteConfigFile(configFilePath, conf)
		// Fall through, just so that its parsed into memory.
	}

	if conf == nil {
		conf, err = tcmd.ParseConfig()
	}
	return
}

func newAsyncLogger(conf *cfg.Config) log.Logger {
	return blog.NewAsyncFileLogger(path.Join(conf.RootDir, DefaultLogFile), DefaultLogBuffSize)
}

// PersistentPreRunEFn returns a PersistentPreRunE function for cobra
// that initailizes the passed in context with a properly configured
// logger and config objecy
func PersistentPreRunEFn(context *sdk_server.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == version.VersionCmd.Name() {
			return nil
		}
		config, err := interceptLoadConfig()
		if err != nil {
			return err
		}

		// TODO: add config for logging to stdout for debug sake
		logger := newAsyncLogger(config)
		logger, err = tmflags.ParseLogLevel(config.LogLevel, logger, cfg.DefaultLogLevel())
		if err != nil {
			return err
		}
		if viper.GetBool(cli.TraceFlag) {
			logger = log.NewTracingLogger(logger)
		}
		logger = logger.With("module", "main")
		blog.Init(logger)

		context.Config = config
		context.Logger = logger
		return nil
	}
}
