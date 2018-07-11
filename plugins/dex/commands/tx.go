package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/wire"
)

// MakeOfferCmd -
func makeOfferCmd(cdc *wire.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "offer [whatever]",
		Short: "Make an offer (dex)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || len(args[0]) == 0 {
				return errors.New("You must provide a whatever")
			}
			return nil
		},
	}
}

// FillOfferCmd -
func fillOfferCmd(cdc *wire.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "fill [order id]",
		Short: "Fill an offer (dex)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || len(args[0]) == 0 {
				return errors.New("You must provide a whatever")
			}
			return nil
		},
	}
}

// CancelOfferCmd -
func cancelOfferCmd(cdc *wire.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel [order id]",
		Short: "Cancel an offer (dex)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || len(args[0]) == 0 {
				return errors.New("You must provide a whatever")
			}
			return nil
		},
	}
}
