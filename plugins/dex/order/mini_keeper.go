package order

import (
	"fmt"
	bnclog "github.com/binance-chain/node/common/log"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	dexUtils "github.com/binance-chain/node/plugins/dex/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strings"
	"sync"
)

const (
	defaultMiniBlockMatchInterval = 16
	defaultActiveMiniSymbolCount  = 8
)

//order keeper for mini-token
type MiniOrderKeeper struct {
	BaseOrderKeeper
}

var _ IDexOrderKeeper = &MiniOrderKeeper{}

// NewBEP2OrderKeeper - Returns the MiniToken BEP2OrderKeeper
func NewMiniOrderKeeper() IDexOrderKeeper {
	logger := bnclog.With("module", "dexMiniKeeper")
	return &MiniOrderKeeper{
		BaseOrderKeeper{
			allOrders:        make(map[string]map[string]*OrderInfo, 256), // need to init the nested map when a new symbol added.
			OrderChangesMtx:  &sync.Mutex{},
			OrderChanges:     make(OrderChanges, 0),
			OrderInfosForPub: make(OrderInfoForPublish),
			roundOrders:      make(map[string][]string, 256),
			roundIOCOrders:   make(map[string][]string, 256),
			logger:           logger,
			symbolSelector:   &MiniSymbolSelector{make(map[string]uint32, 256), make([]string, 0, 256)},},
	}
}

//override
func (kp *MiniOrderKeeper) support(pair string) bool {
	if !sdk.IsUpgradeHeight(upgrade.BEP8) {
		return false
	}
	return dexUtils.IsMiniTokenTradingPair(pair)
}

//override
func (kp *MiniOrderKeeper) supportUpgradeVersion() bool {
	return sdk.IsUpgradeHeight(upgrade.BEP8)
}

func (kp *MiniOrderKeeper) supportPairType(pairType SymbolPairType) bool {
	return PairType.MINI == pairType
}

// override
func (kp *MiniOrderKeeper) initOrders(symbol string) {
	kp.allOrders[symbol] = map[string]*OrderInfo{}
	kp.symbolSelector.AddSymbolHash(symbol)
}

// override
func (kp *MiniOrderKeeper) validateOrder(dexKeeper *DexKeeper, ctx sdk.Context, acc sdk.Account, msg NewOrderMsg) error {

	err := kp.BaseOrderKeeper.validateOrder(dexKeeper, ctx, acc, msg)
	if err != nil {
		return err
	}
	coins := acc.GetCoins()
	symbol := strings.ToUpper(msg.Symbol)
	var quantityBigEnough bool
	if msg.Side == Side.BUY {
		quantityBigEnough = msg.Quantity >= types.MiniTokenMinTotalSupply
	} else if msg.Side == Side.SELL {
		quantityBigEnough = (msg.Quantity >= types.MiniTokenMinTotalSupply) || coins.AmountOf(symbol) == msg.Quantity
	}
	if !quantityBigEnough {
		return fmt.Errorf("quantity is too small, the min quantity is %d or total free balance of the mini token",
			types.MiniTokenMinTotalSupply)
	}
	return nil
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
		if _, exists := kp.OrderInfosForPub[orderInfo.Id]; !exists {
			bnclog.Debug("add order to order changes map, during load snapshot, from active orders", "orderId", orderInfo.Id)
			kp.OrderInfosForPub[orderInfo.Id] = orderInfo
		}
	}
}
