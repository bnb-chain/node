package app

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tendermint/tendermint/abci/server"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	tmcfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/app/config"
	bnclog "github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/BiJie/BinanceChain/plugins/tokens/freeze"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
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
		return bnclog.NewAsyncFileLogger(path.Join(ctx.Config.RootDir, ctx.LogConfig.LogFilePath), ctx.LogConfig.LogBuffSize)
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

func collectInvolvedAddresses(ctx sdk.Context, msg sdk.Msg) (newCtx sdk.Context) {
	switch ct := msg.(type) {
	case list.ListMsg:
		newCtx = addInvolvedAddressesToCtx(ctx, ct.From)
	case order.NewOrderMsg:
		newCtx = addInvolvedAddressesToCtx(ctx, ct.Sender)
	case order.CancelOrderMsg:
		newCtx = addInvolvedAddressesToCtx(ctx, ct.Sender)
	case issue.IssueMsg:
		newCtx = addInvolvedAddressesToCtx(ctx, ct.From)
	case burn.BurnMsg:
		newCtx = addInvolvedAddressesToCtx(ctx, ct.From)
	case freeze.FreezeMsg:
		newCtx = addInvolvedAddressesToCtx(ctx, ct.From)
	case freeze.UnfreezeMsg:
		newCtx = addInvolvedAddressesToCtx(ctx, ct.From)
	case bank.MsgSend:
		newCtx = addInvolvedAddressesToCtx(ctx, ct.Inputs[0].Address, ct.Outputs[0].Address)
	default:
		// TODO(#66): correct error handling
	}
	return
}

func addInvolvedAddressesToCtx(ctx sdk.Context, addresses ...sdk.AccAddress) (newCtx sdk.Context) {
	var newAddress []string
	if addresses, ok := ctx.Value(baseapp.InvolvedAddressKey).([]string); ok {
		newAddress = addresses
	} else {
		newAddress = make([]string, 0)
	}
	for _, address := range addresses {
		newAddress = append(newAddress, string(address.Bytes()))
	}
	newCtx = ctx.WithValue(baseapp.InvolvedAddressKey, newAddress)
	return
}
