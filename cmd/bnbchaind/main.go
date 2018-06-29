package main

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/wire"
	abci "github.com/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tmlibs/cli"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/log"

	"github.com/BiJie/BinanceChain/app"
	"github.com/cosmos/cosmos-sdk/server"
)

// AppInit init parameters
var AppInit = server.AppInit{
	AppGenState: AppGenState,
	AppGenTx:    server.SimpleAppGenTx,
}

// AppGenState sets up the app_state and appends the cool app state
func AppGenState(cdc *wire.Codec, appGenTxs []json.RawMessage) (appState json.RawMessage, err error) {
	appState, err = server.SimpleAppGenState(cdc, appGenTxs)
	if err != nil {
		return
	}
	// dex fee settings - established in genesis block
	// feeFactor: 25 => 0.25% (0.0025)
	// maxFee: 50/10000 = 0.5% (0.005)
	// nativeTokenDiscount: 1/2 => 50%
	// volumeBucketDuration: 82800secs = 23hrs
	key := "dex"
	value := json.RawMessage(`{
		"makerFee": 25,
		"takerFee": 30,
		"feeFactor": 10000,
		"maxFee": 5000,
		"nativeTokenDiscount": 2,
		"volumeBucketDuration": 82800
      }`)
	appState, err = server.AppendJSON(cdc, appState, key, value)
	return
}

func newApp(logger log.Logger, db dbm.DB) abci.Application {
	return app.NewBasecoinApp(logger, db)
}

func exportAppStateAndTMValidators(logger log.Logger, db dbm.DB) (json.RawMessage, []tmtypes.GenesisValidator, error) {
	dapp := app.NewBasecoinApp(logger, db)
	return dapp.ExportAppStateAndValidators()
}

func main() {
	cdc := app.MakeCodec()
	ctx := server.NewDefaultContext()

	rootCmd := &cobra.Command{
		Use:               "bnbchaind",
		Short:             "BNBChain Daemon (server)",
		PersistentPreRunE: server.PersistentPreRunEFn(ctx),
	}

	server.AddCommands(ctx, cdc, rootCmd, AppInit,
		server.ConstructAppCreator(newApp, "bnbchain"),
		server.ConstructAppExporter(exportAppStateAndTMValidators, "bnbchain"))

	// prepare and add flags
	rootDir := os.ExpandEnv("$HOME/.bnbchaind")
	executor := cli.PrepareBaseCmd(rootCmd, "BC", rootDir)
	executor.Execute()
}
