package commands

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/keys"
	txutils "github.com/cosmos/cosmos-sdk/client/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"

	"github.com/BiJie/BinanceChain/common/client"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/common/utils"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/wire"
)

const (
	flagRefId       = "refid"
	flagPrice       = "price"
	flagQty         = "qty"
	flagSide        = "side"
	flagTimeInForce = "tif"
	flagDryRun      = "dry"
)

func newOrderCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order -l <pair> -s <side> -p <price> -q <qty> -t <timeInForce>",
		Short: "Submit a new order",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txBldr := client.PrepareCtx(cdc)
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			symbol := viper.GetString(flagSymbol)
			err = validatePairSymbol(symbol)
			if err != nil {
				return err
			}

			symbol = strings.ToUpper(symbol)

			priceStr := viper.GetString(flagPrice)
			price, err := utils.ParsePrice(priceStr)
			if err != nil {
				return err
			}

			qtyStr := viper.GetString(flagQty)
			qty, err := utils.ParsePrice(qtyStr)
			if err != nil {
				return err
			}

			tif, err := order.TifStringToTifCode(viper.GetString(flagTimeInForce))
			if err != nil {
				return err
			}
			side := int8(viper.GetInt(flagSide))

			// avoids an ugly panic on sequence 0 with --dry
			acc, err := cliCtx.GetAccount(from)
			if acc == nil || err != nil {
				fmt.Println("No transactions involving this address yet. Using sequence 0.")
				txBldr = txBldr.WithSequence(0)
			} else {
				err = client.EnsureSequence(cliCtx, &txBldr)
				if err != nil {
					return err
				}
			}

			msg, err := order.NewNewOrderMsgAuto(txBldr, from, side, symbol, price, qty)
			if err != nil {
				return err
			}

			msg.TimeInForce = tif
			msgs := []sdk.Msg{msg}

			if viper.GetBool(flagDryRun) {
				fmt.Println("Performing dry run; will not broadcast the transaction.")
				name, _ := cliCtx.GetFromName()
				passphrase, err := keys.GetPassphrase(name)
				txBytes, err := txBldr.BuildAndSign(name, passphrase, msgs)
				if err != nil {
					return err
				}
				var tx auth.StdTx
				if err = txBldr.Codec.UnmarshalBinary(txBytes, &tx); err == nil {
					json, err := txBldr.Codec.MarshalJSON(tx)
					if err == nil {
						fmt.Printf("TX JSON: %s\n", json)
					}
				}
				hexBytes := make([]byte, len(txBytes)*2)
				hex.Encode(hexBytes, txBytes)
				fmt.Printf("TX Hex: %s\n", hexBytes)
				return nil
			}

			err = client.SendOrPrintTx(cliCtx, txBldr, msg)
			if err != nil {
				return err
			}

			fmt.Printf("Msg [%v] was sent.\n", msg)
			return nil
		},
	}
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	cmd.Flags().StringP(flagSide, "s", "", "side (buy as 1 or sell as 2) of the order")
	cmd.Flags().StringP(flagPrice, "p", "", "price for the order")
	cmd.Flags().StringP(flagQty, "q", "", "quantity for the order")
	cmd.Flags().StringP(flagTimeInForce, "t", "gtc", "TimeInForce for the order (gtc or ioc)")
	cmd.Flags().BoolP(flagDryRun, "d", false, "Generate and return the tx bytes (do not broadcast)")
	return cmd
}

func showOrderBookCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show -l <listed pair>",
		Short: "Show order book of the listed currency pair",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCLIContext().WithAccountDecoder(types.GetAccountDecoder(cdc))

			symbol := viper.GetString(flagSymbol)
			err := validatePairSymbol(symbol)
			if err != nil {
				return err
			}

			ob, err := store.GetOrderBook(cdc, ctx, symbol)
			if err != nil {
				return err
			}
			levels := ob.Levels

			fmt.Printf("%16v|%16v|%16v|%16v\n", "SellQty", "SellPrice", "BuyPrice", "BuyQty")
			for _, l := range levels {
				fmt.Printf("%16v|%16v|%16v|%16v\n", l.SellQty, l.SellPrice, l.BuyPrice, l.BuyQty)
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	return cmd
}

func cancelOrderCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel -l <trading pair> -f <ref order id>",
		Short: "Cancel an order",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := txbuilder.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().WithCodec(cdc).WithAccountDecoder(types.GetAccountDecoder(cdc))
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}
			symbol := viper.GetString(flagSymbol)
			err = validatePairSymbol(symbol)
			if err != nil {
				return err
			}
			refId := viper.GetString(flagRefId)
			if refId == "" {
				return errors.New("please input reference order id")
			}
			msg := order.NewCancelOrderMsg(from, symbol, refId)
			if cliCtx.GenerateOnly {
				return txutils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg}, false)
			}

			err = txutils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
			if err != nil {
				return err
			}
			fmt.Printf("Msg [%v] was sent.\n", msg)
			return nil
		},
	}
	cmd.Flags().StringP(flagSymbol, "l", "", "the listed trading pair, such as ADA_BNB")
	cmd.Flags().StringP(flagRefId, "f", "", "id string of the order")
	return cmd
}

func validatePairSymbol(symbol string) error {
	return store.ValidatePairSymbol(symbol)
}
