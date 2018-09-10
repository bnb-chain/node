package order

// The types here are shared between order and pub package

type ChangeType int

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

/**
 * The data structure used for publication
 * Fields are come from different parts (abci order handler, matcheng, fee calc, expire handling, transfer logic)
 * 	within life cycle of an order
 * The main reason we need this "wrapper" type is
 * Field marked with [static] won't change after initialize/first set in OrderChange
 * Field marked with [dynamic] directly modify items in orderchangesmap, should only be updated in publishing go routine (but can initialized in abci EndBlock goroutine)
 *
 * OrderMsg 			- [static] init on new order msgs are added. The msg can come from live and during replay
 * TxHash 				- [static] tx of the NewOrderMsg and CancelOrderMsg, only added during we runMsg via order handler
 * Tpe 					- [static] type of order change, we only append new orderchange to `orderchanges` with new Tpe. The field in `orderchangesmap` doesn't need to change
 * Fee 					- [static] fee of specific order change.
 *							for trade fee, we update the field via pub.Trade's fee related fields.
 * 							for iocnofill/expire/cancel with no execution, we update the field via callback in transfer process
 * FeeAsset 			- [static] same with Fee
 * LeavesQty 			- [dynamic] accumulate leaves qty of an order
 * CumQty 				- [dynamic] accumulate executed qty of an order - ideally should be OrderMsg.Quantiyt - LeavesQty
 * CumQuoteAssetQty 	- [dynamic] accumulate cumulative quote asset quantity, should be historical
 * creationTime 		- [static] timestamp of block created this order change
 */
type OrderChange struct {
	OrderMsg         NewOrderMsg // we need maintain a copy of NewOrderMsg in addition to kp.allOrders because on order removal (expire or cancel), the NewOrderMsg would be deleted before we publish
	TxHash           string      // TODO(#66): cannot recover from restart:(
	Tpe              ChangeType
	Fee              int64
	FeeAsset         string
	LeavesQty        int64
	CumQty           int64
	CumQuoteAssetQty int64 // TODO(#66): cannot recover from restart for buy order:(
	creationTime     int64 // TODO(#66): cannot recover from restart:(
}

func (o *OrderChange) SetCreationTime(t int64) {
	o.creationTime = t
}

func (o *OrderChange) CreationTime() int64 {
	return o.creationTime
}

// provide an easy way to retrieve order related static fields during generate executed order status
type OrderChangesMap map[string]*OrderChange
type OrderChanges []OrderChange // clean after publish each block's EndBlock and before next block's BeginBlock

type ChangedPriceLevels map[string]ChangedPriceLevelsPerSymbol

type ChangedPriceLevelsPerSymbol struct {
	Buys  map[int64]int64
	Sells map[int64]int64
}
