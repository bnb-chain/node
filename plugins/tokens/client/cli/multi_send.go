package commands

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/wire"
)

const (
	flagTo = "to"
)

// MultiSendCmd will create a send tx and sign it with the given key
func MultiSendCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-send",
		Short: "Create and sign a send tx",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := txbuilder.NewTxBuilderFromCLI().WithCodec(cdc)
			ctx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			if err := ctx.EnsureAccountExists(); err != nil {
				return err
			}

			// get the from/to address
			from, err := ctx.GetFromAddress()
			if err != nil {
				return err
			}

			toStr := viper.GetString(flagTo)

			toAddrsStr := strings.Split(toStr, ":")

			toAddrs := make([]sdk.AccAddress, 0, len(toAddrsStr))
			for _, toAddr := range toAddrsStr {
				println(toAddr)
				to, err := sdk.AccAddressFromBech32(toAddr)
				if err != nil {
					return err
				}

				toAddrs = append(toAddrs, to)
			}

			// parse toCoins trying to be sent
			amount := viper.GetString(flagAmount)
			toCoins, err := sdk.ParseCoins(amount)
			if err != nil {
				return err
			}

			fromCoins := sdk.Coins{}
			for _, toCoin := range toCoins {
				fromCoin := toCoin
				fromCoin.Amount = fromCoin.Amount.Mul(sdk.NewInt(int64(len(toAddrs))))
				fromCoins = append(fromCoins, fromCoin)
			}

			// ensure account has enough toCoins
			account, err := ctx.GetAccount(from)
			if err != nil {
				return err
			}

			if !account.GetCoins().IsGTE(fromCoins) {
				return errors.Errorf("Address %s doesn't have enough toCoins to pay for this transaction.", from)
			}

			// build and sign the transaction, then broadcast to Tendermint
			msg := BuildMsg(from, fromCoins, toAddrs, toCoins)
			return utils.CompleteAndBroadcastTxCli(txBldr, ctx, []sdk.Msg{msg})

		},
	}

	cmd.Flags().String(flagTo, "", "Address to send coins")
	cmd.Flags().String(flagAmount, "", "Amount of coins to send")

	return cmd
}

func BuildMsg(from sdk.AccAddress, fromCoins sdk.Coins, toAddrs []sdk.AccAddress, toCoins sdk.Coins) sdk.Msg {
	input := bank.NewInput(from, fromCoins)

	output := make([]bank.Output, 0, len(toAddrs))
	for _, toAddr := range toAddrs {
		output = append(output, bank.NewOutput(toAddr, toCoins))
	}
	msg := bank.NewMsgSend([]bank.Input{input}, output)

	return msg
}
