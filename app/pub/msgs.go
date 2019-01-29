package pub

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

type msgType int8

const (
	accountsTpe = iota
	booksTpe
	executionResultTpe
	blockFeeTpe
	transferType
)

// the strings should be keep consistence with top level record name in schemas.go
// !!!NOTE!!! Changes of these strings should notice consumers of kafka publisher
func (this msgType) String() string {
	switch this {
	case accountsTpe:
		return "Accounts"
	case booksTpe:
		return "Books"
	case executionResultTpe:
		return "ExecutionResults"
	case blockFeeTpe:
		return "BlockFee"
	case transferType:
		return "Transfers"
	default:
		return "Unknown"
	}
}

type AvroOrJsonMsg interface {
	ToNativeMap() map[string]interface{}
	String() string
}

type ExecutionResults struct {
	Height    int64
	Timestamp int64 // milli seconds since Epoch
	NumOfMsgs int   // number of individual messages we published, consumer can verify messages they received against this field to make sure they does not miss messages
	Trades    trades
	Orders    Orders
	Proposals Proposals
}

func (msg *ExecutionResults) String() string {
	return fmt.Sprintf("ExecutionResult at height: %d, numOfMsgs: %d", msg.Height, msg.NumOfMsgs)
}

func (msg *ExecutionResults) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	native["timestamp"] = msg.Timestamp
	native["numOfMsgs"] = msg.NumOfMsgs
	if msg.Trades.NumOfMsgs > 0 {
		native["trades"] = map[string]interface{}{"org.binance.dex.model.avro.Trades": msg.Trades.ToNativeMap()}
	}
	if msg.Orders.NumOfMsgs > 0 {
		native["orders"] = map[string]interface{}{"org.binance.dex.model.avro.Orders": msg.Orders.ToNativeMap()}
	}
	if msg.Proposals.NumOfMsgs > 0 {
		native["proposals"] = map[string]interface{}{"org.binance.dex.model.avro.Proposals": msg.Proposals.ToNativeMap()}
	}
	return native
}

type trades struct {
	NumOfMsgs int
	Trades    []*Trade
}

func (msg *trades) String() string {
	return fmt.Sprintf("Trades numOfMsgs: %d", msg.NumOfMsgs)
}

func (msg *trades) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.NumOfMsgs
	ts := make([]map[string]interface{}, len(msg.Trades), len(msg.Trades))
	for idx, trade := range msg.Trades {
		ts[idx] = trade.toNativeMap()
	}
	native["trades"] = ts
	return native
}

type Trade struct {
	Id     string
	Symbol string
	Price  int64
	Qty    int64
	Sid    string
	Bid    string
	Sfee   string
	Bfee   string
	SAddr  string // string representation of AccAddress
	BAddr  string // string representation of AccAddress
}

func (msg *Trade) MarshalJSON() ([]byte, error) {
	type Alias Trade
	return json.Marshal(&struct {
		*Alias
		SAddr string
		BAddr string
	}{
		Alias: (*Alias)(msg),
		SAddr: sdk.AccAddress(msg.SAddr).String(),
		BAddr: sdk.AccAddress(msg.BAddr).String(),
	})
}

func (msg *Trade) String() string {
	return fmt.Sprintf("Trade: %v", msg.toNativeMap())
}

func (msg *Trade) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["id"] = msg.Id
	native["symbol"] = msg.Symbol
	native["price"] = msg.Price
	native["qty"] = msg.Qty
	native["sid"] = msg.Sid
	native["bid"] = msg.Bid
	native["sfee"] = msg.Sfee
	native["bfee"] = msg.Bfee
	native["saddr"] = sdk.AccAddress(msg.SAddr).String()
	native["baddr"] = sdk.AccAddress(msg.BAddr).String()
	return native
}

type Orders struct {
	NumOfMsgs int
	Orders    []*Order
}

func (msg *Orders) String() string {
	return fmt.Sprintf("Orders numOfMsgs: %d", msg.NumOfMsgs)
}

func (msg *Orders) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.NumOfMsgs
	os := make([]map[string]interface{}, len(msg.Orders), len(msg.Orders))
	for idx, o := range msg.Orders {
		os[idx] = o.toNativeMap()
	}
	native["orders"] = os
	return native
}

type Order struct {
	Symbol               string
	Status               orderPkg.ChangeType
	OrderId              string
	TradeId              string
	Owner                string
	Side                 int8
	OrderType            int8
	Price                int64
	Qty                  int64
	LastExecutedPrice    int64
	LastExecutedQty      int64
	CumQty               int64
	Fee                  string
	OrderCreationTime    int64
	TransactionTime      int64
	TimeInForce          int8
	CurrentExecutionType orderPkg.ExecutionType
	TxHash               string
}

func (msg *Order) String() string {
	return fmt.Sprintf("Order: %v", msg.toNativeMap())
}

