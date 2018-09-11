package types

import (
	"encoding/json"

	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/BiJie/BinanceChain/wire"
)

// ChainApp represents the main ABCI application
type ChainApp interface {
	GetCodec() *wire.Codec
	GetContextForCheckState() Context
	Query(req abci.RequestQuery) (res abci.ResponseQuery)
	RegisterQueryHandler(prefix string, handler AbciQueryHandler)
	ExportAppStateAndValidators() (appState json.RawMessage, validators []tmtypes.GenesisValidator, err error)
	EndBlocker(ctx Context, req abci.RequestEndBlock) abci.ResponseEndBlock
}
