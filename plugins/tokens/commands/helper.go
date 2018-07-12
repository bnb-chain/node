package commands

import (
	"errors"
	"strconv"
	"strings"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Commander struct {
	Cdc *wire.Codec
}

type msgBuilder func(from sdk.AccAddress, symbol string, amount int64) sdk.Msg

func (c Commander) checkAndSendTx(cmd *cobra.Command, args []string, builder msgBuilder) error {
	ctx := context.NewCoreContextFromViper().WithDecoder(types.GetAccountDecoder(c.Cdc))

	from, err := ctx.GetFromAddress()
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
	return c.sendTx(ctx, msg)
}

func (c Commander) sendTx(ctx context.CoreContext, msg sdk.Msg) error {
	err := ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, c.Cdc)
	if err != nil {
		return err
	}

	return nil
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
