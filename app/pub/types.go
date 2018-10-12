package pub

import (
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

// intermediate data structures to deal with concurrent publication between main thread and publisher thread
type BlockInfoToPublish struct {
	height             int64
	timestamp          int64
	tradesToPublish    []Trade
	orderChanges       orderPkg.OrderChanges
	orderChangesMap    orderPkg.OrderInfoForPublish
	accounts           map[string]Account
	latestPricesLevels orderPkg.ChangedPriceLevelsMap
}

func NewBlockInfoToPublish(
	height int64,
	timestamp int64,
	tradesToPublish []Trade,
	orderChanges orderPkg.OrderChanges,
	orderChangesMap orderPkg.OrderInfoForPublish,
	accounts map[string]Account,
	latestPriceLevels orderPkg.ChangedPriceLevelsMap) BlockInfoToPublish {
	return BlockInfoToPublish{
		height,
		timestamp,
		tradesToPublish,
		orderChanges,
		orderChangesMap,
		accounts,
		latestPriceLevels}
}
