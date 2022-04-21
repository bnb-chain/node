package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetCmdCreateSideChainValidator(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bsc-create-validator",
		Short: "create new validator for side chain initialized with a self-delegation to it",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
		cliCtx := context.NewCLIContext().
			WithCodec(cdc).
			WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

		amountStr := viper.GetString(FlagAmount)
		if amountStr == "" {
			return fmt.Errorf("Must specify amount to stake using --amount")
		}
		amount, err := sdk.ParseCoin(amountStr)
		if err != nil {
			return err
		}

		valAddr, err := cliCtx.GetFromAddress()
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

		sideChainId, sideConsAddr, sideFeeAddr, err := getSideChainInfo(true, true)
		if err != nil {
			return err
		}

		var msg sdk.Msg
		if viper.GetString(FlagAddressDelegator) != "" {
			delAddr, err := sdk.AccAddressFromBech32(viper.GetString(FlagAddressDelegator))
			if err != nil {
				return err
			}

			msg = stake.NewMsgCreateSideChainValidatorOnBehalfOf(delAddr, sdk.ValAddress(valAddr), amount, description,
				commissionMsg, sideChainId, sideConsAddr, sideFeeAddr)
		} else {
			msg = stake.NewMsgCreateSideChainValidator(
				sdk.ValAddress(valAddr), amount, description, commissionMsg, sideChainId, sideConsAddr, sideFeeAddr)
		}

		return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
	}

	cmd.Flags().AddFlagSet(fsAmount)
	cmd.Flags().AddFlagSet(fsDescriptionCreate)
	cmd.Flags().AddFlagSet(fsCommissionCreate)
	cmd.Flags().AddFlagSet(fsDelegator)
	cmd.Flags().AddFlagSet(fsSideChainFull)
	cmd.MarkFlagRequired(client.FlagFrom)
	return cmd
}

func GetCmdEditSideChainValidator(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bsc-edit-validator",
		Short: "edit an existing side chain validator",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
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

		sideChainId, _, sideFeeAddr, err := getSideChainInfo(false, false)
		if err != nil {
			return err
		}
		msg := stake.NewMsgEditSideChainValidator(sideChainId, sdk.ValAddress(valAddr), description, newRate, sideFeeAddr)
		return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
	}

	cmd.Flags().AddFlagSet(fsDescriptionEdit)
	cmd.Flags().AddFlagSet(fsCommissionUpdate)
	cmd.Flags().AddFlagSet(fsSideChainEdit)
	return cmd
}

func GetCmdSideChainDelegate(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bsc-delegate",
		Short: "delegate liquid tokens to a side chain validator",
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

			valAddr, err := getValidatorAddr(FlagAddressValidator)
			if err != nil {
				return err
			}

			sideChainId, err := getSideChainId()
			if err != nil {
				return err
			}

			msg := stake.NewMsgSideChainDelegate(sideChainId, delAddr, valAddr, amount)
			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(fsAmount)
	cmd.Flags().AddFlagSet(fsValidator)
	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

func GetCmdSideChainRedelegate(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bsc-redelegate",
		Short: "Redelegate illiquid tokens from one validator to another",

		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			delAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			valSrcAddr, err := getValidatorAddr(FlagAddressValidatorSrc)
			if err != nil {
				return err
			}

			valDstAddr, err := getValidatorAddr(FlagAddressValidatorDst)
			if err != nil {
				return err
			}

			amount, err := getAmount()
			if err != nil {
				return err
			}

			sideChainId, err := getSideChainId()
			if err != nil {
				return err
			}

			msg := stake.NewMsgSideChainRedelegate(sideChainId, delAddr, valSrcAddr, valDstAddr, amount)
			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(fsAmount)
	cmd.Flags().AddFlagSet(fsRedelegation)
	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

func GetCmdSideChainUnbond(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bsc-unbond",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			delAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}
			valAddr, err := getValidatorAddr(FlagAddressValidator)
			if err != nil {
				return err
			}

			amount, err := getAmount()
			if err != nil {
				return err
			}

			sideChainId, err := getSideChainId()
			if err != nil {
				return err
			}

			msg := stake.NewMsgSideChainUndelegate(sideChainId, delAddr, valAddr, amount)
			return utils.GenerateOrBroadcastMsgs(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().AddFlagSet(fsAmount)
	cmd.Flags().AddFlagSet(fsValidator)
	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

func getSideChainId() (sideChainId string, err error) {
	sideChainId = viper.GetString(FlagSideChainId)
	if len(sideChainId) == 0 {
		err = fmt.Errorf("%s is required", FlagSideChainId)
	}
	return
}

func getSideChainInfo(requireConsAddr, requireFeeAddr bool) (sideChainId string, sideConsAddr, sideFeeAddr []byte, err error) {
	sideChainId, err = getSideChainId()
	if err != nil {
		return
	}

	sideConsAddrStr := viper.GetString(FlagSideConsAddr)
	if len(sideConsAddrStr) == 0 {
		if requireConsAddr {
			err = fmt.Errorf("%s is required", FlagSideConsAddr)
			return
		}
	} else {
		sideConsAddr, err = sdk.HexDecode(sideConsAddrStr)
		if err != nil {
			return
		}
	}

	sideFeeAddrStr := viper.GetString(FlagSideFeeAddr)
	if len(sideFeeAddrStr) == 0 {
		if requireFeeAddr {
			err = fmt.Errorf("%s is required", FlagSideFeeAddr)
			return
		}
	} else {
		sideFeeAddr, err = sdk.HexDecode(sideFeeAddrStr)
		if err != nil {
			return
		}
	}
	return
}

func getValidatorAddr(flagName string) (valAddr sdk.ValAddress, err error) {
	valAddrStr := viper.GetString(flagName)
	if len(valAddrStr) == 0 {
		err = fmt.Errorf("%s is required", flagName)
		return
	}
	return sdk.ValAddressFromBech32(valAddrStr)
}
