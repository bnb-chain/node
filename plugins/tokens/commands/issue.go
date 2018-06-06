package commands

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
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
	flagSymbol    = "symbol"
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
	cmd.Flags().String(flagDecimal, "", "")

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

func validateSymbol(symbol string) error {
	if len(symbol) == 0 {
		return errors.New("you must provide the symbol of the tokens")
	}

	//TODO: check symbol uniqueness
	return nil
}

func parseSupply(supply string) (*big.Int, error) {
	if len(supply) == 0 {
		return nil, errors.New("you must provide total supply of the tokens")
	}

	n := new(big.Int)
	n, ok := n.SetString(supply, 10)
	if !ok {
		return nil, errors.New("invalid supply number")
	}

	return n, nil
}

func parseDecimal(decimal string) (*big.Int, error) {
	if len(decimal) == 0 {
		return nil, errors.New("you must provide the decimal of the tokens")
	}

	n := new(big.Int)
	n, ok := n.SetString(decimal, 10)
	if !ok {
		return nil, errors.New("invalid supply number")
	}

	return n, nil
}

func buildMsg(addr sdk.Address, name string, symbol string, supply *big.Int, decimal *big.Int) sdk.Msg {
	token := types.Token{Name: name, Symbol: symbol, Supply: types.NewNumber(supply), Decimals: types.NewNumber(decimal)}
	return tokens.NewIssueMsg(addr, token)
}
