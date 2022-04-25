package main

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/tendermint/tendermint/libs/cli"

	"github.com/bnb-chain/node/app"
)

func main() {
	cdc := app.Codec
	ctx := app.ServerContext

	rootCmd := &cobra.Command{
		Use:               "bnbsentry",
		Short:             "BNBChain Sentry (server)",
		PersistentPreRunE: app.PersistentPreRunEFn(ctx),
	}

	server.AddCommands(ctx.ToCosmosServerCtx(), cdc, rootCmd, nil)
	startCmd := server.StartCmd(ctx.ToCosmosServerCtx(), app.NewSentryApplication)

	startCmd.Flags().IntVarP(&app.SentryAppConfig.CacheSize, "cache_size", "c", app.DefaultCacheSize, "The cache size of sentry node")
	startCmd.Flags().IntVarP(&app.SentryAppConfig.MaxSurvive, "max_survive", "s", app.DefaultMaxSurvive, "The max survive of sentry node")
	rootCmd.AddCommand(startCmd)
	// prepare and add flags
	executor := cli.PrepareBaseCmd(rootCmd, "BC", app.DefaultNodeHome)

	_ = executor.Execute()
}
