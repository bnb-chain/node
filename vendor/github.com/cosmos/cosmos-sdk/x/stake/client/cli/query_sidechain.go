package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strconv"

	"github.com/tendermint/tendermint/libs/cli"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

func GetCmdQuerySideValidator(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-validator [operator-addr]",
		Short: "Query a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := sdk.ValAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}
			key := append(sideChainStorePrefix, stake.GetValidatorKey(addr)...)

			res, err := cliCtx.QueryStore(key, storeName)
			if err != nil {
				return err
			} else if len(res) == 0 {
				return fmt.Errorf("No validator found with address %s ", args[0])
			}

			validator, err := types.UnmarshalValidator(cdc, res)
			if err != nil {
				return err
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				human, err := validator.HumanReadableString()
				if err != nil {
					return err
				}
				fmt.Println(human)

			case "json":
				// parse out the validator
				output, err := codec.MarshalJSONIndent(cdc, validator)
				if err != nil {
					return err
				}

				fmt.Println(string(output))
			}

			// TODO: output with proofs / machine parseable etc.
			return nil
		},
	}

	cmd.Flags().AddFlagSet(fsSideChainId)

	return cmd
}

func GetCmdQuerySideChainDelegation(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-delegation [delegator-addr] [operator-addr]",
		Short: "Query a delegation based on address and validator address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			delAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			valAddr, err := sdk.ValAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			sideChainId, _, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			params := stake.QueryBondsParams{
				DelegatorAddr: delAddr,
				ValidatorAddr: valAddr,
				BaseParams:    stake.NewBaseParams(sideChainId),
			}

			bz, err := json.Marshal(params)
			if err != nil {
				return err
			}

			response, err := cliCtx.QueryWithData("custom/stake/delegation", bz)
			if err != nil {
				return err
			} else if len(response) == 0 {
				return fmt.Errorf("No delegation found ")
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				var delResponse types.DelegationResponse
				if err := cdc.UnmarshalJSON(response, &delResponse); err != nil {
					return err
				}
				resp, err := delResponse.HumanReadableString()
				if err != nil {
					return err
				}

				fmt.Println(resp)
			case "json":
				fmt.Println(string(response))
				return nil
			}

			return nil
		},
	}

	cmd.Flags().AddFlagSet(fsValidator)
	cmd.Flags().AddFlagSet(fsDelegator)
	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

func GetCmdQuerySideChainDelegations(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-delegations [delegator-addr]",
		Short: "Query all delegations made from one delegator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			delegatorAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			sideChainId, _, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			params := stake.QueryDelegatorParams{
				DelegatorAddr: delegatorAddr,
				BaseParams:    stake.NewBaseParams(sideChainId),
			}

			bz, err := json.Marshal(params)
			if err != nil {
				return err
			}

			response, err := cliCtx.QueryWithData("custom/stake/delegatorDelegations", bz)
			if err != nil {
				return err
			} else if len(response) == 0 {
				return fmt.Errorf("No delegation found with delegator-addr %s ", args[0])
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				var delegationResponses []types.DelegationResponse
				if err := cdc.UnmarshalJSON(response, &delegationResponses); err != nil {
					return err
				}
				for _, dr := range delegationResponses {
					resp, err := dr.HumanReadableString()
					if err != nil {
						return err
					}
					fmt.Println(resp)
					fmt.Println()
				}
			case "json":
				fmt.Println(string(response))
				return nil
			}

			// TODO: output with proofs / machine parseable etc.
			return nil
		},
	}

	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

// GetCmdQueryUnbondingDelegation implements the command to query a single
// unbonding-delegation record.
func GetCmdQuerySideChainUnbondingDelegation(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-unbonding-delegation [delegator-addr] [operator-addr]",
		Short: "Query an unbonding-delegation record based on delegator and validator address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			delAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			valAddr, err := sdk.ValAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			ubdKey := stake.GetUBDKey(delAddr, valAddr)
			key := append(sideChainStorePrefix, ubdKey...)
			res, err := cliCtx.QueryStore(key, storeName)
			if err != nil {
				return err
			} else if len(res) == 0 {
				return fmt.Errorf("No unbonding-delegation found ")
			}

			// parse out the unbonding delegation
			ubd := types.MustUnmarshalUBD(cdc, ubdKey, res)

			switch viper.Get(cli.OutputFlag) {
			case "text":
				resp, err := ubd.HumanReadableString()
				if err != nil {
					return err
				}

				fmt.Println(resp)
			case "json":
				output, err := codec.MarshalJSONIndent(cdc, ubd)
				if err != nil {
					return err
				}

				fmt.Println(string(output))
				return nil
			}

			return nil
		},
	}

	cmd.Flags().AddFlagSet(fsValidator)
	cmd.Flags().AddFlagSet(fsDelegator)
	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

