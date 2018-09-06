package types

import (
	abci "github.com/tendermint/tendermint/abci/types"
)

// AbciQueryHandler represents an abci query handler, registered by a plugin's InitPlugin.
type AbciQueryHandler func(app ChainApp, req abci.RequestQuery, path []string) (res *abci.ResponseQuery)
