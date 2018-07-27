package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/spf13/cobra"

	"github.com/BiJie/BinanceChain/common/types"
)

// GetAccountCmd returns a query account that will display the
// state of the account at a given address
func GetAccountCmd(storeName string, cdc *wire.Codec, decoder auth.AccountDecoder) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [address]",
		Short: "Query account balance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// find the key to look up the account
			addr := args[0]

			key, err := getAccAddress(addr)
			if err != nil {
				return nil
			}

			// perform query
			ctx := context.NewCoreContextFromViper()
			res, err := ctx.QueryStore(auth.AddressStoreKey(key), storeName)
			if err != nil {
				return err
			}

			// Check if account was found
			if res == nil {
				return errors.New("No account was found with address " + addr)
			}

			// decode the value
			account, err := decoder(res)
			if err != nil {
				return err
			}

			fmt.Println(account)
			return nil
		},
	}

	return cmd
}

func getAccAddress(bechAddr string) (sdk.AccAddress, error) {
	var prefix string
	if strings.HasPrefix(bechAddr, types.Bech32PrefixAccAddr) {
		prefix = types.Bech32PrefixAccAddr
	} else if strings.HasPrefix(bechAddr, sdk.Bech32PrefixAccAddr) {
		prefix = sdk.Bech32PrefixAccAddr
	} else {
		return nil, errors.New("unknown address")
	}

	bz, err := sdk.GetFromBech32(bechAddr, prefix)
	if err != nil {
		return nil, err
	}

	return sdk.AccAddress(bz), nil
}
