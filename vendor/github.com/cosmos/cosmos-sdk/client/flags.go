package client

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// nolint
const (
	FlagUseLedger      = "ledger"
	FlagUseTss         = "tss"
	FlagChainID        = "chain-id"
	FlagNode           = "node"
	FlagHeight         = "height"
	FlagTrustNode      = "trust-node"
	FlagFrom           = "from"
	FlagName           = "name"
	FlagAccountNumber  = "account-number"
	FlagSequence       = "sequence"
	FlagMemo           = "memo"
	FlagSource         = "source"
	FlagAsync          = "async"
	FlagJson           = "json"
	FlagPrintResponse  = "print-response"
	FlagDryRun         = "dry-run"
	FlagDry            = "dry"
	FlagOffline        = "offline"
	FlagGenerateOnly   = "generate-only"
	FlagIndentResponse = "indent"
)

// LineBreak can be included in a command list to provide a blank line
// to help with readability
var (
	LineBreak = &cobra.Command{Run: func(*cobra.Command, []string) {}}
)

// GetCommands adds common flags to query commands
func GetCommands(cmds ...*cobra.Command) []*cobra.Command {
	for _, c := range cmds {
		c.Flags().Bool(FlagIndentResponse, false, "Add indent to JSON response")
		c.Flags().Bool(FlagTrustNode, false, "Trust connected full node (don't verify proofs for responses)")
		c.Flags().Bool(FlagUseLedger, false, "Use a connected Ledger device")
		c.Flags().String(FlagChainID, "", "Chain ID of tendermint node")
		c.Flags().String(FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
		c.Flags().Int64(FlagHeight, 0, "block height to query, omit to get most recent provable block")
		viper.BindPFlag(FlagTrustNode, c.Flags().Lookup(FlagTrustNode))
		viper.BindPFlag(FlagUseLedger, c.Flags().Lookup(FlagUseLedger))
		viper.BindPFlag(FlagChainID, c.Flags().Lookup(FlagChainID))
		viper.BindPFlag(FlagNode, c.Flags().Lookup(FlagNode))
	}
	return cmds
}

// PostCommands adds common flags for commands to post tx
func PostCommands(cmds ...*cobra.Command) []*cobra.Command {
	for _, c := range cmds {
		c.Flags().Bool(FlagIndentResponse, false, "Add indent to JSON response")
		c.Flags().String(FlagFrom, "", "Name or address of private key with which to sign")
		c.Flags().Int64(FlagAccountNumber, 0, "AccountNumber number to sign the tx")
		c.Flags().Int64(FlagSequence, 0, "Sequence number to sign the tx")
		c.Flags().String(FlagMemo, "", "Memo to send along with transaction")
		c.Flags().Int64(FlagSource, 0, "Source of tx")
		c.Flags().String(FlagChainID, "", "Chain ID of tendermint node")
		c.Flags().String(FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
		c.Flags().Bool(FlagUseLedger, false, "Use a connected Ledger device")
		c.Flags().Bool(FlagUseTss, false, "Use a tss vault")
		c.Flags().Bool(FlagAsync, false, "broadcast transactions asynchronously")
		c.Flags().Bool(FlagJson, false, "return output in json format")
		c.Flags().Bool(FlagPrintResponse, true, "return tx response (only works with async = false)")
		c.Flags().Bool(FlagTrustNode, true, "Trust connected full node (don't verify proofs for responses)")
		c.Flags().Bool(FlagDryRun, false, "ignore the perform a simulation of a transaction, but don't broadcast it")
		c.Flags().Bool(FlagDry, false, "Generate and return the tx bytes (do not broadcast)")
		c.Flags().Bool(FlagOffline, false, "Offline mode. Do not query blockchain data")
		c.Flags().Bool(FlagGenerateOnly, false, "build an unsigned transaction and write it to STDOUT")
		viper.BindPFlag(FlagTrustNode, c.Flags().Lookup(FlagTrustNode))
		viper.BindPFlag(FlagUseLedger, c.Flags().Lookup(FlagUseLedger))
		viper.BindPFlag(FlagUseTss, c.Flags().Lookup(FlagUseTss))
		viper.BindPFlag(FlagChainID, c.Flags().Lookup(FlagChainID))
		viper.BindPFlag(FlagNode, c.Flags().Lookup(FlagNode))
	}
	return cmds
}
