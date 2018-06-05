package matcheng

import "math"

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

type MatchEng struct {
	Book            OrderBookInterface
	overLappedLevel []OverLappedLevel //buffer
	maxExec         LevelIndex
	maxSurplus      LevelIndex
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
	return 0
}

func getPriceCloseToRef(overlapped *[]OverLappedLevel, index []int, refPrice float64) float64 {
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
	return (*overlapped)[j].Price
}

func getTradePrice(overlapped *[]OverLappedLevel, maxExec *LevelIndex,
	maxSurplus *SurplusIndex, refPrice float64) float64 {
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
		return (*overlapped)[maxExec.index[0]].Price
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
		return (*overlapped)[maxSurplus.index[0]].Price
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
		return (*overlapped)[maxSurplus.index[len(maxSurplus.index)-1]].Price
	}
	if !buy && sell { // return hightest
		return (*overlapped)[maxSurplus.index[0]].Price
	}
	if (buy && sell) || (!buy && !sell) {
		return getPriceCloseToRef(overlapped, maxSurplus.index, refPrice)
	}
	return math.MaxFloat64
}
