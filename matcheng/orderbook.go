package matcheng

import (
	"bytes"
	"errors"
	"fmt"

	bt "github.com/google/btree"
)

// import ""

const (
	BUYSIDE  = 1
	SELLSIDE = 2
)

// PRECISION is the last effective decimal digit of the price of currency pair
const PRECISION = 0.000000005

type OrderPart struct {
	id   string
	time uint64
	qty  float64
}

type PriceLevel struct {
	Price  float64
	orders []OrderPart
}

type PriceLevelInterface interface {
	addOrder(id string, time uint64, qty float64) (int, error)
	removeOrder(id string) (OrderPart, int, error)
	Less(than bt.Item) bool
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

func (l *PriceLevel) String() string {
	return fmt.Sprintf("%.8f->[%v]", l.Price, l.orders)
}

//addOrder would implicitly called with sequence of 'time' parameter
func (l *PriceLevel) addOrder(id string, time uint64, qty float64) (int, error) {
	// TODO: need benchmark - queue is not expected to be very long (less than hundreds)
	for _, o := range l.orders {
		if o.id == id {
			return 0, fmt.Errorf("Order %s has existed in the price level.", id)
		}
	}
	l.orders = append(l.orders, OrderPart{id, time, qty})
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

type OverLappedLevel struct {
	Price                 float64
	BuyOrders             []OrderPart
	SellOrders            []OrderPart
	SellTotal             float64
	AccumulatedSell       float64
	BuyTotal              float64
	AccumulatedBuy        float64
	AccumulatedExecutions float64
	BuySellSurplus        float64
}

// OrderBookInterface is a generic sequenced order to quickly get the spread to match.
// It can be implemented in different structures but here a fast unrolled-linked list,
// or/and google/B-Tree are chosen, still need performance benchmark to justify this.
type OrderBookInterface interface {
	GetOverlappedRange(overlapped *[]OverLappedLevel, buyBuf *[]PriceLevel, sellBuf *[]PriceLevel) int
	//TODO: especially for ULList, it might be faster by inserting multiple orders in one go then
	//looping through InsertOrder() one after another.
	InsertOrder(id string, side int, time uint, price float64, qty float64) (*PriceLevel, error)
	RemoveOrder(id string, side int, price float64) (OrderPart, error)
	ShowDepth(numOfLevels int, iter func(price float64, buyTotal float64, sellTotal float64))
}

type OrderBookOnULList struct {
	buyQueue  *ULList
	sellQueue *ULList
}

type OrderBookOnBTree struct {
	buyQueue  *bt.BTree
	sellQueue *bt.BTree
}

func (ob *OrderBookOnULList) String() string {
	return fmt.Sprintf("buyQueue: [%v]\nsellQueue:[%v]", ob.buyQueue, ob.sellQueue)
}

func (ob *OrderBookOnULList) getSideQueue(side int) *ULList {
	switch side {
	case BUYSIDE:
		return ob.buyQueue
	case SELLSIDE:
		return ob.sellQueue
	}
	return nil
}

func NewOrderBookOnULList(capacity int, bucketSize int) *OrderBookOnULList {
	//TODO: find out the best degree
	// 16 is my magic number, hopefully the real overlapped levels are less
	return &OrderBookOnULList{NewULList(capacity, bucketSize, compareBuy),
		NewULList(capacity, bucketSize, compareSell)}
}

func mergeLevels(buyLevels []PriceLevel, sellLevels []PriceLevel, overlapped *[]OverLappedLevel) {
	*overlapped = (*overlapped)[:0]
	var i, j, bN int = 0, len(sellLevels) - 1, len(buyLevels)
	for i < bN && j >= 0 {
		b, s := buyLevels[i].Price, sellLevels[j].Price
		switch compareBuy(b, s) {
		case 0:
			*overlapped = append(*overlapped,
				OverLappedLevel{Price: b, BuyOrders: buyLevels[i].orders,
					SellOrders: sellLevels[j].orders})
			i++
			j--
		case -1:
			*overlapped = append(*overlapped, OverLappedLevel{Price: s,
				SellOrders: sellLevels[j].orders})
			j--
		case 1:
			*overlapped = append(*overlapped, OverLappedLevel{Price: b,
				BuyOrders: buyLevels[i].orders})
			i++
		}
	}
	for i < bN {
		b := buyLevels[i].Price
		*overlapped = append(*overlapped, OverLappedLevel{Price: b,
			BuyOrders: buyLevels[i].orders})
		i++
	}
	for j >= 0 {
		s := sellLevels[j].Price
		*overlapped = append(*overlapped, OverLappedLevel{Price: s,
			SellOrders: sellLevels[j].orders})
		j--
	}
}

func (ob *OrderBookOnULList) GetOverlappedRange(overlapped *[]OverLappedLevel, buyBuf *[]PriceLevel, sellBuf *[]PriceLevel) int {
	//clear return
	*overlapped = (*overlapped)[:0]
	*buyBuf = (*buyBuf)[:0]
	*sellBuf = (*sellBuf)[:0]
	// we may need more buffer to prevent memory allocating
	buyTop := ob.buyQueue.GetTop()
	if buyTop == nil { // one side market
		return 0
	}
	sellTop := ob.sellQueue.GetTop()
	if sellTop == nil { // on side market
		return 0
	}
	var p2, p1 float64 = buyTop.Price, sellTop.Price
	if compareBuy(p2, p1) < 0 { //p2 < p1
		return 0 // not overlapped
	}
	buyLevels := ob.buyQueue.GetPriceRange(p2, p1, buyBuf)
	sellLevels := ob.sellQueue.GetPriceRange(p1, p2, sellBuf)
	mergeLevels(buyLevels, sellLevels, overlapped)
	return len(*overlapped)
}

func (ob *OrderBookOnULList) InsertOrder(id string, side int, time uint64, price float64, qty float64) (*PriceLevel, error) {
	q := ob.getSideQueue(side)
	var pl *PriceLevel
	if pl = q.GetPriceLevel(price); pl == nil {
		// price level not exist, insert a new one
		pl = &PriceLevel{price, []OrderPart{{id, time, qty}}}
		if !q.AddPriceLevel(pl) {
			return pl, fmt.Errorf("Failed to insert order %s at price %f", id, price)
		}
		return pl, nil
	} else {
		if _, err := pl.addOrder(id, time, qty); err != nil {
			return pl, err
		}
		return pl, nil
	}
}

//TODO: InsertOrder and RemoveOrder should be faster if done in batch with multiple orders
func (ob *OrderBookOnULList) RemoveOrder(id string, side int, price float64) (OrderPart, error) {
	q := ob.getSideQueue(side)
	var pl *PriceLevel
	if pl = q.GetPriceLevel(price); pl == nil {
		return OrderPart{}, fmt.Errorf("order price %f doesn't exist at side %d.", price, side)
	}
	op, total, ok := pl.removeOrder(id)
	if ok != nil {
		return op, ok
	}
	//price level is gone
	if total == 0 {
		q.DeletePriceLevel(pl.Price)
	}
	return op, ok
}

type BuyPriceLevel struct {
	PriceLevel
}

type SellPriceLevel struct {
	PriceLevel
}

func (l *BuyPriceLevel) Less(than bt.Item) bool {
	return (l.Price - than.(*BuyPriceLevel).Price) >= PRECISION
}

func (l *SellPriceLevel) Less(than bt.Item) bool {
	return (than.(*SellPriceLevel).Price - l.Price) >= PRECISION
}

/*
func (l *BuyPriceLevel) addOrder(id string, time uint64, qty float64) (float64, error) {
	return l.Price.addOrder(id, time, qty)
} */

func newPriceLevelBySide(price float64, orders []OrderPart, side int) PriceLevelInterface {
	switch side {
	case BUYSIDE:
		return &BuyPriceLevel{PriceLevel{price, orders}}
	case SELLSIDE:
		return &SellPriceLevel{PriceLevel{price, orders}}
	}
	return &BuyPriceLevel{PriceLevel{price, orders}}
}

func newPriceLevelKey(price float64, side int) PriceLevelInterface {
	switch side {
	case BUYSIDE:
		return &BuyPriceLevel{PriceLevel{Price: price}}
	case SELLSIDE:
		return &SellPriceLevel{PriceLevel{Price: price}}
	}
	return &BuyPriceLevel{PriceLevel{Price: price}}
}

func NewOrderBookOnBTree(d int) *OrderBookOnBTree {
	//TODO: find out the best degree
	// 16 is my magic number, hopefully the real overlapped levels are less
	return &OrderBookOnBTree{bt.New(8), bt.New(8)}
}

func (ob *OrderBookOnBTree) getSideQueue(side int) *bt.BTree {
	switch side {
	case BUYSIDE:
		return ob.buyQueue
	case SELLSIDE:
		return ob.sellQueue
	}
	return nil
}

func (ob *OrderBookOnBTree) GetOverlappedRange(overlapped *[]OverLappedLevel, buyLevels *[]PriceLevel, sellLevels *[]PriceLevel) int {
	//clear return
	*overlapped = (*overlapped)[:0]
	*buyLevels = (*buyLevels)[:0]
	*sellLevels = (*sellLevels)[:0]
	bItem := ob.buyQueue.Min()
	if bItem == nil {
		return 0
	}
	buyTop, ok := bItem.(*BuyPriceLevel)
	if !ok {
		return 0
	}
	sItem := ob.sellQueue.Min()
	if sItem == nil {
		return 0
	}
	sellTop, ok := sItem.(*SellPriceLevel)
	if !ok {
		return 0
	}
	var p2, p1 float64 = buyTop.Price, sellTop.Price
	if compareBuy(p2, p1) < 0 { //p2 < p1
		return 0 // not overlapped
	}
	//PRECISION has to be added due to AscendRange is a range [GreaterOrEqual, LessThan)
	ob.buyQueue.AscendRange(&BuyPriceLevel{PriceLevel{Price: p2}}, &BuyPriceLevel{PriceLevel{Price: p1 - PRECISION}},
		func(i bt.Item) bool {
			p, ok := i.(*BuyPriceLevel)
			if ok {
				*buyLevels = append(*buyLevels, p.PriceLevel)
			}
			return true
		})
	ob.sellQueue.AscendRange(&SellPriceLevel{PriceLevel{Price: p1}}, &SellPriceLevel{PriceLevel{Price: p2 + PRECISION}},
		func(i bt.Item) bool {
			p, ok := i.(*SellPriceLevel)
			if ok {
				*sellLevels = append(*sellLevels, p.PriceLevel)
			}
			return true
		})

	mergeLevels(*buyLevels, *sellLevels, overlapped)
	return len(*overlapped)
}

func toPriceLevel(pi PriceLevelInterface, side int) *PriceLevel {
	switch side {
	case BUYSIDE:
		if pl, ok := pi.(*BuyPriceLevel); ok {
			return &pl.PriceLevel
		}
	case SELLSIDE:
		if pl, ok := pi.(*SellPriceLevel); ok {
			return &pl.PriceLevel
		}
	}
	return nil
}

func printOrderQueueString(q *bt.BTree, side int) string {
	var buffer bytes.Buffer
	q.Ascend(func(i bt.Item) bool {
		buffer.WriteString(fmt.Sprintf("%v, ", toPriceLevel(i.(PriceLevelInterface), side)))
		return true
	})
	return buffer.String()
}

func (ob *OrderBookOnBTree) InsertOrder(id string, side int, time uint64, price float64, qty float64) (*PriceLevel, error) {
	q := ob.getSideQueue(side)

	if pl := q.Get(newPriceLevelKey(price, side)); pl == nil {
		// price level not exist, insert a new one
		pl2 := newPriceLevelBySide(price, []OrderPart{{id, time, qty}}, side)
		if q.ReplaceOrInsert(pl2) != nil {
			return toPriceLevel(pl2, side), fmt.Errorf("Severe error: data consistence break when insert %v @ %v orderbook", id, price)
		}
		return toPriceLevel(pl2, side), nil
	} else {
		if pl2, ok := pl.(PriceLevelInterface); !ok {
			return nil, errors.New("Severe error: Wrong type item inserted into OrderBook")
		} else {
			_, e := pl2.addOrder(id, time, qty)
			return toPriceLevel(pl2, side), e
		}
	}

}

func (ob *OrderBookOnBTree) RemoveOrder(id string, side int, price float64) (OrderPart, error) {
	q := ob.getSideQueue(side)
	var pl bt.Item
	if pl = q.Get(newPriceLevelKey(price, side)); pl == nil {
		return OrderPart{}, fmt.Errorf("order price %f doesn't exist at side %d.", price, side)
	}
	if pl2, ok := pl.(PriceLevelInterface); !ok {
		return OrderPart{}, errors.New("Severe error: Wrong type item inserted into OrderBook")
	} else {
		op, total, err := pl2.removeOrder(id)
		if err != nil {
			return op, err
		}
		//price level is gone
		if total == 0 {
			q.Delete(pl)
		}
		return op, err
	}
}
