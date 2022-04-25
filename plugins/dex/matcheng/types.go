package matcheng

import (
	"fmt"
	"sort"

	bt "github.com/google/btree"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	UNKNOWN  int8 = 0
	BUYSIDE  int8 = 1
	SELLSIDE int8 = 2
)

// PRECISION is the last effective decimal digit of the price of currency pair
const PRECISION = 1

// Trade status
const (
	Unknown = iota
	SellTaker
	BuyTaker
	BuySurplus
	SellSurplus
	Neutral
)

//Trade stores an execution between 2 orders on a *currency pair*.
//3 things needs attention:
// - srcId and oid are just different names; actually no concept of source or destination;
// - one trade would be implemented via TWO transfer transactions on each currency of the pair;
// - the trade would be uniquely identifiable via the two order id. UUID generation cannot be used here.
type Trade struct {
	Sid        string // sell order id
	LastPx     int64  // execution price
	LastQty    int64  // execution quantity
	BuyCumQty  int64  // cumulative executed quantity for the buy order
	SellCumQty int64  // cumulative executed quantity for the sell order
	Bid        string // buy order Id
	TickType   int8
	SellerFee  *sdk.Fee // seller's fee
	BuyerFee   *sdk.Fee // buyer's fee
}

type OrderPart struct {
	Id       string
	Time     int64
	Qty      int64
	CumQty   int64
	nxtTrade int64
}

func (o *OrderPart) LeavesQty() int64 {
	if o.CumQty >= o.Qty {
		return 0
	} else {
		return o.Qty - o.CumQty
	}
}

type PriceLevelInterface interface {
	addOrder(id string, time int64, qty int64) (int, error)
	removeOrder(id string) (OrderPart, int, error)
	removeOrders(beforeTime int64, callback func(OrderPart))
	getOrder(id string) (OrderPart, error)
	Less(than bt.Item) bool
	TotalLeavesQty() int64
}

type PriceLevel struct {
	Price  int64
	Orders []OrderPart
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

func (l *SellPriceLevel) Less(than bt.Item) bool {
	return (than.(*SellPriceLevel).Price - l.Price) >= PRECISION
}

func (l *PriceLevel) String() string {
	return fmt.Sprintf("%d->[%v]", l.Price, l.Orders)
}

//addOrder would implicitly called with sequence of 'time' parameter
func (l *PriceLevel) addOrder(id string, time int64, qty int64) (int, error) {
	// TODO: need benchmark - queue is not expected to be very long (less than hundreds)
	for _, o := range l.Orders {
		if o.Id == id {
			return 0, fmt.Errorf("Order %s has existed in the price level.", id)
		}
	}
	l.Orders = append(l.Orders, OrderPart{id, time, qty, 0, 0})
	return len(l.Orders), nil
}

func (l *PriceLevel) removeOrder(id string) (OrderPart, int, error) {
	for i, o := range l.Orders {
		if o.Id == id {
			k := len(l.Orders)
			if i == k-1 {
				l.Orders = l.Orders[:i]
			} else if i == 0 {
				l.Orders = l.Orders[1:]
			} else {
				l.Orders = append(l.Orders[:i], l.Orders[i+1:]...)
			}
			return o, k - 1, nil
		}
	}
	// not found
	return OrderPart{}, 0, fmt.Errorf("order %s doesn't exist.", id)
}

// since the orders in one PriceLevel are sorted by time(height), the orders to be removed are all in the front of the slice.
func (l *PriceLevel) removeOrders(beforeTime int64, callback func(OrderPart)) {
	i := sort.Search(len(l.Orders), func(i int) bool {
		return l.Orders[i].Time >= beforeTime
	})

	if callback != nil {
		for _, ord := range l.Orders[:i] {
			callback(ord)
		}
	}
	l.Orders = l.Orders[i:]
}

func (l *PriceLevel) getOrder(id string) (OrderPart, error) {
	for _, o := range l.Orders {
		if o.Id == id {
			return o, nil
		}
	}
	// not found
	return OrderPart{}, fmt.Errorf("order %s doesn't exist.", id)
}

func (l *PriceLevel) TotalLeavesQty() int64 {
	var total int64 = 0
	for _, o := range l.Orders {
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

	BuyTakerStartIdx  int
	SellTakerStartIdx int
	BuyMakerTotal     int64
	SellMakerTotal    int64
}

func (overlapped *OverLappedLevel) HasBuyMaker() bool {
	return overlapped.BuyTakerStartIdx > 0 && overlapped.BuyMakerTotal > 0
}

func (overlapped *OverLappedLevel) HasBuyTaker() bool {
	return overlapped.BuyTakerStartIdx < len(overlapped.BuyOrders)
}

func (overlapped *OverLappedLevel) HasSellMaker() bool {
	return overlapped.SellTakerStartIdx > 0 && overlapped.SellMakerTotal > 0
}

func (overlapped *OverLappedLevel) HasSellTaker() bool {
	return overlapped.SellTakerStartIdx < len(overlapped.SellOrders)
}

type LevelIter func(priceLevel *PriceLevel, levelIndex int)

type MergedPriceLevel struct {
	price    int64
	orders   []*OrderPart
	totalQty int64
}

func NewMergedPriceLevel(price int64) *MergedPriceLevel {
	return &MergedPriceLevel{
		price:    price,
		orders:   make([]*OrderPart, 0),
		totalQty: 0,
	}
}

func (l *MergedPriceLevel) AddOrder(order *OrderPart) {
	l.orders = append(l.orders, order)
	l.totalQty += order.nxtTrade
}

func (l *MergedPriceLevel) AddOrders(orders []*OrderPart) {
	l.orders = append(l.orders, orders...)
	for _, order := range orders {
		l.totalQty += order.nxtTrade
	}
}

type TakerSideOrders struct {
	*MergedPriceLevel
}
