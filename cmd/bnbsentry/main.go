package main

import (
	"github.com/spf13/cobra"

	"github.com/BiJie/BinanceChain/app"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/cli"
)

const flagSequentialABCI = "seq-abci"

func main() {
	cdc := app.Codec
	ctx := app.ServerContext
	config := sdk.GetConfig()

	config.SetBech32PrefixForAccount(ctx.Bech32PrefixAccAddr, ctx.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(ctx.Bech32PrefixValAddr, ctx.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(ctx.Bech32PrefixConsAddr, ctx.Bech32PrefixConsPub)
	config.Seal()

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
	startCmd.Flags().Bool(flagSequentialABCI, true, "whether check tx in sequentially ad")
	// prepare and add flags
	executor := cli.PrepareBaseCmd(rootCmd, "BC", app.DefaultNodeHome)

	executor.Execute()
}
