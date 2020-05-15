package matcheng

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/upgrade"
	"github.com/binance-chain/node/common/utils"
)

func (me *MatchEng) Match(height int64, lastMatchedHeight int64) bool {
	if !sdk.IsUpgrade(upgrade.BEP19) {
		return me.MatchBeforeGalileo(height)
	}
	me.logger.Debug("match starts...", "height", height)
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

	if err := me.dropRedundantQty(index); err != nil {
		me.logger.Error("dropRedundantQty failed", "error", err)
		return false
	}
	//If order height > the last Match height, then it's maker.
	// Block Height cannot be used here since mini-token is not matched in every block
	takerSide, err := me.determineTakerSide(lastMatchedHeight, index)
	if err != nil {
		me.logger.Error("determineTakerSide failed", "error", err)
		return false
	}
	takerSideOrders := mergeTakerSideOrders(takerSide, tradePrice, me.overLappedLevel, index)
	surplus := me.overLappedLevel[index].BuySellSurplus
	me.fillOrdersNew(takerSide, takerSideOrders, index, tradePrice, surplus)
	me.LastTradePrice = tradePrice
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
			// it can be proved that redundant qty only exists in the last non-empty line of the overlapped buy price level
			if me.overLappedLevel[i].BuyTotal != 0 {
				return dropRedundantQty(me.overLappedLevel[i].BuyOrders, qBuy-totalExec, me.LotSize)
			}
		}
	} else if compareBuy(qSell, totalExec) > 0 {
		length := len(me.overLappedLevel)
		for i := tradePriceLevelIdx; i < length; i++ {
			// it can be proved that redundant qty only exists in the first non-empty line of the overlapped sell price level
			if me.overLappedLevel[i].SellTotal != 0 {
				return dropRedundantQty(me.overLappedLevel[i].SellOrders, qSell-totalExec, me.LotSize)
			}
		}
	}
	return fmt.Errorf("internal error! invalud AccumulatedExecutions found, "+
		"AccumulatedBuy=%v, AccumulatedSell=%v, AccumulatedExecutions=%v", qBuy, qSell, totalExec)
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
		if order.Time != currTime {
			if ok := allocateResidual(&residual, orders[currStartIdx:i], lotSize); !ok {
				return fmt.Errorf("allocate residual failed, residual=%v", residual)
			}
			if residual <= 0 {
				for ; i < n; i++ {
					orders[i].nxtTrade = 0
				}
				return nil
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

func findTakerStartIdx(lastMatchHeight int64, orders []OrderPart) (idx int, makerTotal int64) {
	i, k := 0, len(orders)
	for ; i < k; i++ {
		if orders[i].Time <= lastMatchHeight {
			makerTotal += orders[i].nxtTrade
		} else {
			return i, makerTotal
		}
	}
	return i, makerTotal
}

func (me *MatchEng) determineTakerSide(lastMatchHeight int64, tradePriceIdx int) (int8, error) {
	makerSide := UNKNOWN
	for i := 0; i <= tradePriceIdx; i++ {
		l := &me.overLappedLevel[i]
		l.BuyTakerStartIdx, l.BuyMakerTotal = findTakerStartIdx(lastMatchHeight, l.BuyOrders)
		if l.HasBuyMaker() {
			makerSide = BUYSIDE
		}
	}

	for i := len(me.overLappedLevel) - 1; i >= tradePriceIdx; i-- {
		l := &me.overLappedLevel[i]
		l.SellTakerStartIdx, l.SellMakerTotal = findTakerStartIdx(lastMatchHeight, l.SellOrders)
		if l.HasSellMaker() {
			if makerSide == BUYSIDE {
				return UNKNOWN, errors.New("both buy side and sell side have maker orders.")
			}
			makerSide = SELLSIDE
		}
	}

	if makerSide == BUYSIDE {
		return SELLSIDE, nil
	} else {
		// UNKNOWN or SELLSIDE
		// if no maker orders, choose SELLSIDE as the maker side. No impact on the final result.
		return BUYSIDE, nil
	}
}

func mergeTakerSideOrders(side int8, concludedPrice int64, overlapped []OverLappedLevel, tradePriceIdx int) TakerSideOrders {
	merged := NewMergedPriceLevel(concludedPrice)
	if side == BUYSIDE {
		for i := 0; i <= tradePriceIdx; i++ {
			mergeOneTakerLevel(side, &overlapped[i], merged)
		}
	} else {
		for i := len(overlapped) - 1; i >= tradePriceIdx; i-- {
			mergeOneTakerLevel(side, &overlapped[i], merged)
		}
	}
	return TakerSideOrders{merged}
}

func mergeOneTakerLevel(side int8, priceLevel *OverLappedLevel, merged *MergedPriceLevel) {
	var orders []OrderPart
	if side == BUYSIDE {
		orders = priceLevel.BuyOrders[priceLevel.BuyTakerStartIdx:]
	} else {
		orders = priceLevel.SellOrders[priceLevel.SellTakerStartIdx:]
	}
	if len(orders) == 0 {
		return
	}

	sortOrders := func(o []*OrderPart) {
		sort.SliceStable(o, func(i, j int) bool {
			return o[i].Qty > o[j].Qty
		})
	}

	takerOrders := make([]*OrderPart, 0)
	for i, order := range orders {
		if order.nxtTrade > 0 {
			takerOrders = append(takerOrders, &orders[i])
		}
		// else cannot "break". In some edge cases, we may have such nxtTrade sequence: 2, 0, 5
	}

	if len(takerOrders) != 0 {
		sortOrders(takerOrders)
		merged.AddOrders(takerOrders)
	}
}

func (me *MatchEng) fillOrdersNew(takerSide int8, takerSideOrders TakerSideOrders, tradePriceIdx int, concludedPrice, surplus int64) {
	takers := takerSideOrders.orders
	totalTakerQty := takerSideOrders.totalQty
	nTakers := len(takers)
	// we need to keep a copy of original nxtTrade as order.nxtTrade would be changed when filled
	proportion := make([]int64, nTakers)
	for i := 0; i < nTakers; i++ {
		proportion[i] = takers[i].nxtTrade
	}

	genTrades := func(makers []OrderPart, makerPrice int64, toFillQty []int64) {
		nMakers := len(makers)
		for mIndex, tIndex := 0, 0; mIndex < nMakers && tIndex < nTakers; {
			maker := &makers[mIndex]
			if compareBuy(maker.nxtTrade, 0) == 0 {
				mIndex++
				continue
			}
			taker := takers[tIndex]
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
				LastPx:  makerPrice,
				LastQty: filledQty,
			}
			if surplus < 0 {
				trade.TickType = SellSurplus
			} else if surplus > 0 {
				trade.TickType = BuySurplus
			} else {
				trade.TickType = Neutral
			}

			if takerSide == SELLSIDE {
				trade.Sid, trade.Bid = taker.Id, maker.Id
				trade.SellCumQty, trade.BuyCumQty = taker.CumQty, maker.CumQty
				if maker.Time < taker.Time {
					trade.TickType = SellTaker
				}
			} else {
				trade.Sid, trade.Bid = maker.Id, taker.Id
				trade.SellCumQty, trade.BuyCumQty = maker.CumQty, taker.CumQty
				if maker.Time < taker.Time {
					trade.TickType = BuyTaker
				}
			}
			me.Trades = append(me.Trades, trade)
		}
	}

	// always reuse this slice to avoid malloc
	toFillQty := make([]int64, nTakers)
	if takerSide == SELLSIDE {
		// first round is for maker orders
		for i := 0; i <= tradePriceIdx; i++ {
			overlapped := &me.overLappedLevel[i]
			if !overlapped.HasBuyMaker() || overlapped.Price == concludedPrice {
				continue
			}
			calcFillQty(toFillQty, overlapped.BuyMakerTotal, takers, proportion, totalTakerQty, me.LotSize)
			genTrades(overlapped.BuyOrders[:overlapped.BuyTakerStartIdx], overlapped.Price, toFillQty)
		}
		// second round for taker orders
		for j := 0; j < nTakers; j++ {
			toFillQty[j] = takers[j].nxtTrade
		}
		for i := 0; i <= tradePriceIdx; i++ {
			overlapped := &me.overLappedLevel[i]
			genTrades(overlapped.BuyOrders, concludedPrice, toFillQty)
		}
	} else {
		// first round is for maker orders
		for i := len(me.overLappedLevel) - 1; i >= tradePriceIdx; i-- {
			overlapped := me.overLappedLevel[i]
			if !overlapped.HasSellMaker() || overlapped.Price == concludedPrice {
				continue
			}
			calcFillQty(toFillQty, overlapped.SellMakerTotal, takers, proportion, totalTakerQty, me.LotSize)
			genTrades(overlapped.SellOrders[:overlapped.SellTakerStartIdx], overlapped.Price, toFillQty)
		}
		// second round for taker orders
		for j := 0; j < nTakers; j++ {
			toFillQty[j] = takers[j].nxtTrade
		}
		for i := len(me.overLappedLevel) - 1; i >= tradePriceIdx; i-- {
			overlapped := &me.overLappedLevel[i]
			genTrades(overlapped.SellOrders, concludedPrice, toFillQty)
		}
	}
}

// the logic is similar to `allocateResidual`.
func calcFillQty(toFillQty []int64, makerQty int64, takers []*OrderPart, proportion []int64, totalTakerQty int64, lotSize int64) {
	residual := makerQty
	nLot := residual / lotSize
	n := len(takers)
	for i := 0; i < n; i++ {
		nxtTrade := lotSize * calcNumOfLot(nLot, proportion[i], totalTakerQty)
		nxtTrade = utils.MinInt(nxtTrade, takers[i].nxtTrade)
		// we must have nxtTrade < residual
		residual -= nxtTrade
		toFillQty[i] = nxtTrade
	}

	for i := 0; residual > 0; i = (i + 1) % n {
		order := takers[i]
		toAdd := utils.MinInt(order.nxtTrade-toFillQty[i], utils.MinInt(residual, lotSize))
		residual -= toAdd
		toFillQty[i] += toAdd
	}
}
