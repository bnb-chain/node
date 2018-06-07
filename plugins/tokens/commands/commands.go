package commands

import (
	"errors"
	"strconv"

	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/spf13/cobra"
)

const (
	flagSymbol = "symbol"
	flagAmount = "amount"
)

type Commander struct {
	Cdc *wire.Codec
}

func AddCommands(cmd *cobra.Command, cdc *wire.Codec) {

	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "issue or view tokens",
		Long:  ``,
	}

	cmdr := Commander{Cdc: cdc}
	tokenCmd.AddCommand(
		client.PostCommands(
			issueTokenCmd(cmdr),
			burnTokenCmd(cmdr))...)
	tokenCmd.AddCommand(
		client.GetCommands(
			listTokensCmd,
			getTokenInfoCmd(cmdr))...)

	tokenCmd.AddCommand(client.LineBreak)

	cmd.AddCommand(tokenCmd)
}

func validateSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("you must provide the symbol of the tokens")
	}

	if !utils.IsAlphaNum(symbol) {
		return errors.New("the symbol should be alphanumeric")
	}

	return nil
}

func parseAmount(amountStr string) (int64, error) {
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return 0, err
	}

	if amount <= 0 {
		return amount, errors.New("the amount should be greater than 0")
	}

	return amount, nil
}