func (msg *Order) effectQtyToOrderBook() int64 {
	switch msg.Status {
	case orderPkg.Ack:
		return msg.Qty
	case orderPkg.FullyFill, orderPkg.PartialFill:
		return -msg.LastExecutedQty
	case orderPkg.Expired, orderPkg.IocNoFill, orderPkg.Canceled:
		return msg.CumQty - msg.Qty // deliberated be negative value
	default:
		Logger.Error("does not supported order status", "order", msg.String())
		return 0
	}
}

func (msg *Order) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["symbol"] = msg.Symbol
	native["status"] = msg.Status.String()
	native["orderId"] = msg.OrderId
	native["tradeId"] = msg.TradeId
	native["owner"] = msg.Owner
	native["side"] = int(msg.Side) // without conversion avro encoder would complain: value does not match its schema: cannot encode binary int: expected: Go numeric; received: int8
	native["orderType"] = int(msg.OrderType)
	native["price"] = msg.Price
	native["qty"] = msg.Qty
	native["lastExecutedPrice"] = msg.LastExecutedPrice
	native["lastExecutedQty"] = msg.LastExecutedQty
	native["cumQty"] = msg.CumQty
	native["fee"] = msg.Fee
	native["orderCreationTime"] = msg.OrderCreationTime
	native["transactionTime"] = msg.TransactionTime
	native["timeInForce"] = int(msg.TimeInForce)
	native["currentExecutionType"] = msg.CurrentExecutionType.String()
	native["txHash"] = msg.TxHash
	return native
}

type Proposals struct {
	NumOfMsgs int
	Proposals []*Proposal
}

func (msg *Proposals) String() string {
	return fmt.Sprintf("Proposals numOfMsgs: %d", msg.NumOfMsgs)
}

func (msg *Proposals) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.NumOfMsgs
	ps := make([]map[string]interface{}, len(msg.Proposals), len(msg.Proposals))
	for idx, p := range msg.Proposals {
		ps[idx] = p.toNativeMap()
	}
	native["proposals"] = ps
	return native
}

type ProposalStatus uint8

const (
	Succeed ProposalStatus = iota
	Failed
)

func (this ProposalStatus) String() string {
	switch this {
	case Succeed:
		return "S"
	case Failed:
		return "F"
	default:
		return "Unknown"
	}
}

type Proposal struct {
	Id     int64
	Status ProposalStatus
}

func (msg *Proposal) String() string {
	return fmt.Sprintf("Proposal: %v", msg.toNativeMap())
}

func (msg *Proposal) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["id"] = msg.Id
	native["status"] = msg.Status.String()
	return native
}

type PriceLevel struct {
	Price   int64
	LastQty int64
}

func (msg *PriceLevel) String() string {
	return fmt.Sprintf("priceLevel: %s", msg.ToNativeMap())
}

func (msg *PriceLevel) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["price"] = msg.Price
	native["lastQty"] = msg.LastQty
	return native
}

type OrderBookDelta struct {
	Symbol string
	Buys   []PriceLevel
	Sells  []PriceLevel
}

func (msg *OrderBookDelta) String() string {
	return fmt.Sprintf("orderBookDelta for: %s, num of buys prices: %d, num of sell prices: %d", msg.Symbol, len(msg.Buys), len(msg.Sells))
}

func (msg *OrderBookDelta) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["symbol"] = msg.Symbol
	bs := make([]map[string]interface{}, len(msg.Buys), len(msg.Buys))
	for idx, buy := range msg.Buys {
		bs[idx] = buy.ToNativeMap()
	}
	native["buys"] = bs
	ss := make([]map[string]interface{}, len(msg.Sells), len(msg.Sells))
	for idx, sell := range msg.Sells {
		ss[idx] = sell.ToNativeMap()
	}
	native["sells"] = ss
	return native
}

type Books struct {
	Height    int64
	Timestamp int64
	NumOfMsgs int
	Books     []OrderBookDelta
}

func (msg *Books) String() string {
	return fmt.Sprintf("Books at height: %d, numOfMsgs: %d", msg.Height, msg.NumOfMsgs)
}

func (msg *Books) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	native["timestamp"] = msg.Timestamp
	native["numOfMsgs"] = msg.NumOfMsgs
	if msg.NumOfMsgs > 0 {
		bs := make([]map[string]interface{}, len(msg.Books), len(msg.Books))
		for idx, book := range msg.Books {
			bs[idx] = book.ToNativeMap()
		}
		native["books"] = bs
	}
	return native
}

type AssetBalance struct {
	Asset  string
	Free   int64
	Frozen int64
	Locked int64
}

func (msg *AssetBalance) String() string {
	return fmt.Sprintf("AssetBalance: %s", msg.ToNativeMap())
}

func (msg *AssetBalance) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["asset"] = msg.Asset
	native["free"] = msg.Free
	native["frozen"] = msg.Frozen
	native["locked"] = msg.Locked
	return native
}

type Account struct {
	Owner    string // string representation of AccAddress
	Fee      string
	Balances []*AssetBalance
}

