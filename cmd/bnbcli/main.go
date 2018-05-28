package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tendermint/tmlibs/cli"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/lcd"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/cosmos/cosmos-sdk/version"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/commands"
	bankcmd "github.com/cosmos/cosmos-sdk/x/bank/commands"
	ibccmd "github.com/cosmos/cosmos-sdk/x/ibc/commands"
	simplestakingcmd "github.com/cosmos/cosmos-sdk/x/simplestake/commands"

	"github.com/BiJie/BinanceChain/app"
	dexcmd "github.com/BiJie/BinanceChain/plugins/dex/commands"
	icocmd "github.com/BiJie/BinanceChain/plugins/ico/commands"
	"github.com/BiJie/BinanceChain/types"
)

// rootCmd is the entry point for this binary
var (
	rootCmd = &cobra.Command{
		Use:   "bnbcli",
		Short: "BNBChain cli client",
	}
)

func main() {
	// disable sorting
	cobra.EnableCommandSorting = false

	// get the codec
	cdc := app.MakeCodec()

	// TODO: setup keybase, viper object, etc. to be passed into
	// the below functions and eliminate global vars, like we do
	// with the cdc

	// add standard rpc, and tx commands
	rpc.AddCommands(rootCmd)
	rootCmd.AddCommand(client.LineBreak)
	tx.AddCommands(rootCmd, cdc)
	rootCmd.AddCommand(client.LineBreak)

	// add query/post commands (custom to binary)
	// start with commands common to basecoin
	rootCmd.AddCommand(
		client.GetCommands(
			authcmd.GetAccountCmd("main", cdc, types.GetAccountDecoder(cdc)),
		)...)
	rootCmd.AddCommand(
		client.PostCommands(
			bankcmd.SendTxCmd(cdc),
		)...)
	rootCmd.AddCommand(
		client.PostCommands(
			ibccmd.IBCTransferCmd(cdc),
		)...)
	rootCmd.AddCommand(
		client.PostCommands(
			ibccmd.IBCRelayCmd(cdc),
			simplestakingcmd.BondTxCmd(cdc),
		)...)
	rootCmd.AddCommand(
		client.PostCommands(
			simplestakingcmd.UnbondTxCmd(cdc),
		)...)

	// dex specific commands
	rootCmd.AddCommand(
		client.PostCommands(
			client.LineBreak,
			dexcmd.MakeOfferCmd(cdc),
			dexcmd.FillOfferCmd(cdc),
			dexcmd.CancelOfferCmd(cdc),
		)...)

	// add proxy, version and key info
	rootCmd.AddCommand(
		client.LineBreak,
		lcd.ServeCommand(cdc),
		keys.Commands(),
		client.LineBreak,
		version.VersionCmd,
	)

	icocmd.AddCommands(rootCmd, cdc)

	// prepare and add flags
	executor := cli.PrepareMainCmd(rootCmd, "BC", os.ExpandEnv("$HOME/.bnbcli"))
	executor.Execute()
}
