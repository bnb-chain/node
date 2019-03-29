package order

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/types"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
	"github.com/binance-chain/node/plugins/dex/utils"
)

type transferEventType uint8

const (
	eventFilled transferEventType = iota
	eventFullyExpire
	eventPartiallyExpire
	eventIOCFullyExpire
	eventIOCPartiallyExpire
	eventFullyCancel
	eventPartiallyCancel
	eventCancelForMatchFailure
)

// Transfer represents a transfer between trade currencies
type Transfer struct {
	Oid        string
	eventType  transferEventType
	accAddress sdk.AccAddress
	inAsset    string
	in         int64
	outAsset   string
	out        int64
	unlock     int64
	Fee        types.Fee
	Trade      *me.Trade
	Symbol     string
}

func (tran Transfer) FeeFree() bool {
	return tran.eventType == eventPartiallyExpire ||
		tran.eventType == eventIOCPartiallyExpire ||
		tran.eventType == eventPartiallyCancel ||
		tran.eventType == eventCancelForMatchFailure
}

func (tran Transfer) IsExpire() bool {
	return tran.eventType == eventIOCFullyExpire ||
		tran.eventType == eventIOCPartiallyExpire ||
		tran.eventType == eventPartiallyExpire ||
		tran.eventType == eventFullyExpire
}

func (tran Transfer) IsExpiredWithFee() bool {
	return tran.eventType == eventFullyExpire || tran.eventType == eventIOCFullyExpire
}

func (tran *Transfer) String() string {
	return fmt.Sprintf("Transfer[eventType:%v, oid:%v, inAsset:%v, inQty:%v, outAsset:%v, outQty:%v, unlock:%v, fee:%v]",
		tran.eventType, tran.Oid, tran.inAsset, tran.in, tran.outAsset, tran.out, tran.unlock, tran.Fee)
}

func TransferFromTrade(trade me.Trade, symbol string, orderMap map[string]*OrderInfo) (Transfer, Transfer) {
	baseAsset, quoteAsset, _ := utils.TradingPair2Assets(symbol)
	seller := orderMap[trade.Sid].Sender
	buyer := orderMap[trade.Bid].Sender
	quoteQty := utils.CalBigNotionalInt64(trade.LastPx, trade.LastQty)
	unlock := utils.CalBigNotionalInt64(trade.OrigBuyPx, trade.BuyCumQty) - utils.CalBigNotionalInt64(trade.OrigBuyPx, trade.BuyCumQty-trade.LastQty)
	return Transfer{
			Oid:        trade.Sid,
			eventType:  eventFilled,
			accAddress: seller,
			inAsset:    quoteAsset,
			in:         quoteQty,
			outAsset:   baseAsset,
			out:        trade.LastQty,
			unlock:     trade.LastQty,
			Fee:        types.Fee{},
			Trade:      &trade,
			Symbol:     symbol,
		}, Transfer{
			Oid:        trade.Bid,
			eventType:  eventFilled,
			accAddress: buyer,
			inAsset:    baseAsset,
			in:         trade.LastQty,
			outAsset:   quoteAsset,
			out:        quoteQty,
			unlock:     unlock,
			Fee:        types.Fee{},
			Trade:      &trade,
			Symbol:     symbol,
		}
}

func TransferFromExpired(ord me.OrderPart, ordMsg OrderInfo) Transfer {
	var tranEventType transferEventType
	if ord.CumQty != 0 {
		if ordMsg.TimeInForce == TimeInForce.IOC {
			tranEventType = eventIOCPartiallyExpire // IOC partially filled
		} else {
			tranEventType = eventPartiallyExpire
		}
	} else {
		if ordMsg.TimeInForce == TimeInForce.IOC {
			tranEventType = eventIOCFullyExpire
		} else {
			tranEventType = eventFullyExpire
		}
	}

	return transferFromOrderRemoved(ord, ordMsg, tranEventType)
}

func TransferFromCanceled(ord me.OrderPart, ordMsg OrderInfo, isMatchFailure bool) Transfer {
	var tranEventType transferEventType
	if isMatchFailure {
		tranEventType = eventCancelForMatchFailure
	} else {
		if ord.CumQty != 0 {
			tranEventType = eventPartiallyCancel
		} else {
			tranEventType = eventFullyCancel
		}
	}

	return transferFromOrderRemoved(ord, ordMsg, tranEventType)
}

func transferFromOrderRemoved(ord me.OrderPart, ordMsg OrderInfo, tranEventType transferEventType) Transfer {
	//here is a trick to use the same currency as in and out ccy to simulate cancel
	qty := ord.LeavesQty()
	baseAsset, quoteAsset, _ := utils.TradingPair2Assets(ordMsg.Symbol)
	var unlock int64
	var unlockAsset string
	if ordMsg.Side == Side.BUY {
		unlockAsset = quoteAsset
		unlock = utils.CalBigNotionalInt64(ordMsg.Price, ordMsg.Quantity) - utils.CalBigNotionalInt64(ordMsg.Price, ordMsg.Quantity-qty)
	} else {
		unlockAsset = baseAsset
		unlock = qty
	}

	return Transfer{
		Oid:        ordMsg.Id,
		eventType:  tranEventType,
		accAddress: ordMsg.Sender,
		inAsset:    unlockAsset,
		in:         unlock,
		outAsset:   unlockAsset,
		out:        unlock,
		unlock:     unlock,
	}
}

type sortedAsset struct {
	native int64
	// coins are sorted.
	tokens sdk.Coins
}

// not thread safe
func (s *sortedAsset) addAsset(asset string, amt int64) {
	if asset == types.NativeTokenSymbol {
		s.native += amt
	} else {
		if s.tokens == nil {
			s.tokens = sdk.Coins{}
		}
		s.tokens = s.tokens.Plus(sdk.Coins{{Denom: asset, Amount: amt}})
	}
}
