package main

import (
	"encoding/json"
	"io"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/cli"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/server"

	"github.com/BiJie/BinanceChain/app"
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
	rootCmd := app.RootCmd

	server.AddCommands(ctx, cdc, rootCmd, app.BinanceAppInit(),
		server.ConstructAppCreator(newApp, "bnbchain"),
		server.ConstructAppExporter(exportAppStateAndTMValidators, "bnbchain"))

	// prepare and add flags
	executor := cli.PrepareBaseCmd(rootCmd, "BC", app.DefaultNodeHome)
	executor.Execute()
}
