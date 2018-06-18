package matcheng

import (
	"math"
	"sort"
)

type LevelIndex struct {
	value float64
	index []int
}

type SurplusIndex struct {
	LevelIndex
	surplus []float64
}

func (li *LevelIndex) clear() {
	li.value = 0.0
	li.index = li.index[:0]
}

func (li *SurplusIndex) clear() {
	li.value = math.MaxFloat64
	li.index = li.index[:0]
	li.surplus = li.surplus[:0]
}

//Trade stores an execution between 2 orders on a *currency pair*.
//3 things needs attention:
// - srcId and oid are just different names; actually no concept of source or destination;
// - one trade would be implemented via TWO transfer transactions on each currency of the pair;
// - the trade would be uniquely identifiable via the two order id. UUID generation cannot be used here.
type Trade struct {
	oid     string  // order id
	lastPx  float64 // execution price
	lastQty float64 // execution quantity
	srcOId  string  // source order id allocated from
}

type MatchEng struct {
	Book OrderBookInterface
	// LotSize may be based on price level, which can be set
	// before any match() call
	LotSize         float64
	overLappedLevel []OverLappedLevel //buffer
	buyBuf          []PriceLevel
	sellBuf         []PriceLevel
	maxExec         LevelIndex
	leastSurplus    SurplusIndex
	trades          []Trade
	lastTradePrice  float64
}

func NewMatchEng(basePrice, lotSize float64) *MatchEng {
	return &MatchEng{LotSize: lotSize, overLappedLevel: make([]OverLappedLevel, 0, 16),
		buyBuf: make([]PriceLevel, 16), sellBuf: make([]PriceLevel, 16),
		maxExec: LevelIndex{0.0, make([]int, 8)}, leastSurplus: SurplusIndex{LevelIndex{math.MaxFloat64, make([]int, 8)}, make([]float64, 8)},
		trades: make([]Trade, 0, 64), lastTradePrice: basePrice}
}

//sumOrdersTotalLeft() returns the total value left that can be traded in this block round.
//recalNxtTrade should be true at the begining and false when nxtTrade is changed by allocation logic
func sumOrdersTotalLeft(orders []OrderPart, recalNxtTrade bool) float64 {
	var s float64
	k := len(orders)
	for i := 0; i < k; i++ {
		o := &orders[i]
		if recalNxtTrade {
			o.nxtTrade = o.qty - o.cumQty
		}
		s += o.nxtTrade
	}
	return s
}

func prepareMatch(overlapped *[]OverLappedLevel) int {
	var accu float64
	k := len(*overlapped)
	for i := k - 1; i >= 0; i-- {
		l := &(*overlapped)[i]
		l.SellTotal = sumOrdersTotalLeft(l.SellOrders, true)
		accu += l.SellTotal
		l.AccumulatedSell = accu
	}
	accu = 0.0
	for i := 0; i < k; i++ {
		l := &(*overlapped)[i]
		l.BuyTotal = sumOrdersTotalLeft(l.BuyOrders, true)
		accu += l.BuyTotal
		l.AccumulatedBuy = accu
		l.AccumulatedExecutions = math.Min(l.AccumulatedBuy, l.AccumulatedSell)
		l.BuySellSurplus = l.AccumulatedBuy - l.AccumulatedSell
	}
	return k
}

func getPriceCloseToRef(overlapped []OverLappedLevel, index []int, refPrice float64) (float64, int) {
	var j int
	var diff float64 = math.MaxFloat64
	for _, i := range index {
		p := overlapped[i].Price
		d := math.Abs(p - refPrice)
		if compareBuy(diff, d) > 0 {
			// do not count == case, when more than one has the same diff, return the largest price, i.e. the 1st
			diff = d
			j = i
		}
	}
	return overlapped[j].Price, j
}

