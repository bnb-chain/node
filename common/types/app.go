package types

import (
	"encoding/json"

	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/wire"
)

const (
	TxHashKey = "txHash" // we pass txHash of current handling message via context so that we can publish it as metadata of Msg
)

// ChainApp represents the main ABCI application
type ChainApp interface {
	GetCodec() *wire.Codec
	GetRouter() baseapp.Router
	GetContextForCheckState() sdk.Context
	Query(req abci.RequestQuery) (res abci.ResponseQuery)
	RegisterQueryHandler(prefix string, handler AbciQueryHandler)
	ExportAppStateAndValidators() (appState json.RawMessage, validators []tmtypes.GenesisValidator, err error)
	EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock
}
