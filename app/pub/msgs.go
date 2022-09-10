package pub

import (
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bnb-chain/node/common/types"
	orderPkg "github.com/bnb-chain/node/plugins/dex/order"
)

type msgType int8

const (
	accountsTpe = iota
	booksTpe
	executionResultTpe
	blockFeeTpe
	transferTpe
	blockTpe
	stakingTpe
	distributionTpe
	slashingTpe
	crossTransferTpe
	mirrorTpe
	sideProposalType
	breatheBlockTpe
)

var (
	nativeBlockMetaKey   = fmt.Sprintf("%sBlockMeta", strings.ToLower(types.NativeTokenSymbol))
	nativeTransactionKey = fmt.Sprintf("%sTransaction", strings.ToLower(types.NativeTokenSymbol))
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
	case transferTpe:
		return "Transfers"
	case blockTpe:
		return "Block"
	case stakingTpe:
		return "Staking"
	case distributionTpe:
		return "Distribution"
	case slashingTpe:
		return "Slashing"
	case crossTransferTpe:
		return "CrossTransfer"
	case mirrorTpe:
		return "Mirror"
	case sideProposalType:
		return "SideProposal"
	case breatheBlockTpe:
		return "BreatheBlock"
	default:
		return "Unknown"
	}
}

// Versions of schemas
// This field is used to generate last component of key in kafka message and helps consumers
// figure out which version of writer schema to use.
// This allows consumers be deployed independently (in advance) with publisher
var latestSchemaVersions = map[msgType]int{
	accountsTpe:        1,
	booksTpe:           0,
	executionResultTpe: 1,
	blockFeeTpe:        0,
	transferTpe:        1,
	blockTpe:           0,
	stakingTpe:         1,
	distributionTpe:    1,
	slashingTpe:        0,
	crossTransferTpe:   0,
	mirrorTpe:          0,
	sideProposalType:   0,
	breatheBlockTpe:    0,
}

type AvroOrJsonMsg interface {
	ToNativeMap() map[string]interface{}
	String() string
}

// EssMsg is a type when AvroOrJsonMsg failed to publish
// Not all AvroOrJsonMsg implemented Ess because:
//
// for transfer:
//
// 1. qs doesn't subscribe to its topic (risk control is relying on that)
// 2. risk control can recover from explorer indexed transfers (pull mode)
// 3. we don't have a unique representation of transfer like order-id (we didn't save txhash in message)
//
// for trade:
// the problem is same with above point 3, (trade id is only generated during publication, not persisted anywhere).
// If we keep qty, price, sid, bid for a trade, it would be too much,
// in this case we should recover from local publisher
type EssMsg interface {
	AvroOrJsonMsg

	// a string that carry essential msg used to make up downstream service on kafka issue
	// this string would be persisted into file
	EssentialMsg() string

	// an empty message of original `AvroOrJsonMsg` to make downstream logic not broken
	EmptyCopy() AvroOrJsonMsg
}

type ExecutionResults struct {
	Height       int64
	Timestamp    int64 // milli seconds since Epoch
	NumOfMsgs    int   // number of individual messages we published, consumer can verify messages they received against this field to make sure they does not miss messages
	Trades       trades
	Orders       Orders
	Proposals    Proposals
	StakeUpdates StakeUpdates
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
	if msg.StakeUpdates.NumOfMsgs > 0 {
		native["stakeUpdates"] = map[string]interface{}{"org.binance.dex.model.avro.StakeUpdates": msg.StakeUpdates.ToNativeMap()}
	}

	return native
}

func (msg *ExecutionResults) EssentialMsg() string {
	// mainly used to recover for large breathe block expiring message, there should be no trade on breathe block
	orders := msg.Orders.EssentialMsg()
	return fmt.Sprintf("height:%d\ntime:%d\norders:\n%s\n", msg.Height, msg.Timestamp, orders)
}

