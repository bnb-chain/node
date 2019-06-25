//nolint
package version

import "fmt"

var (
	// GitCommit is the current HEAD set using ldflags.
	GitCommit         string
	CosmosRelease     string
	TendermintRelease string

	Version string
)

const NodeVersion = "0.6.0"

func init() {
	Version = fmt.Sprintf("Binance Chain Release: %s;", NodeVersion)
	if GitCommit != "" {
		Version += fmt.Sprintf("Binance Chain Commit: %s;", GitCommit)
	}
	if CosmosRelease != "" {
		Version += fmt.Sprintf(" Cosmos Release: %s;", CosmosRelease)
	}
	if TendermintRelease != "" {
		Version += fmt.Sprintf(" Tendermint Release: %s;", TendermintRelease)
	}
}
