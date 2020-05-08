package dex

import (
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/plugins/dex/types"
)

// type MsgList = list.Msg
// type TradingPair = types.TradingPair

type TradingPairMapper = store.TradingPairMapper
type BEP2OrderKeeper = order.BEP2OrderKeeper
type MiniOrderKeeper = order.MiniOrderKeeper
type IDexOrderKeeper = order.IDexOrderKeeper
type DexKeeper = order.DexKeeper
type SymbolPairType = order.SymbolPairType

var NewTradingPairMapper = store.NewTradingPairMapper
var NewOrderKeeper = order.NewBEP2OrderKeeper
var NewMiniOrderKeeper = order.NewMiniOrderKeeper
var NewDexKeeper = order.NewDexKeeper
var PairType = order.PairType

const DefaultCodespace = types.DefaultCodespace
