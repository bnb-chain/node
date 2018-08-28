package pub

import (
	"fmt"

	"github.com/linkedin/goavro"

	"github.com/tendermint/tendermint/libs/log"
)

var (
	logger              log.Logger
	tradesCodec         *goavro.Codec
	ordersCodec         *goavro.Codec
	booksCodec          *goavro.Codec
	accountCodec        *goavro.Codec
	transactionCodec    *goavro.Codec
	blockCommittedCodec *goavro.Codec
)

type msgType int8

const (
	tradesTpe msgType = iota
	ordersTpe
	booksTpe
	blockCommitedTpe
)

// the strings should be keep consistence with top level record name in schemas.go
func (this msgType) String() string {
	switch this {
	case tradesTpe:
		return "Trades"
	case ordersTpe:
		return "Orders"
	case booksTpe:
		return "Books"
	case blockCommitedTpe:
		return "BlockCommitted"
	default:
		return "Unknown"
	}
}

type AvroMsg interface {
	ToNativeMap() map[string]interface{}
}

func marshal(msg AvroMsg, tpe msgType) ([]byte, error) {
	native := msg.ToNativeMap()
	var codec *goavro.Codec
	switch tpe {
	case tradesTpe:
		codec = tradesCodec
	case ordersTpe:
		codec = ordersCodec
	case booksTpe:
		codec = booksCodec
	case blockCommitedTpe:
		codec = blockCommittedCodec
	default:
		return nil, fmt.Errorf("doesn't support marshal kafka msg tpe: %s", tpe.String())
	}
	bb, err := codec.BinaryFromNative(nil, native)
	if err != nil {
		logger.Error(fmt.Sprint("failed to serialize message: %s", msg), "err", err)
	}
	return bb, err
}

type blockCommitted struct {
	height    int64
	msg       string
	timestamp int64 // milli seconds since Epoch
	numOfMsgs int   // number of individual messages we published, consumer can verify messages they received against this field to make sure they does not miss messages
}

func (msg *blockCommitted) String() string {
	return fmt.Sprint("BlockCommitted: %s", msg.ToNativeMap())
}

func (msg *blockCommitted) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.height
	native["msg"] = msg.msg
	native["timestamp"] = msg.timestamp
	native["numOfMsgs"] = msg.numOfMsgs
	return native
}

type trades struct {
	blockHeight int64
	numOfMsgs   int
	trades      []trade
}

func (msg *trades) String() string {
	return fmt.Sprintf("Trades at: %d, numOfMsgs: %d", msg.blockHeight, msg.numOfMsgs)
}

func (msg *trades) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["blockHeight"] = msg.blockHeight
	native["numOfMsgs"] = msg.numOfMsgs
	ts := make([]map[string]interface{}, len(msg.trades), len(msg.trades))
	for idx, trade := range msg.trades {
		ts[idx] = trade.toNativeMap()
	}
	native["trades"] = ts
	return native
}

type trade struct {
	id     string
	symbol string
	price  int64
	qty    int64
	sid    string
	bid    string
	sfee   int64
	bfee   int64
}

func (msg *trade) String() string {
	return fmt.Sprint("Trade: %s", msg.toNativeMap())
}

func (msg *trade) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["id"] = msg.id
	native["symbol"] = msg.symbol
	native["price"] = msg.price
	native["qty"] = msg.qty
	native["sid"] = msg.sid
	native["bid"] = msg.bid
	native["sfee"] = msg.sfee
	native["bfee"] = msg.bfee
	return native
}

type orders struct {
	blockHeight int64
	numOfMsgs   int
	orders      []order
}

func (msg *orders) String() string {
	return fmt.Sprintf("Orders at %d, numOfMsgs: %d", msg.blockHeight, msg.numOfMsgs)
}

func (msg *orders) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["blockHeight"] = msg.blockHeight
	native["numOfMsgs"] = msg.numOfMsgs
	os := make([]map[string]interface{}, len(msg.orders), len(msg.orders))
	for idx, o := range msg.orders {
		os[idx] = o.toNativeMap()
	}
	native["orders"] = os
	return native
}

type order struct {
	symbol               string
	status               string
	orderId              string
	tradeId              string
	owner                string
	side                 string
	orderType            string
	price                int64
	qty                  int64
	lastExecutedPrice    int64
	lastExecutedQty      int64
	cumQty               int64
	cumQuoteAssetQty     int64
	fee                  int64
	feeAsset             string
	orderCreationTime    int64
	transactionTime      int64
	timeInForce          string
	currentExecutionType string
}

func (msg *order) String() string {
	return fmt.Sprint("Order: %s", msg.toNativeMap())
}

