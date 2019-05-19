package matcheng

import (
	"fmt"
	"github.com/pkg/errors"
	"sort"

	"github.com/binance-chain/node/common/utils"
)

const upgradeHeight = 1

func (me *MatchEng) Match(height int64) bool {
	if height < upgradeHeight {
		return me.MatchDeprecated()
	}

	me.Trades = me.Trades[:0]
	r := me.Book.GetOverlappedRange(&me.overLappedLevel, &me.buyBuf, &me.sellBuf)
	if r <= 0 {
		return true
	}
	prepareMatch(&me.overLappedLevel)
	tradePrice, index := getTradePrice(&me.overLappedLevel, &me.maxExec, &me.leastSurplus, me.LastTradePrice, me.PriceLimitPct)
	if index < 0 {
		return false
	}
	me.LastTradePrice = tradePrice

	// 1. drop redundant qty
	// 2. rearrange the orders by their trade price and time.
	// 3. fill orders and generate trades
	if err := me.dropRedundantQty(index); err != nil {
		me.logger.Error("dropRedundantQty failed", "error", err)
		return false
	}
	makerTakerOrders, err := createMakerTakerOrders(height, me.overLappedLevel, tradePrice, index)
	if err != nil {
		me.logger.Error("createMakerTakerOrders failed", "error", err)
		return false
	}
	me.fillOrdersNew(makerTakerOrders)
	return true
}

func (me *MatchEng) dropRedundantQty(tradePriceLevelIdx int) error {
	tradePriceLevel := me.overLappedLevel[tradePriceLevelIdx]
	totalExec := tradePriceLevel.AccumulatedExecutions
	qBuy := tradePriceLevel.AccumulatedBuy
	qSell := tradePriceLevel.AccumulatedSell
	if qBuy == qSell {
		return nil
	}

	if compareBuy(qBuy, totalExec) > 0 {
		for i := tradePriceLevelIdx; i >= 0; i-- {
			// it can be proved that redundant qty only exists in the last line of the overlapped buy price level
			if me.overLappedLevel[i].BuyTotal != 0 {
				return dropRedundantQty(me.overLappedLevel[i].BuyOrders, qBuy-totalExec, me.LotSize)
			}
		}
	} else if compareBuy(qSell, totalExec) > 0 {
		length := len(me.overLappedLevel)
		for i := tradePriceLevelIdx; i < length; i++ {
			// it can be proved that redundant qty only exists in the first line of the overlapped sell price level
			if me.overLappedLevel[i].SellTotal != 0 {
				return dropRedundantQty(me.overLappedLevel[i].SellOrders, qSell-totalExec, me.LotSize)
			}
		}
	}
	return nil
}

// assume the `orders` are sorted by time
func dropRedundantQty(orders []OrderPart, toDropQty, lotSize int64) error {
	if toDropQty <= 0 {
		return fmt.Errorf("invalid quantity to drop, toDropQty=%v", toDropQty)
	}
	n := len(orders)
	if n == 0 {
		return fmt.Errorf("no orders found, toDropQty=%v", toDropQty)
	}
	totalQty := sumOrdersTotalLeft(orders, false)
	if totalQty < toDropQty {
		return fmt.Errorf("no enough quantity can be dropped, toDropQty=%v, totalQty=%v", toDropQty, totalQty)
	}

	residual := totalQty - toDropQty
	currTime := orders[0].Time
	currStartIdx := 0
	for i := 0; i < n; i++ {
		order := &orders[i]
		if residual <= 0 {
			order.nxtTrade = 0
			continue
		}
		if order.Time != currTime {
			if ok := allocateResidual(&residual, orders[currStartIdx:i], lotSize); !ok {
				return fmt.Errorf("allocate residual failed, residual=%v", residual)
			}
			currStartIdx = i
			currTime = order.Time
		}
		if i == n-1 {
			if ok := allocateResidual(&residual, orders[currStartIdx:], lotSize); !ok {
				return fmt.Errorf("allocate residual failed, residual=%v", residual)
			}
		}
	}


	return nil
}

