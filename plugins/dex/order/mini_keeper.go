package order

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	bnclog "github.com/bnb-chain/node/common/log"
	"github.com/bnb-chain/node/common/upgrade"
	dexUtils "github.com/bnb-chain/node/plugins/dex/utils"
)

const (
	defaultMiniBlockMatchInterval int = 16
	defaultActiveMiniSymbolCount  int = 8
)

//order keeper for mini-token
type MiniOrderKeeper struct {
	BaseOrderKeeper
	symbolSelector MiniSymbolSelector
}

var _ DexOrderKeeper = &MiniOrderKeeper{}

// NewBEP2OrderKeeper - Returns the MiniToken orderKeeper
func NewMiniOrderKeeper() DexOrderKeeper {
	return &MiniOrderKeeper{
		BaseOrderKeeper: NewBaseOrderKeeper("dexMiniKeeper"),
		symbolSelector: MiniSymbolSelector{
			make(map[string]uint32, 256),
			make([]string, 0, 256),
		},
	}
}

//override
func (kp *MiniOrderKeeper) support(pair string) bool {
	if !sdk.IsUpgrade(upgrade.BEP8) {
		return false
	}
	return dexUtils.IsMiniTokenTradingPair(pair)
}

//override
func (kp *MiniOrderKeeper) supportUpgradeVersion() bool {
	return sdk.IsUpgrade(upgrade.BEP8)
}

func (kp *MiniOrderKeeper) supportPairType(pairType SymbolPairType) bool {
	return PairType.MINI == pairType
}

// override
func (kp *MiniOrderKeeper) initOrders(symbol string) {
	kp.allOrders[symbol] = map[string]*OrderInfo{}
	kp.symbolSelector.addSymbolHash(symbol)
}

func (kp *MiniOrderKeeper) clearAfterMatch() {
	kp.logger.Debug("clearAfterMatchMini...")
	for _, symbol := range kp.symbolSelector.roundSelectedSymbols {
		delete(kp.roundOrders, symbol)
		delete(kp.roundIOCOrders, symbol)
	}
	kp.symbolSelector.clearRoundMatchSymbol()
}

func (kp *MiniOrderKeeper) iterateRoundSelectedPairs(iter func(string)) {
	for _, symbol := range kp.symbolSelector.roundSelectedSymbols {
		iter(symbol)
	}
}

func (kp *MiniOrderKeeper) getRoundOrdersNum() int {
	n := 0
	kp.iterateRoundSelectedPairs(func(symbol string) {
		n += len(kp.roundOrders[symbol])
	})
	return n
}

func (kp *MiniOrderKeeper) reloadOrder(symbol string, orderInfo *OrderInfo, height int64) {
	kp.allOrders[symbol][orderInfo.Id] = orderInfo
	//TODO confirm no round orders for mini symbol
	if kp.collectOrderInfoForPublish {
		if _, exists := kp.orderInfosForPub[orderInfo.Id]; !exists {
			bnclog.Debug("add order to order changes map, during load snapshot, from active orders", "orderId", orderInfo.Id)
			kp.orderInfosForPub[orderInfo.Id] = orderInfo
		}
	}
}

func (kp *MiniOrderKeeper) selectSymbolsToMatch(height int64, matchAllSymbols bool) []string {
	return kp.symbolSelector.SelectSymbolsToMatch(kp.roundOrders, height, matchAllSymbols)
}