func (msg *ExecutionResults) EmptyCopy() AvroOrJsonMsg {
	var nonExpiredOrders []*Order
	for _, order := range msg.Orders.Orders {
		if order.Status != orderPkg.Expired {
			nonExpiredOrders = append(nonExpiredOrders, order)
		}
	}

	return &ExecutionResults{
		msg.Height,
		msg.Timestamp,
		msg.Proposals.NumOfMsgs + msg.StakeUpdates.NumOfMsgs + len(nonExpiredOrders),
		trades{}, // no trades on breathe block
		Orders{len(nonExpiredOrders), nonExpiredOrders},
		msg.Proposals,
		msg.StakeUpdates,
	}
}

// deliberated not implemented Ess
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
	ts := make([]map[string]interface{}, len(msg.Trades))
	for idx, trade := range msg.Trades {
		ts[idx] = trade.toNativeMap()
	}
	native["trades"] = ts
	return native
}

type Trade struct {
	Id         string
	Symbol     string
	Price      int64
	Qty        int64
	Sid        string
	Bid        string
	Sfee       string // DEPRECATING(Galileo): seller's total fee in this block, in future we should use SSingleFee which is more precise
	Bfee       string // DEPRECATING(Galileo): buyer's total fee in this block, in future we should use BSingleFee which is more precise
	SAddr      string // string representation of AccAddress
	BAddr      string // string representation of AccAddress
	SSrc       int64  // sell order source - ADDED Galileo
	BSrc       int64  // buy order source - ADDED Galileo
	SSingleFee string // seller's fee for this trade - ADDED Galileo
	BSingleFee string // buyer's fee for this trade - ADDED Galileo
	TickType   int    // ADDED Galileo
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
	native["ssrc"] = msg.SSrc
	native["bsrc"] = msg.BSrc
	native["ssinglefee"] = msg.SSingleFee
	native["bsinglefee"] = msg.BSingleFee
	native["tickType"] = msg.TickType
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
	os := make([]map[string]interface{}, len(msg.Orders))
	for idx, o := range msg.Orders {
		os[idx] = o.toNativeMap()
	}
	native["orders"] = os
	return native
}

func (msg *Orders) EssentialMsg() string {
	expiredOrders := &strings.Builder{}
	for _, order := range msg.Orders {
		// we only log expired orders in essential file
		// and publish other types of message via kafka
		if order.Status == orderPkg.Expired {
			fmt.Fprintf(expiredOrders, "%s %s %s\n", order.OrderId, order.Owner, order.Fee)
		}
	}
	return expiredOrders.String()
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
	Fee                  string // DEPRECATING(Galileo): total fee for Owner in this block, should use SingleFee in future
	OrderCreationTime    int64
	TransactionTime      int64
	TimeInForce          int8
	CurrentExecutionType orderPkg.ExecutionType
	TxHash               string
	SingleFee            string // fee for this order update - ADDED Galileo
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
	case orderPkg.Expired, orderPkg.IocExpire, orderPkg.IocNoFill, orderPkg.Canceled, orderPkg.FailedMatching:
		return msg.CumQty - msg.Qty // deliberated be negative value
	case orderPkg.FailedBlocking:
		return 0
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
	native["singlefee"] = msg.SingleFee
	return native
}

func (msg Order) isChargedCancel() bool {
	return msg.CumQty == 0 && msg.Status == orderPkg.Canceled
}

func (msg Order) isChargedExpire() bool {
	return msg.CumQty == 0 && (msg.Status == orderPkg.IocNoFill || msg.Status == orderPkg.Expired)
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
	ps := make([]map[string]interface{}, len(msg.Proposals))
	for idx, p := range msg.Proposals {
		ps[idx] = p.toNativeMap()
	}
	native["proposals"] = ps
	return native
}

type SideProposals struct {
	Height    int64
	Timestamp int64
	NumOfMsgs int
	Proposals []*SideProposal
}

func (msg *SideProposals) String() string {
	return fmt.Sprintf("SideProposals in block: %d, numOfMsgs: %d", msg.Height, msg.NumOfMsgs)
}

