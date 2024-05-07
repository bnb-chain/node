// nolint
package version

import "fmt"

var (
	// GitCommit is the current HEAD set using ldflags.
	GitCommit         string
	CosmosRelease     string
	TendermintRelease string

	Version string
)

const NodeVersion = "v0.10.21"

func init() {
	Version = fmt.Sprintf("BNB Beacon Chain Release: %s;", NodeVersion)
	if GitCommit != "" {
		Version += fmt.Sprintf("BNB Beacon Chain Commit: %s;", GitCommit)
	}
	if CosmosRelease != "" {
		Version += fmt.Sprintf(" Cosmos Release: %s;", CosmosRelease)
	}
	if TendermintRelease != "" {
		Version += fmt.Sprintf(" Tendermint Release: %s;", TendermintRelease)
	}
}
