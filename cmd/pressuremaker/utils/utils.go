package utils

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"

	"github.com/binance-chain/node/app/pub"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
)

type MessageGenerator struct {
	NumOfTradesPerBlock   int
	NumOfTransferPerBlock int
	NumOfBlocks           int
	OrderChangeMap        orderPkg.OrderInfoForPublish

	buyerAddrs  []sdk.AccAddress
	sellerAddrs []sdk.AccAddress

	sendAddrs     []sdk.AccAddress
	receiverAddrs []sdk.AccAddress

	orderChanges orderPkg.OrderChanges
	trades       []*pub.Trade

	TimeStart time.Time
}

func (mg *MessageGenerator) Setup() {
	coins := sdk.Coins{sdk.NewCoin("BNB", 10000000000000000), sdk.NewCoin("NNB", 10000000000000000)}
	_, mg.buyerAddrs, _, _ = mock.CreateGenAccounts(mg.NumOfTradesPerBlock, coins)
	_, mg.sellerAddrs, _, _ = mock.CreateGenAccounts(mg.NumOfTradesPerBlock, coins)
	_, mg.sendAddrs, _, _ = mock.CreateGenAccounts(mg.NumOfTransferPerBlock, coins)
	_, mg.receiverAddrs, _, _ = mock.CreateGenAccounts(mg.NumOfTransferPerBlock*2, coins)
}

// each trade has two equal quantity order
// the price is a sin function of time, 4 full sin curve (0 - 2 * pi) within an hour :P
func (mg *MessageGenerator) OneOnOneMessages(height int, timeNow time.Time) (tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, accounts map[string]pub.Account, transfers *pub.Transfers) {
	tradesToPublish = make([]*pub.Trade, mg.NumOfTradesPerBlock)
	accounts = make(map[string]pub.Account, mg.NumOfTradesPerBlock*2)
	orderChanges = make(orderPkg.OrderChanges, mg.NumOfTradesPerBlock*2)

	timeSeed := float64(timeNow.UnixNano()-mg.TimeStart.UnixNano()) / (float64(time.Hour) / 4)
	timePub := timeNow.Unix()
	pi2 := 2.0 * math.Pi
	price := (30 + int64(20*math.Sin(float64(timeSeed)*pi2))) * 100000000
	amount := int64(100000000 * rand.Intn(10))

	for i := 0; i < mg.NumOfTradesPerBlock; i++ {
		seq := height
		buyOrder := makeOrderInfo(mg.buyerAddrs[i], 1, int64(height), price, amount, amount, timePub)
		sellOrder := makeOrderInfo(mg.sellerAddrs[i], 2, int64(height), price, amount, amount, timePub)
		mg.OrderChangeMap[buyOrder.Id] = &buyOrder
		mg.OrderChangeMap[sellOrder.Id] = &sellOrder

		orderChanges[i*2] = orderPkg.OrderChange{buyOrder.Id, orderPkg.Ack, nil}
		orderChanges[i*2+1] = orderPkg.OrderChange{sellOrder.Id, orderPkg.Ack, nil}

		tradesToPublish[i] = makeTradeToPub(fmt.Sprintf("%d-%d", height, i), sellOrder.Id, buyOrder.Id, mg.sellerAddrs[i].String(), mg.buyerAddrs[i].String(), price, amount)

		accounts[mg.buyerAddrs[i].String()] = pub.Account{string(mg.buyerAddrs[i]), "", []*pub.AssetBalance{{"NNB", 10000000000000000 + 100000000*int64(seq), 0, 0}, {"BNB", 10000000000000000 - 100000000*int64(seq), 0, 0}}}
		accounts[mg.sellerAddrs[i].String()] = pub.Account{string(mg.sellerAddrs[i]), "", []*pub.AssetBalance{{"NNB", 10000000000000000 - 100000000*int64(seq), 0, 0}, {"BNB", 10000000000000000 + 100000000*int64(seq), 0, 0}}}
	}
	transfers = &pub.Transfers{Height: int64(height), Num: 0, Transfers: []pub.Transfer{}}
	for i := 0; i < mg.NumOfTransferPerBlock; i++ {
		t := pub.Transfer{From: mg.sendAddrs[i].String(), To: []pub.Receiver{{mg.receiverAddrs[i].String(), []pub.Coin{{"BNB", rand.Int63n(math.MaxInt64)}}}, {mg.receiverAddrs[2*i+1].String(), []pub.Coin{{"BTC", rand.Int63n(math.MaxInt64)}}}}}
		transfers.Transfers = append(transfers.Transfers, t)
	}

	return
}

