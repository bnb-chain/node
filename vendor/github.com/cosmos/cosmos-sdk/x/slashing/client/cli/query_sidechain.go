package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"
)

// GetCmdQuerySideChainSigningInfo implements the command to query signing info.
func GetCmdQuerySideChainSigningInfo(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-signing-info [validator-sideConsAddr]",
		Short: "Query a validator's signing information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sideConsAddr, err := sdk.HexDecode(args[0])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)

			if err != nil {
				return err
			}

			key := append(sideChainStorePrefix, slashing.GetValidatorSigningInfoKey(sdk.ConsAddress(sideConsAddr))...)

			res, err := cliCtx.QueryStore(key, storeName)
			if err != nil {
				return err
			} else if len(res) == 0 {
				return fmt.Errorf("No signing info found with sideConsAddr %s ", args[0])
			}

			signingInfo := new(slashing.ValidatorSigningInfo)
			cdc.MustUnmarshalBinaryLengthPrefixed(res, signingInfo)

			switch viper.Get(cli.OutputFlag) {

			case "text":
				human := signingInfo.HumanReadableString()
				fmt.Println(human)

			case "json":
				// parse out the signing info
				output, err := codec.MarshalJSONIndent(cdc, signingInfo)
				if err != nil {
					return err
				}
				fmt.Println(string(output))
			}

			return nil
		},
	}
	cmd.Flags().String(FlagSideChainId, "", "chain-id of the side chain the validator belongs to")
	return cmd
}

func GetCmdQueryAllSideSlashRecords(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-all-slash-histories",
		Short: "Query all slash histories on side chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			key := append(sideChainStorePrefix, slashing.SlashRecordKey...)
			resKVs, err := cliCtx.QuerySubspace(key, storeName)
			if err != nil {
				return err
			} else if len(resKVs) == 0 {
				return fmt.Errorf("No slash histories found ")
			}

			var slashRecords []slashing.SlashRecord
			for _, kv := range resKVs {
				k := kv.Key[len(sideChainStorePrefix):] // remove side chain prefix bytes
				sr := slashing.MustUnmarshalSlashRecord(cdc, k, kv.Value)
				slashRecords = append(slashRecords, sr)
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				for _, sr := range slashRecords {
					resp, err := sr.HumanReadableString()
					if err != nil {
						return err
					}
					fmt.Println(resp)
				}
			case "json":
				output, err := codec.MarshalJSONIndent(cdc, slashRecords)
				if err != nil {
					return err
				}
				fmt.Println(string(output))
				return nil
			}

			return nil
		},
	}

	cmd.Flags().String(FlagSideChainId, "", "chain-id of the side chain the validator belongs to")
	return cmd
}

// GetCmdQuerySideChainSlashRecord implements the command to query slash Record
func GetCmdQuerySideChainSlashRecord(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-slash-history [validator-sideConsAddr]",
		Short: "Query a validator's slash history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sideConsAddr, err := sdk.HexDecode(args[0])
			if err != nil {
				return err
			}

			infractionType := viper.GetString(FlagInfractionType)
			resType, err := convertInfractionType(infractionType)
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			_, sideChainStorePrefix, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}
			height := viper.GetUint64(FlagInfractionHeight)

			key := append(sideChainStorePrefix, slashing.GetSlashRecordKey(sideConsAddr, resType, height)...)
			res, err := cliCtx.QueryStore(key, storeName)
			if err != nil {
				return err
			} else if len(res) == 0 {
				return fmt.Errorf("no slash history found with sideConsAddr %s ", args[0])
			}

			slashRecord := new(slashing.SlashRecord)
			cdc.MustUnmarshalBinaryLengthPrefixed(res, slashRecord)

			switch viper.Get(cli.OutputFlag) {

			case "text":
				human, err := slashRecord.HumanReadableString()
				if err != nil {
					return err
				}
				fmt.Println(human)

			case "json":
				// parse out the signing info
				output, err := codec.MarshalJSONIndent(cdc, slashRecord)
				if err != nil {
					return err
				}
				fmt.Println(string(output))
			}
			return nil
		},
	}
	cmd.Flags().String(FlagInfractionType, "", "infraction type, 'DoubleSign;Downtime'")
	cmd.Flags().Int64(FlagInfractionHeight, 0, "infraction height")
	cmd.Flags().String(FlagSideChainId, "", "chain-id of the side chain the validator belongs to")
	cmd.MarkFlagRequired(FlagInfractionType)
	cmd.MarkFlagRequired(FlagInfractionHeight)
	return cmd
}

// GetCmdQuerySideChainSlashRecords implements the command to query slash Records
func GetCmdQuerySideChainSlashRecords(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "side-slash-histories [validator-sideConsAddr]",
		Short: "Query a validator's slash histories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sideConsAddr, err := sdk.HexDecode(args[0])
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			sideChainId, _, err := getSideChainConfig(cliCtx)
			if err != nil {
				return err
			}

			infractionType := viper.GetString(FlagInfractionType)
			var response []byte
			if len(infractionType) == 0 {
				params := slashing.QueryConsAddrParams{
					BaseParams: slashing.NewBaseParams(sideChainId),
					ConsAddr:   sideConsAddr,
				}

				bz, err := json.Marshal(params)
				if err != nil {
					return err
				}
				response, err = cliCtx.QueryWithData("custom/slashing/consAddrSlashHistories", bz)
				if err != nil {
					return err
				}
			} else {
				params := slashing.QueryConsAddrTypeParams{
					BaseParams: slashing.NewBaseParams(sideChainId),
					ConsAddr:   sideConsAddr,
				}
				infractionType := viper.GetString(FlagInfractionType)
				resType, err := convertInfractionType(infractionType)
				if err != nil {
					return err
				}
				params.InfractionType = resType
				bz, err := json.Marshal(params)
				if err != nil {
					return err
				}
				response, err = cliCtx.QueryWithData("custom/slashing/consAddrTypeSlashHistories", bz)
				if err != nil {
					return err
				}
			}

			if len(response) == 0 {
				return fmt.Errorf("no slash history found with sideConsAddr = %s\n", args[0])
			}

			switch viper.Get(cli.OutputFlag) {
			case "text":
				var slashRecords []slashing.SlashRecord
				if err = cdc.UnmarshalJSON(response, &slashRecords); err != nil {
					return err
				}
				for _, sr := range slashRecords {
					resp, err := sr.HumanReadableString()
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

	cmd.Flags().String(FlagInfractionType, "", "infraction type, 'DoubleSign;Downtime'")
	cmd.Flags().String(FlagSideChainId, "", "chain-id of the side chain the validator belongs to")
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

func convertInfractionType(infractionTypeS string) (byte, error) {
	var res byte
	if infractionTypeS == "DoubleSign" {
		res = slashing.DoubleSign
	} else if infractionTypeS == "Downtime" {
		res = slashing.Downtime
	} else {
		return 0, errors.New("unknown infraction type")
	}
	return res, nil
}
