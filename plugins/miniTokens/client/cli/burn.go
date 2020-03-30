package commands

import (
	"github.com/binance-chain/node/common/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/binance-chain/node/plugins/miniTokens/burn"
)

func burnTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn",
		Short: "burn some amount of mini-token",
		RunE:  cmdr.burnToken,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token to be burnt")
	cmd.Flags().StringP(flagAmount, "n", "", "amount of the token to be burnt")

	return cmd
}

func (c Commander) burnToken(cmd *cobra.Command, args []string) error {
	symbol := viper.GetString(flagSymbol)
	err := types.ValidateMapperMiniTokenSymbol(symbol)
	if err != nil {
		return err
	}
	burnMsgBuilder := func(from sdk.AccAddress, symbol string, amount int64) sdk.Msg {
		return burn.NewMsg(from, symbol, amount)
	}

	return c.checkAndSendTx(cmd, args, burnMsgBuilder)
}
