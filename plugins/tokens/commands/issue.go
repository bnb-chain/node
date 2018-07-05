package commands

import (
	"strconv"
	"strings"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagTotalSupply    = "total-supply"
	flagTokenName = "token-name"
)

func issueTokenCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "issue a new token",
		RunE:  cmdr.issueToken,
	}

	cmd.Flags().String(flagTokenName, "", "name of the new token")
	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the new token")
	cmd.Flags().StringP(flagTotalSupply, "n", "", "total supply of the new token")

	return cmd
}

func (c Commander) issueToken(cmd *cobra.Command, args []string) error {
	ctx := context.NewCoreContextFromViper().WithDecoder(authcmd.GetAccountDecoder(c.Cdc))

	from, err := ctx.GetFromAddress()
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

	supplyStr := viper.GetString(flagTotalSupply)
	supply, err := parseSupply(supplyStr)
	if err != nil {
		return err
	}

	// build message
	msg := buildMsg(from, name, symbol, supply)
	return c.sendTx(ctx, msg)
}

func parseSupply(supply string) (int64, error) {
	if len(supply) == 0 {
		return 0, errors.New("you must provide total supply of the tokens")
	}

	n, err := strconv.ParseInt(supply, 10, 64)
	if err != nil || n < 0 || n > types.MaxTotalSupply {
		return 0, errors.New("invalid supply number")
	}

	return n, nil
}

func buildMsg(addr sdk.Address, name string, symbol string, supply int64) sdk.Msg {
	return issue.NewMsg(addr, name, symbol, supply)
}
