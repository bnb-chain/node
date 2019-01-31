package matcheng

import (
	"math"
	"sort"

	"github.com/binance-chain/node/common/utils"
)

type LevelIndex struct {
	value float64
	index []int
}

func (li *LevelIndex) clear() {
	li.value = 0
	li.index = li.index[:0]
}

type SurplusIndex struct {
	LevelIndex
	surplus []int64
}

func (li *SurplusIndex) clear() {
	li.value = math.MaxFloat64
	li.index = li.index[:0]
	li.surplus = li.surplus[:0]
}

//sumOrdersTotalLeft() returns the total value left that can be traded in this block round.
//reCalNxtTrade should be true at the beginning and false when nxtTrade is changed by allocation logic
func sumOrdersTotalLeft(orders []OrderPart, reCalNxtTrade bool) int64 {
	var s int64
	k := len(orders)
	for i := 0; i < k; i++ {
		o := &orders[i]
		if reCalNxtTrade {
			o.nxtTrade = o.Qty - o.CumQty
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
	return math.MaxInt64, -1
}

// allocateResidual() assumes toAlloc is less than sum of quantity in orders.
// It would try best to evenly allocate toAlloc among orders in proportion of order qty meanwhile by whole lot
// Due to lotsize change, it is possible the order would not be allocated with a full lot.
func allocateResidual(toAlloc *int64, orders []OrderPart, lotSize int64) bool {
	if len(orders) == 1 {
		qty := math.Min(*toAlloc, orders[0].nxtTrade)
		orders[0].nxtTrade = qty
		*toAlloc -= qty
		return true
	}

	t := sumOrdersTotalLeft(orders, false)

	// orders should have the same time, sort here to get deterministic sequence
	sort.Slice(orders, func(i, j int) bool { return orders[i].Id < orders[j].Id })
	residual := *toAlloc
	halfLot := lotSize / 2

	if compareBuy(t, residual) > 0 { // not enough to allocate
		// It is assumed here toAlloc is lot size rounded, so that the below code
		// should leave nothing not allocated
		nLot := math.Floor((residual + halfLot) / lotSize)
		k := len(orders)
		i := 0
		for i = 0; i < k; i++ {
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
		for j := i % k; j < k; j++ {
			if residual > lotSize { // remainder distribution, every one can only get 1 lot or zero
				orders[j].nxtTrade += lotSize
				residual -= lotSize
				if j == k-1 { //restart from the beginning
					i = 0
				}
			} else { // residual may has odd lot remainder
				orders[j].nxtTrade += residual
				residual = 0
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
