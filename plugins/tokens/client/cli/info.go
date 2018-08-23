package commands

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

func getTokenInfoCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <symbol>",
		Short: "Query token info",
		RunE:  cmdr.runGetToken,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token")
	return cmd
}

func (c Commander) runGetToken(cmd *cobra.Command, args []string) error {
	ctx := context.NewCoreContextFromViper()

	symbol := viper.GetString(flagSymbol)
	if len(symbol) == 0 {
		return errors.New("you must provide the symbol")
	}

	key := []byte(strings.ToUpper(symbol))

	res, err := ctx.QueryStore(key, common.TokenStoreName)
	if err != nil {
		return err
	}

	if len(res) == 0 {
		fmt.Printf("No such token(%v) exists\n", symbol)
		return nil
	}

	// decode the value
	token := new(types.Token)
	err = c.Cdc.UnmarshalBinaryBare(res, &token)
	if err != nil {
		return err
	}

	// print out the toke info
	output, err := wire.MarshalJSONIndent(c.Cdc, token)
	if err != nil {
		return err
	}

	fmt.Println(string(output))

	return nil
}
