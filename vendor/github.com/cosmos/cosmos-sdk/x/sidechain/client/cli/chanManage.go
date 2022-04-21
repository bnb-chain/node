package cli

import (
	"errors"
	"fmt"
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
	"github.com/cosmos/cosmos-sdk/x/sidechain/types"
)

const (
	flagChannelId     = "channel-id"
	flagChannelEnable = "enable"
)

func SubmitChannelManageProposalCmd(cdc *codec.Codec) *cobra.Command {
	var channelSetting types.ChanPermissionSetting
	cmd := &cobra.Command{
		Use:   "submit-channel-manage-proposal",
		Short: "Submit a cross chain channel management proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))
			title := viper.GetString(flagTitle)
			initialDeposit := viper.GetString(flagDeposit)
			votingPeriodInSeconds := viper.GetInt64(flagVotingPeriod)
			channelId := viper.GetUint(flagChannelId)
			sideChainId := viper.GetString(flagSideChainId)
			if sideChainId == "" {
				return fmt.Errorf("missing side-chain-id")
			}

			channelSetting.ChannelId = sdk.ChannelID(channelId)
			channelSetting.SideChainId = sideChainId
			enable := viper.GetBool(flagChannelEnable)
			if enable {
				channelSetting.Permission = sdk.ChannelAllow
			} else {
				channelSetting.Permission = sdk.ChannelForbidden
			}

			err := channelSetting.Check()
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
			cscParamsBz, err := cdc.MarshalJSON(channelSetting)
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

			msg := gov.NewMsgSubmitProposal(title, string(cscParamsBz), gov.ProposalTypeManageChanPermission, fromAddr, amount, votingPeriod)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}
			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}
	cmd.Flags().Uint8(flagChannelId, 0, "the the channel id that want to manage")
	cmd.Flags().Bool(flagChannelEnable, true, "enable the channel or not")
	cmd.Flags().String(flagTitle, "", "title of proposal")
	cmd.Flags().Int64(flagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().String(flagDeposit, "", "deposit of proposal")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain")
	return cmd
}
func ShowChannelPermissionCmd(cdc *amino.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-channel-permissions",
		Short: "Show channel permissions of side chain",
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))
			sideChainId := viper.GetString(flagSideChainId)
			if sideChainId == "" {
				return fmt.Errorf("missing side-chain-id")
			}

			queryData, err := cdc.MarshalJSON(sideChainId)
			if err != nil {
				return err
			}

			bz, err := cliCtx.Query(fmt.Sprintf("custom/sideChain/channelSettings"), queryData)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}

	cmd.Flags().String(flagSideChainId, "", "the id of side chain")
	return cmd
}
