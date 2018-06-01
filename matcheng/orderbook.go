package matcheng

import (
	"errors"
	"fmt"
)

// import ""

const (
	BUYSIDE  = 1
	SELLSIDE = 2
)

// PRECISION is the last effective decimal digit of the price of currency pair
const PRECISION = 0.00000001

type OrderPart struct {
	id   string
	time uint64
	qty  float64
}

// OrderBookInterface is a generic sequenced order to quickly get the spread to match.
// It can be implemented in different structures but here a fast adaptive-radix-tree is chosen,
// Still need performance benchmark to justify this.
type OrderBookInterface interface {
	GetOverlappedRange() []PriceLevel
	InsertOrder(id string, side int, time uint64, price float64, qty float64) (*PriceLevel, error)
	RemoveOrder(id string, side int, price float64) (OrderPart, error)
}

type OrderQueue struct {
	totalQty float64
	orders   []OrderPart
}

type PriceLeveL struct {
	price float64
	queue OrderQueue
}

type BuyPriceLevel PriceLevel
type SellPriceLevel PriceLevel

func (l *BuyPriceLevel) Less(than Item) bool {
	return (than.(PriceLevel).price - l.price) >= PRECISION
}

func (l *SellPriceLevel) Less(than Item) bool {
	return (l.price - than.(PriceLevel).price) >= PRECISION
}

func newPriceLevel(price float64, orders []OrderPart, side int) {
	t := 0
	for _, o := range orders {
		t += o.qty
	}
	switch side {
	case BUYSIDE:
		return BuyPriceLevel{price, OrderQueue{t, orders}}
	case SELLSIDE:
		return SellPriceLevel{price, OrderQueue{t, orders}}
	}
}

func newPriceLevelKey(price float64, side int) {
	switch side {
	case BUYSIDE:
		return BuyPriceLevel{price: price}
	case SELLSIDE:
		return SellPriceLevel{price: price}
	}
}

type PriceLevelInterface interface {
	addOrder(id string, time uint64, qty float64) (float64, error)
	removeOrder(id string) (OrderPart, float64, error)
}

func (l *PriceLevel) addOrder(id string, time uint64, qty float64) (float64, error) {
	for _, o := range l.queue.orders {
		if o.id == id {
			return 0, fmt.Errorf("Order %s has existed in the price level.", id)
		}
	}
	l.queue.totalQty += qty
	append(l.queue.orders, OrderPart{id, time, qty})
	return l.queue.totalQty, nil

}

func (l *PriceLevel) removeOrder(id string) (OrderPart, float64, error) {
	for i, o := range l.queue.orders {
		if o.id == id {
			l.queue.orders = append(l.queue.order[:i], l.queue.order[i+1])
			l.queue.totalQty -= o.qty
			return o, totalQty, nil
		}
	}
	// not found
	return OrderPart{}, fmt.Errorf("order %s doesn't exist.", id)
}

type OrderBook struct {
	buyQueue  art.Tree
	sellQueue art.Tree
}

func (ob *OrderBook) getSideQueue(side int) *BTree {
	switch side {
	case BUYSIDE:
		return ob.buyQueue
	case SELLSIDE:
		return ob.sellQueue
	}
}

func NewOrderBook(d int) *OrderBook {
	return &OrderBook{art.New(), art.New()} //TODO: find out the best degree
}

func (ob *OrderBook) GetOverlappedRange() []PriceLevel {
	levels = make([]PriceLevel, 16) // 16 is my magic number, hopefully the real overlapped levels are less

}

func (ob *OrderBook) InsertOrder(id string, side int, time uint64, price float64, qty float64) (*PriceLevel, error) {
	q := ob.getSideQueue(side)
	if pl := q.Get(newPriceLevelKey(price, side)); pl == nil {
		// price level not exist, insert a new one
		pl = newPriceLevel(price, []OrderParts{OrderPart{id, time, qty}}, side)
	} else {
		if pl2, ok := pl.(PriceLevelInterface); !ok {
			return nil, errors.New("Severe error: Wrong type item inserted into OrderBook")
		} else {
			if f, e := pl2.addOrder(id, time, qty); e != nil {
				return &pl2, e
			}
		}
	}
	if q.ReplaceOrInsert(pl) == nil {
		return pl, fmt.Errorf("Failed to insert order %s at price %d", id, price)
	}
	return &pl, nil
}

func (ob *OrderBook) RemoveOrder(id string, side int, price float64) (OrderPart, error) {
	q := ob.getSideQueue(side)
	if pl := q.Get(newPriceLevelKey(price, side)); pl == nil {
		return OrderPart{}, fmt.Errorf("order price %d doesn't exist at side %d.", price, side)
	}
	if pl2, ok := pl.(PriceLevelInterface); !ok {
		return OrderPart{}, errors.New("Severe error: Wrong type item inserted into OrderBook")
	} else {
		op, total, ok = pl2.removeOrder(id)
		if ok != nil {
			return op, ok
		}
		//price level is gone
		if total == 0.0 {
			q.Delete(pl)
			return op, ok
		}
	}
}
