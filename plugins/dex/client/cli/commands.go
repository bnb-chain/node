package commands

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/binance-chain/node/wire"
)

const (
	flagSymbol = "symbol"
	flagLevels = "levels"
)

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	dexCmd := &cobra.Command{
		Use:   "dex",
		Short: "dex commands",
	}

	dexCmd.AddCommand(
		client.PostCommands(
			listTradingPairCmd(cdc),
			listMiniTradingPairCmd(cdc),
			client.LineBreak,
			newOrderCmd(cdc),
			cancelOrderCmd(cdc))...)
	dexCmd.AddCommand(
		client.GetCommands(
			showOrderBookCmd(cdc))...)

	dexCmd.AddCommand(client.LineBreak)
	cmd.AddCommand(dexCmd)
}
