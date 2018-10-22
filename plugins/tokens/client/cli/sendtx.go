package commands

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/wire"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagTo = "to"
)

// SendTxCmd will create a send tx and sign it with the given key
func MultiSendTxCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-send",
		Short: "Create and sign a send tx",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCoreContextFromViper().WithDecoder(authcmd.GetAccountDecoder(cdc))

			// get the from/to address
			from, err := ctx.GetFromAddress()
			if err != nil {
				return err
			}

			fromAcc, err := ctx.QueryStore(auth.AddressStoreKey(from), ctx.AccountStore)
			if err != nil {
				return err
			}

			// Check if account was found
			if fromAcc == nil {
				return errors.Errorf("No account with address %s was found in the state.\nAre you sure there has been a transaction involving it?", from)
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

			// parse coins trying to be sent
			amount := viper.GetString(flagAmount)
			coins, err := sdk.ParseCoins(amount)
			if err != nil {
				return err
			}

			// ensure account has enough coins
			account, err := ctx.Decoder(fromAcc)
			if err != nil {
				return err
			}

			fromCoins := sdk.Coins{}
			for _, toCoin := range coins {
				fromCoin := toCoin
				fromCoin.Amount = fromCoin.Amount.Mul(sdk.NewInt(int64(len(toAddrs))))
				fromCoins = append(fromCoins, fromCoin)
			}

			if !account.GetCoins().IsGTE(fromCoins) {
				return errors.Errorf("Address %s doesn't have enough coins to pay for this transaction.", from)
			}

			// build and sign the transaction, then broadcast to Tendermint
			msg := BuildMsg(from, fromCoins, toAddrs, coins)

			err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc)
			if err != nil {
				return err
			}
			return nil

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
