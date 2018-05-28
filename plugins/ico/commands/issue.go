package commands

import (
	"fmt"
	"github.com/BiJie/BinanceChain/plugins/ico"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/commands"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"math/big"
)

const (
	flagSupply    = "supply"
	flagTokenName = "token-name"
	flagSymbol    = "symbol"
	flagDecimal   = "decimal"
)

func issueTokenCmd(cdc *wire.Codec) *cobra.Command {
	cmdr := Commander{cdc}
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

type Commander struct {
	Cdc *wire.Codec
}

func (c Commander) issueToken(cmd *cobra.Command, args []string) error {
	ctx := context.NewCoreContextFromViper().WithDecoder(authcmd.GetAccountDecoder(c.Cdc))

	// get the banker address
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
	msg := BuildMsg(from, name, symbol, supply, decimal)

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
		return errors.New("you must provide the symbol of the token")
	}

	//TODO: check symbol uniqueness
	return nil
}

// TODO: type amount as big.Int
func parseSupply(supply string) (*big.Int, error) {
	if len(supply) == 0 {
		return nil, errors.New("you must provide total supply of the token")
	}

	n := new(big.Int)
	n, ok := n.SetString(supply, 10)
	if !ok {
		return nil, errors.New("invalid supply number")
	}

	return n, nil
}

// TODO: type decimal as big.Int
func parseDecimal(decimal string) (*big.Int, error) {
	if len(decimal) == 0 {
		return nil, errors.New("you must provide the decimal of the token")
	}

	n := new(big.Int)
	n, ok := n.SetString(decimal, 10)
	if !ok {
		return nil, errors.New("invalid supply number")
	}

	return n, nil
}

func BuildMsg(addr sdk.Address, name string, symbol string, supply *big.Int, decimal *big.Int) sdk.Msg {
	amount := new(big.Int)
	amount.Mul(amount.Exp(big.NewInt(10), decimal, nil), supply)

	// TODO: will change the type of Coin.Amount to *Big.Int
	coin := sdk.Coin{Denom: symbol, Amount: amount.Int64()}

	return ico.NewIssueMsg(addr, coin)
}
