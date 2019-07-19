package account

import (
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/binance-chain/node/plugins/account/scripts"
	app "github.com/binance-chain/node/common/types"
)

func InitPlugin(appp app.ChainApp, accountKeeper auth.AccountKeeper) {
	// add msg handlers
	for route, handler := range routes(accountKeeper) {
		appp.GetRouter().AddRoute(route, handler)
	}

	//register transfer memo checker
	scripts.RegisterTransferMemoCheckScript(accountKeeper)
}