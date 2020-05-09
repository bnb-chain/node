package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/binance-chain/node/plugins/param"
	"github.com/binance-chain/node/wire"
	"io/ioutil"
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
	flagSCParamFile = "sc-param-file"
)

func SubmitSCParamChangeProposalCmd(cdc *codec.Codec) *cobra.Command {
	scParams := types.SCChangeParams{}
	cmd := &cobra.Command{
		Use:   "submit-sc-change-proposal",
		Short: "Submit a side chain param change proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))
			title := viper.GetString(flagTitle)
			initialDeposit := viper.GetString(flagDeposit)
			scParamFile := viper.GetString(flagSCParamFile)
			scParams.Description = viper.GetString(flagDescription)
			votingPeriodInSeconds := viper.GetInt64(flagVotingPeriod)
			sideChainId := viper.GetString(flagSideChainId)
			if sideChainId == "" {
				return fmt.Errorf("missing side-chain-id")
			}
			if scParamFile == "" {
				return errors.New("sc-param-file is missing")
			}

			bz, err := ioutil.ReadFile(scParamFile)
			if err != nil {
				return err
			}
			err = cdc.UnmarshalJSON(bz, &(scParams.SCParams))
			if err != nil {
				return err
			}
			err = scParams.Check()
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
			// scParams get interface field, use amino
			scParamsBz, err := app.Codec.MarshalJSON(scParams)
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

			msg := gov.NewMsgSideChainSubmitProposal(title, string(scParamsBz), gov.ProposalTypeSCParamsChange, fromAddr, amount, votingPeriod, sideChainId)
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
	cmd.Flags().String(flagSCParamFile, "", "the file of Side Chain params (json format)")
	cmd.Flags().String(flagTitle, "", "title of proposal")
	cmd.Flags().Int64(flagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().String(flagDescription, "", "description of proposal")
	cmd.Flags().String(flagDeposit, "", "deposit of proposal")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain")
	return cmd
}

func ShowSideChainParamsCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-params",
		Short: "Show the params of the side chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			sideChainId := viper.GetString(flagSideChainId)
			if sideChainId == "" {
				return fmt.Errorf("missing side-chain-id")
			}
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))
			format := viper.GetString(flagFormat)
			if format != types.JSONFORMAT && format != types.AMINOFORMAT {
				return fmt.Errorf("format %s is not supported, options [%s, %s] ", format, types.JSONFORMAT, types.AMINOFORMAT)
			}

			bz, err := cliCtx.Query(fmt.Sprintf("%s/fees", param.AbciQueryPrefix), nil)
			if err != nil {
				return err
			}
			var params []types.SCParam
			err = cdc.UnmarshalBinaryLengthPrefixed(bz, &params)
			if err != nil {
				return err
			}

			var output []byte
			if format == types.JSONFORMAT {
				output, err = json.MarshalIndent(params, "", "\t")
			} else if format == types.AMINOFORMAT {
				output, err = cdc.MarshalJSONIndent(params, "", "\t")
			}
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().String(flagFormat, types.AMINOFORMAT, fmt.Sprintf("the response format, options: [%s, %s]", types.AMINOFORMAT, types.JSONFORMAT))
	cmd.Flags().String(flagSideChainId, "", "the id of side chain")
	return cmd
}
