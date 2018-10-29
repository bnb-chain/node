package main

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/cli"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	gaiaInit "github.com/cosmos/cosmos-sdk/cmd/gaia/init"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/BiJie/BinanceChain/app"
)

func newApp(logger log.Logger, db dbm.DB, storeTracer io.Writer) abci.Application {
	return app.NewBinanceChain(logger, db, storeTracer, baseapp.SetPruning(viper.GetString("pruning")))
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
	rootCmd.AddCommand(gaiaInit.InitCmd(ctx.ToCosmosServerCtx(), cdc, appInit))
	server.AddCommands(ctx.ToCosmosServerCtx(), cdc, rootCmd, appInit, newApp, exportAppStateAndTMValidators)

	// prepare and add flags
	executor := cli.PrepareBaseCmd(rootCmd, "BC", app.DefaultNodeHome)
	executor.Execute()
}
