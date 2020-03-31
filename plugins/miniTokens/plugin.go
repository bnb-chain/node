package minitokens

import (
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	app "github.com/binance-chain/node/common/types"
)

const abciQueryPrefix = "minitokens"

// InitPlugin initializes the plugin.
func InitPlugin(
	appp app.ChainApp, mapper MiniTokenMapper, accKeeper auth.AccountKeeper, coinKeeper bank.Keeper) {
	// add msg handlers
	for route, handler := range Routes(mapper, accKeeper, coinKeeper) {
		appp.GetRouter().AddRoute(route, handler)
	}

	// add abci handlers
	handler := createQueryHandler(mapper)
	appp.RegisterQueryHandler(abciQueryPrefix, handler)
}

func createQueryHandler(mapper MiniTokenMapper) app.AbciQueryHandler {
	return createAbciQueryHandler(mapper)
}
