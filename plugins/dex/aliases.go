package dex

import (
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/dex/store"
	"github.com/binance-chain/node/plugins/dex/types"
)

// type MsgList = list.Msg
// type TradingPair = types.TradingPair

type TradingPairMapper = store.TradingPairMapper
type IDexOrderKeeper = order.IDexOrderKeeper
type DexKeeper = order.DexKeeper
type SymbolPairType = order.SymbolPairType

var NewTradingPairMapper = store.NewTradingPairMapper
var NewDexKeeper = order.NewDexKeeper
var PairType = order.PairType

const DefaultCodespace = types.DefaultCodespace
