package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/cosmos/cosmos-sdk/bsc"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagEvidence     = "evidence"
	flagEvidenceFile = "evidence-file"

	flagSideChainId = "side-chain-id"
)

// GetCmdSubmitEvidence implements the submit evidence command handler.
func GetCmdBscSubmitEvidence(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bsc-submit-evidence",
		Short: "submit evidence against the malicious validator on bsc",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			if err := cliCtx.EnsureAccountExists(); err != nil {
				return err
			}

			// get the from/to address
			from, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			filePath := viper.GetString(flagEvidenceFile)
			evidenceBytes := make([]byte, 0)
			if filePath != "" {
				evidenceBytes, err = ioutil.ReadFile(filePath)
				if err != nil {
					return err
				}
			} else {
				txStr := viper.GetString(flagEvidence)
				if txStr == "" {
					return errors.New(fmt.Sprintf("either %s or %s is required", flagEvidenceFile, flagEvidence))
				}
				evidenceBytes = []byte(txStr)
			}

			headers := make([]bsc.Header, 0)
			err = json.Unmarshal(evidenceBytes, &headers)
			if err != nil {
				return err
			}

			if len(headers) != 2 {
				return errors.New(fmt.Sprintf("must have 2 headers exactly"))
			}

			msg := slashing.NewMsgBscSubmitEvidence(from, headers)

			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}
	cmd.Flags().String(flagEvidence, "", "Evidence details, including two headers with json format, e.g. [{\"difficulty\":\"0x2\",\"extraData\":\"0xd98301...},{\"difficulty\":\"0x3\",\"extraData\":\"0xd64372...}]")
	cmd.Flags().String(flagEvidenceFile, "", "File of evidence details, if evidence-file is not empty, --evidence will be ignored")
	return cmd
}

// GetCmdSideChainUnjail implements the create unjail validator command.
func GetCmdSideChainUnjail(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-unjail",
		Args:  cobra.NoArgs,
		Short: "unjail side validator previously jailed",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			valAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			sideChainId, err := getSideChainId()
			if err != nil {
				return err
			}

			msg := slashing.NewMsgSideChainUnjail(sdk.ValAddress(valAddr), sideChainId)

			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}
	cmd.Flags().String(FlagSideChainId, "", "chain-id of the side chain the validator belongs to")
	return cmd
}

func getSideChainId() (sideChainId string, err error) {
	sideChainId = viper.GetString(flagSideChainId)
	if len(sideChainId) == 0 {
		err = fmt.Errorf("%s is required", flagSideChainId)
	}
	return
}
