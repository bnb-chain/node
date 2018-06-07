package commands

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/issue"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/commands"
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
	token := types.Token{Name: name, Symbol: symbol, Supply: supply, Decimal: decimal}
	return issue.NewMsg(addr, token)
}