func calMaxExec(overlapped *[]OverLappedLevel, maxExec *LevelIndex) {
	for i, l := range *overlapped {
		r := compareBuy(l.AccumulatedExecutions, maxExec.value)
		if r > 0 {
			maxExec.value = l.AccumulatedExecutions
			maxExec.index = maxExec.index[:0]
			maxExec.index = append(maxExec.index, i)
		} else if r == 0 {
			maxExec.index = append(maxExec.index, i)
		}
	}
}

func calLeastSurplus(overlapped *[]OverLappedLevel, maxExec *LevelIndex,
	leastSurplus *SurplusIndex) {
	for _, j := range maxExec.index {
		surplus := (*overlapped)[j].BuySellSurplus
		abSurplus := math.Abs(surplus)
		r := compareBuy(abSurplus, leastSurplus.value)
		if r < 0 {
			leastSurplus.value = abSurplus
			leastSurplus.index = leastSurplus.index[:0]
			leastSurplus.surplus = leastSurplus.surplus[:0]
			leastSurplus.index = append(leastSurplus.index, j)
			leastSurplus.surplus = append(leastSurplus.surplus, surplus)
		} else if r == 0 {
			leastSurplus.index = append(leastSurplus.index, j)
			leastSurplus.surplus = append(leastSurplus.surplus, surplus)
		}
	}
}

func getTradePrice(overlapped *[]OverLappedLevel, maxExec *LevelIndex,
	leastSurplus *SurplusIndex, refPrice float64) (float64, int) {
	maxExec.clear()
	leastSurplus.clear()
	calMaxExec(overlapped, maxExec)
	if len(maxExec.index) == 1 {
		i := maxExec.index[0]
		return (*overlapped)[i].Price, i
	}
	calLeastSurplus(overlapped, maxExec, leastSurplus)
	if len(leastSurplus.index) == 1 {
		i := leastSurplus.index[0]
		return (*overlapped)[i].Price, i
	}
	var buySurplus, sellSurplus bool
	for _, i := range leastSurplus.surplus {
		if i < 0 {
			sellSurplus = true
		}
		if i > 0 {
			buySurplus = true
		}
		if buySurplus && sellSurplus { // just a short cut
			break
		}
	}
	// only buy side surplus exist, buying pressure
	if buySurplus && !sellSurplus { // return hightest
		i := leastSurplus.index[0]
		return (*overlapped)[i].Price, i
	}
	// only sell side surplus exist, selling pressure
	if !buySurplus && sellSurplus { // return lowest
		i := leastSurplus.index[len(leastSurplus.index)-1]
		return (*overlapped)[i].Price, i
	}
	if (buySurplus && sellSurplus) || (!buySurplus && !sellSurplus) {
		return getPriceCloseToRef(*overlapped, leastSurplus.index, refPrice)
	}
	//never reach here
	return math.MaxFloat64, -1
}

// fillOrders would fill the orders at BuyOrders[i] and SellOrders[j] against each other.
// At least one side would be fully filled.
func (me *MatchEng) fillOrders(i int, j int) {
	var k, h int
	buys := me.overLappedLevel[i].BuyOrders
	sells := me.overLappedLevel[j].SellOrders
	// sort 1st to get the same seq of fills across different nodes
	// TODO: duplicated sort called here via multiple call of fillOrders on the same i or j
	// not a big deal so far since re-sort on a sorted slice is fast.
	// stable sort is not used here to prevent sort-multiple-times changing the sequence
	// because order id should be always different
	sort.Slice(buys, func(i, j int) bool { return buys[i].id < buys[j].id })
	sort.Slice(sells, func(i, j int) bool { return sells[i].id < sells[j].id })
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
			me.trades = append(me.trades, Trade{sells[h].id, me.lastTradePrice, trade, buys[k].id})
			h++
		case r < 0:
			trade := buys[k].nxtTrade
			sells[h].nxtTrade -= trade
			buys[k].nxtTrade = 0
			me.trades = append(me.trades, Trade{sells[h].id, me.lastTradePrice, trade, buys[k].id})
			k++
		case r == 0:
			trade := sells[h].nxtTrade
			buys[k].nxtTrade = 0
			sells[h].nxtTrade = 0
			me.trades = append(me.trades, Trade{sells[h].id, me.lastTradePrice, trade, buys[k].id})
			h++
			k++
		}
	}
	me.overLappedLevel[i].BuyTotal = sumOrdersTotalLeft(buys, false)
	me.overLappedLevel[i].SellTotal = sumOrdersTotalLeft(sells, false)
}

