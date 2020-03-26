package bridge

import (
	app "github.com/binance-chain/node/common/types"
)

func InitPlugin(appp app.ChainApp, keeper Keeper) {
	for route, handler := range Routes(keeper) {
		appp.GetRouter().AddRoute(route, handler)
	}
}