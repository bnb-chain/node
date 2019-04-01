package pub

import (
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
)

// intermediate data structures to deal with concurrent publication between main thread and publisher thread
type BlockInfoToPublish struct {
	height               int64
	timestamp            int64
	tradesToPublish      []*Trade
	proposalsToPublish   *Proposals
	stakeUpdatedAccounts *StakeUpdatedAccounts
	orderChanges         orderPkg.OrderChanges
	orderInfos           orderPkg.OrderInfoForPublish
	accounts             map[string]Account
	latestPricesLevels   orderPkg.ChangedPriceLevelsMap
	blockFee             BlockFee
	feeHolder            orderPkg.FeeHolder
	transfers            *Transfers
}

func NewBlockInfoToPublish(
	height int64,
	timestamp int64,
	tradesToPublish []*Trade,
	proposalsToPublish *Proposals,
	stakeUpdatedAccounts *StakeUpdatedAccounts,
	orderChanges orderPkg.OrderChanges,
	orderInfos orderPkg.OrderInfoForPublish,
	accounts map[string]Account,
	latestPriceLevels orderPkg.ChangedPriceLevelsMap,
	blockFee BlockFee,
	feeHolder orderPkg.FeeHolder, transfers *Transfers) BlockInfoToPublish {
	return BlockInfoToPublish{
		height,
		timestamp,
		tradesToPublish,
		proposalsToPublish,
		stakeUpdatedAccounts,
		orderChanges,
		orderInfos,
		accounts,
		latestPriceLevels,
		blockFee,
		feeHolder,
		transfers}
}
