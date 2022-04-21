package cmd

import (
	"github.com/spf13/cobra"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/server"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:    "server",
	Short:  "bootstrap and relay server helps node (dynamic ip) discovery and NAT traversal",
	Long:   "bootstrap and relay server helps node (dynamic ip) discovery and NAT traversal",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		server.NewTssBootstrapServer(common.TssCfg.Home, common.TssCfg.P2PConfig)
		select {}
	},
}
