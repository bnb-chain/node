package order

import "sync"

type OrderKeeper interface {
	IterateRoundPairs(func(string))
	GetRoundPairsNum() int
	GetRoundOrdersNum () int
	GetAllOrdersForPair(pair string) map[string]*OrderInfo
	GetRoundOrdersForPair(pair string) []string
	GetRoundIOCOrdersForPair(pair string) []string
	AppendOrderChange(change OrderChange)
	Support(pair string) bool

}

type BEP2OrderKeeper struct {
	allOrders                  map[string]map[string]*OrderInfo // symbol -> order ID -> order
	OrderChangesMtx            *sync.Mutex                      // guard OrderChanges and OrderInfosForPub during PreDevlierTx (which is async)
	OrderChanges               OrderChanges                     // order changed in this block, will be cleaned before matching for new block
	OrderInfosForPub           OrderInfoForPublish              // for publication usage
	roundOrders                map[string][]string              // limit to the total tx number in a block
	roundIOCOrders             map[string][]string
}

func NewBEP2OrderKeeper() *BEP2OrderKeeper {
	return &BEP2OrderKeeper{
		allOrders:                  make(map[string]map[string]*OrderInfo, 256), // need to init the nested map when a new symbol added.
		OrderChangesMtx:            &sync.Mutex{},
		OrderChanges:               make(OrderChanges, 0),
		OrderInfosForPub:           make(OrderInfoForPublish),
		roundOrders:                make(map[string][]string, 256),
		roundIOCOrders:             make(map[string][]string, 256),
	}
}

func (k BEP2OrderKeeper) Support(pair string) bool {
	return true
}

func (k BEP2OrderKeeper) IterateRoundPairs(iter func(string)) {
	for symbol := range k.roundOrders {
		iter(symbol)
	}
}


func (k BEP2OrderKeeper) GetRoundPairsNum() int {
	return len(k.roundOrders)
}

func (k BEP2OrderKeeper) GetRoundOrdersNum() int {
	n := 0
	for _, orders :=range k.roundOrders {
		n += len(orders)
	}
	return n
}

func (k BEP2OrderKeeper) GetRoundOrdersForPair(pair string) []string {
	return k.roundOrders[pair]
}

func (k BEP2OrderKeeper) GetRoundIOCOrdersForPair(pair string) []string {
	return k.roundIOCOrders[pair]
}

func (k BEP2OrderKeeper) GetAllOrdersForPair(pair string) map[string]*OrderInfo {
	return k.allOrders[pair]
}

func (k *BEP2OrderKeeper) AppendOrderChange(change OrderChange) {
	k.OrderChangesMtx.Lock()
	k.OrderChanges = append(k.OrderChanges, change)
	k.OrderChangesMtx.Unlock()
}