// GetCmdQueryUnbondingDelegations implements the command to query all the
// unbonding-delegation records for a delegator.
func GetCmdQuerySideChainUnbondingDelegations(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-unbonding-delegations [delegator-addr]",
		Short: "Query all unbonding-delegations records for one delegator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			delegatorAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			key := append(sideChainStorePrefix, stake.GetUBDsKey(delegatorAddr)...)

			resKVs, err := cliCtx.QuerySubspace(key, storeName)
			if err != nil {
				return err
			} else if len(resKVs) == 0 {
				return fmt.Errorf("No unbonding-delegations found with delegator %s ", args[0])
			}

			// parse out the validators
			var ubds []stake.UnbondingDelegation
			for _, kv := range resKVs {
				k := kv.Key[len(sideChainStorePrefix):] // remove side chain prefix bytes
				ubd := types.MustUnmarshalUBD(cdc, k, kv.Value)
				ubds = append(ubds, ubd)
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				for _, ubd := range ubds {
					resp, err := ubd.HumanReadableString()
					if err != nil {
						return err
					}
					fmt.Println(resp)
					fmt.Println()
				}
			case "json":
				output, err := codec.MarshalJSONIndent(cdc, ubds)
				if err != nil {
					return err
				}

				fmt.Println(string(output))
				return nil
			}

			// TODO: output with proofs / machine parseable etc.
			return nil
		},
	}

	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

// GetCmdQueryRedelegation implements the command to query a single
// redelegation record.
func GetCmdQuerySideChainRedelegation(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-redelegation [delegator-addr] [src-operator-addr] [dst-operator-addr]",
		Short: "Query a redelegation record based on delegator and a source and destination validator address",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			delAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			valSrcAddr, err := sdk.ValAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			valDstAddr, err := sdk.ValAddressFromBech32(args[2])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			redKey := stake.GetREDKey(delAddr, valSrcAddr, valDstAddr)
			key := append(sideChainStorePrefix, redKey...)
			res, err := cliCtx.QueryStore(key, storeName)
			if err != nil {
				return err
			} else if len(res) == 0 {
				return fmt.Errorf("No redelegation found ")
			}

			// parse out the unbonding delegation
			red := types.MustUnmarshalRED(cdc, redKey, res)

			switch viper.Get(cli.OutputFlag) {
			case "text":
				resp, err := red.HumanReadableString()
				if err != nil {
					return err
				}

				fmt.Println(resp)
			case "json":
				output, err := codec.MarshalJSONIndent(cdc, red)
				if err != nil {
					return err
				}

				fmt.Println(string(output))
				return nil
			}

			return nil
		},
	}

	cmd.Flags().AddFlagSet(fsRedelegation)
	cmd.Flags().AddFlagSet(fsDelegator)
	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

// GetCmdQueryRedelegations implements the command to query all the
// redelegation records for a delegator.
func GetCmdQuerySideChainRedelegations(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-redelegations [delegator-addr]",
		Short: "Query all redelegations records for one delegator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			delegatorAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			key := append(sideChainStorePrefix, stake.GetREDsKey(delegatorAddr)...)
			resKVs, err := cliCtx.QuerySubspace(key, storeName)
			if err != nil {
				return err
			} else if len(resKVs) == 0 {
				return fmt.Errorf("No redelegations found ")
			}

			// parse out the validators
			var reds []stake.Redelegation
			for _, kv := range resKVs {
				k := kv.Key[len(sideChainStorePrefix):]
				red := types.MustUnmarshalRED(cdc, k, kv.Value)
				reds = append(reds, red)
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				for _, red := range reds {
					resp, err := red.HumanReadableString()
					if err != nil {
						return err
					}
					fmt.Println(resp)
					fmt.Println()
				}
			case "json":
				output, err := codec.MarshalJSONIndent(cdc, reds)
				if err != nil {
					return err
				}

				fmt.Println(string(output))
				return nil
			}

			// TODO: output with proofs / machine parseable etc.
			return nil
		},
	}

	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

func GetCmdQuerySideChainUnbondingDelegationsByValidator(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-val-unbonding-delegations [operator-addr]",
		Short: "Query all unbonding-delegations records for one validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			valAddr, err := sdk.ValAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			sideChainId, _, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			params := stake.QueryValidatorParams{
				ValidatorAddr: valAddr,
				BaseParams:    stake.NewBaseParams(sideChainId),
			}

			bz, err := json.Marshal(params)
			if err != nil {
				return err
			}

			response, err := cliCtx.QueryWithData("custom/stake/validatorUnbondingDelegations", bz)
			if err != nil {
				return err
			} else if len(response) == 0 {
				return fmt.Errorf("No unbounding delegations found with operator address %s ", args[0])
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				var ubds []types.UnbondingDelegation
				if err = cdc.UnmarshalJSON(response, &ubds); err != nil {
					return err
				}
				for _, ubd := range ubds {
					resp, err := ubd.HumanReadableString()
					if err != nil {
						return err
					}
					fmt.Println(resp)
					fmt.Println()
				}
			case "json":
				fmt.Println(string(response))
				return nil
			}
			return nil
		},
	}
	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

