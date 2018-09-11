package tokens

import (
	app "github.com/BiJie/BinanceChain/common/types"
)

const abciQueryPrefix = "tokens"

// InitPlugin initializes the plugin.
func InitPlugin(appp app.ChainApp, mapper Mapper) {
	handler := createQueryHandler(mapper)
	appp.RegisterQueryHandler(abciQueryPrefix, handler)
}

func createQueryHandler(mapper Mapper) app.AbciQueryHandler {
	return createAbciQueryHandler(mapper)
}
