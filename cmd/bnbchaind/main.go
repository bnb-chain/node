package main

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/cli"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/server"

	"github.com/bnb-chain/node/app"
	bnbInit "github.com/bnb-chain/node/cmd/bnbchaind/init"
	"github.com/bnb-chain/node/version"
)

func newApp(logger log.Logger, db dbm.DB, storeTracer io.Writer) abci.Application {
	return app.NewBinanceChain(logger, db, storeTracer)
}

func exportAppStateAndTMValidators(logger log.Logger, db dbm.DB, storeTracer io.Writer) (json.RawMessage, []tmtypes.GenesisValidator, error) {
	dapp := app.NewBinanceChain(logger, db, storeTracer)
	return dapp.ExportAppStateAndValidators()
}

func main() {
	cdc := app.Codec
	ctx := app.ServerContext

	rootCmd := &cobra.Command{
		Use:               "bnbchaind",
		Short:             "BNBChain Daemon (server)",
		PersistentPreRunE: app.PersistentPreRunEFn(ctx),
	}

	appInit := app.BinanceAppInit()
	rootCmd.AddCommand(bnbInit.InitCmd(ctx.ToCosmosServerCtx(), cdc, appInit))
	rootCmd.AddCommand(bnbInit.TestnetFilesCmd(ctx.ToCosmosServerCtx(), cdc, appInit))
	rootCmd.AddCommand(bnbInit.CollectGenTxsCmd(cdc, appInit))
	rootCmd.AddCommand(version.VersionCmd)
	server.AddCommands(ctx.ToCosmosServerCtx(), cdc, rootCmd, exportAppStateAndTMValidators)
	startCmd := server.StartCmd(ctx.ToCosmosServerCtx(), newApp)
	startCmd.Flags().Int64VarP(&ctx.PublicationConfig.FromHeightInclusive, "fromHeight", "f", 1, "from which height (inclusive) we want publish market data")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(bnbInit.SnapshotCmd(ctx.ToCosmosServerCtx(), cdc))

	// prepare and add flags
	executor := cli.PrepareBaseCmd(rootCmd, "BC", app.DefaultNodeHome)
	_ = executor.Execute()
}
