package matcheng

import (
	"math"
	"math/big"
	"sort"

	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
)

type LevelIndex struct {
	value int64
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
	li.value = math.MaxInt64
	li.index = li.index[:0]
	li.surplus = li.surplus[:0]
}

//sumOrdersTotalLeft() returns the total value left that can be traded in this block round.
//reCalNxtTrade should be true at the beginning and false when nxtTrade is changed by allocation logic
//note: the result would never overflow because we have checked when place order.
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
	var accum int64
	k := len(*overlapped)
	for i := k - 1; i >= 0; i-- {
		l := &(*overlapped)[i]
		l.SellTotal = sumOrdersTotalLeft(l.SellOrders, true)
		if accum+l.SellTotal < 0 {
			// overflow
			// actually, for sell orders, we would never reach here because of the limit of total supply
			accum = math.MaxInt64
		} else {
			accum += l.SellTotal
		}
		l.AccumulatedSell = accum
	}
	accum = 0
	for i := 0; i < k; i++ {
		l := &(*overlapped)[i]
		l.BuyTotal = sumOrdersTotalLeft(l.BuyOrders, true)
		if accum+l.BuyTotal < 0 {
			// overflow, it's safe to use MaxInt64 because the final execution would never exceed the total supply of the base asset
			accum = math.MaxInt64
		} else {
			accum += l.BuyTotal
		}
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
	if buySurplus && !sellSurplus { // return highest
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

// allocateResidual() assumes toAlloc is less than sum of quantity in orders.
// It would try best to evenly allocate toAlloc among orders in proportion of order qty meanwhile by whole lot
// Due to lotsize change, it is possible the order would not be allocated with a full lot.
func allocateResidual(toAlloc *int64, orders []OrderPart, lotSize int64) bool {
	if len(orders) == 1 {
		qty := utils.MinInt(*toAlloc, orders[0].nxtTrade)
		orders[0].nxtTrade = qty
		*toAlloc -= qty
		return true
	}

	t := sumOrdersTotalLeft(orders, false)

	upgrade.FixOrderSeqInPriceLevel(func() {
		sort.Slice(orders, func(i, j int) bool { return orders[i].Id < orders[j].Id })
	}, nil, nil)

	residual := *toAlloc

	if compareBuy(t, residual) > 0 { // not enough to allocate
		// It is assumed here toAlloc is lot size rounded, so that the below code
		// should leave nothing not allocated
		nLot := residual / lotSize
		k := len(orders)
		i := 0
		for i = 0; i < k; i++ {
			a := calcNumOfLot(nLot, orders[i].nxtTrade, t) * lotSize // this is supposed to be the main portion
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

// totalLot * orderLeft / totalLeft, orderLeft <= totalLeft
func calcNumOfLot(totalLot, orderLeft, totalLeft int64) int64 {
	if tmp, ok := utils.Mul64(totalLot, orderLeft); ok {
		return tmp / totalLeft
	} else {
		var res big.Int
		res.Quo(res.Mul(big.NewInt(totalLot), big.NewInt(orderLeft)), big.NewInt(totalLeft))
		return res.Int64()
	}
}
