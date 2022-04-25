package main

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	bankcmd "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	govcmd "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	sidecmd "github.com/cosmos/cosmos-sdk/x/sidechain/client/cli"

	paramcmd "github.com/cosmos/cosmos-sdk/x/paramHub/client/cli"
	slashingcmd "github.com/cosmos/cosmos-sdk/x/slashing/client/cli"
	stakecmd "github.com/cosmos/cosmos-sdk/x/stake/client/cli"

	"github.com/tendermint/tendermint/libs/cli"

	"github.com/bnb-chain/node/admin"
	"github.com/bnb-chain/node/app"
	"github.com/bnb-chain/node/common"
	"github.com/bnb-chain/node/common/types"
	accountcmd "github.com/bnb-chain/node/plugins/account/client/cli"
	apiserv "github.com/bnb-chain/node/plugins/api"
	bridgecmd "github.com/bnb-chain/node/plugins/bridge/client/cli"
	dexcmd "github.com/bnb-chain/node/plugins/dex/client/cli"
	tokencmd "github.com/bnb-chain/node/plugins/tokens/client/cli"
	"github.com/bnb-chain/node/version"
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
	if app.Bech32PrefixAccAddr != "" {
		ctx.Bech32PrefixAccAddr = app.Bech32PrefixAccAddr
	}
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
			bankcmd.GetBroadcastCommand(cdc),
			authcmd.GetSignCommand(cdc, types.GetAccountDecoder(cdc)),
		)...)

	// add proxy, version and key info
	rootCmd.AddCommand(
		client.LineBreak,
		apiserv.ServeCommand(cdc),
		keys.Commands(),
		client.LineBreak,
		version.VersionCmd,
	)

	tokencmd.AddCommands(rootCmd, cdc)
	accountcmd.AddCommands(rootCmd, cdc)
	dexcmd.AddCommands(rootCmd, cdc)
	paramcmd.AddCommands(rootCmd, cdc)

	stakecmd.AddCommands(rootCmd, cdc)
	slashingcmd.AddCommands(rootCmd, cdc)
	govcmd.AddCommands(rootCmd, cdc)
	admin.AddCommands(rootCmd, cdc)
	bridgecmd.AddCommands(rootCmd, cdc)
	sidecmd.AddCommands(rootCmd, cdc)

	// prepare and add flags
	executor := cli.PrepareMainCmd(rootCmd, "BC", app.DefaultCLIHome)
	_ = executor.Execute()
}
