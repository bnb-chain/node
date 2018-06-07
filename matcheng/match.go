package matcheng

import (
	"math"
	"sort"

	"github.com/google/uuid"
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

type Trade struct {
	oid     string  // order id
	tid     string  // trade id
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
	maxExec         LevelIndex
	maxSurplus      SurplusIndex
	trades          []Trade
	lastTradePrice  float64
}

func NewMatchEng(basePrice, lotSize float64) *MatchEng {
	return &MatchEng{LotSize: lotSize, overLappedLevel: make([]OverLappedLevel, 0, 16),
		trades: make([]Trade, 0, 32), lastTradePrice: basePrice}
}

func (me *MatchEng) clearData() {
	me.overLappedLevel = me.overLappedLevel[:0]
	me.maxExec.clear()
	me.maxSurplus.clear()
}

func sumOrders(orders []OrderPart) float64 {
	var s float64
	for _, o := range orders {
		s += o.qty
	}
	return s
}

func prepareMatch(overlapped *[]OverLappedLevel) int {
	var accu float64
	for i := len(*overlapped) - 1; i >= 0; i-- {
		l := (*overlapped)[i]
		l.SellTotal = sumOrders(l.SellOrders)
		accu += l.SellTotal
		l.AccumulatedSell = accu
	}
	accu = 0.0
	for _, l := range *overlapped {
		l.BuyTotal = sumOrders(l.BuyOrders)
		accu += l.BuyTotal
		l.AccumulatedBuy = accu
		l.AccumulatedExecutions = math.Min(l.AccumulatedBuy, l.AccumulatedSell)
		l.BuySellSurplus = l.AccumulatedBuy - l.AccumulatedSell
	}
	return len(*overlapped)
}

func getPriceCloseToRef(overlapped *[]OverLappedLevel, index []int, refPrice float64) (float64, int) {
	var j int
	var diff float64 = math.MaxFloat64
	for _, i := range index {
		p := (*overlapped)[i].Price
		d := math.Abs(p - refPrice)
		if compareBuy(diff, d) > 0 {
			// do not count == case, when more than one has the same diff, return the largest
			diff = d
			j = i
		}
	}
	return (*overlapped)[j].Price, j
}

func getTradePrice(overlapped *[]OverLappedLevel, maxExec *LevelIndex,
	maxSurplus *SurplusIndex, refPrice float64) (float64, int) {
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
	if len(maxExec.index) == 1 {
		i := maxExec.index[0]
		return (*overlapped)[i].Price, i
	}
	for _, j := range maxExec.index {
		surplus := (*overlapped)[j].BuySellSurplus
		abSurplus := math.Abs(surplus)
		r := compareBuy(surplus, maxSurplus.value)
		if r > 0 {
			maxSurplus.value = abSurplus
			maxSurplus.index = maxSurplus.index[:0]
			maxSurplus.surplus = maxSurplus.surplus[:0]
			maxSurplus.index = append(maxSurplus.index, j)
			maxSurplus.surplus = append(maxSurplus.surplus, surplus)
		} else if r == 0 {
			maxSurplus.index = append(maxSurplus.index, j)
			maxSurplus.surplus = append(maxSurplus.surplus, surplus)
		}
	}
	if len(maxSurplus.index) == 1 {
		i := maxSurplus.index[0]
		return (*overlapped)[i].Price, i
	}
	var buy, sell bool
	for _, i := range maxSurplus.surplus {
		if i < 0 {
			sell = true
		}
		if i > 0 {
			buy = true
		}
		if buy && sell {
			break
		}
	}
	if buy && !sell { // return lowest
		i := maxSurplus.index[len(maxSurplus.index)-1]
		return (*overlapped)[i].Price, i
	}
	if !buy && sell { // return hightest
		i := maxSurplus.index[0]
		return (*overlapped)[i].Price, i
	}
	if (buy && sell) || (!buy && !sell) {
		return getPriceCloseToRef(overlapped, maxSurplus.index, refPrice)
	}
	return -math.MaxFloat64, -1
}

func generateTradeID() string {
	return uuid.New().String()
}

func (me *MatchEng) fillOrders(i int, j int) {
	var k, h int
	buys := me.overLappedLevel[i].BuyOrders
	sells := me.overLappedLevel[j].SellOrders
	// sort 1st to get the same seq of fills across different nodes
	// TODO: duplicated sort called here via multiple call of fillOrders on the same i or j
	// not a big deal so far since re-sort on a sorted is
	// stable sort is not used here to prevent sort-multiple-times changing the sequence
	// because order id should be always different
	sort.Slice(buys, func(i, j int) bool { return buys[i].id < buys[j].id })
	sort.Slice(sells, func(i, j int) bool { return sells[i].id < sells[j].id })
	for k < len(buys) && h < len(sells) {
		if buys[k].qty == 0 {
			k++
			continue
		}
		if sells[h].qty == 0 {
			h++
			continue
		}
		r := compareBuy(buys[k].qty, sells[h].qty)
		switch {
		case r > 0:
			trade := sells[h].qty
			buys[k].qty -= trade
			sells[h].qty = 0
			h++
			tid := generateTradeID()
			me.trades = append(me.trades, Trade{sells[h].id, tid, me.lastTradePrice, trade, buys[k].id})
		case r < 0:
			trade := buys[k].qty
			sells[h].qty -= trade
			buys[k].qty = 0
			k++
			tid := generateTradeID()
			me.trades = append(me.trades, Trade{buys[k].id, tid, me.lastTradePrice, trade, sells[h].id})
		case r == 0:
			trade := sells[h].qty
			buys[k].qty = 0
			sells[h].qty = 0
			h++
			k++
			tid := generateTradeID()
			me.trades = append(me.trades, Trade{sells[h].id, tid, me.lastTradePrice, trade, buys[k].id})
		}
	}
	me.overLappedLevel[i].BuyTotal = sumOrders(buys)
	me.overLappedLevel[i].SellTotal = sumOrders(sells)
}

func (me *MatchEng) allocateResidual(toAlloc *float64, orders []OrderPart) bool {
	if len(orders) == 1 {
		qty := math.Min(*toAlloc, orders[0].qty)
		orders[0].qty = qty
		*toAlloc -= qty
		return true
	}
	t := sumOrders(orders)
	// orders should have the same time, sort here to get deterministic sequence
	sort.Slice(orders, func(i, j int) bool { return orders[i].id < orders[j].id })
	residual := *toAlloc
	halfLot := me.LotSize / 2

	if compareBuy(t, residual) > 0 { // not enough to allocate
		// It is assumed here toAlloc is lot size rounded, so that the below code
		// should leave nothing not allocated
		nLot := math.Floor((residual + halfLot) / me.LotSize)
		remainderLot := math.Floor(math.Mod(nLot, float64(len(orders))) + 0.5)
		for _, o := range orders {
			a := math.Floor((nLot*o.qty+halfLot)/me.LotSize) * me.LotSize
			if compareBuy(remainderLot, 0) > 0 {
				a += me.LotSize
				remainderLot -= me.LotSize
				o.qty = a
			}
			residual -= a
		}
		*toAlloc = residual
		//assert *toAlloc == 0
		if compareBuy(*toAlloc, 0) != 0 {
			return false
		}
	}
	return true
}

func (me *MatchEng) reserveQty(residual float64, orders []OrderPart) bool {
	//orders should be sorted by time already, since they are added as time squence
	//no fill should happen on any in orders before this call, so that so other sorting happens
	if len(orders) == 1 {
		orders[0].qty = residual
		return true
	}
	nt := orders[0].time
	j, k := 1, 1
	toAlloc := residual
	for j < len(orders) || toAlloc <= 0 {
		if orders[j].time == nt {
			k++
			if j == len(orders)-1 { // last one
				return me.allocateResidual(&toAlloc, orders[j-k:])
			} else {
				j++
			}
		} else {
			if !me.allocateResidual(&toAlloc, orders[j-k:j]) {
				return false
			}
			if j == len(orders)-1 {
				return me.allocateResidual(&toAlloc, orders[j:])
			} else {
				k = 1
				j++
			}
		}
	}
	return true
}

// Match() return false mean there is orders in the book the current MatchEngine cannot handle.
// in such case, there should be alerts and all the new orders in this round should be rejected and dropped from order books
func (me *MatchEng) Match() bool {
	r := me.Book.GetOverlappedRange(&me.overLappedLevel)
	if r <= 0 {
		return true
	}
	prepareMatch(&me.overLappedLevel)
	lastPx, index := getTradePrice(&me.overLappedLevel, &me.maxExec, &me.maxSurplus, me.lastTradePrice)
	if index < 0 {
		return false
	}
	totalExec := me.overLappedLevel[index].AccumulatedExecutions
	me.trades = me.trades[:0]
	me.lastTradePrice = lastPx
	i, j := 0, len(me.overLappedLevel)-1
	//sell above price at index or buy below would not get filled
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
