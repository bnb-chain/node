package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/core"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Commander struct {
	Cdc *wire.Codec
}

type msgBuilder func(from sdk.Address, symbol string, amount int64) sdk.Msg

func (c Commander) checkAndSendTx(cmd *cobra.Command, args []string, builder msgBuilder) error {
	ctx := context.NewCoreContextFromViper().WithDecoder(types.GetAccountDecoder(c.Cdc))

	from, err := ctx.GetFromAddress()
	if err != nil {
		return err
	}

	symbol := viper.GetString(flagSymbol)
	err = validateSymbol(symbol)
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

func (c Commander) sendTx(ctx core.CoreContext, msg sdk.Msg) error {
	// default to next sequence number if none provided
	ctx, err := context.EnsureSequence(ctx)
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

	if !utils.IsAlphaNum(symbol) {
		return errors.New("the symbol should be alphanumeric")
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
