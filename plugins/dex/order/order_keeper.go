package order

import (
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/bnb-chain/node/common/log"
	"github.com/bnb-chain/node/common/utils"
	me "github.com/bnb-chain/node/plugins/dex/matcheng"
	"github.com/bnb-chain/node/plugins/dex/store"
	dexUtils "github.com/bnb-chain/node/plugins/dex/utils"
)

const (
	numPricesStored  = 2000
	pricesStoreEvery = 1000
	minimalNumPrices = 500
)

type DexOrderKeeper interface {
	initOrders(symbol string)
	addOrder(symbol string, info OrderInfo, isRecovery bool)
	reloadOrder(symbol string, orderInfo *OrderInfo, height int64)
	removeOrder(dexKeeper *DexKeeper, id string, symbol string) (ord me.OrderPart, err error)
	orderExists(symbol, id string) (OrderInfo, bool)
	getOpenOrders(pair string, addr sdk.AccAddress) []store.OpenOrder
	getAllOrders() map[string]map[string]*OrderInfo
	deleteOrdersForPair(pair string)

	iterateRoundSelectedPairs(func(string))
	iterateAllOrders(func(symbol string, id string))

	getRoundOrdersNum() int
	getAllOrdersForPair(pair string) map[string]*OrderInfo
	getRoundOrdersForPair(pair string) []string
	getRoundIOCOrdersForPair(pair string) []string
	clearAfterMatch()
	selectSymbolsToMatch(height int64, matchAllSymbols bool) []string

	// publish
	enablePublish()
	appendOrderChangeSync(change OrderChange)
	getOrderChanges() OrderChanges
	clearOrderChanges()
	getOrderInfosForPub() OrderInfoForPublish
	removeOrderInfosForPub(orderId string)

	support(pair string) bool
	supportUpgradeVersion() bool
	supportPairType(pairType SymbolPairType) bool
}

// in the future, this may be distributed via Sharding
type BaseOrderKeeper struct {
	allOrders      map[string]map[string]*OrderInfo // symbol -> order ID -> order
	roundOrders    map[string][]string              // limit to the total tx number in a block
	roundIOCOrders map[string][]string

	collectOrderInfoForPublish bool
	orderChangesMtx            *sync.Mutex         // guard orderChanges and orderInfosForPub during PreDevlierTx (which is async)
	orderChanges               OrderChanges        // order changed in this block, will be cleaned before matching for new block
	orderInfosForPub           OrderInfoForPublish // for publication usage

	logger tmlog.Logger
}

func NewBaseOrderKeeper(moduleName string) BaseOrderKeeper {
	logger := bnclog.With("module", moduleName)
	return BaseOrderKeeper{
		// need to init the nested map when a new symbol added.
		allOrders:      make(map[string]map[string]*OrderInfo, 256),
		roundOrders:    make(map[string][]string, 256),
		roundIOCOrders: make(map[string][]string, 256),

		collectOrderInfoForPublish: false, // default to false, need a explicit set if needed
		orderChangesMtx:            &sync.Mutex{},
		orderChanges:               make(OrderChanges, 0),
		orderInfosForPub:           make(OrderInfoForPublish),
		logger:                     logger,
	}
}

func (kp *BaseOrderKeeper) addOrder(symbol string, info OrderInfo, isRecovery bool) {
	if kp.collectOrderInfoForPublish {
		change := OrderChange{info.Id, Ack, "", nil}
		// deliberately not add this message to orderChanges
		if !isRecovery {
			kp.orderChanges = append(kp.orderChanges, change)
		}
		kp.logger.Debug("add order to order changes map", "orderId", info.Id, "isRecovery", isRecovery)
		kp.orderInfosForPub[info.Id] = &info
	}

	kp.allOrders[symbol][info.Id] = &info
	kp.addRoundOrders(symbol, info)
}

func (kp *BaseOrderKeeper) addRoundOrders(symbol string, info OrderInfo) {
	if ids, ok := kp.roundOrders[symbol]; ok {
		kp.roundOrders[symbol] = append(ids, info.Id)
	} else {
		newIds := make([]string, 0, 16)
		kp.roundOrders[symbol] = append(newIds, info.Id)
	}
	if info.TimeInForce == TimeInForce.IOC {
		kp.roundIOCOrders[symbol] = append(kp.roundIOCOrders[symbol], info.Id)
	}
}

func (kp *BaseOrderKeeper) orderExists(symbol, id string) (OrderInfo, bool) {
	if orders, ok := kp.allOrders[symbol]; ok {
		if msg, ok := orders[id]; ok {
			return *msg, ok
		}
	}
	return OrderInfo{}, false
}

func (kp *BaseOrderKeeper) removeOrder(dexKeeper *DexKeeper, id string, symbol string) (ord me.OrderPart, err error) {
	ordMsg, ok := kp.orderExists(symbol, id)
	if !ok {
		return me.OrderPart{}, orderNotFound(symbol, id)
	}
	eng, ok := dexKeeper.engines[symbol]
	if !ok {
		return me.OrderPart{}, orderNotFound(symbol, id)
	}
	delete(kp.allOrders[symbol], id)
	return eng.Book.RemoveOrder(id, ordMsg.Side, ordMsg.Price)
}

func (kp *BaseOrderKeeper) deleteOrdersForPair(pair string) {
	delete(kp.allOrders, pair)
}

