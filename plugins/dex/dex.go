package dex

import (
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
)

// type MsgList = list.Msg
// type TradingPair = types.TradingPair

type TradingPairMapper = store.TradingPairMapper
type OrderKeeper = order.Keeper

var NewTradingPairMapper = store.NewTradingPairMapper
var NewOrderKeeper = order.NewKeeper
