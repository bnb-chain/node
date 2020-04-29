package order

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The types here are shared between order and pub package

type ChangeType uint8

const (
	Ack            ChangeType = iota // new order tx
	Canceled                         // cancel order tx
	Expired                          // expired for gte order
	IocNoFill                        // ioc order is not filled expire
	IocExpire                        // ioc order is partial filled expire
	PartialFill                      // order is partial filled, derived from trade
	FullyFill                        // order is fully filled, derived from trade
	FailedBlocking                   // order tx is failed blocking, we only publish essential message
	FailedMatching                   // order failed matching
)

// True for should not remove order in these status from OrderInfoForPub
// False for remove
func (tpe ChangeType) IsOpen() bool {
	// FailedBlocking tx doesn't effect OrderInfoForPub, should not be put into closedToPublish
	return tpe == Ack ||
		tpe == PartialFill ||
		tpe == FailedBlocking
}

func (tpe ChangeType) String() string {
	switch tpe {
	case Ack:
		return "Ack"
	case Canceled:
		return "Canceled"
	case Expired:
		return "Expired"
	case IocNoFill:
		return "IocNoFill"
	case IocExpire:
		return "IocExpire"
	case PartialFill:
		return "PartialFill"
	case FullyFill:
		return "FullyFill"
	case FailedBlocking:
		return "FailedBlocking"
	case FailedMatching:
		return "FailedMatching"
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
	Id             string
	Tpe            ChangeType
	SingleFee      string
	MsgForFailedTx interface{} // pointer to NewOrderMsg or CancelOrderMsg
}

func (oc OrderChange) String() string {
	return fmt.Sprintf("id: %s, tpe: %s", oc.Id, oc.Tpe.String())
}

func (oc OrderChange) failedBlockingMsg() *OrderInfo {
	switch msg := oc.MsgForFailedTx.(type) {
	case NewOrderMsg:
		return &OrderInfo{
			NewOrderMsg: msg,
		}
	case CancelOrderMsg:
		return &OrderInfo{
			NewOrderMsg: NewOrderMsg{Sender: msg.Sender, Id: msg.RefId, Symbol: msg.Symbol},
		}
	default:
		return nil
	}
}

func (oc OrderChange) ResolveOrderInfo(orderInfos OrderInfoForPublish) *OrderInfo {
	switch oc.Tpe {
	case FailedBlocking:
		return oc.failedBlockingMsg()
	default:
		return orderInfos[oc.Id]
	}
}

// provide an easy way to retrieve order related static fields during generate executed order status
type OrderInfoForPublish map[string]*OrderInfo
type OrderChanges []OrderChange // clean after publish each block's EndBlock and before next block's BeginBlock

type ChangedPriceLevelsMap map[string]ChangedPriceLevelsPerSymbol

type ChangedPriceLevelsPerSymbol struct {
	Buys  map[int64]int64
	Sells map[int64]int64
}

type ExpireHolder struct {
	OrderId string
	Reason  ChangeType
	Fee     string
}

type FeeHolder map[string]*sdk.Fee
