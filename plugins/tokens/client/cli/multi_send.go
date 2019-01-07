package commands

import (
	"encoding/json"
	"io/ioutil"

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
	flagTx     = "tx"
	flagTxPath = "tx-path"
)

type Tx struct {
	To     string `json:"to"`
	Amount string `json:"amount"`
}

type Txs []Tx

// MultiSendCmd will create a send tx and sign it with the given key
func MultiSendCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-send",
		Short: "Create and sign a multi send tx",
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

			txPath := viper.GetString(flagTxPath)
			txBytes := make([]byte, 0)
			if txPath != "" {
				txBytes, err = ioutil.ReadFile(txPath)
				if err != nil {
					return err
				}
			} else {
				txStr := viper.GetString(flagTx)
				txBytes = []byte(txStr)
			}

			txs := Txs{}
			err = json.Unmarshal(txBytes, &txs)
			if err != nil {
				return err
			}

			if len(txs) == 0 {
				return errors.New("tx is empty")
			}

			toAddrs := make([]sdk.AccAddress, 0, len(txs))
			toCoins := make([]sdk.Coins, 0, len(txs))

			for _, tx := range txs {
				to, err := sdk.AccAddressFromBech32(tx.To)
				if err != nil {
					return err
				}
				toAddrs = append(toAddrs, to)

				toCoin, err := sdk.ParseCoins(tx.Amount)
				if err != nil {
					return err
				}
				toCoins = append(toCoins, toCoin)
			}

			fromCoins := sdk.Coins{}
			for _, toCoin := range toCoins {
				fromCoins = fromCoins.Plus(toCoin)
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
			msg := BuildMultiSendMsg(from, fromCoins, toAddrs, toCoins)
			return utils.CompleteAndBroadcastTxCli(txBldr, ctx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagTx, "", "Tx details, format: [{\"to\": \"addr\", \"amount\": \"1:BNB,2:BTC\"}, ...]")
	cmd.Flags().String(flagTxPath, "", "Path of tx details, if path is not empty, --tx will be ignored")

	return cmd
}

func BuildMultiSendMsg(from sdk.AccAddress, fromCoins sdk.Coins, toAddrs []sdk.AccAddress, toCoins []sdk.Coins) sdk.Msg {
	input := bank.NewInput(from, fromCoins)

	output := make([]bank.Output, 0, len(toAddrs))
	for idx, toAddr := range toAddrs {
		output = append(output, bank.NewOutput(toAddr, toCoins[idx]))
	}
	msg := bank.NewMsgSend([]bank.Input{input}, output)
	return msg
}
