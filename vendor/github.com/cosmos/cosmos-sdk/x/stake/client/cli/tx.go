package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GetCmdCreateValidator implements the create validator command handler.
func GetCmdCreateValidator(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-validator",
		Short: "create new validator initialized with a self-delegation to it",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			amount, err := getAmount()
			if err != nil {
				return err
			}

			valAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			pkStr := viper.GetString(FlagPubKey)
			if len(pkStr) == 0 {
				return fmt.Errorf("must use --pubkey flag")
			}

			pk, err := sdk.GetConsPubKeyBech32(pkStr)
			if err != nil {
				return err
			}

			if viper.GetString(FlagMoniker) == "" {
				return fmt.Errorf("please enter a moniker for the validator using --moniker")
			}

			description := stake.Description{
				Moniker:  viper.GetString(FlagMoniker),
				Identity: viper.GetString(FlagIdentity),
				Website:  viper.GetString(FlagWebsite),
				Details:  viper.GetString(FlagDetails),
			}

			// get the initial validator commission parameters
			rateStr := viper.GetString(FlagCommissionRate)
			maxRateStr := viper.GetString(FlagCommissionMaxRate)
			maxChangeRateStr := viper.GetString(FlagCommissionMaxChangeRate)
			commissionMsg, err := buildCommissionMsg(rateStr, maxRateStr, maxChangeRateStr)
			if err != nil {
				return err
			}

			var msg sdk.Msg
			if viper.GetString(FlagAddressDelegator) != "" {
				delAddr, err := sdk.AccAddressFromBech32(viper.GetString(FlagAddressDelegator))
				if err != nil {
					return err
				}

				msg = stake.NewMsgCreateValidatorOnBehalfOf(
					delAddr, sdk.ValAddress(valAddr), pk, amount, description, commissionMsg,
				)
			} else {
				msg = stake.NewMsgCreateValidator(
					sdk.ValAddress(valAddr), pk, amount, description, commissionMsg,
				)
			}

			if viper.GetBool(FlagGenesisFormat) {
				ip := viper.GetString(FlagIP)
				nodeID := viper.GetString(FlagNodeID)
				if nodeID != "" && ip != "" {
					txBldr = txBldr.WithMemo(fmt.Sprintf("%s@%s:26656", nodeID, ip))
				}
			}

			proposalId := viper.GetInt64(FlagProposalID)
			if proposalId == -1 {
				depositStr := viper.GetString(FlagDeposit)
				if depositStr == "" {
					return fmt.Errorf("must specify deposit amount when proposalId is zero using --deposit")
				}
				deposit, err := sdk.ParseCoin(depositStr)
				if err != nil {
					return err
				}
				title := fmt.Sprintf("create validator %s", valAddr.String())

				description, err := json.Marshal(msg)
				if err != nil {
					return err
				}
				votingPeriodInSeconds := viper.GetInt64(FlagVotingPeriod)

				if votingPeriodInSeconds <= 0 {
					return errors.New("voting period should be positive")
				}

				votingPeriod := time.Duration(votingPeriodInSeconds) * time.Second
				if votingPeriod > gov.MaxVotingPeriod {
					return errors.New(fmt.Sprintf("voting period should be less than %d seconds", gov.MaxVotingPeriod/time.Second))
				}

				msg = gov.NewMsgSubmitProposal(title, string(description),
					gov.ProposalTypeCreateValidator, valAddr, sdk.Coins{deposit}, votingPeriod)
			} else {
				msg = stake.MsgCreateValidatorProposal{
					MsgCreateValidator: msg.(stake.MsgCreateValidator),
					ProposalId:         proposalId,
				}
			}

			if viper.GetBool(FlagGenesisFormat) || cliCtx.GenerateOnly {
				//Enable offline mode
				viper.Set(FlagOffline, true)
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			// build and sign the transaction, then broadcast to Tendermint
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().Int64(FlagProposalID, -1, "id of the CreateValidator proposal")
	cmd.Flags().Int64(FlagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().AddFlagSet(fsPk)
	cmd.Flags().AddFlagSet(fsAmount)
	cmd.Flags().String(FlagDeposit, "", "deposit token amount")
	cmd.Flags().AddFlagSet(fsDescriptionCreate)
	cmd.Flags().AddFlagSet(fsCommissionCreate)
	cmd.Flags().AddFlagSet(fsDelegator)
	cmd.Flags().Bool(FlagGenesisFormat, false, "Export the transaction in gen-tx format; it implies --generate-only")
	cmd.Flags().String(FlagIP, "", fmt.Sprintf("Node's public IP. It takes effect only when used in combination with --%s", FlagGenesisFormat))
	cmd.Flags().String(FlagNodeID, "", "Node's ID")
	cmd.MarkFlagRequired(client.FlagFrom)

	return cmd
}

// GetCmdEditValidator implements the create edit validator command.
func GetCmdRemoveValidator(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-validator",
		Short: "remove validator",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			launcher, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			validatorAddr, err := sdk.ValAddressFromBech32(viper.GetString(FlagAddressValidator))
			if err != nil {
				return err
			}
			validatorConsAddr, err := sdk.ConsAddressFromBech32(viper.GetString(FlagConsAddrValidator))
			if err != nil {
				return err
			}
			proposalId := viper.GetInt64(FlagProposalID)

			var msg sdk.Msg
			msg = stake.NewMsgRemoveValidator(launcher, validatorAddr, validatorConsAddr, proposalId)
			if proposalId == 0 {
				depositStr := viper.GetString(FlagDeposit)
				if depositStr == "" {
					return fmt.Errorf("must specify deposit amount when proposalId is zero using --deposit")
				}
				deposit, err := sdk.ParseCoin(depositStr)
				if err != nil {
					return err
				}
				title := "remove validator"
				description, err := json.Marshal(msg)
				if err != nil {
					return err
				}

				votingPeriodInSeconds := viper.GetInt64(FlagVotingPeriod)
				if votingPeriodInSeconds <= 0 {
					return errors.New("voting period should be positive")
				}
				votingPeriod := time.Duration(votingPeriodInSeconds) * time.Second
				if votingPeriod > gov.MaxVotingPeriod {
					return errors.New(fmt.Sprintf("voting period should be less than %d seconds", gov.MaxVotingPeriod/time.Second))
				}

				msg = gov.NewMsgSubmitProposal(title, string(description),
					gov.ProposalTypeRemoveValidator, launcher, sdk.Coins{deposit}, votingPeriod)
			}

			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().Int64(FlagProposalID, 0, "id of the remove validator proposal")
	cmd.Flags().Int64(FlagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().String(FlagAddressValidator, "", "validator address")
	cmd.Flags().String(FlagConsAddrValidator, "", "validator consensus address")
	cmd.Flags().String(FlagDeposit, "", "deposit token amount")

	return cmd
}

// GetCmdEditValidator implements the create edit validator command.
func GetCmdEditValidator(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-validator",
		Short: "edit and existing validator account",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			valAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			description := stake.Description{
				Moniker:  viper.GetString(FlagMoniker),
				Identity: viper.GetString(FlagIdentity),
				Website:  viper.GetString(FlagWebsite),
				Details:  viper.GetString(FlagDetails),
			}

			var newRate *sdk.Dec

			commissionRate := viper.GetString(FlagCommissionRate)
			if commissionRate != "" {
				rate, err := sdk.NewDecFromStr(commissionRate)
				if err != nil {
					return fmt.Errorf("invalid new commission rate: %v", err)
				}

				newRate = &rate
			}

			msg := stake.NewMsgEditValidator(sdk.ValAddress(valAddr), description, newRate)
			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(fsDescriptionEdit)
	cmd.Flags().AddFlagSet(fsCommissionUpdate)

	return cmd
}

// GetCmdDelegate implements the delegate command.
func GetCmdDelegate(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate",
		Short: "delegate liquid tokens to a validator",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			amount, err := getAmount()
			if err != nil {
				return err
			}

			delAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			valAddr, err := sdk.ValAddressFromBech32(viper.GetString(FlagAddressValidator))
			if err != nil {
				return err
			}

			msg := stake.NewMsgDelegate(delAddr, valAddr, amount)
			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(fsAmount)
	cmd.Flags().AddFlagSet(fsValidator)
	return cmd
}

// GetCmdRedelegate implements the redelegate validator command.
func GetCmdRedelegate(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redelegate",
		Short: "redelegate illiquid tokens from one validator to another",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			var err error

			delAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			valSrcAddr, err := sdk.ValAddressFromBech32(viper.GetString(FlagAddressValidatorSrc))
			if err != nil {
				return err
			}

			valDstAddr, err := sdk.ValAddressFromBech32(viper.GetString(FlagAddressValidatorDst))
			if err != nil {
				return err
			}

			// get the shares amount
			sharesAmountStr := viper.GetString(FlagSharesAmount)
			sharesPercentStr := viper.GetString(FlagSharesPercent)
			sharesAmount, err := getShares(
				storeName, cdc, sharesAmountStr, sharesPercentStr,
				delAddr, valSrcAddr,
			)
			if err != nil {
				return err
			}

			msg := stake.NewMsgBeginRedelegate(delAddr, valSrcAddr, valDstAddr, sharesAmount)
			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(fsShares)
	cmd.Flags().AddFlagSet(fsRedelegation)
	return cmd
}

// GetCmdUnbond implements the unbond validator command.
func GetCmdUnbond(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbond",
		Short: "unbond shares from a validator",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			delAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			valAddr, err := sdk.ValAddressFromBech32(viper.GetString(FlagAddressValidator))
			if err != nil {
				return err
			}

			// get the shares amount
			sharesAmountStr := viper.GetString(FlagSharesAmount)
			sharesPercentStr := viper.GetString(FlagSharesPercent)
			sharesAmount, err := getShares(
				storeName, cdc, sharesAmountStr, sharesPercentStr,
				delAddr, valAddr,
			)
			if err != nil {
				return err
			}

			msg := stake.NewMsgBeginUnbonding(delAddr, valAddr, sharesAmount)
			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(fsShares)
	cmd.Flags().AddFlagSet(fsValidator)
	return cmd
}

func getAmount() (sdk.Coin, error) {
	amountStr := viper.GetString(FlagAmount)
	if amountStr == "" {
		return sdk.Coin{}, fmt.Errorf("%s is required", FlagAmount)
	}
	amount, err := sdk.ParseCoin(amountStr)
	if err != nil {
		return sdk.Coin{}, err
	}

	return amount, nil
}
