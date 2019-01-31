package commands

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/binance-chain/node/wire"
)

const (
	flagSymbol = "symbol"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	dexCmd := &cobra.Command{
		Use:   "dex",
		Short: "dex commands",
	}

	dexCmd.AddCommand(
		client.PostCommands(
			listTradingPairCmd(cdc),
			client.LineBreak,
			newOrderCmd(cdc),
			showOrderBookCmd(cdc),
			cancelOrderCmd(cdc))...)
	// dexCmd.AddCommand(
	// 	client.GetCommands()...)

	dexCmd.AddCommand(client.LineBreak)
	cmd.AddCommand(dexCmd)
}
