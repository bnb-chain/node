package dex

import (
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/plugins/dex/types"
)

// type MsgList = list.Msg
// type TradingPair = types.TradingPair

type TradingPairMapper = store.TradingPairMapper
type DexKeeper = order.Keeper
type DexMiniTokenKeeper = order.MiniKeeper
type DexOrderKeeper = order.DexOrderKeeper
type DexGlobalKeeper = order.GlobalKeeper


var NewTradingPairMapper = store.NewTradingPairMapper
var NewOrderKeeper = order.NewKeeper
var NewMiniKeeper = order.NewMiniKeeper
var NewGlobalKeeper = order.NewGlobalKeeper

const DefaultCodespace = types.DefaultCodespace