// each big order eat two small orders
func (mg *MessageGenerator) TwoOnOneMessages(height int, timeNow time.Time) (tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, accounts map[string]pub.Account, transfers *pub.Transfers) {
	timePub := timeNow.Unix()
	if height%2 != 0 {
		// place small buy orders
		tradesToPublish = make([]*pub.Trade, 0)
		orderChanges = make(orderPkg.OrderChanges, mg.NumOfTradesPerBlock, mg.NumOfTradesPerBlock)
		accounts = make(map[string]pub.Account)

		for i := 0; i < mg.NumOfTradesPerBlock; i++ {
			buyOrder := makeOrderInfo(mg.buyerAddrs[i], 1, int64(height), 100000000, 100000000, 0, timePub)
			mg.OrderChangeMap[buyOrder.Id] = &buyOrder
			orderChanges[i] = orderPkg.OrderChange{buyOrder.Id, orderPkg.Ack, nil}
		}
	} else {
		// place big sell orders
		tradesToPublish = make([]*pub.Trade, mg.NumOfTradesPerBlock, mg.NumOfTradesPerBlock)
		orderChanges = make(orderPkg.OrderChanges, mg.NumOfTradesPerBlock/2, mg.NumOfTradesPerBlock/2)
		accounts = make(map[string]pub.Account, mg.NumOfTradesPerBlock)

		for i := 0; i < mg.NumOfTradesPerBlock; i++ {
			buyOrder := makeOrderInfo(mg.buyerAddrs[i], 1, int64(height/2), 100000000, 100000000, 100000000, timePub)
			var cumQty int64
			if i%2 == 0 {
				cumQty = 100000000
			} else {
				cumQty = 200000000
			}
			sellOrder := makeOrderInfo(mg.sellerAddrs[i/2], 2, int64(height), 100000000, 200000000, cumQty, timePub)
			if i%2 == 0 {
				orderChanges[i/2] = orderPkg.OrderChange{sellOrder.Id, orderPkg.Ack, nil}
			}
			tradesToPublish[i] = makeTradeToPub(fmt.Sprintf("%d-%d", height, i), buyOrder.Id, sellOrder.Id, mg.sellerAddrs[i].String(),
				mg.buyerAddrs[i].String(), 100000000, 100000000)
			mg.OrderChangeMap[buyOrder.Id] = &buyOrder
			mg.OrderChangeMap[sellOrder.Id] = &sellOrder
			accounts[mg.buyerAddrs[i/2].String()] = pub.Account{string(mg.buyerAddrs[i].String()), "", []*pub.AssetBalance{{"NNB", 10000000000000000 + 100000000*int64(height), 0, 0}, {"BNB", 10000000000000000 - 100000000*int64(height), 0, 0}}}
			accounts[mg.sellerAddrs[i].String()] = pub.Account{string(mg.sellerAddrs[i].String()), "", []*pub.AssetBalance{{"NNB", 10000000000000000 - 200000000*int64(height), 0, 0}, {"BNB", 10000000000000000 + 200000000*int64(height), 0, 0}}}
		}
	}
	transfers = &pub.Transfers{Height: int64(height), Num: 0, Transfers: []pub.Transfer{}}
	for i := 0; i < mg.NumOfTransferPerBlock; i++ {
		t := pub.Transfer{From: mg.sendAddrs[i].String(), To: []pub.Receiver{{mg.receiverAddrs[i].String(), []pub.Coin{{"BNB", rand.Int63n(math.MaxInt64)}}}, {mg.receiverAddrs[2*i+1].String(), []pub.Coin{{"BTC", rand.Int63n(math.MaxInt64)}}}}}
		transfers.Transfers = append(transfers.Transfers, t)
	}
	return
}

// simulate 1 million expire orders to publish at breathe block
func (mg *MessageGenerator) ExpireMessages(height int, timeNow time.Time) (tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, accounts map[string]pub.Account) {
	timePub := timeNow.Unix()
	tradesToPublish = make([]*pub.Trade, 0)
	orderChanges = make(orderPkg.OrderChanges, 0, 100000)
	accounts = make(map[string]pub.Account)

	for i := 0; i < 100000; i++ {
		o := makeOrderInfo(mg.buyerAddrs[0], 1, int64(height), 1000000000, 1000000000, 500000000, timePub)
		mg.OrderChangeMap[fmt.Sprintf("%d", i)] = &o
		orderChanges = append(orderChanges, orderPkg.OrderChange{fmt.Sprintf("%d", i), orderPkg.Expired, nil})
	}
	return
}

func (mg MessageGenerator) Publish(height, timePub int64, tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, orderChangesMap orderPkg.OrderInfoForPublish, accounts map[string]pub.Account, transfers *pub.Transfers) {
	orderChangesCopy := make(orderPkg.OrderInfoForPublish, len(orderChangesMap))
	for k, v := range orderChangesMap {
		orderChangesCopy[k] = v
	}
	pub.ToPublishCh <- pub.NewBlockInfoToPublish(
		height,
		timePub,
		tradesToPublish,
		new(pub.Proposals),
		orderChanges,
		orderChangesCopy,
		accounts,
		nil,
		pub.BlockFee{},
		nil,
		transfers)
}

func makeOrderInfo(sender sdk.AccAddress, side int8, height, price, qty, cumQty, timePub int64) orderPkg.OrderInfo {
	return orderPkg.OrderInfo{
		NewOrderMsg: orderPkg.NewOrderMsg{
			Sender:      sender,
			Id:          orderPkg.GenerateOrderID(height, sender),
			Symbol:      "NNB_BNB",
			OrderType:   0,
			Side:        side,
			Price:       price,
			Quantity:    qty,
			TimeInForce: 1,
		},
		CreatedHeight:        height,
		CreatedTimestamp:     timePub,
		LastUpdatedHeight:    height,
		LastUpdatedTimestamp: timePub,
		CumQty:               cumQty,
		TxHash:               "5C151DB68ACDF5745C45732C7F3ECA0D223EC555",
	}
}

func makeTradeToPub(id, sid, bid, saddr, baddr string, price, qty int64) *pub.Trade {
	return &pub.Trade{
		id,
		"NNB_BNB",
		price,
		qty,
		sid,
		bid,
		"",
		"",
		saddr,
		baddr,
	}
}
