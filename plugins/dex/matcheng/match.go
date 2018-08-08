package matcheng

import (
	"math"
	"sort"
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
//recalNxtTrade should be true at the begining and false when nxtTrade is changed by allocation logic
func sumOrdersTotalLeft(orders []OrderPart, recalNxtTrade bool) float64 {
	var s float64
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
func allocateResidual(toAlloc *float64, orders []OrderPart, lotSize float64) bool {
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
