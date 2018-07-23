package matcheng

import (
	"math"
	"sort"

	"github.com/BiJie/BinanceChain/common/utils"
)

type LevelIndex struct {
	value int64
	index []int
}

type SurplusIndex struct {
	LevelIndex
	surplus []int64
}

func (li *LevelIndex) clear() {
	li.value = 0
	li.index = li.index[:0]
}

func (li *SurplusIndex) clear() {
	li.value = math.MaxInt64
	li.index = li.index[:0]
	li.surplus = li.surplus[:0]
}

//Trade stores an execution between 2 orders on a *currency pair*.
//3 things needs attention:
// - srcId and oid are just different names; actually no concept of source or destination;
// - one trade would be implemented via TWO transfer transactions on each currency of the pair;
// - the trade would be uniquely identifiable via the two order id. UUID generation cannot be used here.
type Trade struct {
	SId     string // sell order id
	LastPx  int64  // execution price
	LastQty int64  // execution quantity
	BId     string // buy order Id
}

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

//sumOrdersTotalLeft() returns the total value left that can be traded in this block round.
//reCalNxtTrade should be true at the begining and false when nxtTrade is changed by allocation logic
func sumOrdersTotalLeft(orders []OrderPart, reCalNxtTrade bool) int64 {
	var s int64
	k := len(orders)
	for i := 0; i < k; i++ {
		o := &orders[i]
		if reCalNxtTrade {
			o.nxtTrade = o.qty - o.cumQty
		}
		s += o.nxtTrade
	}
	return s
}

func prepareMatch(overlapped *[]OverLappedLevel) int {
	var accum int64
	k := len(*overlapped)
	for i := k - 1; i >= 0; i-- {
		l := &(*overlapped)[i]
		l.SellTotal = sumOrdersTotalLeft(l.SellOrders, true)
		accum += l.SellTotal
		l.AccumulatedSell = accum
	}
	accum = 0
	for i := 0; i < k; i++ {
		l := &(*overlapped)[i]
		l.BuyTotal = sumOrdersTotalLeft(l.BuyOrders, true)
		accum += l.BuyTotal
		l.AccumulatedBuy = accum
		l.AccumulatedExecutions = utils.MinInt(l.AccumulatedBuy, l.AccumulatedSell)
		l.BuySellSurplus = l.AccumulatedBuy - l.AccumulatedSell
	}
	return k
}

