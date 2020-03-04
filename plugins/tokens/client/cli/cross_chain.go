package commands

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/spf13/viper"

	"github.com/binance-chain/node/common/types"

	"github.com/binance-chain/node/plugins/tokens/cross_chain"

	"github.com/spf13/cobra"

	"github.com/binance-chain/node/common/client"
)

const (
	flagContractAddress = "contract-address"
	flagContractDecimal = "contract-decimal"
	flagToAddress       = "to"
)

func bindCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cross-bind",
		Short: "bind smart chain token to bep2 token",
		RunE:  cmdr.bind,
	}

	cmd.Flags().String(flagContractAddress, "", "contract address")
	cmd.Flags().Int(flagContractDecimal, 0, "contract token decimal")
	cmd.Flags().String(flagSymbol, "", "symbol")

	return cmd
}

func (c Commander) bind(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	contractAddress := viper.GetString(flagContractAddress)
	contractDecimal := viper.GetInt(flagContractDecimal)
	symbol := viper.GetString(flagSymbol)

	// build message
	msg := cross_chain.NewBindMsg(from, symbol, types.NewEthereumAddress(contractAddress), contractDecimal)

	sdkErr := msg.ValidateBasic()
	if sdkErr != nil {
		return fmt.Errorf("%v", sdkErr.Data())
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}

func transferCmd(cmdr Commander) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cross-transfer",
		Short: "transfer bep2 token to smart chain",
		RunE:  cmdr.bind,
	}

	cmd.Flags().String(flagContractAddress, "", "contract address")
	cmd.Flags().String(flagToAddress, "", "smart chain address")
	cmd.Flags().String(flagAmount, "", "amount")

	return cmd
}

func (c Commander) transfer(cmd *cobra.Command, args []string) error {
	cliCtx, txBldr := client.PrepareCtx(c.Cdc)

	from, err := cliCtx.GetFromAddress()
	if err != nil {
		return err
	}

	contractAddress := viper.GetString(flagContractAddress)
	to := viper.GetString(flagToAddress)
	amount := viper.GetString(flagAmount)

	amountToTransfer, err := sdk.ParseCoin(amount)
	if err != nil {
		return err
	}

	// build message
	msg := cross_chain.NewTransferMsg(from, types.NewEthereumAddress(contractAddress), types.NewEthereumAddress(to), amountToTransfer)

	sdkErr := msg.ValidateBasic()
	if sdkErr != nil {
		return fmt.Errorf("%v", sdkErr.Data())
	}
	return client.SendOrPrintTx(cliCtx, txBldr, msg)
}
