package matcheng

import (
	"fmt"

	bt "github.com/google/btree"
)

const (
	BUYSIDE  int8 = 1
	SELLSIDE int8 = 2
)

// PRECISION is the last effective decimal digit of the price of currency pair
const PRECISION = 1

//Trade stores an execution between 2 orders on a *currency pair*.
//3 things needs attention:
// - srcId and oid are just different names; actually no concept of source or destination;
// - one trade would be implemented via TWO transfer transactions on each currency of the pair;
// - the trade would be uniquely identifiable via the two order id. UUID generation cannot be used here.
type Trade struct {
	SId       string // sell order id
	LastPx    int64  // execution price
	LastQty   int64  // execution quantity
	OrigBuyPx int64  // original intended price for the trade
	BuyCumQty int64  // original intended price for the trade
	BId       string // buy order Id
}

type OrderPart struct {
	id       string
	time     int64
	qty      int64
	cumQty   int64
	nxtTrade int64
}

func (o *OrderPart) LeavesQty() int64 {
	if o.cumQty >= o.qty {
		return 0
	} else {
		return o.qty - o.cumQty
	}
}

func (l *SellPriceLevel) Less(than bt.Item) bool {
	return (than.(*SellPriceLevel).Price - l.Price) >= PRECISION
}

type PriceLevelInterface interface {
	addOrder(id string, time int64, qty int64) (int, error)
	removeOrder(id string) (OrderPart, int, error)
	getOrder(id string) (OrderPart, error)
	Less(than bt.Item) bool
	totalLeavesQty() int64
}

type PriceLevel struct {
	Price  int64
	orders []OrderPart
}

type BuyPriceLevel struct {
	PriceLevel
}

func (l *BuyPriceLevel) Less(than bt.Item) bool {
	return (l.Price - than.(*BuyPriceLevel).Price) >= PRECISION
}

type SellPriceLevel struct {
	PriceLevel
}

func (l *PriceLevel) String() string {
	return fmt.Sprintf("%d->[%v]", l.Price, l.orders)
}

//addOrder would implicitly called with sequence of 'time' parameter
func (l *PriceLevel) addOrder(id string, time int64, qty int64) (int, error) {
	// TODO: need benchmark - queue is not expected to be very long (less than hundreds)
	for _, o := range l.orders {
		if o.id == id {
			return 0, fmt.Errorf("Order %s has existed in the price level.", id)
		}
	}
	l.orders = append(l.orders, OrderPart{id, time, qty, 0, 0})
	return len(l.orders), nil
}

func (l *PriceLevel) removeOrder(id string) (OrderPart, int, error) {
	for i, o := range l.orders {
		if o.id == id {
			k := len(l.orders)
			if i == k-1 {
				l.orders = l.orders[:i]
			} else if i == 0 {
				l.orders = l.orders[1:]
			} else {
				l.orders = append(l.orders[:i], l.orders[i+1:]...)
			}
			return o, k - 1, nil
		}
	}
	// not found
	return OrderPart{}, 0, fmt.Errorf("order %s doesn't exist.", id)
}

func (l *PriceLevel) getOrder(id string) (OrderPart, error) {
	for _, o := range l.orders {
		if o.id == id {
			return o, nil
		}
	}
	// not found
	return OrderPart{}, fmt.Errorf("order %s doesn't exist.", id)
}

func (l *PriceLevel) totalLeavesQty() int64 {
	var total int64 = 0
	for _, o := range l.orders {
		total += o.LeavesQty()
	}
	return total
}

type OverLappedLevel struct {
	Price                 int64
	BuyOrders             []OrderPart
	SellOrders            []OrderPart
	SellTotal             int64
	AccumulatedSell       int64
	BuyTotal              int64
	AccumulatedBuy        int64
	AccumulatedExecutions int64
	BuySellSurplus        int64
}

type LevelIter func(price int64, total int64)
