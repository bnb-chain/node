package commands

import (
	"errors"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	cli "github.com/cosmos/cosmos-sdk/client/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authctx "github.com/cosmos/cosmos-sdk/x/auth/client/context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/wire"
)

type Commander struct {
	Cdc *wire.Codec
}

type msgBuilder func(from sdk.AccAddress, symbol string, amount int64) sdk.Msg

func (c Commander) checkAndSendTx(cmd *cobra.Command, args []string, builder msgBuilder) error {
	txCtx := authctx.NewTxContextFromCLI().WithCodec(c.Cdc)
	cliCtx := context.NewCLIContext().WithAccountDecoder(types.GetAccountDecoder(c.Cdc))

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	symbol := viper.GetString(flagSymbol)
	err = types.ValidateSymbol(symbol)
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
	msg := builder(from, symbol, amount)
	return cli.SendTx(txCtx, cliCtx, []sdk.Msg{msg})
}

func parseAmount(amountStr string) (int64, error) {
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return 0, err
	}

	if amount <= 0 {
		return amount, errors.New("the amount should be greater than 0")
	}

	return amount, nil
}
