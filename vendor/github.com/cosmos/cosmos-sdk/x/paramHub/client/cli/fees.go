package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/go-amino"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/paramHub"
	"github.com/cosmos/cosmos-sdk/x/paramHub/types"
)

const (
	//Fee flag
	flagFeeParamFile = "fee-param-file"
	flagFormat       = "format"
)

func SubmitFeeChangeProposalCmd(cdc *codec.Codec) *cobra.Command {
	feeParam := types.FeeChangeParams{FeeParams: make([]types.FeeParam, 0)}
	cmd := &cobra.Command{
		Use:   "submit-fee-change-proposal",
		Short: "Submit a fee or fee rate change proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))
			title := viper.GetString(flagTitle)
			initialDeposit := viper.GetString(flagDeposit)
			feeParamFile := viper.GetString(flagFeeParamFile)
			feeParam.Description = viper.GetString(flagDescription)
			votingPeriodInSeconds := viper.GetInt64(flagVotingPeriod)
			if feeParamFile == "" {
				return errors.New("fee-param-file is missing")
			}

			bz, err := ioutil.ReadFile(feeParamFile)
			if err != nil {
				return err
			}
			err = cdc.UnmarshalJSON(bz, &(feeParam.FeeParams))
			if err != nil {
				return err
			}
			err = feeParam.Check()
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
			// feeParam get interface field, use amino
			feeParamsBz, err := cdc.MarshalJSON(feeParam)
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

			msg := gov.NewMsgSubmitProposal(title, string(feeParamsBz), gov.ProposalTypeFeeChange, fromAddr, amount, votingPeriod)
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
	cmd.Flags().String(flagFeeParamFile, "", "the file of fee params (json format)")
	cmd.Flags().String(flagTitle, "", "title of proposal")
	cmd.Flags().Int64(flagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().String(flagDescription, "", "description of proposal")
	cmd.Flags().String(flagDeposit, "", "deposit of proposal")
	return cmd
}

func ShowFeeParamsCmd(cdc *amino.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-fees",
		Short: "Show order book of the listed currency pair",
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))
			format := viper.GetString(flagFormat)
			if format != types.JSONFORMAT && format != types.AMINOFORMAT {
				return fmt.Errorf("format %s is not supported, options [%s, %s] ", format, types.JSONFORMAT, types.AMINOFORMAT)
			}

			bz, err := cliCtx.Query(fmt.Sprintf("%s/fees", paramHub.AbciQueryPrefix), nil)
			if err != nil {
				return err
			}
			var fees []types.FeeParam
			err = cdc.UnmarshalBinaryLengthPrefixed(bz, &fees)
			if err != nil {
				return err
			}

			var output []byte
			if format == types.JSONFORMAT {
				output, err = json.MarshalIndent(fees, "", "\t")
			} else if format == types.AMINOFORMAT {
				output, err = cdc.MarshalJSONIndent(fees, "", "\t")
			}
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().String(flagFormat, types.AMINOFORMAT, fmt.Sprintf("the response format, options: [%s, %s]", types.AMINOFORMAT, types.JSONFORMAT))
	return cmd
}
