package utils

import (
	"github.com/spf13/cobra"
)

func Commands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "utils",
		Short: "Utilities for interacting with BNB Beacon Chain",
	}
	cmd.AddCommand(
		genConsensusKeyCommand(),
	)
	return cmd
}
