package commands

import (

	"github.com/spf13/cobra"
)

var listTokensCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tokens",
	Long: "Return a list of all tokens",
	//RunE: runListCmd,
}

// func runListCmd(cmd *cobra.Command, args []string) error {
// 	ctx := context.NewCoreContextFromViper()
//
// 	res, err := ctx.Query(key, c.storeName)
// 	if err != nil {
// 		return err
// 	}
// 	return err
// }