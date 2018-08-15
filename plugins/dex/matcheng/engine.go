package matcheng

import (
	"math"
	"sort"
)

type MatchEng struct {
	Book OrderBookInterface
	// LotSize may be based on price level, which can be set
	// before any match() call
	LotSize int64
	// PriceLimit is a percentage use to calculate the range of price
	// in order to determine the trade price. Though it is saved as int64,
	// it would be converted into a float when the match engine is created.
	PriceLimitPct float64
	// all the below are buffers
	overLappedLevel []OverLappedLevel
	buyBuf          []PriceLevel
	sellBuf         []PriceLevel
	maxExec         LevelIndex
	leastSurplus    SurplusIndex
	Trades          []Trade
	LastTradePrice  int64
}

func NewMatchEng(basePrice, lotSize int64, priceLimit float64) *MatchEng {
	return &MatchEng{Book: NewOrderBookOnULList(10000, 16), LotSize: lotSize, PriceLimitPct: priceLimit, overLappedLevel: make([]OverLappedLevel, 0, 16),
		buyBuf: make([]PriceLevel, 16), sellBuf: make([]PriceLevel, 16),
		maxExec: LevelIndex{0, make([]int, 8)}, leastSurplus: SurplusIndex{LevelIndex{math.MaxInt64, make([]int, 8)}, make([]int64, 8)},
		Trades: make([]Trade, 0, 64), LastTradePrice: basePrice}
}

// fillOrders would fill the orders at BuyOrders[i] and SellOrders[j] against each other.
// At least one side would be fully filled.
func (me *MatchEng) fillOrders(i int, j int) {
	var k, h int
	buys := me.overLappedLevel[i].BuyOrders
	sells := me.overLappedLevel[j].SellOrders
	origBuyPx := me.overLappedLevel[i].Price
	// sort 1st to get the same seq of fills across different nodes
	// TODO: duplicated sort called here via multiple call of fillOrders on the same i or j
	// not a big deal so far since re-sort on a sorted slice is fast.
	// stable sort is not used here to prevent sort-multiple-times changing the sequence
	// because order id should be always different
	sort.Slice(buys, func(i, j int) bool { return buys[i].Id < buys[j].Id })
	sort.Slice(sells, func(i, j int) bool { return sells[i].Id < sells[j].Id })
	bLength := len(buys)
	sLength := len(sells)
	for k < bLength && h < sLength {
		if compareBuy(buys[k].nxtTrade, 0) == 0 {
			k++
			continue
		}
		if compareBuy(sells[h].nxtTrade, 0) == 0 {
			h++
			continue
		}
		r := compareBuy(buys[k].nxtTrade, sells[h].nxtTrade)
		switch {
		case r > 0:
			trade := sells[h].nxtTrade
			buys[k].nxtTrade -= trade
			sells[h].nxtTrade = 0
			buys[k].CumQty += trade
			sells[h].CumQty += trade
			me.Trades = append(me.Trades, Trade{sells[h].Id, me.LastTradePrice, trade, origBuyPx, buys[k].CumQty, buys[k].Id})
			h++
		case r < 0:
			trade := buys[k].nxtTrade
			sells[h].nxtTrade -= trade
			buys[k].nxtTrade = 0
			buys[k].CumQty += trade
			sells[h].CumQty += trade
			me.Trades = append(me.Trades, Trade{sells[h].Id, me.LastTradePrice, trade, origBuyPx, buys[k].CumQty, buys[k].Id})
			k++
		case r == 0:
			trade := sells[h].nxtTrade
			buys[k].nxtTrade = 0
			sells[h].nxtTrade = 0
			buys[k].CumQty += trade
			sells[h].CumQty += trade
			me.Trades = append(me.Trades, Trade{sells[h].Id, me.LastTradePrice, trade, origBuyPx, buys[k].CumQty, buys[k].Id})
			h++
			k++
		}
	}
	me.overLappedLevel[i].BuyTotal = sumOrdersTotalLeft(buys, false)
	me.overLappedLevel[j].SellTotal = sumOrdersTotalLeft(sells, false)
}