type MakerTakerOrders struct {
	isBuySideMaker bool
	makerSide      MakerSideOrders
	takerSide      TakerSideOrders
}

func createMakerTakerOrders(height int64, overlapped []OverLappedLevel, concludedPrice int64, tradePriceLevelIdx int) (*MakerTakerOrders, error) {
	//makerSide := SELLSIDE // if no maker orders, use SELLSIDE as the maker side. No impact on the final result.

	var makerTakerOrders MakerTakerOrders
	buySideIsMakerSide, mergedBuyLevels := mergeSidePriceLevels(BUYSIDE, height, concludedPrice, tradePriceLevelIdx, overlapped)
	sellSideIsMakerSide, mergedSellLevels := mergeSidePriceLevels(SELLSIDE, height, concludedPrice, tradePriceLevelIdx, overlapped)

	if buySideIsMakerSide && sellSideIsMakerSide {
		// impossible, never reach here
		return nil, errors.New("both buy side and sell side have maker orders.")
	} else if buySideIsMakerSide && !sellSideIsMakerSide {
		makerTakerOrders.isBuySideMaker = true
		makerTakerOrders.makerSide = MakerSideOrders{mergedBuyLevels}
		makerTakerOrders.takerSide = TakerSideOrders{mergedSellLevels[0]}
	} else {
		makerTakerOrders.isBuySideMaker = false
		// buySideIsMakerSide == false
		makerTakerOrders.makerSide = MakerSideOrders{mergedSellLevels}
		makerTakerOrders.takerSide = TakerSideOrders{mergedBuyLevels[0]}
		// note if both buy side and sell side are taker side, that means no leftover orders would be matched in this round.
		// choosing whichever side to be the maker side is fine, since all of orders will be applied with the same price(concluded price).
	}

	return &makerTakerOrders, nil
}

func mergeSidePriceLevels(side int8, height int64, concludedPrice int64, tradePriceLevelIdx int, levels []OverLappedLevel) (isMakerSide bool, mergedLevels []*MergedPriceLevel) {
	makerLevels := make([]*MergedPriceLevel, 0)
	// concludedPriceLevel is mixed with taker orders and some maker orders whose price is equal to the concludedPrice
	concludedPriceLevel := NewMergedPriceLevel(concludedPrice)
	levelsLength := len(levels)
	// merge from the better price to worse
	if side == BUYSIDE {
		for i := 0; i <= tradePriceLevelIdx; i++ {
			mergeOnePriceLevel(side, height, &levels[i], &makerLevels, concludedPriceLevel)
		}
	} else {
		for i := levelsLength - 1; i >= tradePriceLevelIdx; i-- {
			mergeOnePriceLevel(side, height, &levels[i], &makerLevels, concludedPriceLevel)
		}
	}

	if len(makerLevels) == 0 {
		// it's taker side
		return false, []*MergedPriceLevel{concludedPriceLevel}
	}

	// it's maker side
	if concludedPriceLevel.totalQty != 0 {
		// it's an optimization to merge concludedPriceLevel into the worst maker level if possible
		// last maker level has the worst price
		lastMakerLevel := makerLevels[len(makerLevels)-1]
		if compareBuy(lastMakerLevel.price, concludedPrice) == 0 {
			lastMakerLevel.AddOrders(concludedPriceLevel.orders)
		} else {
			makerLevels = append(makerLevels, concludedPriceLevel)
		}
	}
	return true, makerLevels
}

