package main

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	bankcmd "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	govcmd "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	ibccmd "github.com/cosmos/cosmos-sdk/x/ibc/client/cli"
	"github.com/spf13/cobra"

	"github.com/tendermint/tendermint/libs/cli"

	"github.com/BiJie/BinanceChain/admin"
	"github.com/BiJie/BinanceChain/app"
	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	apiserv "github.com/BiJie/BinanceChain/plugins/api"
	dexcmd "github.com/BiJie/BinanceChain/plugins/dex/client/cli"
	paramcmd "github.com/BiJie/BinanceChain/plugins/param/client/cli"
	tokencmd "github.com/BiJie/BinanceChain/plugins/tokens/client/cli"
)

// rootCmd is the entry point for this binary
var (
	rootCmd = &cobra.Command{
		Use:   "bnbcli",
		Short: "BNBChain light-client",
	}
)

func main() {
	// disable sorting
	cobra.EnableCommandSorting = false

	// get the codec
	cdc := app.Codec
	ctx := app.ServerContext

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(ctx.Bech32PrefixAccAddr, ctx.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(ctx.Bech32PrefixValAddr, ctx.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(ctx.Bech32PrefixConsAddr, ctx.Bech32PrefixConsPub)
	config.Seal()

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
			authcmd.GetAccountCmd(common.AccountStoreName, cdc, types.GetAccountDecoder(cdc)),
		)...)
	rootCmd.AddCommand(
		client.PostCommands(
			bankcmd.SendTxCmd(cdc),
		)...)
	rootCmd.AddCommand(
		client.PostCommands(
			ibccmd.IBCTransferCmd(cdc),
		)...)

	// temp. disabled staking commands
	// rootCmd.AddCommand(
	// 	client.PostCommands(
	// 		ibccmd.IBCRelayCmd(cdc),
	// 		simplestakingcmd.BondTxCmd(cdc),
	// 	)...)
	// rootCmd.AddCommand(
	// 	client.PostCommands(
	// 		simplestakingcmd.UnbondTxCmd(cdc),
	// 	)...)

	// add proxy, version and key info
	rootCmd.AddCommand(
		client.LineBreak,
		apiserv.ServeCommand(cdc),
		keys.Commands(),
		client.LineBreak,
		version.VersionCmd,
	)

	tokencmd.AddCommands(rootCmd, cdc)
	dexcmd.AddCommands(rootCmd, cdc)
	paramcmd.AddCommands(rootCmd, cdc)

	govcmd.AddCommands(rootCmd, cdc)
	admin.AddCommands(rootCmd, cdc)

	// prepare and add flags
	executor := cli.PrepareMainCmd(rootCmd, "BC", app.DefaultCLIHome)
	executor.Execute()
}
