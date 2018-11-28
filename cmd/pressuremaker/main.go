package main

import (
	"fmt"
	"github.com/BiJie/BinanceChain/app/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"os"
	"strconv"

	"github.com/BiJie/BinanceChain/app/pub"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

const (
	defaultNumOfTradesPerBlock = 6000
	defaultBlocksToBePublished = 500
)

func main() {
	blocksToBePublished := defaultBlocksToBePublished
	if len(os.Args) > 1 {
		if b, err := strconv.Atoi(os.Args[1]); err != nil {
			fmt.Println("first argument should be num of blocks to be published")
			panic(err)
		} else {
			blocksToBePublished = b
		}
	}

	//publicationConfig := config.PublicationConfig{
	//	PublishOrderUpdates: true,
	//	OrderUpdatesTopic:   "test_perf",
	//	OrderUpdatesKafka:   "127.0.0.1:9092",
	//
	//	PublishAccountBalance: true,
	//	AccountBalanceTopic:   "test_perf",
	//	AccountBalanceKafka:   "127.0.0.1:9092",
	//}

	publicationConfig := config.PublicationConfig{
		PublishOrderUpdates: true,
		OrderUpdatesTopic:   "ptest",
		OrderUpdatesKafka:   "172.31.47.173:9092",

		PublishAccountBalance: true,
		AccountBalanceTopic:   "ptest",
		AccountBalanceKafka:   "172.31.47.173:9092",
	}

	//publicationConfig := config.PublicationConfig{
	//	PublishOrderUpdates: true,
	//	OrderUpdatesTopic:   "test_perf_ws",
	//	OrderUpdatesKafka:   "172.22.41.119:9092",
	//
	//	PublishAccountBalance: true,
	//	AccountBalanceTopic:   "test_perf_ws",
	//	AccountBalanceKafka:   "172.22.41.119:9092",
	//}

	finishSignal := make(chan struct{})
	publisher := pub.NewKafkaMarketDataPublisher(&publicationConfig)

	generator := MessageGenerator{
		numOfTradesPerBlock: defaultNumOfTradesPerBlock,
		numOfBlocks:         blocksToBePublished,
		orderChangeMap:      make(orderPkg.OrderInfoForPublish, 0),
	}
	generator.setup()

	for h := 1; h <= generator.numOfBlocks; h++ {
		//trades, orders, accounts := generator.makeMessage(h)
		trades, orders, accounts := generator.makeMessage2(h)
		generator.publish(int64(h), trades, orders, generator.orderChangeMap, accounts)
	}

	<-finishSignal
	publisher.Stop()
}

type MessageGenerator struct {
	numOfTradesPerBlock int
	numOfBlocks         int
	orderChangeMap      orderPkg.OrderInfoForPublish

	buyerAddrs  []sdk.AccAddress
	sellerAddrs []sdk.AccAddress

	orderChanges orderPkg.OrderChanges
	trades       []*pub.Trade
}

func (mg *MessageGenerator) setup() {
	coins := sdk.Coins{sdk.NewCoin("BNB", sdk.NewInt(10000000000000000)), sdk.NewCoin("NNB", sdk.NewInt(10000000000000000))}
	_, mg.buyerAddrs, _, _ = mock.CreateGenAccounts(mg.numOfTradesPerBlock, coins)
	_, mg.sellerAddrs, _, _ = mock.CreateGenAccounts(mg.numOfTradesPerBlock, coins)

	// generate some giant orders only make trades
	mg.orderChanges = make(orderPkg.OrderChanges, mg.numOfBlocks*mg.numOfTradesPerBlock*2, mg.numOfBlocks*mg.numOfTradesPerBlock*2)
	mg.trades = make([]*pub.Trade, 0, mg.numOfTradesPerBlock*mg.numOfBlocks)

	for h := 1; h <= mg.numOfBlocks; h++ {
		for i := 0; i < mg.numOfTradesPerBlock; i++ {
			buyOrder := makeOrderInfo(mg.buyerAddrs[i], 1, int64(h), 100000000, 100000000, 100000000)
			sellOrder := makeOrderInfo(mg.sellerAddrs[i], 2, int64(h), 100000000, 100000000, 100000000)
			mg.orderChangeMap[buyOrder.Id] = &buyOrder
			mg.orderChangeMap[sellOrder.Id] = &sellOrder

			mg.orderChanges[(h-1)*mg.numOfTradesPerBlock*2+i*2] = orderPkg.OrderChange{buyOrder.Id, orderPkg.Ack}
			mg.orderChanges[(h-1)*mg.numOfTradesPerBlock*2+i*2+1] = orderPkg.OrderChange{sellOrder.Id, orderPkg.Ack}

			mg.trades = append(mg.trades, makeTradeToPub(fmt.Sprintf("%d-%d", h, i), sellOrder.Id, buyOrder.Id, mg.sellerAddrs[i].String(), mg.buyerAddrs[i].String()))
		}
	}

}

func makeOrderInfo(sender sdk.AccAddress, side int8, height, price, qty, cumQty int64) orderPkg.OrderInfo {
	return orderPkg.OrderInfo{
		NewOrderMsg: orderPkg.NewOrderMsg{
			Sender:      sender,
			Id:          orderPkg.GenerateOrderID(height, sender),
			Symbol:      "NNB_BNB",
			OrderType:   0,
			Side:        side,
			Price:       price,
			Quantity:    qty,
			TimeInForce: 0,
		},
		CreatedHeight:        height,
		CreatedTimestamp:     height,
		LastUpdatedHeight:    height,
		LastUpdatedTimestamp: height,
		CumQty:               cumQty,
		TxHash:               "5C151DB68ACDF5745C45732C7F3ECA0D223EC555",
	}
}

func makeTradeToPub(id, sid, bid, saddr, baddr string) *pub.Trade {
	return &pub.Trade{
		id,
		"NNB_BNB",
		100000000,
		100000000,
		sid,
		bid,
		"",
		"",
		saddr,
		baddr,
	}
}

// each trade has two equal quantity order
func (mg *MessageGenerator) makeMessage(height int) (tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, accounts map[string]pub.Account) {
	tradesToPublish = mg.trades[(height-1)*mg.numOfTradesPerBlock : height*mg.numOfTradesPerBlock]
	accounts = make(map[string]pub.Account, mg.numOfTradesPerBlock*2)

	for i := 0; i < mg.numOfTradesPerBlock; i++ {
		seq := height

		accounts[mg.buyerAddrs[i].String()] = pub.Account{mg.buyerAddrs[i].String(), []*pub.AssetBalance{{"NNB", 10000000000000000 + 100000000*int64(seq), 0, 0}, {"BNB", 10000000000000000 - 100000000*int64(seq), 0, 0}}}
		accounts[mg.sellerAddrs[i].String()] = pub.Account{mg.sellerAddrs[i].String(), []*pub.AssetBalance{{"NNB", 10000000000000000 - 100000000*int64(seq), 0, 0}, {"BNB", 10000000000000000 + 100000000*int64(seq), 0, 0}}}
	}

	orderChanges = mg.orderChanges[(height-1)*mg.numOfTradesPerBlock*2 : height*mg.numOfTradesPerBlock*2]

	return
}

// each big order eat two small orders
func (mg *MessageGenerator) makeMessage2(height int) (tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, accounts map[string]pub.Account) {
	if height%2 != 0 {
		// place small buy orders
		tradesToPublish = make([]*pub.Trade, 0)
		orderChanges = make(orderPkg.OrderChanges, mg.numOfTradesPerBlock, mg.numOfTradesPerBlock)
		accounts = make(map[string]pub.Account)

		for i := 0; i < mg.numOfTradesPerBlock; i++ {
			buyOrder := makeOrderInfo(mg.buyerAddrs[i], 1, int64(height), 100000000, 100000000, 0)
			mg.orderChangeMap[buyOrder.Id] = &buyOrder
			orderChanges[i] = orderPkg.OrderChange{buyOrder.Id, orderPkg.Ack}
		}
	} else {
		// place big sell orders
		tradesToPublish = make([]*pub.Trade, mg.numOfTradesPerBlock, mg.numOfTradesPerBlock)
		orderChanges = make(orderPkg.OrderChanges, mg.numOfTradesPerBlock/2, mg.numOfTradesPerBlock/2)
		accounts = make(map[string]pub.Account, mg.numOfTradesPerBlock)

		for i := 0; i < mg.numOfTradesPerBlock; i++ {
			buyOrder := makeOrderInfo(mg.buyerAddrs[i], 1, int64(height/2), 100000000, 100000000, 100000000)
			var cumQty int64
			if i%2 == 0 {
				cumQty = 100000000
			} else {
				cumQty = 200000000
			}
			sellOrder := makeOrderInfo(mg.sellerAddrs[i/2], 2, int64(height), 100000000, 200000000, cumQty)
			if i%2 == 0 {
				orderChanges[i/2] = orderPkg.OrderChange{sellOrder.Id, orderPkg.Ack}
			}
			tradesToPublish[i] = makeTradeToPub(fmt.Sprintf("%d-%d", height, i), buyOrder.Id, sellOrder.Id, mg.sellerAddrs[i].String(),
				mg.buyerAddrs[i].String())
			accounts[mg.buyerAddrs[i/2].String()] = pub.Account{mg.buyerAddrs[i].String(), []*pub.AssetBalance{{"NNB", 10000000000000000 + 100000000*int64(height), 0, 0}, {"BNB", 10000000000000000 - 100000000*int64(height), 0, 0}}}
			accounts[mg.sellerAddrs[i].String()] = pub.Account{mg.sellerAddrs[i].String(), []*pub.AssetBalance{{"NNB", 10000000000000000 - 200000000*int64(height), 0, 0}, {"BNB", 10000000000000000 + 200000000*int64(height), 0, 0}}}
		}
	}
	return
}

func (mg MessageGenerator) publish(height int64, tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, orderChangesMap orderPkg.OrderInfoForPublish, accounts map[string]pub.Account) {
	orderChangesCopy := make(orderPkg.OrderInfoForPublish, len(orderChangesMap))
	for k, v := range orderChangesMap {
		orderChangesCopy[k] = v
	}
	pub.ToPublishCh <- pub.NewBlockInfoToPublish(
		height,
		height,
		tradesToPublish,
		orderChanges,
		orderChangesCopy,
		accounts,
		nil,
		pub.BlockFee{},
		nil)
}
