package dex

import (
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	"github.com/BiJie/BinanceChain/plugins/dex/types"
)

// type MsgList = list.Msg
// type TradingPair = types.TradingPair

type TradingPairMapper = store.TradingPairMapper
type DexKeeper = order.Keeper

var NewTradingPairMapper = store.NewTradingPairMapper
var NewOrderKeeper = order.NewKeeper

const DefaultCodespace = types.DefaultCodespace
