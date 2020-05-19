package commands

import (
	"bytes"
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/wire"
)

const miniTokenKeyPrefix = "mini"

type Commander struct {
	Cdc *wire.Codec
}

type msgBuilder func(from sdk.AccAddress, symbol string, amount int64) sdk.Msg

func (c Commander) checkAndSendTx(cmd *cobra.Command, args []string, builder msgBuilder) error {
	cliCtx, txBuilder := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	symbol := viper.GetString(flagSymbol)
	if !types.IsMiniTokenSymbol(symbol) {
		err = types.ValidateMapperTokenSymbol(symbol)
		if err != nil {
			return err
		}
	}

	symbol = strings.ToUpper(symbol)

	amountStr := viper.GetString(flagAmount)
	amount, err := parseAmount(amountStr)
	if err != nil {
		return err
	}

	// build message
	msg := builder(from, symbol, amount)
	return client.SendOrPrintTx(cliCtx, txBuilder, msg)
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

func validateTokenURI(uri string) error {
	if len(uri) > 2048 {
		return errors.New("uri cannot be longer than 2048 characters")
	}
	return nil
}

func calcMiniTokenKey(symbol string) []byte {
	var buf bytes.Buffer
	buf.WriteString(miniTokenKeyPrefix)
	buf.WriteString(":")
	buf.WriteString(symbol)
	return buf.Bytes()
}
