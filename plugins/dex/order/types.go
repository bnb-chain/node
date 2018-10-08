package order

import "fmt"

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
	Id       string
	Tpe      ChangeType
	Fee      int64
	FeeAsset string
}

// provide an easy way to retrieve order related static fields during generate executed order status
type OrderInfoForPublish map[string]*OrderInfo
type OrderChanges []OrderChange // clean after publish each block's EndBlock and before next block's BeginBlock

type ChangedPriceLevelsMap map[string]ChangedPriceLevelsPerSymbol

type ChangedPriceLevelsPerSymbol struct {
	Buys  map[int64]int64
	Sells map[int64]int64
}

type TradeFeeHolder struct {
	SId  string
	BId  string // still need two ids here to link with correct trade
	Side int8
	Fee
}

func (fh TradeFeeHolder) String() string {
	return fmt.Sprintf("sid: %s, bid: %s, side: %s, fee: %s", fh.SId, fh.BId, IToSide(fh.Side), fh.Fee)
}

type ExpireFeeHolder struct {
	OrderId string
	Fee
}

func (fh ExpireFeeHolder) String() string {
	return fmt.Sprintf("order: %s, fee: %s", fh.OrderId, fh.Fee)
}

type Fee struct {
	Amount int64
	Asset  string
}

func (fee Fee) String() string {
	return fmt.Sprintf("%d%s", fee.Amount, fee.Asset)
}