func mergeOnePriceLevel(side int8, height int64, priceLevel *OverLappedLevel,
	makerLevels *[]*MergedPriceLevel, concludedPriceLevel *MergedPriceLevel) {
	var orders []OrderPart
	if side == BUYSIDE {
		orders = priceLevel.BuyOrders
	} else {
		orders = priceLevel.SellOrders
	}
	if len(orders) == 0 {
		return
	}

	sortOrders := func(o []*OrderPart) {
		sort.SliceStable(o, func(i, j int) bool {
			return o[i].nxtTrade > o[j].nxtTrade
		})
	}

	takerOrders := make([]*OrderPart, 0)
	makerOrders := make([]*OrderPart, 0)
	for i, order := range orders {
		if order.nxtTrade <= 0 {
			// cannot "break". In some edge cases, we may have such nxtTrade sequence: 2, 0, 5
			continue
		}
		if order.Time < height {
			makerOrders = append(makerOrders, &orders[i])
		} else {
			takerOrders = append(takerOrders, &orders[i])
		}
	}

	if len(makerOrders) != 0 {
		makerLevel := NewMergedPriceLevel(priceLevel.Price)
		sortOrders(makerOrders)
		makerLevel.AddOrders(makerOrders)
		*makerLevels = append(*makerLevels, makerLevel)
	}

	if len(takerOrders) != 0 {
		sortOrders(takerOrders)
		concludedPriceLevel.AddOrders(takerOrders)
	}
}

func (me *MatchEng) fillOrdersNew(makerTakerOrders *MakerTakerOrders) {
	takers := makerTakerOrders.takerSide.orders
	totalTakerQty := makerTakerOrders.takerSide.totalQty
	nTakers := len(takers)
	// we need to keep a copy of original nxtTrade as order.nxtTrade would be changed when filled
	proportion := make([]int64, nTakers)
	for i := 0; i < nTakers; i++ {
		proportion[i] = takers[i].nxtTrade
	}

	for _, makerLevel := range makerTakerOrders.makerSide.priceLevels {
		toFillQty := calcFillQty(makerLevel.totalQty, takers, proportion, totalTakerQty, me.LotSize)
		makers := makerLevel.orders
		nMakers := len(makers)
		for mIndex, tIndex := 0,0; mIndex < nMakers && tIndex < nTakers; {
			maker := makers[mIndex]
			taker := takers[tIndex]
			if compareBuy(maker.nxtTrade, 0) == 0 {
				mIndex++
				continue
			}
			if compareBuy(toFillQty[tIndex], 0) == 0 {
				tIndex++
				continue
			}

			filledQty := utils.MinInt(maker.nxtTrade, toFillQty[tIndex])
			toFillQty[tIndex] -= filledQty
			taker.nxtTrade -= filledQty
			taker.CumQty += filledQty
			maker.nxtTrade -= filledQty
			maker.CumQty += filledQty
			trade := Trade{
				LastPx:  makerLevel.price,
				LastQty: filledQty,
			}
			if makerTakerOrders.isBuySideMaker {
				trade.Sid, trade.Bid = taker.Id, maker.Id
				trade.SellCumQty, trade.BuyCumQty = taker.CumQty, maker.CumQty
			} else {
				trade.Sid, trade.Bid = maker.Id, taker.Id
				trade.SellCumQty, trade.BuyCumQty = maker.CumQty, taker.CumQty
			}
			me.Trades = append(me.Trades, trade)
		}
	}
}

// the logic is similar to `allocateResidual`.
func calcFillQty(makerQty int64, takers []*OrderPart, proportion []int64, totalTakerQty int64, lotSize int64) ([]int64) {
	residual := makerQty
	nLot := residual / lotSize
	n := len(takers)
	fillQty := make([]int64, n)
	for i := 0; i < n; i++ {
		nxtTrade := lotSize * calcNumOfLot(nLot, proportion[i], totalTakerQty)
		nxtTrade = utils.MinInt(nxtTrade, takers[i].nxtTrade)
		// we must have nxtTrade < residual
		residual -= nxtTrade
		fillQty[i] = nxtTrade
	}

	for i := 0; residual > 0; i = (i + 1) % n {
		order := takers[i]
		toAdd := utils.MinInt(order.nxtTrade-fillQty[i], utils.MinInt(residual, lotSize))
		residual -= toAdd
		fillQty[i] += toAdd
	}
	return fillQty
}
