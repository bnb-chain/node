package server

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/server/concurrent"

	"github.com/tendermint/tendermint/abci/server"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	pvm "github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	tmstore "github.com/tendermint/tendermint/store"
)

const (
	flagWithTendermint = "with-tendermint"
	flagAddress        = "address"
	flagTraceStore     = "trace-store"
	flagPruning        = "pruning"
	flagSequentialABCI = "seq-abci"
)

var BlockStore *tmstore.BlockStore

// StartCmd runs the service passed in, either stand-alone or in-process with
// Tendermint.
func StartCmd(ctx *Context, appCreator AppCreator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !viper.GetBool(flagWithTendermint) {
				ctx.Logger.Info("Starting ABCI without Tendermint")
				return startStandAlone(ctx, appCreator)
			}

			ctx.Logger.Info("Starting ABCI with Tendermint")

			_, err := startInProcess(ctx, appCreator)
			return err
		},
	}

	// core flags for the ABCI application
	cmd.Flags().Bool(flagWithTendermint, true, "Run abci app embedded in-process with tendermint")
	cmd.Flags().String(flagAddress, "tcp://0.0.0.0:26658", "Listen address")
	cmd.Flags().String(flagTraceStore, "", "Enable KVStore tracing to an output file")
	cmd.Flags().Bool(flagSequentialABCI, false, "Run abci app in sync mode")
	cmd.Flags().String(flagPruning, "syncable", "Pruning strategy: syncable, nothing, everything")

	// add support for all Tendermint-specific command line options
	tcmd.AddNodeFlags(cmd)
	return cmd
}

func startStandAlone(ctx *Context, appCreator AppCreator) error {
	addr := viper.GetString(flagAddress)
	home := viper.GetString("home")
	traceWriterFile := viper.GetString(flagTraceStore)

	db, err := openDB(home)
	if err != nil {
		return err
	}
	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		return err
	}

	app := appCreator(ctx.Logger, db, traceWriter)

	svr, err := server.NewServer(addr, "socket", app)
	if err != nil {
		return errors.Errorf("error creating listener: %v\n", err)
	}

	svr.SetLogger(ctx.Logger.With("module", "abci-server"))

	err = svr.Start()
	if err != nil {
		cmn.Exit(err.Error())
	}

	// wait forever
	cmn.TrapSignal(ctx.Logger, func() {
		// cleanup
		err = svr.Stop()
		if err != nil {
			cmn.Exit(err.Error())
		}
	})
	select {}
	return nil
}

// nolint: unparam
func startInProcess(ctx *Context, appCreator AppCreator) (*node.Node, error) {
	cfg := ctx.Config
	traceWriterFile := viper.GetString(flagTraceStore)
	isSequentialABCI := viper.GetBool(flagSequentialABCI)

	dbProvider := node.DefaultDBProvider
	db, err := dbProvider(&node.DBContext{"application", cfg})
	if err != nil {
		return nil, err
	}
	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		return nil, err
	}

	app := appCreator(ctx.Logger, db, traceWriter)

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return nil, err
	}

	var cliCreator proxy.ClientCreator
	if isSequentialABCI {
		cliCreator = proxy.NewLocalClientCreator(app)
	} else {
		cliCreator = concurrent.NewAsyncLocalClientCreator(app,
			ctx.Logger.With("module", "abciCli"))
	}

	// create & start tendermint node
	tmNode, err := node.NewNode(
		cfg,
		pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		cliCreator,
		node.DefaultGenesisDocProviderFunc(cfg),
		dbProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		ctx.Logger.With("module", "node"),
	)
	if err != nil {
		return nil, err
	}

	BlockStore = tmNode.BlockStore()

	err = tmNode.Start()
	if err != nil {
		return nil, err
	}

	TrapSignal(func() {
		if tmNode.IsRunning() {
			_ = tmNode.Stop()
		}
	})

	// run forever (the node will not be returned)
	select {}
}