func (msg *Account) MarshalJSON() ([]byte, error) {
	type Alias Account
	return json.Marshal(&struct {
		*Alias
		Owner string
	}{
		Alias: (*Alias)(msg),
		Owner: sdk.AccAddress(msg.Owner).String(),
	})
}

func (msg *Account) String() string {
	return fmt.Sprintf("Account of: %s, fee: %s, num of balance changes: %d", msg.Owner, msg.Fee, len(msg.Balances))
}

func (msg *Account) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["owner"] = sdk.AccAddress(msg.Owner).String()
	bs := make([]map[string]interface{}, len(msg.Balances), len(msg.Balances))
	for idx, b := range msg.Balances {
		bs[idx] = b.ToNativeMap()
	}
	native["fee"] = msg.Fee
	native["balances"] = bs
	return native
}

type Accounts struct {
	Height    int64
	NumOfMsgs int
	Accounts  []Account
}

func (msg *Accounts) String() string {
	return fmt.Sprintf("Accounts at height: %d, numOfMsgs: %d", msg.Height, msg.NumOfMsgs)
}

func (msg *Accounts) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	native["numOfMsgs"] = msg.NumOfMsgs
	if msg.NumOfMsgs > 0 {
		as := make([]map[string]interface{}, len(msg.Accounts), len(msg.Accounts))
		for idx, a := range msg.Accounts {
			as[idx] = a.ToNativeMap()
		}
		native["accounts"] = as
	}
	return native
}

type BlockFee struct {
	Height     int64
	Fee        string
	Validators []string // slice of string wrappers of bytes representation of sdk.AccAddress
}

func (msg BlockFee) MarshalJSON() ([]byte, error) {
	bech32Strs := make([]string, len(msg.Validators), len(msg.Validators))
	for id, val := range msg.Validators {
		bech32Strs[id] = sdk.AccAddress(val).String()
	}
	type Alias BlockFee
	return json.Marshal(&struct {
		Alias
		Validators []string
	}{
		Alias:      (Alias)(msg),
		Validators: bech32Strs,
	})
}

func (msg BlockFee) String() string {
	return fmt.Sprintf("Blockfee at height: %d, fee: %s, validators: %v", msg.Height, msg.Fee, msg.Validators)
}

func (msg BlockFee) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	native["fee"] = msg.Fee
	validators := make([]string, len(msg.Validators), len(msg.Validators))
	for idx, addr := range msg.Validators {
		validators[idx] = sdk.AccAddress(addr).String()
	}
	native["validators"] = validators
	return native
}

type Coin struct {
	Denom  string `json:"denom"`
	Amount int64  `json:"amount"`
}

func (coin Coin) String() string {
	return fmt.Sprintf("%d%s", coin.Amount, coin.Denom)
}

func (msg Coin) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["denom"] = msg.Denom
	native["amount"] = msg.Amount
	return native
}

type Receiver struct {
	Addr  string
	Coins []Coin
}

func (msg Receiver) MarshalJSON() ([]byte, error) {
	type Alias Receiver
	return json.Marshal(&struct {
		Alias
		Addr string
	}{
		Alias: (Alias)(msg),
		Addr:  msg.Addr,
	})
}

func (msg Receiver) String() string {
	return fmt.Sprintf("Transfer receiver %s get coin %v", msg.Addr, msg.Coins)
}

func (msg Receiver) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["addr"] = msg.Addr
	coins := make([]map[string]interface{}, len(msg.Coins), len(msg.Coins))
	for idx, c := range msg.Coins {
		coins[idx] = c.ToNativeMap()
	}
	native["coins"] = coins
	return native
}

type Transfer struct {
	From string
	To   []Receiver
}

func (msg Transfer) MarshalJSON() ([]byte, error) {
	type Alias Transfer
	return json.Marshal(&struct {
		Alias
		From string
	}{
		Alias: (Alias)(msg),
		From:  msg.From,
	})
}

func (msg Transfer) String() string {
	return fmt.Sprintf("Transfer : from: %s, to: %v", msg.From, msg.To)
}

func (msg Transfer) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["from"] = msg.From
	to := make([]map[string]interface{}, len(msg.To), len(msg.To))
	for idx, t := range msg.To {
		to[idx] = t.ToNativeMap()
	}
	native["to"] = to
	return native
}

type Transfers struct {
	Height    int64
	Num       int
	Timestamp int64
	Transfers []Transfer
}

func (msg Transfers) String() string {
	return fmt.Sprintf("Transfers in block %d, num: %d", msg.Height, msg.Num)
}

func (msg Transfers) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	transfers := make([]map[string]interface{}, len(msg.Transfers), len(msg.Transfers))
	for idx, t := range msg.Transfers {
		transfers[idx] = t.ToNativeMap()
	}
	native["timestamp"] = msg.Timestamp
	native["num"] = msg.Num
	native["transfers"] = transfers
	return native
}