func (msg *SideProposals) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.NumOfMsgs
	native["height"] = msg.Height
	native["timestamp"] = msg.Timestamp
	ps := make([]map[string]interface{}, len(msg.Proposals))
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

type SideProposal struct {
	Id      int64
	ChainId string
	Status  ProposalStatus
}

func (msg *SideProposal) String() string {
	return fmt.Sprintf("SideProposal: %v", msg.toNativeMap())
}

func (msg *SideProposal) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["id"] = msg.Id
	native["chainid"] = msg.ChainId
	native["status"] = msg.Status.String()
	return native
}

type StakeUpdates struct {
	NumOfMsgs                     int
	CompletedUnbondingDelegations []*CompletedUnbondingDelegation
}

func (msg *StakeUpdates) String() string {
	return fmt.Sprintf("StakeUpdates numOfMsgs: %d", msg.NumOfMsgs)
}

func (msg *StakeUpdates) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.NumOfMsgs
	ps := make([]map[string]interface{}, len(msg.CompletedUnbondingDelegations))
	for idx, p := range msg.CompletedUnbondingDelegations {
		ps[idx] = p.toNativeMap()
	}
	native["completedUnbondingDelegations"] = ps
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
	bs := make([]map[string]interface{}, len(msg.Buys))
	for idx, buy := range msg.Buys {
		bs[idx] = buy.ToNativeMap()
	}
	native["buys"] = bs
	ss := make([]map[string]interface{}, len(msg.Sells))
	for idx, sell := range msg.Sells {
		ss[idx] = sell.ToNativeMap()
	}
	native["sells"] = ss
	return native
}

// deliberated not implemented Ess
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
		bs := make([]map[string]interface{}, len(msg.Books))
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
	Sequence int64

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
	bs := make([]map[string]interface{}, len(msg.Balances))
	for idx, b := range msg.Balances {
		bs[idx] = b.ToNativeMap()
	}
	native["sequence"] = msg.Sequence
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
		as := make([]map[string]interface{}, len(msg.Accounts))
		for idx, a := range msg.Accounts {
			as[idx] = a.ToNativeMap()
		}
		native["accounts"] = as
	}
	return native
}

func (msg *Accounts) EssentialMsg() string {
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "height:%d\n", msg.Height)
	for _, acc := range msg.Accounts {
		fmt.Fprintf(&builder, "%s\n", sdk.AccAddress(acc.Owner).String())
	}
	return builder.String()
}

func (msg *Accounts) EmptyCopy() AvroOrJsonMsg {
	return &Accounts{
		msg.Height,
		0,
		[]Account{},
	}
}

// deliberated not implemented Ess
type BlockFee struct {
	Height     int64
	Fee        string
	Validators []string // slice of string wrappers of bytes representation of sdk.AccAddress
}

func (msg BlockFee) MarshalJSON() ([]byte, error) {
	bech32Strs := make([]string, len(msg.Validators))
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
	validators := make([]string, len(msg.Validators))
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

func (msg Receiver) String() string {
	return fmt.Sprintf("Transfer receiver %s get coin %v", msg.Addr, msg.Coins)
}

func (msg Receiver) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["addr"] = msg.Addr
	coins := make([]map[string]interface{}, len(msg.Coins))
	for idx, c := range msg.Coins {
		coins[idx] = c.ToNativeMap()
	}
	native["coins"] = coins
	return native
}

type Transfer struct {
	TxHash string
	Memo   string // Added for BEP39
	From   string
	To     []Receiver
}

func (msg Transfer) String() string {
	return fmt.Sprintf("Transfer: txHash: %s, memo: %s, from: %s, to: %v", msg.TxHash, msg.Memo, msg.From, msg.To)
}

func (msg Transfer) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["txhash"] = msg.TxHash
	native["memo"] = msg.Memo
	native["from"] = msg.From
	to := make([]map[string]interface{}, len(msg.To))
	for idx, t := range msg.To {
		to[idx] = t.ToNativeMap()
	}
	native["to"] = to
	return native
}

