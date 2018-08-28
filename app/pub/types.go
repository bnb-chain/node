package pub

import (
	"github.com/BiJie/BinanceChain/plugins/dex/matcheng"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

// intermediate data structures to deal with concurrent publication between main thread and publisher thread
type BlockInfoToPublish struct {
	height             int64
	timestamp          int64
	tradesForAllPairs  map[string][]matcheng.Trade
	orderChanges       orderPkg.OrderChanges
	orderChangesMap    orderPkg.OrderChangesMap
	latestPricesLevels orderPkg.ChangedPriceLevels
}

func NewBlockInfoToPublish(height int64, timestamp int64, tradesForAllPairs *map[string][]matcheng.Trade, orderChanges orderPkg.OrderChanges, orderChangesMap orderPkg.OrderChangesMap, latestPriceLevels orderPkg.ChangedPriceLevels) BlockInfoToPublish {
	return BlockInfoToPublish{height, timestamp, *tradesForAllPairs, orderChanges, orderChangesMap, latestPriceLevels}
}