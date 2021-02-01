package order

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/upgrade"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
)

const (
	defaultMiniBlockMatchInterval int = 16
	defaultActiveMiniSymbolCount  int = 8
)

//order keeper for growth market
type GrowthMarketOrderKeeper struct {
	BaseOrderKeeper
	symbolSelector MiniSymbolSelector
}

var _ DexOrderKeeper = &GrowthMarketOrderKeeper{}

// NewMainMarketOrderKeeper - Returns the MiniToken orderKeeper
func NewGrowthMarketOrderKeeper() DexOrderKeeper {
	return &GrowthMarketOrderKeeper{
		BaseOrderKeeper: NewBaseOrderKeeper("growthMarketKeeper"),
		symbolSelector: MiniSymbolSelector{
			make(map[string]uint32, 256),
			make([]string, 0, 256),
		},
	}
}

//override
func (kp *GrowthMarketOrderKeeper) support(pair string) bool {
	if !sdk.IsUpgrade(upgrade.BEP8) {
		return false
	}
	return dexUtils.IsMiniTokenTradingPair(pair)
}

//override
func (kp *GrowthMarketOrderKeeper) supportUpgradeVersion() bool {
	return sdk.IsUpgrade(upgrade.BEP8)
}

func (kp *GrowthMarketOrderKeeper) supportPairType(pairType dexTypes.SymbolPairType) bool {
	if sdk.IsUpgrade(upgrade.BEPX) {
		return dexTypes.PairType.GROWTH == pairType
	}
	return dexTypes.PairType.MINI == pairType
}

// override
func (kp *GrowthMarketOrderKeeper) initOrders(symbol string) {
	kp.allOrders[symbol] = map[string]*OrderInfo{}
	kp.symbolSelector.addSymbolHash(symbol)
}

func (kp *GrowthMarketOrderKeeper) clearAfterMatch() {
	kp.logger.Debug("clearAfterMatchMini...")
	for _, symbol := range kp.symbolSelector.roundSelectedSymbols {
		delete(kp.roundOrders, symbol)
		delete(kp.roundIOCOrders, symbol)
	}
	kp.symbolSelector.clearRoundMatchSymbol()
}

func (kp *GrowthMarketOrderKeeper) iterateRoundSelectedPairs(iter func(string)) {
	for _, symbol := range kp.symbolSelector.roundSelectedSymbols {
		iter(symbol)
	}
}

func (kp *GrowthMarketOrderKeeper) getRoundPairsNum() int {
	return len(kp.symbolSelector.roundSelectedSymbols)
}

func (kp *GrowthMarketOrderKeeper) getRoundOrdersNum() int {
	n := 0
	kp.iterateRoundSelectedPairs(func(symbol string) {
		n += len(kp.roundOrders[symbol])
	})
	return n
}

func (kp *GrowthMarketOrderKeeper) reloadOrder(symbol string, orderInfo *OrderInfo, height int64) {
	kp.allOrders[symbol][orderInfo.Id] = orderInfo
	//TODO confirm no round orders for mini symbol
	if kp.collectOrderInfoForPublish {
		if _, exists := kp.orderInfosForPub[orderInfo.Id]; !exists {
			bnclog.Debug("add order to order changes map, during load snapshot, from active orders", "orderId", orderInfo.Id)
			kp.orderInfosForPub[orderInfo.Id] = orderInfo
		}
	}
}

func (kp *GrowthMarketOrderKeeper) selectSymbolsToMatch(height int64, matchAllSymbols bool) []string {
	return kp.symbolSelector.SelectSymbolsToMatch(kp.roundOrders, height, matchAllSymbols)
}
