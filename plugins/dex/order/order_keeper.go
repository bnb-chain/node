package order

import (
	"errors"
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	tmlog "github.com/tendermint/tendermint/libs/log"

	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/store"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
	"github.com/binance-chain/node/wire"
)

const (
	numPricesStored  = 2000
	pricesStoreEvery = 1000
	minimalNumPrices = 500
)

type IDexOrderKeeper interface {
	addOrder(symbol string, info OrderInfo, collectOrderInfoForPublish bool, isRecovery bool)
	removeOrder(dexKeeper *DexKeeper, id string, symbol string, postCancelHandler func(ord me.OrderPart)) (err error)
	orderExists(symbol, id string) (OrderInfo, bool)
	getOpenOrders(pair string, addr sdk.AccAddress) []store.OpenOrder
	getAllOrders() map[string]map[string]*OrderInfo
	deleteOrdersForPair(pair string)
	clearOrderChanges()
	getOrderChanges() OrderChanges
	getOrderInfosForPub() OrderInfoForPublish
	appendOrderChange(change OrderChange)
	initOrders(symbol string)
	support(pair string) bool
	supportUpgradeVersion() bool
	supportPairType(pairType SymbolPairType) bool
	validateOrder(dexKeeper *DexKeeper, context sdk.Context, account sdk.Account, msg NewOrderMsg) error
	iterateRoundPairs(func(string))
	iterateAllOrders(func(symbol string, id string))
	reloadOrder(symbol string, orderInfo *OrderInfo, height int64, collectOrderInfoForPublish bool)
	getRoundPairsNum() int
	getRoundOrdersNum() int
	getAllOrdersForPair(pair string) map[string]*OrderInfo
	getRoundOrdersForPair(pair string) []string
	getRoundIOCOrdersForPair(pair string) []string
	clearAfterMatch()
	selectSymbolsToMatch(height, timestamp int64, matchAllSymbols bool) []string
	appendOrderChangeSync(change OrderChange)
}

type BEP2OrderKeeper struct {
	BaseOrderKeeper
}

var _ IDexOrderKeeper = &BEP2OrderKeeper{}

// in the future, this may be distributed via Sharding
type BaseOrderKeeper struct {
	allOrders        map[string]map[string]*OrderInfo // symbol -> order ID -> order
	OrderChangesMtx  *sync.Mutex                      // guard OrderChanges and OrderInfosForPub during PreDevlierTx (which is async)
	OrderChanges     OrderChanges                     // order changed in this block, will be cleaned before matching for new block
	OrderInfosForPub OrderInfoForPublish              // for publication usage
	roundOrders      map[string][]string              // limit to the total tx number in a block
	roundIOCOrders   map[string][]string
	poolSize         uint // number of concurrent channels, counted in the pow of 2
	cdc              *wire.Codec
	logger           tmlog.Logger
	symbolSelector   SymbolSelector
}

// NewBEP2OrderKeeper - Returns the BEP2OrderKeeper
func NewBEP2OrderKeeper() IDexOrderKeeper {
	logger := bnclog.With("module", "Bep2OrderKeeper")
	return &BEP2OrderKeeper{
		BaseOrderKeeper{
			allOrders: make(map[string]map[string]*OrderInfo, 256),
			// need to init the nested map when a new symbol added.
			OrderChangesMtx:  &sync.Mutex{},
			OrderChanges:     make(OrderChanges, 0),
			OrderInfosForPub: make(OrderInfoForPublish),
			roundOrders:      make(map[string][]string, 256),
			roundIOCOrders:   make(map[string][]string, 256),
			logger:           logger,
			symbolSelector:   &BEP2SymbolSelector{},
		},
	}
}