func (msg *order) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["symbol"] = msg.symbol
	native["status"] = msg.status
	native["orderId"] = msg.orderId
	native["tradeId"] = msg.tradeId
	native["owner"] = msg.owner
	native["side"] = msg.side
	native["orderType"] = msg.orderType
	native["price"] = msg.price
	native["qty"] = msg.qty
	native["lastExecutedPrice"] = msg.lastExecutedPrice
	native["lastExecutedQty"] = msg.lastExecutedQty
	native["cumQty"] = msg.cumQty
	native["cumQuoteAssetQty"] = msg.cumQuoteAssetQty
	native["fee"] = msg.fee
	native["feeAsset"] = msg.feeAsset
	native["orderCreationTime"] = msg.orderCreationTime
	native["transactionTime"] = msg.transactionTime
	native["timeInForce"] = msg.timeInForce
	native["currentExecutionType"] = msg.currentExecutionType
	return native
}

type priceLevel struct {
	price   int64
	lastQty int64
}

func (msg *priceLevel) String() string {
	return fmt.Sprintf("priceLevel: %s", msg.ToNativeMap())
}

func (msg *priceLevel) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["price"] = msg.price
	native["lastQty"] = msg.lastQty
	return native
}

type orderBookDelta struct {
	symbol string
	buys   []priceLevel
	sells  []priceLevel
}

func (msg *orderBookDelta) String() string {
	return fmt.Sprintf("orderBookDelta for: %s, num of buys prices: %d, num of sell prices: %d", msg.symbol, len(msg.buys), len(msg.sells))
}

func (msg *orderBookDelta) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["symbol"] = msg.symbol
	bs := make([]map[string]interface{}, len(msg.buys), len(msg.buys))
	for idx, buy := range msg.buys {
		bs[idx] = buy.ToNativeMap()
	}
	native["buys"] = bs
	ss := make([]map[string]interface{}, len(msg.sells), len(msg.sells))
	for idx, sell := range msg.sells {
		ss[idx] = sell.ToNativeMap()
	}
	native["sells"] = ss
	return native
}

type books struct {
	blockHeight int64
	numOfMsgs   int
	books       []orderBookDelta
}

func (msg *books) String() string {
	return fmt.Sprintf("books for %d symbols at height: %d", msg.numOfMsgs, msg.blockHeight)
}

func (msg *books) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["blockHeight"] = msg.blockHeight
	native["numOfMsgs"] = msg.numOfMsgs
	bs := make([]map[string]interface{}, len(msg.books), len(msg.books))
	for idx, book := range msg.books {
		bs[idx] = book.ToNativeMap()
	}
	native["books"] = bs
	return native
}

type assetBalance struct {
	asset  string
	free   int64
	frozen int64
	locked int64
}

func (msg *assetBalance) String() string {
	return fmt.Sprint("AssetBalance: %s", msg.ToNativeMap())
}

func (msg *assetBalance) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["asset"] = msg.asset
	native["free"] = msg.free
	native["frozen"] = msg.frozen
	native["locked"] = msg.locked
	return native
}

type account struct {
	owner    string
	balances []assetBalance
}

func (msg *account) String() string {
	return fmt.Sprint("Account: %s", msg.ToNativeMap())
}

func (msg *account) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["owner"] = msg.owner
	bs := make([]map[string]interface{}, len(msg.balances), len(msg.balances))
	for idx, b := range msg.balances {
		bs[idx] = b.ToNativeMap()
	}
	native["balances"] = bs
	return native
}

type accounts struct {
	blockHeight int64
	numOfMsgs   int
	accounts    []account
}

func (msg *accounts) String() string {
	return fmt.Sprint("Accounts at: %d, numOfMsgs: %d", msg.blockHeight, msg.numOfMsgs)
}

func (msg *accounts) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["blockHeight"] = msg.blockHeight
	native["numOfMsgs"] = msg.numOfMsgs
	as := make([]map[string]interface{}, len(msg.accounts), len(msg.accounts))
	for idx, a := range msg.accounts {
		as[idx] = a.ToNativeMap()
	}
	native["accounts"] = as
	return native
}

type transaction struct {
	id    string
	from  string
	to    string
	asset string
	qty   int64
	tpe   string
}

func (msg *transaction) String() string {
	return fmt.Sprint("Transaction: %s", msg.ToNativeMap())
}

func (msg *transaction) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["id"] = msg.id
	native["from"] = msg.from
	native["to"] = msg.to
	native["asset"] = msg.asset
	native["qty"] = msg.qty
	native["tpe"] = msg.tpe
	return native
}

func initAvroCodecs(logg log.Logger) (res error) {
	logger = logg
	if blockCommittedCodec, res = goavro.NewCodec(blockCommittedSchema); res != nil {
		return res
	} else if tradesCodec, res = goavro.NewCodec(tradesSchema); res != nil {
		return res
	} else if ordersCodec, res = goavro.NewCodec(ordersSchema); res != nil {
		return res
	} else if booksCodec, res = goavro.NewCodec(booksSchema); res != nil {
		return res
	} else if accountCodec, res = goavro.NewCodec(accountSchema); res != nil {
		return res
	} else if transactionCodec, res = goavro.NewCodec(transactionSchema); res != nil {
		return res
	}
	return nil
}
