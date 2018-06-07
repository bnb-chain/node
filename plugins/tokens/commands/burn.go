package commands

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/BiJie/BinanceChain/plugins/tokens/burn"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	authcmd "github.com/cosmos/cosmos-sdk/x/auth/commands"
)

func burnTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn",
		Short: "burn some amount of token",
		RunE:  cmdr.burnToken,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token to be burnt")
	cmd.Flags().StringP(flagAmount, "n", "", "amount of the token to be burnt")

	return cmd
}

func (c Commander) burnToken(cmd *cobra.Command, args []string) error {
	ctx := context.NewCoreContextFromViper().WithDecoder(authcmd.GetAccountDecoder(c.Cdc))

	// get the banker address
	from, err := ctx.GetFromAddress()
	fmt.Println(hex.EncodeToString(from))
	if err != nil {
		return err
	}

	symbol := viper.GetString(flagSymbol)
	err = validateSymbol(symbol)
	if err != nil {
		return err
	}

	symbol = strings.ToUpper(symbol)

	amountStr := viper.GetString(flagAmount)
	amount, err := parseAmount(amountStr)
	if err != nil {
		return err
	}

	// build message
	msg := burn.NewMsg(from, symbol, amount)

	// default to next sequence number if none provided
	ctx, err = context.EnsureSequence(ctx)
	if err != nil {
		return err
	}

	// build and sign the transaction, then broadcast to Tendermint
	res, err := ctx.SignBuildBroadcast(ctx.FromAddressName, msg, c.Cdc)
	if err != nil {
		return err
	}

	fmt.Printf("Committed at block %d. Hash: %s\n", res.Height, res.Hash.String())
	return nil
}
