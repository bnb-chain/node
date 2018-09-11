package commands

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	authctx "github.com/cosmos/cosmos-sdk/x/auth/client/context"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/BiJie/BinanceChain/common/types"
)

const (
	flagTo = "to"
)

// SendTxCmd will create a send tx and sign it with the given key.
func transferCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "transfer tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			txCtx := authctx.NewTxContextFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().WithAccountDecoder(types.GetAccountDecoder(cdc))

			if err := cliCtx.EnsureAccountExists(); err != nil {
				return err
			}

			toStr := viper.GetString(flagTo)
			to, err := sdk.AccAddressFromBech32(toStr)
			if err != nil {
				return err
			}

			amount := viper.GetString(flagAmount)
			tokens, err := parseTokens(amount)
			if err != nil {
				return err
			}

			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			account, err := cliCtx.GetAccount(from)
			if err != nil {
				return err
			}

			// ensure account has enough coins
			if !account.GetCoins().IsGTE(tokens) {
				return errors.Errorf("Address %s doesn't have enough coins to pay for this transaction.", from)
			}

			// build and sign the transaction, then broadcast to Tendermint
			input := bank.NewInput(from, tokens)
			output := bank.NewOutput(to, tokens)
			msg := bank.NewMsgSend([]bank.Input{input}, []bank.Output{output})

			return utils.SendTx(txCtx, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagTo, "", "Address to send tokens")
	cmd.Flags().String(flagAmount, "", "Amount of tokens to send")

	return cmd
}

// ParseCoin parses out one token with its amount, returning errors if invalid.
// This returns an error on an empty string as well.
func parseToken(tokenStr string) (coin sdk.Coin, err error) {
	tokenStr = strings.TrimSpace(tokenStr)
	strs := strings.Split(tokenStr, ":")
	if len(strs) != 2 {
		err = fmt.Errorf("invalid token expression: %s", tokenStr)
		return
	}
	tokenSymbol, amountStr := strs[0], strs[1]
	err = types.ValidateSymbol(tokenSymbol)
	if err != nil {
		return
	}

	tokenSymbol = strings.ToUpper(tokenSymbol)
	amount, err := parseAmount(amountStr)
	if err != nil {
		return
	}

	return sdk.NewInt64Coin(tokenSymbol, amount), nil
}

// ParseCoins will parse out a list of tokens separated by commas.
// If nothing is provided, it returns nil Coins.
// Returned coins are sorted.
func parseTokens(tokensStr string) (tokens sdk.Coins, err error) {
	tokensStr = strings.TrimSpace(tokensStr)
	if len(tokensStr) == 0 {
		return nil, nil
	}

	tokenStrs := strings.Split(tokensStr, ",")
	for _, tokenStr := range tokenStrs {
		token, err := parseToken(tokenStr)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	// Sort coins for determinism.
	tokens.Sort()

	// Validate coins before returning.
	if !tokens.IsValid() {
		return nil, fmt.Errorf("parseTokens invalid: %#v", tokens)
	}

	return tokens, nil
}
