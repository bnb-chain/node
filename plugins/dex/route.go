package dex

import (
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/plugins/dex/list"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens"
)

func Routes(tradingPairMapper TradingPairMapper, orderKeeper OrderKeeper, tokenMapper tokens.Mapper, accountMapper auth.AccountMapper, keeper bank.Keeper) map[string]tx.Handler {
	routes := make(map[string]tx.Handler)
	routes[order.Route] = order.NewHandler(orderKeeper, accountMapper)
	routes[list.Route] = list.NewHandler(tradingPairMapper, tokenMapper)
	return routes
}
