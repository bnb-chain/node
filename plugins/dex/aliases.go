package dex

import (
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/plugins/dex/types"
)

// type MsgList = list.Msg
// type TradingPair = types.TradingPair

type TradingPairMapper = store.TradingPairMapper
type DexKeeper = order.DexKeeper

var NewTradingPairMapper = store.NewTradingPairMapper
var NewDexKeeper = order.NewDexKeeper
var InitOrders = order.Init

const DefaultCodespace = types.DefaultCodespace