func GetCmdQuerySideChainReDelegationsByValidator(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-val-redelegations [operator-addr]",
		Short: "Query all redelegations records for one validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			valAddr, err := sdk.ValAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			sideChainId, _, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}
			params := stake.QueryValidatorParams{
				ValidatorAddr: valAddr,
				BaseParams:    stake.NewBaseParams(sideChainId),
			}

			bz, err := json.Marshal(params)
			if err != nil {
				return err
			}

			response, err := cliCtx.QueryWithData("custom/stake/validatorRedelegations", bz)
			if err != nil {
				return err
			} else if len(response) == 0 {
				return fmt.Errorf("No re-delegations found with operator address %s ", args[0])
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				var reds []types.Redelegation
				if err = cdc.UnmarshalJSON(response, &reds); err != nil {
					return err
				}
				for _, red := range reds {
					resp, err := red.HumanReadableString()
					if err != nil {
						return err
					}
					fmt.Println(resp)
					fmt.Println()
				}
			case "json":
				fmt.Println(string(response))
				return nil
			}
			return nil
		},
	}
	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

func GetCmdQuerySideChainPool(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-pool",
		Short: "Query the current staking pool values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			key := append(sideChainStorePrefix, stake.PoolKey...)
			res, err := cliCtx.QueryStore(key, storeName)
			if err != nil {
				return err
			} else if len(res) == 0 {
				return fmt.Errorf("No pool found ")
			}

			pool := types.MustUnmarshalPool(cdc, res)

			switch viper.Get(cli.OutputFlag) {
			case "text":
				human := pool.HumanReadableString()

				fmt.Println(human)

			case "json":
				// parse out the pool
				output, err := codec.MarshalJSONIndent(cdc, pool)
				if err != nil {
					return err
				}

				fmt.Println(string(output))
			}
			return nil
		},
	}

	cmd.Flags().AddFlagSet(fsSideChainId)
	return cmd
}

func GetCmdQuerySideChainTopValidators(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-top-validators",
		Short: "Query top N validators at current time",
		RunE: func(cmd *cobra.Command, args []string) error {
			topS := viper.GetString("top")
			var top int
			if len(topS) != 0 {
				topI, err := strconv.Atoi(topS)
				if err != nil {
					return err
				}
				if topI > 50 || topI < 1 {
					return errors.New("top must be between 1 and 50")
				}
				top = topI
			}
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			sideChainId, _, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}
			params := stake.QueryTopValidatorsParams{
				BaseParams: stake.NewBaseParams(sideChainId),
				Top:        top,
			}

			bz, err := json.Marshal(params)
			if err != nil {
				return err
			}

			response, err := cliCtx.QueryWithData("custom/stake/topValidators", bz)
			if err != nil {
				return err
			} else if len(response) == 0 {
				return fmt.Errorf("No validators found ")
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				var vals []types.Validator
				if err = cdc.UnmarshalJSON(response, &vals); err != nil {
					return err
				}
				for _, val := range vals {
					resp, err := val.HumanReadableString()
					if err != nil {
						return err
					}
					fmt.Println(resp)
				}
			case "json":
				fmt.Println(string(response))
				return nil
			}
			return nil
		},
	}
	cmd.Flags().AddFlagSet(fsSideChainId)
	cmd.Flags().String("top", "", "")
	return cmd
}

func GetCmdQuerySideAllValidatorsCount(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-validators-count",
		Short: "Query all validators count",
		RunE: func(cmd *cobra.Command, args []string) error {
			jailInvolved := viper.GetBool("jail-involved")

			cliCtx := context.NewCLIContext().WithCodec(cdc)

			sideChainId, _, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			params := stake.BaseParams{
				SideChainId: sideChainId,
			}

			bz, err := json.Marshal(params)
			if err != nil {
				return err
			}

			path := "custom/stake/allUnJailValidatorsCount"
			if jailInvolved {
				path = "custom/stake/allValidatorsCount"
			}
			response, err := cliCtx.QueryWithData(path, bz)
			if err != nil {
				return err
			} else if len(response) == 0 {
				response = []byte{}
			}

			fmt.Println(string(response))

			return nil
		},
	}
	cmd.Flags().AddFlagSet(fsSideChainId)
	cmd.Flags().Bool("jail-involved", false, "")
	return cmd
}

func getSideChainConfig(cliCtx context.CLIContext) (sideChainId string, prefix []byte, error error) {
	sideChainId, error = getSideChainId()
	if error != nil {
		return sideChainId, nil, error
	}

	prefix, error = cliCtx.QueryStore(sidechain.GetSideChainStorePrefixKey(sideChainId), scStoreKey)
	if error != nil {
		return sideChainId, nil, error
	} else if len(prefix) == 0 {
		return sideChainId, nil, fmt.Errorf("Invalid side-chain-id %s ", sideChainId)
	}
	return sideChainId, prefix, error
}
