package order

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/upgrade"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
)

const (
	defaultMiniBlockMatchInterval = 16
	defaultActiveMiniSymbolCount  = 8
)

//order keeper for mini-token
type MiniOrderKeeper struct {
	BaseOrderKeeper
}

var _ DexOrderKeeper = &MiniOrderKeeper{}

// NewBEP2OrderKeeper - Returns the MiniToken orderKeeper
func NewMiniOrderKeeper() DexOrderKeeper {
	return &MiniOrderKeeper{
		NewBaseOrderKeeper("dexMiniKeeper",
			&MiniSymbolSelector{make(map[string]uint32, 256), make([]string, 0, 256)}),
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
	kp.symbolSelector.AddSymbolHash(symbol)
}

func (kp *MiniOrderKeeper) clearAfterMatch() {
	kp.logger.Debug("clearAfterMatchMini...")
	for _, symbol := range *kp.symbolSelector.GetRoundMatchSymbol() {
		delete(kp.roundOrders, symbol)
		delete(kp.roundIOCOrders, symbol)
	}
	clearedRoundMatchSymbols := make([]string, 0)
	kp.symbolSelector.SetRoundMatchSymbol(clearedRoundMatchSymbols)
}

func (kp *MiniOrderKeeper) iterateRoundPairs(iter func(string)) {
	for _, symbol := range *kp.symbolSelector.GetRoundMatchSymbol() {
		iter(symbol)
	}
}

func (kp *MiniOrderKeeper) getRoundPairsNum() int {
	return len(*kp.symbolSelector.GetRoundMatchSymbol())
}

func (kp *MiniOrderKeeper) getRoundOrdersNum() int {
	n := 0
	kp.iterateRoundPairs(func(symbol string) {
		n += len(kp.roundOrders[symbol])
	})
	return n
}

func (kp *MiniOrderKeeper) reloadOrder(symbol string, orderInfo *OrderInfo, height int64, collectOrderInfoForPublish bool) {
	kp.allOrders[symbol][orderInfo.Id] = orderInfo
	//TODO confirm no active orders for mini symbol
	if collectOrderInfoForPublish {
		if _, exists := kp.orderInfosForPub[orderInfo.Id]; !exists {
			bnclog.Debug("add order to order changes map, during load snapshot, from active orders", "orderId", orderInfo.Id)
			kp.orderInfosForPub[orderInfo.Id] = orderInfo
		}
	}
}
