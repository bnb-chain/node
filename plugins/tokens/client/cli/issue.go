package commands

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	cli "github.com/cosmos/cosmos-sdk/client/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authctx "github.com/cosmos/cosmos-sdk/x/auth/client/context"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
)

const (
	flagTotalSupply = "total-supply"
	flagTokenName   = "token-name"
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
	txCtx := authctx.NewTxContextFromCLI().WithCodec(c.Cdc)
	cliCtx := context.NewCLIContext().WithAccountDecoder(types.GetAccountDecoder(c.Cdc))

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	name := viper.GetString(flagTokenName)
	if len(name) == 0 {
		return errors.New("you must provide the name of the token")
	}

	symbol := viper.GetString(flagSymbol)
	err = types.ValidateSymbol(symbol)
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
	return cli.SendTx(txCtx, cliCtx, []sdk.Msg{msg})
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

func buildMsg(addr sdk.AccAddress, name string, symbol string, supply int64) sdk.Msg {
	return issue.NewMsg(addr, name, symbol, supply)
}