// deliberated not implemented Ess
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
	transfers := make([]map[string]interface{}, len(msg.Transfers))
	for idx, t := range msg.Transfers {
		transfers[idx] = t.ToNativeMap()
	}
	native["timestamp"] = msg.Timestamp
	native["num"] = msg.Num
	native["transfers"] = transfers
	return native
}

type Block struct {
	ChainID     string
	CryptoBlock CryptoBlock
}

func (msg Block) String() string {
	return fmt.Sprintf("Block: blockHash: %s, blockHeihgt: %d, numofTx: %d", msg.CryptoBlock.BlockHash, msg.CryptoBlock.BlockHeight, len(msg.CryptoBlock.Transactions))
}

func (msg Block) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["chainId"] = msg.ChainID
	native["cryptoBlock"] = msg.CryptoBlock.ToNativeMap()
	return native
}

type CryptoBlock struct {
	BlockHash   string
	ParentHash  string
	BlockHeight int64
	Timestamp   string
	TxTotal     int64

	BlockMeta    NativeBlockMeta
	Transactions []Transaction
}

func (msg CryptoBlock) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})

	native["blockHash"] = msg.BlockHash
	native["parentHash"] = msg.ParentHash
	native["blockHeight"] = msg.BlockHeight
	native["timestamp"] = msg.Timestamp
	native["txTotal"] = msg.TxTotal
	native[nativeBlockMetaKey] = msg.BlockMeta.ToNativeMap()

	transactions := make([]map[string]interface{}, 0, len(msg.Transactions))
	for _, t := range msg.Transactions {
		transactions = append(transactions, t.ToNativeMap())
	}
	native["transactions"] = transactions
	return native
}

func (msg CryptoBlock) String() string {
	return fmt.Sprintf("CryptoBlock: blockHash: %s, blockHeihgt: %d, numofTx: %d", msg.BlockHash, msg.BlockHeight, len(msg.Transactions))
}

type NativeBlockMeta struct {
	LastCommitHash     string
	DataHash           string
	ValidatorsHash     string
	NextValidatorsHash string
	ConsensusHash      string
	AppHash            string
	LastResultsHash    string
	EvidenceHash       string
	ProposerAddress    string
}

func (msg NativeBlockMeta) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["lastCommitHash"] = msg.LastCommitHash
	native["dataHash"] = msg.DataHash
	native["validatorsHash"] = msg.ValidatorsHash
	native["nextValidatorsHash"] = msg.NextValidatorsHash
	native["consensusHash"] = msg.ConsensusHash
	native["appHash"] = msg.AppHash
	native["lastResultsHash"] = msg.LastResultsHash
	native["evidenceHash"] = msg.EvidenceHash
	native["proposerAddress"] = msg.ProposerAddress
	return native
}

func (msg NativeBlockMeta) String() string {
	return fmt.Sprintf("NativeBlockMeta: dataHash: %s, appHash: %s, proposerAddress: %s", msg.DataHash, msg.AppHash, msg.ProposerAddress)
}

type Transaction struct {
	TxHash    string
	Fee       string
	Timestamp string

	Inputs  []Input
	Outputs []Output

	NativeTransaction NativeTransaction
}

func (msg Transaction) String() string {
	return fmt.Sprintf("Transaction: txHash: %s, fee: %s, source: %d, type: %s, data: %s", msg.TxHash, msg.Fee, msg.NativeTransaction.Source, msg.NativeTransaction.TxType, msg.NativeTransaction.Data)
}

func (msg Transaction) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["txHash"] = msg.TxHash
	native["fee"] = msg.Fee
	inputs := make([]map[string]interface{}, 0, len(msg.Inputs))
	for _, c := range msg.Inputs {
		inputs = append(inputs, c.ToNativeMap())
	}
	native["inputs"] = inputs
	outputs := make([]map[string]interface{}, 0, len(msg.Outputs))
	for _, c := range msg.Outputs {
		outputs = append(outputs, c.ToNativeMap())
	}
	native["outputs"] = outputs
	native[nativeTransactionKey] = msg.NativeTransaction.ToNativeMap()
	return native
}