func (kp *BaseOrderKeeper) getOpenOrders(pair string, addr sdk.AccAddress) []store.OpenOrder {
	openOrders := make([]store.OpenOrder, 0)

	for _, order := range kp.allOrders[pair] {
		if string(order.Sender.Bytes()) == string(addr.Bytes()) {
			openOrders = append(
				openOrders,
				store.OpenOrder{
					order.Id,
					pair,
					utils.Fixed8(order.Price),
					utils.Fixed8(order.Quantity),
					utils.Fixed8(order.CumQty),
					order.CreatedHeight,
					order.CreatedTimestamp,
					order.LastUpdatedHeight,
					order.LastUpdatedTimestamp,
				})
		}
	}

	return openOrders
}

func (kp *BaseOrderKeeper) getAllOrders() map[string]map[string]*OrderInfo {
	return kp.allOrders
}

func (kp *BaseOrderKeeper) clearOrderChanges() {
	kp.orderChanges = kp.orderChanges[:0]
}

func (kp *BaseOrderKeeper) enablePublish() {
	kp.collectOrderInfoForPublish = true
}

func (kp *BaseOrderKeeper) appendOrderChangeSync(change OrderChange) {
	kp.orderChangesMtx.Lock()
	kp.orderChanges = append(kp.orderChanges, change)
	kp.orderChangesMtx.Unlock()
}

func (kp *BaseOrderKeeper) getOrderChanges() OrderChanges {
	return kp.orderChanges
}

func (kp *BaseOrderKeeper) getOrderInfosForPub() OrderInfoForPublish {
	return kp.orderInfosForPub
}

func (kp *BaseOrderKeeper) removeOrderInfosForPub(orderId string) {
	delete(kp.orderInfosForPub, orderId)
}

func (kp *BaseOrderKeeper) getRoundOrdersForPair(pair string) []string {
	return kp.roundOrders[pair]
}

func (kp *BaseOrderKeeper) getRoundIOCOrdersForPair(pair string) []string {
	return kp.roundIOCOrders[pair]
}

func (kp *BaseOrderKeeper) getAllOrdersForPair(pair string) map[string]*OrderInfo {
	return kp.allOrders[pair]
}

func (kp *BaseOrderKeeper) iterateAllOrders(iter func(string, string)) {
	for symbol, orders := range kp.allOrders {
		for orderId := range orders {
			iter(symbol, orderId)
		}
	}
}

//------  BEP2OrderKeeper methods -----
var _ DexOrderKeeper = &BEP2OrderKeeper{}

type BEP2OrderKeeper struct {
	BaseOrderKeeper
	symbolSelector BEP2SymbolSelector
}

// NewBEP2OrderKeeper - Returns the BEP2OrderKeeper
func NewBEP2OrderKeeper() DexOrderKeeper {
	return &BEP2OrderKeeper{
		BaseOrderKeeper: NewBaseOrderKeeper("BEP2OrderKeeper"),
		symbolSelector:  BEP2SymbolSelector{},
	}
}

func (kp *BEP2OrderKeeper) support(pair string) bool {
	if !sdk.IsUpgrade(sdk.BEP8) {
		return true
	}
	return !dexUtils.IsMiniTokenTradingPair(pair)
}

func (kp *BEP2OrderKeeper) supportUpgradeVersion() bool {
	return true
}

func (kp *BEP2OrderKeeper) supportPairType(pairType SymbolPairType) bool {
	return PairType.BEP2 == pairType
}

func (kp *BEP2OrderKeeper) initOrders(symbol string) {
	kp.allOrders[symbol] = map[string]*OrderInfo{}
}

func (kp *BEP2OrderKeeper) clearAfterMatch() {
	kp.logger.Debug("clearAfterMatchBEP2...")
	kp.roundOrders = make(map[string][]string, 256)
	kp.roundIOCOrders = make(map[string][]string, 256)
}

func (kp *BEP2OrderKeeper) iterateRoundSelectedPairs(iter func(string)) {
	for symbol := range kp.roundOrders {
		iter(symbol)
	}
}

func (kp *BEP2OrderKeeper) reloadOrder(symbol string, orderInfo *OrderInfo, height int64) {
	kp.allOrders[symbol][orderInfo.Id] = orderInfo
	if orderInfo.CreatedHeight == height {
		kp.roundOrders[symbol] = append(kp.roundOrders[symbol], orderInfo.Id)
		if orderInfo.TimeInForce == TimeInForce.IOC {
			kp.roundIOCOrders[symbol] = append(kp.roundIOCOrders[symbol], orderInfo.Id)
		}
	}
	if kp.collectOrderInfoForPublish {
		if _, exists := kp.orderInfosForPub[orderInfo.Id]; !exists {
			bnclog.Debug("add order to order changes map, during load snapshot, from active orders", "orderId", orderInfo.Id)
			kp.orderInfosForPub[orderInfo.Id] = orderInfo
		}
	}
}

func (kp *BEP2OrderKeeper) getRoundPairsNum() int {
	return len(kp.roundOrders)
}

func (kp *BEP2OrderKeeper) getRoundOrdersNum() int {
	n := 0
	for _, orders := range kp.roundOrders {
		n += len(orders)
	}
	return n
}

func (kp *BEP2OrderKeeper) selectSymbolsToMatch(height int64, matchAllSymbols bool) []string {
	return kp.symbolSelector.SelectSymbolsToMatch(kp.roundOrders, height, matchAllSymbols)
}
