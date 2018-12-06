package utils

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mock"

	"github.com/BiJie/BinanceChain/app/pub"
	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

type MessageGenerator struct {
	NumOfTradesPerBlock int
	NumOfBlocks         int
	OrderChangeMap      orderPkg.OrderInfoForPublish

	buyerAddrs  []sdk.AccAddress
	sellerAddrs []sdk.AccAddress

	orderChanges orderPkg.OrderChanges
	trades       []*pub.Trade
}

func (mg *MessageGenerator) Setup() {
	coins := sdk.Coins{sdk.NewCoin("BNB", 10000000000000000), sdk.NewCoin("NNB", 10000000000000000)}
	_, mg.buyerAddrs, _, _ = mock.CreateGenAccounts(mg.NumOfTradesPerBlock, coins)
	_, mg.sellerAddrs, _, _ = mock.CreateGenAccounts(mg.NumOfTradesPerBlock, coins)
}

// each trade has two equal quantity order
func (mg *MessageGenerator) OneOnOneMessages(height int) (tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, accounts map[string]pub.Account) {
	tradesToPublish = make([]*pub.Trade, mg.NumOfTradesPerBlock)
	accounts = make(map[string]pub.Account, mg.NumOfTradesPerBlock*2)
	orderChanges = make(orderPkg.OrderChanges, mg.NumOfTradesPerBlock*2)

	for i := 0; i < mg.NumOfTradesPerBlock; i++ {
		seq := height

		buyOrder := makeOrderInfo(mg.buyerAddrs[i], 1, int64(height), 100000000, 100000000, 100000000)
		sellOrder := makeOrderInfo(mg.sellerAddrs[i], 2, int64(height), 100000000, 100000000, 100000000)
		mg.OrderChangeMap[buyOrder.Id] = &buyOrder
		mg.OrderChangeMap[sellOrder.Id] = &sellOrder

		orderChanges[i*2] = orderPkg.OrderChange{buyOrder.Id, orderPkg.Ack}
		orderChanges[i*2+1] = orderPkg.OrderChange{sellOrder.Id, orderPkg.Ack}

		tradesToPublish[i] = makeTradeToPub(fmt.Sprintf("%d-%d", height, i), sellOrder.Id, buyOrder.Id, mg.sellerAddrs[i].String(), mg.buyerAddrs[i].String())

		accounts[mg.buyerAddrs[i].String()] = pub.Account{string(mg.buyerAddrs[i]), "", []*pub.AssetBalance{{"NNB", 10000000000000000 + 100000000*int64(seq), 0, 0}, {"BNB", 10000000000000000 - 100000000*int64(seq), 0, 0}}}
		accounts[mg.sellerAddrs[i].String()] = pub.Account{string(mg.sellerAddrs[i]), "", []*pub.AssetBalance{{"NNB", 10000000000000000 - 100000000*int64(seq), 0, 0}, {"BNB", 10000000000000000 + 100000000*int64(seq), 0, 0}}}
	}

	return
}

// each big order eat two small orders
func (mg *MessageGenerator) TwoOnOneMessages(height int) (tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, accounts map[string]pub.Account) {
	if height%2 != 0 {
		// place small buy orders
		tradesToPublish = make([]*pub.Trade, 0)
		orderChanges = make(orderPkg.OrderChanges, mg.NumOfTradesPerBlock, mg.NumOfTradesPerBlock)
		accounts = make(map[string]pub.Account)

		for i := 0; i < mg.NumOfTradesPerBlock; i++ {
			buyOrder := makeOrderInfo(mg.buyerAddrs[i], 1, int64(height), 100000000, 100000000, 0)
			mg.OrderChangeMap[buyOrder.Id] = &buyOrder
			orderChanges[i] = orderPkg.OrderChange{buyOrder.Id, orderPkg.Ack}
		}
	} else {
		// place big sell orders
		tradesToPublish = make([]*pub.Trade, mg.NumOfTradesPerBlock, mg.NumOfTradesPerBlock)
		orderChanges = make(orderPkg.OrderChanges, mg.NumOfTradesPerBlock/2, mg.NumOfTradesPerBlock/2)
		accounts = make(map[string]pub.Account, mg.NumOfTradesPerBlock)

		for i := 0; i < mg.NumOfTradesPerBlock; i++ {
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
			mg.OrderChangeMap[buyOrder.Id] = &buyOrder
			mg.OrderChangeMap[sellOrder.Id] = &sellOrder
			accounts[mg.buyerAddrs[i/2].String()] = pub.Account{string(mg.buyerAddrs[i].String()), "", []*pub.AssetBalance{{"NNB", 10000000000000000 + 100000000*int64(height), 0, 0}, {"BNB", 10000000000000000 - 100000000*int64(height), 0, 0}}}
			accounts[mg.sellerAddrs[i].String()] = pub.Account{string(mg.sellerAddrs[i].String()), "", []*pub.AssetBalance{{"NNB", 10000000000000000 - 200000000*int64(height), 0, 0}, {"BNB", 10000000000000000 + 200000000*int64(height), 0, 0}}}
		}
	}
	return
}

func (mg MessageGenerator) Publish(height int64, tradesToPublish []*pub.Trade, orderChanges orderPkg.OrderChanges, orderChangesMap orderPkg.OrderInfoForPublish, accounts map[string]pub.Account) {
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