// reserveQty() is called when orders have more leavesQty than the residual execution qty calculated from the matching process,
// so that here is to 'reserve' the necessary qty from orders.
func (me *MatchEng) reserveQty(residual int64, orders []OrderPart) bool {
	//orders should be sorted by time already, since they are added as time sequence
	//no fill should happen on any in the 'orders' before this call, so that no other sorting happens
	// residual must be smaller than the total qty of all orders
	if len(orders) == 1 {
		orders[0].nxtTrade = residual
		return true
	}
	nt := orders[0].Time
	j, k := 1, 1
	toAlloc := residual
	// the below algorithm is to determine the windows by orders' time and
	// allocate residual qty one window after another
	for j < len(orders) && toAlloc > 0 {
		if orders[j].Time == nt {
			if j == len(orders)-1 { // last one, so all the orders are at the same time
				return allocateResidual(&toAlloc, orders[j-k:], me.LotSize)
			} else { // check the next order's time
				j++
				k++
			}
		} else { // the current order time is different from all the past time, j must > 0
			nt = orders[j].Time //set the time for the new orders
			// allocate for the past k orders
			if !allocateResidual(&toAlloc, orders[j-k:j], me.LotSize) {
				return false
			}
			if j == len(orders)-1 { //only one order left
				return allocateResidual(&toAlloc, orders[j:], me.LotSize)
			} else { //start new counting
				k = 1
				j++
			}
		}
	}
	return true
}

// Match() return false mean there is orders in the book the current MatchEngine cannot handle.
// in such case, there should be alerts and all the new orders in this round should be rejected and dropped from order books
// cancel order should be handled 1st before calling Match().
// IOC orders should be handled after Match()
func (me *MatchEng) Match() bool {
	me.Trades = me.Trades[:0]
	r := me.Book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	if r <= 0 {
		return true
	}
	prepareMatch(&me.overLappedLevel)
	lastPx, index := getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, me.LastTradePrice, me.PriceLimitPct)
	if index < 0 {
		return false
	}
	totalExec := me.overLappedLevel[index].AccumulatedExecutions
	me.Trades = me.Trades[:0]
	me.LastTradePrice = lastPx
	i, j := 0, len(me.overLappedLevel)-1
	//sell below the price at index or buy above the price would not get filled
	for i <= index && j >= index && compareBuy(totalExec, 0) > 0 {
		buyTotal := me.overLappedLevel[i].BuyTotal
		sellTotal := me.overLappedLevel[j].SellTotal
		switch {
		case compareBuy(buyTotal, sellTotal) > 0: //fill all sell
			if compareBuy(totalExec, buyTotal) >= 0 { // all buy would be filled later as well
				me.fillOrders(i, j)
			} else {
				//for each BuyOrders or SellOrders, the reserveQty should be called only once
				//and the call should happen before any fill happens on these orders
				if !me.reserveQty(totalExec, me.overLappedLevel[i].BuyOrders) {
					return false
				}
				me.fillOrders(i, j)
			}
			totalExec -= sellTotal
			j--
		case compareBuy(buyTotal, sellTotal) < 0: //fill all buy
			if compareBuy(totalExec, sellTotal) >= 0 { // all buy would be filled later as well
				me.fillOrders(i, j)
			} else {
				if !me.reserveQty(totalExec, me.overLappedLevel[j].SellOrders) {
					return false
				}
				me.fillOrders(i, j)
			}
			totalExec -= buyTotal
			i++
		case compareBuy(buyTotal, sellTotal) == 0: //fill both sides
			me.fillOrders(i, j)
			totalExec -= buyTotal
			i++
			j--
		}
	}

	return true
}

//DropFilledOrder() would clear the order to remove
func (me *MatchEng) DropFilledOrder() int {
	i := 0
	for _, p := range me.overLappedLevel {
		if len(p.BuyOrders) > 0 {
			p.BuyTotal = sumOrdersTotalLeft(p.BuyOrders, true)
			if p.BuyTotal == 0 {
				me.Book.RemovePriceLevel(p.Price, BUYSIDE)
				i++
			} else {
				for _, o := range p.BuyOrders {
					if o.nxtTrade == 0 {
						me.Book.RemoveOrder(o.Id, BUYSIDE, p.Price)
					}
				}
			}
		}
		if len(p.SellOrders) > 0 {
			p.SellTotal = sumOrdersTotalLeft(p.SellOrders, true)
			if p.SellTotal == 0 {
				me.Book.RemovePriceLevel(p.Price, SELLSIDE)
				i++
			} else {
				for _, o := range p.SellOrders {
					if o.nxtTrade == 0 {
						me.Book.RemoveOrder(o.Id, SELLSIDE, p.Price)
					}
				}
			}
		}
	}

	return i
}
