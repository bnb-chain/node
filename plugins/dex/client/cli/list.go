package commands

import (
	"errors"

	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/utils"
	dextypes "github.com/binance-chain/node/plugins/dex/types"
	"github.com/binance-chain/node/wire"
)

const flagBaseAsset = "base-asset-symbol"
const flagQuoteAsset = "quote-asset-symbol"
const flagInitPrice = "init-price"
const flagProposalId = "proposal-id"

func listTradingPairCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list a trading pair, notice: it is unsupported after XX upgrade ", // todo fill the correct upgrade name
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txbldr := client.PrepareCtx(cdc)

			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			baseAsset := viper.GetString(flagBaseAsset)
			err = types.ValidateTokenSymbol(baseAsset)
			if err != nil {
				return err
			}

			quoteAsset := viper.GetString(flagQuoteAsset)
			err = types.ValidateTokenSymbol(quoteAsset)
			if err != nil {
				return err
			}

			baseAsset = strings.ToUpper(baseAsset)
			quoteAsset = strings.ToUpper(quoteAsset)

			initPriceStr := viper.GetString(flagInitPrice)
			initPrice, err := utils.ParsePrice(initPriceStr)
			if err != nil {
				return err
			}

			proposalId := viper.GetInt64(flagProposalId)
			if proposalId <= 0 {
				return errors.New("proposal id should larger than zero")
			}

			msg := dextypes.NewListMsg(from, proposalId, baseAsset, quoteAsset, initPrice)
			err = client.SendOrPrintTx(cliCtx, txbldr, msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagBaseAsset, "s", "", "symbol of the base asset")
	cmd.Flags().String(flagQuoteAsset, "", "symbol of the quote currency")
	cmd.Flags().String(flagInitPrice, "", "init price for this pair")
	cmd.Flags().Int64(flagProposalId, 0, "list proposal id")

	return cmd
}

func listMiniTradingPairCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-mini",
		Short: "list a mini trading pair, notice: it is unsupported after XX upgrade ", // todo fill the correct upgrade name
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txbldr := client.PrepareCtx(cdc)

			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			baseAsset := viper.GetString(flagBaseAsset)
			err = types.ValidateMiniTokenSymbol(baseAsset)
			if err != nil {
				return err
			}

			quoteAsset := viper.GetString(flagQuoteAsset)
			if quoteAsset != types.NativeTokenSymbol && !strings.HasPrefix(quoteAsset, "BUSD") {
				return errors.New("invalid quote asset")
			}

			baseAsset = strings.ToUpper(baseAsset)
			quoteAsset = strings.ToUpper(quoteAsset)

			initPriceStr := viper.GetString(flagInitPrice)
			initPrice, err := utils.ParsePrice(initPriceStr)
			if err != nil {
				return err
			}

			msg := dextypes.NewListMiniMsg(from, baseAsset, quoteAsset, initPrice)
			err = client.SendOrPrintTx(cliCtx, txbldr, msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagBaseAsset, "s", "", "symbol of the base asset")
	cmd.Flags().String(flagQuoteAsset, "", "symbol of the quote currency")
	cmd.Flags().String(flagInitPrice, "", "init price for this pair")

	return cmd
}

func listGrowthMarketCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-growth-market",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, txbldr := client.PrepareCtx(cdc)

			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			baseAsset := viper.GetString(flagBaseAsset)
			if !types.IsValidMiniTokenSymbol(baseAsset) {
				err = types.ValidateTokenSymbol(baseAsset)
				if err != nil {
					return err
				}
			}

			quoteAsset := viper.GetString(flagQuoteAsset)
			if quoteAsset != types.NativeTokenSymbol && !strings.HasPrefix(quoteAsset, "BUSD") {
				return errors.New("invalid quote asset")
			}

			baseAsset = strings.ToUpper(baseAsset)
			quoteAsset = strings.ToUpper(quoteAsset)

			initPriceStr := viper.GetString(flagInitPrice)
			initPrice, err := utils.ParsePrice(initPriceStr)
			if err != nil {
				return err
			}

			msg := dextypes.NewListGrowthMarketMsg(from, baseAsset, quoteAsset, initPrice)
			err = client.SendOrPrintTx(cliCtx, txbldr, msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP(flagBaseAsset, "s", "", "symbol of the base asset")
	cmd.Flags().String(flagQuoteAsset, "", "symbol of the quote currency")
	cmd.Flags().String(flagInitPrice, "", "init price for this pair")

	return cmd
}
