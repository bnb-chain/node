package commands

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagSupply    = "supply"
	flagTokenName = "token-name"
	flagDecimal   = "decimal"
)

func issueTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "issue a new token",
		RunE:  cmdr.issueToken,
	}

	cmd.Flags().String(flagTokenName, "", "name of the new token")
	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the new token")
	cmd.Flags().StringP(flagSupply, "n", "", "total supply of the new token")
	cmd.Flags().String(flagDecimal, "0", "the decimal points of the token")

	return cmd
}

func (c Commander) issueToken(cmd *cobra.Command, args []string) error {
	ctx := context.NewCoreContextFromViper().WithDecoder(authcmd.GetAccountDecoder(c.Cdc))

	// get the banker address
	from, err := ctx.GetFromAddress()
	fmt.Println(hex.EncodeToString(from))
	if err != nil {
		return err
	}

	name := viper.GetString(flagTokenName)
	if len(name) == 0 {
		return errors.New("you must provide the name of the token")
	}

	symbol := viper.GetString(flagSymbol)
	err = validateSymbol(symbol)
	if err != nil {
		return err
	}

	symbol = strings.ToUpper(symbol)

	supplyStr := viper.GetString(flagSupply)
	supply, err := parseSupply(supplyStr)
	if err != nil {
		return err
	}

	decimalStr := viper.GetString(flagDecimal)
	decimal, err := parseDecimal(decimalStr)
	if err != nil {
		return nil
	}

	// build message
	msg := buildMsg(from, name, symbol, supply, decimal)
	return c.sendTx(ctx, msg)
}

func parseSupply(supply string) (int64, error) {
	if len(supply) == 0 {
		return 0, errors.New("you must provide total supply of the tokens")
	}

	n, err := strconv.ParseInt(supply, 10, 64)
	if err != nil || n < 0 {
		return 0, errors.New("invalid supply number")
	}

	return n, nil
}

func parseDecimal(decimal string) (int8, error) {
	if len(decimal) == 0 {
		return 0, errors.New("you must provide the decimal of the tokens")
	}

	n, err := strconv.ParseInt(decimal, 10, 8)
	if err != nil || n < 0 {
		return 0, errors.New("invalid decimal number")
	}

	return int8(n), nil
}

func buildMsg(addr sdk.Address, name string, symbol string, supply int64, decimal int8) sdk.Msg {
	return issue.NewMsg(addr, name, symbol, supply, decimal)
}
