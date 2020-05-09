package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/app"
	"github.com/binance-chain/node/plugins/param/types"
)

const (

	//CSC flag
	flagKey    = "key"
	flagValue  = "value"
	flagTaregt = "target"
)

func SubmitCSCParamChangeProposalCmd(cdc *codec.Codec) *cobra.Command {
	var cscParam types.CSCParamChange
	cmd := &cobra.Command{
		Use:   "submit-cscParam-change-proposal",
		Short: "Submit a cross side chain parameter change proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))
			title := viper.GetString(flagTitle)
			initialDeposit := viper.GetString(flagDeposit)
			votingPeriodInSeconds := viper.GetInt64(flagVotingPeriod)
			cscParam.Key = viper.GetString(flagKey)
			value := viper.GetString(flagValue)
			sideChainId := viper.GetString(flagSideChainId)
			if sideChainId == "" {
				return fmt.Errorf("missing side-chain-id")
			}
			if strings.HasPrefix(value, "0x") {
				value = value[2:]
			}
			cscParam.Value = value

			target := viper.GetString(flagTaregt)
			if strings.HasPrefix(target, "0x") {
				target = target[2:]
			}
			cscParam.Target = target

			err := cscParam.Check()
			if err != nil {
				return err
			}
			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}
			amount, err := sdk.ParseCoins(initialDeposit)
			if err != nil {
				return err
			}
			// cscParam get interface field, use amino
			cscParamsBz, err := app.Codec.MarshalJSON(cscParam)
			if err != nil {
				return err
			}

			if votingPeriodInSeconds <= 0 {
				return errors.New("voting period should be positive")
			}

			votingPeriod := time.Duration(votingPeriodInSeconds) * time.Second
			if votingPeriod > gov.MaxVotingPeriod {
				return fmt.Errorf("voting period should less than %d seconds", gov.MaxVotingPeriod/time.Second)
			}

			msg := gov.NewMsgSideChainSubmitProposal(title, string(cscParamsBz), gov.ProposalTypeCSCParamsChange, fromAddr, amount, votingPeriod, sideChainId)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}
			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}
			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}
	cmd.Flags().String(flagKey, "", "the parameter name on the side chain")
	cmd.Flags().String(flagValue, "", "the specified value of the parameter on side chain, should encoded in hex")
	cmd.Flags().String(flagTaregt, "", "the address of the contract on side chain")
	cmd.Flags().String(flagTitle, "", "title of proposal")
	cmd.Flags().Int64(flagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().String(flagDeposit, "", "deposit of proposal")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain")
	return cmd
}