// allocateResidual() assumes toAlloc is less than sum of quantity in orders.
// It would try best to evenly allocate toAlloc among orders in proportion of order qty meanwhile by whole lot
func allocateResidual(toAlloc *float64, orders []OrderPart, lotSize float64) bool {
	if len(orders) == 1 {
		qty := math.Min(*toAlloc, orders[0].nxtTrade)
		orders[0].nxtTrade = qty
		*toAlloc -= qty
		return true
	}

	t := sumOrdersTotalLeft(orders, false)

	// orders should have the same time, sort here to get deterministic sequence
	sort.Slice(orders, func(i, j int) bool { return orders[i].id < orders[j].id })
	residual := *toAlloc
	halfLot := lotSize / 2

	if compareBuy(t, residual) > 0 { // not enough to allocate
		// It is assumed here toAlloc is lot size rounded, so that the below code
		// should leave nothing not allocated
		nLot := math.Floor((residual + halfLot) / lotSize)
		k := len(orders)
		for i := 0; i < k; i++ {
			a := math.Floor(nLot*orders[i].nxtTrade/t+halfLot) * lotSize // this is supposed to be the main portion
			if compareBuy(a, residual) >= 0 {
				orders[i].nxtTrade = residual
				residual = 0
				break
			} else {
				orders[i].nxtTrade = a
				residual -= a
			}
		}
		remainderLot := math.Floor((residual + halfLot) / lotSize)
		for i := 0; i < k; i++ {
			if remainderLot > 0 { // remainer distribution, every one can only get 1 lot or zero
				orders[i].nxtTrade += lotSize
				remainderLot -= 1
				residual -= lotSize
				if i == k-1 { //restart from the beginning
					i = 0
				}
			} else {
				break
			}
		}
		*toAlloc = residual
		//assert *toAlloc == 0
		if compareBuy(*toAlloc, 0) != 0 {
			return false
		}
		return true
	} else { // t <= *toAlloc
		*toAlloc -= t
		return true
	}
}

// reserveQty() is called when orders have more leavesQty than the residual execution qty calculated from the matching process,
// so that here is to 'reserve' the necessary qtys from orders.
func (me *MatchEng) reserveQty(residual float64, orders []OrderPart) bool {
	//orders should be sorted by time already, since they are added as time squence
	//no fill should happen on any in the 'orders' before this call, so that no other sorting happens
	// resdiual must be smaller than the total qty of all orders
	if len(orders) == 1 {
		orders[0].nxtTrade = residual
		return true
	}
	nt := orders[0].time
	j, k := 1, 1
	toAlloc := residual
	// the below algorithm is to determine the windows by orders' time and
	// allocate residual qty one window after another
	for j < len(orders) && toAlloc > 0 {
		if orders[j].time == nt {
			if j == len(orders)-1 { // last one, so all the orders are at the same time
				return allocateResidual(&toAlloc, orders[j-k:], me.LotSize)
			} else { // check the next order's time
				j++
				k++
			}
		} else { // the current order time is different from all the past time, j must > 0
			nt = orders[j].time //set the time for the new orders
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
	r := me.Book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	if r <= 0 {
		return true
	}
	prepareMatch(&me.overLappedLevel)
	lastPx, index := getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, me.lastTradePrice)
	if index < 0 {
		return false
	}
	totalExec := me.overLappedLevel[index].AccumulatedExecutions
	me.trades = me.trades[:0]
	me.lastTradePrice = lastPx
	i, j := 0, len(me.overLappedLevel)-1
	//sell below the price at index or buy above the price would not get filled
	for i <= index && j >= index {
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
				me.reserveQty(totalExec, me.overLappedLevel[j].SellOrders)
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