func (kp *BaseOrderKeeper) addOrder(symbol string, info OrderInfo, collectOrderInfoForPublish bool, isRecovery bool) {

	if collectOrderInfoForPublish {
		change := OrderChange{info.Id, Ack, "", nil}
		// deliberately not add this message to orderChanges
		if !isRecovery {
			kp.OrderChanges = append(kp.OrderChanges, change)
		}
		kp.logger.Debug("add order to order changes map", "orderId", info.Id, "isRecovery", isRecovery)
		kp.OrderInfosForPub[info.Id] = &info
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

func (kp *BaseOrderKeeper) removeOrder(dexKeeper *DexKeeper, id string, symbol string, postCancelHandler func(ord me.OrderPart)) (err error) {

	ordMsg, ok := kp.orderExists(symbol, id)
	if !ok {
		return orderNotFound(symbol, id)
	}
	eng, ok := dexKeeper.engines[symbol]
	if !ok {
		return orderNotFound(symbol, id)
	}
	delete(kp.allOrders[symbol], id)
	ord, err := eng.Book.RemoveOrder(id, ordMsg.Side, ordMsg.Price)
	if err != nil {
		return err
	}

	if postCancelHandler != nil {
		postCancelHandler(ord)
	}
	return nil
}

func (kp *BaseOrderKeeper) deleteOrdersForPair(pair string) {
	delete(kp.allOrders, pair)
}

func (kp *BaseOrderKeeper) validateOrder(dexKeeper *DexKeeper, ctx sdk.Context, acc sdk.Account, msg NewOrderMsg) error {
	baseAsset, quoteAsset, err := dexUtils.TradingPair2Assets(msg.Symbol)
	if err != nil {
		return err
	}

	seq := acc.GetSequence()
	expectedID := GenerateOrderID(seq, msg.Sender)
	if expectedID != msg.Id {
		return fmt.Errorf("the order ID(%s) given did not match the expected one: `%s`", msg.Id, expectedID)
	}

	pair, err := dexKeeper.PairMapper.GetTradingPair(ctx, baseAsset, quoteAsset)
	if err != nil {
		return err
	}

	if msg.Quantity <= 0 || msg.Quantity%pair.LotSize.ToInt64() != 0 {
		return fmt.Errorf("quantity(%v) is not rounded to lotSize(%v)", msg.Quantity, pair.LotSize.ToInt64())
	}

	if msg.Price <= 0 || msg.Price%pair.TickSize.ToInt64() != 0 {
		return fmt.Errorf("price(%v) is not rounded to tickSize(%v)", msg.Price, pair.TickSize.ToInt64())
	}

	if sdk.IsUpgrade(upgrade.LotSizeOptimization) {
		if dexUtils.IsUnderMinNotional(msg.Price, msg.Quantity) {
			return errors.New("notional value of the order is too small")
		}
	}

	if dexUtils.IsExceedMaxNotional(msg.Price, msg.Quantity) {
		return errors.New("notional value of the order is too large(cannot fit in int64)")
	}

	return nil
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

func (kp *BaseOrderKeeper) clearOrderChanges() {
	kp.OrderChanges = kp.OrderChanges[:0]
}

func (kp *BaseOrderKeeper) getAllOrders() map[string]map[string]*OrderInfo {
	return kp.allOrders
}

func (kp *BaseOrderKeeper) getOrderChanges() OrderChanges {
	return kp.OrderChanges
}

func (kp *BaseOrderKeeper) getOrderInfosForPub() OrderInfoForPublish {
	return kp.OrderInfosForPub
}

func (kp *BaseOrderKeeper) appendOrderChange(change OrderChange) {
	kp.OrderChanges = append(kp.OrderChanges, change)
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

func (kp *BaseOrderKeeper) selectSymbolsToMatch(height, timestamp int64, matchAllSymbols bool) []string {
	return kp.symbolSelector.SelectSymbolsToMatch(kp.roundOrders, height, timestamp, matchAllSymbols)
}

func (kp *BaseOrderKeeper) appendOrderChangeSync(change OrderChange) {
	kp.OrderChangesMtx.Lock()
	kp.OrderChanges = append(kp.OrderChanges, change)
	kp.OrderChangesMtx.Unlock()
}

func (kp *BaseOrderKeeper) iterateAllOrders(iter func(string, string)) {
	//TODO
	for symbol, orders := range kp.allOrders {
		for orderId := range orders {
			iter(symbol, orderId)
		}
	}
}

//------  BEP2OrderKeeper methods -----

func (kp *BEP2OrderKeeper) support(pair string) bool {
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

func (kp *BEP2OrderKeeper) iterateRoundPairs(iter func(string)) {
	for symbol := range kp.roundOrders {
		iter(symbol)
	}
}

func (kp *BEP2OrderKeeper) reloadOrder(symbol string, orderInfo *OrderInfo, height int64, collectOrderInfoForPublish bool) {
	kp.allOrders[symbol][orderInfo.Id] = orderInfo
	if orderInfo.CreatedHeight == height {
		kp.roundOrders[symbol] = append(kp.roundOrders[symbol], orderInfo.Id)
		if orderInfo.TimeInForce == TimeInForce.IOC {
			kp.roundIOCOrders[symbol] = append(kp.roundIOCOrders[symbol], orderInfo.Id)
		}
	}
	if collectOrderInfoForPublish {
		if _, exists := kp.OrderInfosForPub[orderInfo.Id]; !exists {
			bnclog.Debug("add order to order changes map, during load snapshot, from active orders", "orderId", orderInfo.Id)
			kp.OrderInfosForPub[orderInfo.Id] = orderInfo
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