func getPriceCloseToRef(overlapped []OverLappedLevel, index []int, refPrice int64) (int64, int) {
	var j int
	var diff int64 = math.MaxInt64
	refIsSmaller := false
	for _, i := range index {
		p := overlapped[i].Price
		d := p - refPrice
		switch compareBuy(d, 0) {
		case 0:
			return refPrice, i
		case 1:
			refIsSmaller = true
		case -1:
			if refIsSmaller {
				return refPrice, j
			}
			d = -d
		}
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
		abSurplus := utils.AbsInt(surplus)
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

func getTradePriceForMarketPressure(side int8, overlapped *[]OverLappedLevel,
	leastSurplus []int, refPrice float64, priceLimit float64) (int64, int) {
	lowerLimit := int64(math.Floor(refPrice * (1.0 - priceLimit)))
	i := leastSurplus[0] //largest
	if compareBuy(lowerLimit, (*overlapped)[i].Price) > 0 {
		// refPrice is larger than every one
		return (*overlapped)[i].Price, i
	}
	upperLimit := int64(math.Ceil(refPrice * (1.0 + priceLimit)))
	j := leastSurplus[len(leastSurplus)-1] //smallest
	if compareBuy((*overlapped)[j].Price, upperLimit) > 0 {
		// refPrice is less than every one
		return (*overlapped)[j].Price, j
	}
	if side == BUYSIDE {
		if compareBuy(upperLimit, (*overlapped)[i].Price) > 0 {
			return (*overlapped)[i].Price, i
		} else {
			return getPriceCloseToRef(*overlapped, leastSurplus, upperLimit)
		}
	} else {
		if compareBuy(lowerLimit, (*overlapped)[j].Price) < 0 {
			return (*overlapped)[j].Price, j
		} else {
			return getPriceCloseToRef(*overlapped, leastSurplus, lowerLimit)
		}
	}
}

func getTradePrice(overlapped *[]OverLappedLevel, maxExec *LevelIndex,
	leastSurplus *SurplusIndex, refPrice int64, priceLimitPct float64) (int64, int) {
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
		return getTradePriceForMarketPressure(BUYSIDE, overlapped,
			leastSurplus.index, float64(refPrice), priceLimitPct)
	}
	// only sell side surplus exist, selling pressure
	if !buySurplus && sellSurplus { // return lowest
		return getTradePriceForMarketPressure(SELLSIDE, overlapped,
			leastSurplus.index, float64(refPrice), priceLimitPct)
	}
	if (buySurplus && sellSurplus) || (!buySurplus && !sellSurplus) {
		return getPriceCloseToRef(*overlapped, leastSurplus.index, refPrice)
	}
	//never reach here
	return math.MaxInt64, -1
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
			buys[k].cumQty += trade
			sells[h].cumQty += trade
			me.Trades = append(me.Trades, Trade{sells[h].id, me.LastTradePrice, trade, buys[k].id})
			h++
		case r < 0:
			trade := buys[k].nxtTrade
			sells[h].nxtTrade -= trade
			buys[k].nxtTrade = 0
			buys[k].cumQty += trade
			sells[h].cumQty += trade
			me.Trades = append(me.Trades, Trade{sells[h].id, me.LastTradePrice, trade, buys[k].id})
			k++
		case r == 0:
			trade := sells[h].nxtTrade
			buys[k].nxtTrade = 0
			sells[h].nxtTrade = 0
			buys[k].cumQty += trade
			sells[h].cumQty += trade
			me.Trades = append(me.Trades, Trade{sells[h].id, me.LastTradePrice, trade, buys[k].id})
			h++
			k++
		}
	}
	me.overLappedLevel[i].BuyTotal = sumOrdersTotalLeft(buys, false)
	me.overLappedLevel[j].SellTotal = sumOrdersTotalLeft(sells, false)
}

// allocateResidual() assumes toAlloc is less than sum of quantity in orders.
// It would try best to evenly allocate toAlloc among orders in proportion of order qty meanwhile by whole lot
func allocateResidual(toAlloc *int64, orders []OrderPart, lotSize int64) bool {
	if len(orders) == 1 {
		qty := utils.MinInt(*toAlloc, orders[0].nxtTrade)
		orders[0].nxtTrade = qty
		*toAlloc -= qty
		return true
	}

	t := sumOrdersTotalLeft(orders, false)

	// orders should have the same time, sort here to get deterministic sequence
	sort.Slice(orders, func(i, j int) bool { return orders[i].id < orders[j].id })
	residual := *toAlloc

	if compareBuy(t, residual) > 0 { // not enough to allocate
		// It is assumed here toAlloc is lot size rounded, so that the below code
		// should leave nothing not allocated
		nLot := float64(residual / lotSize)
		totalF := float64(t)
		k := len(orders)
		for i := 0; i < k; i++ {
			a := int64(math.Floor(nLot*float64(orders[i].nxtTrade)/totalF)) * lotSize // this is supposed to be the main portion
			if compareBuy(a, residual) >= 0 {
				orders[i].nxtTrade = residual
				residual = 0
				break
			} else {
				orders[i].nxtTrade = a
				residual -= a
			}
		}
		remainderLot := residual / lotSize
		for i := 0; i < k; i++ {
			if remainderLot > 0 { // remainder distribution, every one can only get 1 lot or zero
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
// so that here is to 'reserve' the necessary qty from orders.
func (me *MatchEng) reserveQty(residual int64, orders []OrderPart) bool {
	//orders should be sorted by time already, since they are added as time sequence
	//no fill should happen on any in the 'orders' before this call, so that no other sorting happens
	// residual must be smaller than the total qty of all orders
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
						me.Book.RemoveOrder(o.id, BUYSIDE, p.Price)
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
						me.Book.RemoveOrder(o.id, SELLSIDE, p.Price)
					}
				}
			}
		}
	}

	return i
}
