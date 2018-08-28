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

type OrderChange struct {
	OrderMsg  NewOrderMsg // we need maintain a copy of NewOrderMsg in addition to kp.allOrders because on order removal (expire or cancel), the NewOrderMsg would be deleted before we publish
	Tpe       ChangeType
	Fee       int64
	LeavesQty int64
	CumQty    int64
}

type OrderChangesMap map[string]*OrderChange // provide an easy way to retrieve order related fields during generate (partial) filled order status, clean with OrderChanges
type OrderChanges []OrderChange              // clean after publish each block's EndBlock and before next block's BeginBlock



type ChangedPriceLevels map[string]ChangedPriceLevelsPerSymbol

type ChangedPriceLevelsPerSymbol struct {
	Buys map[int64]int64
	Sells map[int64]int64
}