type Input struct {
	Address string
	Coins   []Coin
}

func (msg Input) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["address"] = msg.Address
	coins := make([]map[string]interface{}, len(msg.Coins))
	for idx, c := range msg.Coins {
		coins[idx] = c.ToNativeMap()
	}
	native["coins"] = coins
	return native
}

func (msg Input) String() string {
	return fmt.Sprintf("Input: address: %s, coins: %v", msg.Address, msg.Coins)
}

type Output struct {
	Address string
	Coins   []Coin
}

func (msg Output) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["address"] = msg.Address
	coins := make([]map[string]interface{}, len(msg.Coins))
	for idx, c := range msg.Coins {
		coins[idx] = c.ToNativeMap()
	}
	native["coins"] = coins
	return native
}

func (msg Output) String() string {
	return fmt.Sprintf("Output: address: %s, coins: %v", msg.Address, msg.Coins)
}

type NativeTransaction struct {
	Source     int64
	TxType     string
	TxAsset    string
	OrderId    string
	Code       uint32
	Data       string
	ProposalId int64
}

func (msg NativeTransaction) String() string {
	return fmt.Sprintf("NativeTransaction: TxType: %s, Code: %d,data: %s", msg.TxType, msg.Code, msg.Data)
}

func (msg NativeTransaction) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["source"] = msg.Source
	native["txType"] = msg.TxType
	native["txAssert"] = msg.TxAsset
	native["orderId"] = msg.OrderId
	native["code"] = int64(msg.Code)
	native["data"] = msg.Data
	native["proposalId"] = msg.ProposalId
	return native
}

// distribution message
type DistributionMsg struct {
	NumOfMsgs     int
	Height        int64
	Timestamp     int64
	Distributions map[string][]*Distribution
}

func (msg *DistributionMsg) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.NumOfMsgs
	native["height"] = msg.Height
	native["timestamp"] = msg.Timestamp

	distributions := make(map[string]interface{})
	for chainId, v := range msg.Distributions {
		items := make([]map[string]interface{}, len(v))
		for idx, item := range v {
			items[idx] = item.toNativeMap()
		}
		distributions[chainId] = items
	}
	native["distributions"] = distributions
	return native
}

func (msg *DistributionMsg) String() string {
	return fmt.Sprintf("DistributionMsg at height: %d, numOfMsgs: %d", msg.Height, msg.NumOfMsgs)
}

func (msg *DistributionMsg) EssentialMsg() string {
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "height:%d\n", msg.Height)
	for chainId, diss := range msg.Distributions {
		fmt.Fprintf(&builder, "chainId:%s\n", chainId)
		for _, dis := range diss {
			fmt.Fprintf(&builder, "validator:%s,rewards count:%d\n", dis.Validator.String(), len(dis.Rewards))
		}
	}
	return builder.String()
}

func (msg *DistributionMsg) EmptyCopy() AvroOrJsonMsg {
	return &DistributionMsg{
		msg.NumOfMsgs,
		msg.Height,
		msg.Timestamp,
		make(map[string][]*Distribution),
	}
}

type Distribution struct {
	Validator      sdk.ValAddress
	SelfDelegator  sdk.AccAddress
	DistributeAddr sdk.AccAddress
	ValTokens      int64
	TotalReward    int64
	Commission     int64
	Rewards        []*Reward
}

func (msg *Distribution) String() string {
	return fmt.Sprintf("Distribution: %v", msg.toNativeMap())
}

func (msg *Distribution) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["validator"] = msg.Validator.String()
	native["selfDelegator"] = msg.SelfDelegator.String()
	native["distributeAddr"] = msg.DistributeAddr.String()
	native["valTokens"] = msg.ValTokens
	native["totalReward"] = msg.TotalReward
	native["commission"] = msg.Commission
	as := make([]map[string]interface{}, len(msg.Rewards))
	for idx, reward := range msg.Rewards {
		as[idx] = reward.toNativeMap()
	}
	native["rewards"] = as
	return native
}

