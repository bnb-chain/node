package order

import (
	"fmt"

	"github.com/binance-chain/node/common/types"
	me "github.com/binance-chain/node/plugins/dex/matcheng"
)

// The types here are shared between order and pub package

type ChangeType uint8

const (
	Ack ChangeType = iota
	Canceled
	Expired
	IocNoFill
	PartialFill
	FullyFill
)

func (this ChangeType) String() string {
	switch this {
	case Ack:
		return "Ack"
	case Canceled:
		return "Canceled"
	case Expired:
		return "Expired"
	case IocNoFill:
		return "IocNoFill"
	case PartialFill:
		return "PartialFill"
	case FullyFill:
		return "FullyFill"
	default:
		return "Unknown"
	}
}

type ExecutionType uint8

const (
	NEW ExecutionType = iota
)

func (this ExecutionType) String() string {
	switch this {
	case NEW:
		return "NEW"
	default:
		return "Unknown"
	}
}

type OrderChange struct {
	Id  string
	Tpe ChangeType
}

func (oc OrderChange) String() string {
	return fmt.Sprintf("id: %s, tpe: %s", oc.Id, oc.Tpe.String())
}

// provide an easy way to retrieve order related static fields during generate executed order status
type OrderInfoForPublish map[string]*OrderInfo
type OrderChanges []OrderChange // clean after publish each block's EndBlock and before next block's BeginBlock

type ChangedPriceLevelsMap map[string]ChangedPriceLevelsPerSymbol

type ChangedPriceLevelsPerSymbol struct {
	Buys  map[int64]int64
	Sells map[int64]int64
}

type TradeHolder struct {
	OId    string
	Trade  *me.Trade
	Symbol string
}

func (fh TradeHolder) String() string {
	return fmt.Sprintf("oid: %s, bid: %s, sid: %s", fh.OId, fh.Trade.Bid, fh.Trade.Sid)
}

type ExpireHolder struct {
	OrderId string
}

type FeeHolder map[string]*types.Fee
