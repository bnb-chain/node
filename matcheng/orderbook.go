package matcheng

import (
	"fmt"

	. "github.com/google/btree"
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
	time uint
	qty  float64
}

type OrderQueue struct {
	totalQty float64
	orders   []OrderPart
}

type PriceLevel struct {
	Price float64
	queue OrderQueue
}

func compareBuy(p1 float64, p2 float64) int {
	d := (p2 - p1)
	switch {
	case d >= PRECISION:
		return -1
	case d <= -PRECISION:
		return 1
	default:
		return 0
	}
}

func compareSell(p1 float64, p2 float64) int {
	return -compareBuy(p1, p2)
}

func newPriceLevel(price float64, orders []OrderPart) *PriceLevel {
	t := 0.0
	for _, o := range orders {
		t += o.qty
	}
	return &PriceLevel{price, OrderQueue{t, orders}}
}

//addOrder would implicitly called with sequence of 'time' parameter
func (l *PriceLevel) addOrder(id string, time uint, qty float64) (float64, error) {
	// TODO: need benchmark - queue is not expected to be very long (less than hundreds)
	for _, o := range l.queue.orders {
		if o.id == id {
			return 0, fmt.Errorf("Order %s has existed in the price level.", id)
		}
	}
	l.queue.totalQty += qty
	l.queue.orders = append(l.queue.orders, OrderPart{id, time, qty})
	return l.queue.totalQty, nil

}

func (l *PriceLevel) removeOrder(id string) (OrderPart, float64, error) {
	for i, o := range l.queue.orders {
		if o.id == id {
			l.queue.orders = append(l.queue.orders[:i], l.queue.orders[i+1])
			l.queue.totalQty -= o.qty
			return o, l.queue.totalQty, nil
		}
	}
	// not found
	return OrderPart{}, l.queue.totalQty, fmt.Errorf("order %s doesn't exist.", id)
}

// OrderBookInterface is a generic sequenced order to quickly get the spread to match.
// It can be implemented in different structures but here a fast unrolled-linked list,
// or/and google/B-Tree are chosen, still need performance benchmark to justify this.
type OrderBookInterface interface {
	GetOverlappedRange() []PriceLevel
	InsertOrder(id string, side int, time uint, price float64, qty float64) (*PriceLevel, error)
	RemoveOrder(id string, side int, price float64) (OrderPart, error)
}

type OrderBookOnULList struct {
	buyQueue   *ULList
	sellQueue  *ULList
	overlapped []PriceLevel
}

type OrderBookOnBTree struct {
	buyQueue   *BTree
	sellQueue  *BTree
	overlapped []PriceLevel
}

func (ob *OrderBookULList) getSideQueue(side int) *ULList {
	switch side {
	case BUYSIDE:
		return ob.buyQueue
	case SELLSIDE:
		return ob.sellQueue
	}
	return nil
}

func NewOrderBookOnULList(d int) *OrderBook {
	//TODO: find out the best degree
	// 16 is my magic number, hopefully the real overlapped levels are less
	return &OrderBook{NewULList(4096, 16, compareBuy),
		NewULList(4096, 16, compareSell),
		make([]PriceLevel, 16)}
}

func (ob *OrderBookULList) GetOverlappedRange() []PriceLevel {
	return ob.overlapped
}

func (ob *OrderBookULList) InsertOrder(id string, side int, time uint, price float64, qty float64) (*PriceLevel, error) {
	q := ob.getSideQueue(side)
	var pl *PriceLevel
	if pl = q.GetPriceLevel(price); pl == nil {
		// price level not exist, insert a new one
		pl = newPriceLevel(price, []OrderPart{OrderPart{id, time, qty}})
	} else {
		if _, e := pl.addOrder(id, time, qty); e != nil {
			return pl, e
		}
	}
	if !q.SetPriceLevel(pl) {
		return pl, fmt.Errorf("Failed to insert order %s at price %f", id, price)
	}
	return pl, nil
}

func (ob *OrderBookOnULList) RemoveOrder(id string, side int, price float64) (OrderPart, error) {
	q := ob.getSideQueue(side)
	var pl *PriceLevel
	if pl := q.GetPriceLevel(price); pl == nil {
		return OrderPart{}, fmt.Errorf("order price %f doesn't exist at side %d.", price, side)
	}
	op, total, ok := pl.removeOrder(id)
	if ok != nil {
		return op, ok
	}
	//price level is gone
	if total == 0.0 {
		q.DeletePriceLevel(pl.Price)
	}
	return op, ok
}
