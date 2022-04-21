package commands

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/context"

	"github.com/bnb-chain/node/common"
	"github.com/bnb-chain/node/common/types"
	"github.com/bnb-chain/node/wire"
)

func getTokenInfoCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <symbol>",
		Short: "Query token/mini-token info",
		RunE:  cmdr.runGetToken,
	}

	cmd.Flags().StringP(flagSymbol, "s", "", "symbol of the token")
	return cmd
}

func (c Commander) runGetToken(cmd *cobra.Command, args []string) error {
	ctx := context.NewCLIContext().WithCodec(c.Cdc)

	symbol := viper.GetString(flagSymbol)
	if len(symbol) == 0 {
		return errors.New("you must provide the symbol")
	}

	var key []byte
	if types.IsMiniTokenSymbol(symbol) {
		key = calcMiniTokenKey(strings.ToUpper(symbol))
	} else {
		key = []byte(strings.ToUpper(symbol))
	}

	res, err := ctx.QueryStore(key, common.TokenStoreName)
	if err != nil {
		return err
	}

	if len(res) == 0 {
		fmt.Printf("No such token(%v) exists\n", symbol)
		return nil
	}

	// decode the value
	var token types.IToken
	err = c.Cdc.UnmarshalBinaryBare(res, &token)
	if err != nil {
		return err
	}

	// print out the toke info
	output, err := wire.MarshalJSONIndent(c.Cdc, &token)
	if err != nil {
		return err
	}

	fmt.Println(string(output))
	return nil
}