type Reward struct {
	Validator sdk.ValAddress
	Delegator sdk.AccAddress
	Tokens    int64
	Amount    int64
}

func (msg *Reward) String() string {
	return fmt.Sprintf("Reward: %v", msg.toNativeMap())
}

func (msg *Reward) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["validator"] = msg.Validator.String()
	native["delegator"] = msg.Delegator.String()
	native["delegationTokens"] = msg.Tokens
	native["reward"] = msg.Amount
	return native
}

// slash message
type SlashMsg struct {
	NumOfMsgs int
	Height    int64
	Timestamp int64
	SlashData map[string][]*Slash
}

func (msg *SlashMsg) String() string {
	return fmt.Sprintf("SlashMsg at height: %d, numOfMsgs: %d, slashData: %v", msg.Height, msg.NumOfMsgs, msg.SlashData)
}

func (msg *SlashMsg) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.NumOfMsgs
	native["height"] = msg.Height
	native["timestamp"] = msg.Timestamp

	slashData := make(map[string]interface{})
	for chainId, v := range msg.SlashData {
		items := make([]map[string]interface{}, len(v))
		for idx, item := range v {
			items[idx] = item.toNativeMap()
		}
		slashData[chainId] = items
	}
	native["slashData"] = slashData
	return native
}

func (msg *SlashMsg) EssentialMsg() string {
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "height:%d\n", msg.Height)
	for chainId, slash := range msg.SlashData {
		fmt.Fprintf(&builder, "chainId:%s\n, slash count: %d\n", chainId, len(slash))
	}
	return builder.String()
}

func (msg *SlashMsg) EmptyCopy() AvroOrJsonMsg {
	return &SlashMsg{
		msg.NumOfMsgs,
		msg.Height,
		msg.Timestamp,
		make(map[string][]*Slash),
	}
}

type Slash struct {
	Validator              sdk.ValAddress
	InfractionType         byte
	InfractionHeight       int64
	JailUtil               int64
	SlashAmount            int64
	ToFeePool              int64
	Submitter              sdk.AccAddress
	SubmitterReward        int64
	ValidatorsCompensation []*AllocatedAmt
}

func (msg *Slash) String() string {
	return fmt.Sprintf("Slash: %v", msg.toNativeMap())
}

func (msg *Slash) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["validator"] = msg.Validator.String()
	native["infractionType"] = int(msg.InfractionType)
	native["infractionHeight"] = msg.InfractionHeight
	native["jailUtil"] = msg.JailUtil
	native["slashAmount"] = msg.SlashAmount
	native["toFeePool"] = msg.ToFeePool
	if msg.Submitter != nil {
		native["submitter"] = msg.Submitter.String()
	} else {
		native["submitter"] = ""
	}
	native["submitterReward"] = msg.SubmitterReward

	vsc := make([]map[string]interface{}, len(msg.ValidatorsCompensation))
	for idx, compensation := range msg.ValidatorsCompensation {
		vsc[idx] = compensation.toNativeMap()
	}
	native["validatorsCompensation"] = vsc
	return native
}

type AllocatedAmt struct {
	Address string
	Amount  int64
}

func (msg *AllocatedAmt) String() string {
	return fmt.Sprintf("AllocatedAmt: %v", msg.toNativeMap())
}

func (msg *AllocatedAmt) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["address"] = msg.Address
	native["amount"] = msg.Amount
	return native
}

type BreatheBlockMsg struct {
	Height    int64
	Timestamp int64
}

func (msg *BreatheBlockMsg) String() string {
	return fmt.Sprintf("BreatheBlockMsg at height: %d", msg.Height)
}

func (msg *BreatheBlockMsg) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	native["timestamp"] = msg.Timestamp
	return native
}

func (msg *BreatheBlockMsg) EssentialMsg() string {
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "height:%d\n", msg.Height)
	return builder.String()
}

func (msg *BreatheBlockMsg) EmptyCopy() AvroOrJsonMsg {
	return &BreatheBlockMsg{
		msg.Height,
		msg.Timestamp,
	}
}
