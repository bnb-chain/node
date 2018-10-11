package pub

import (
	"github.com/BiJie/BinanceChain/common/log"
	"os"
	"testing"

	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

// This test ensures schema or AvroMsg change are consistent and prevent marshal error in runtime

func TestMain(m *testing.M) {
	Logger = log.With("module", "pub")
	initAvroCodecs()
	os.Exit(m.Run())
}

func TestTradesAndOrdersMarshaling(t *testing.T) {
	trades := trades{
		numOfMsgs: 1,
		trades: []Trade{Trade{
			Id: "42-0", Symbol: "NNB_BNB", Price: 100, Qty: 100,
			Sid: "s-1", Bid: "b-1",
			Sfee: 10, SfeeAsset: "NNB", Bfee: 10, BfeeAsset: "BNB"}},
	}
	orders := orders{
		numOfMsgs: 3,
		orders: []order{
			order{"NNB_BNB", orderPkg.Ack, "b-1", "", "b", orderPkg.Side.BUY, orderPkg.OrderType.LIMIT, 100, 100, 0, 0, 0, 0, "", 100, 100, orderPkg.TimeInForce.GTC, orderPkg.NEW, ""},
			order{"NNB_BNB", orderPkg.FullyFill, "b-1", "42-0", "b", orderPkg.Side.BUY, orderPkg.OrderType.LIMIT, 100, 100, 100, 100, 100, 10, "BNB", 100, 100, orderPkg.TimeInForce.GTC, orderPkg.NEW, ""},
			order{"NNB_BNB", orderPkg.FullyFill, "s-1", "42-0", "s", orderPkg.Side.SELL, orderPkg.OrderType.LIMIT, 100, 100, 100, 100, 100, 10, "NNB", 99, 99, orderPkg.TimeInForce.GTC, orderPkg.NEW, ""},
		},
	}
	msg := tradesAndOrders{
		height:    42,
		timestamp: 100,
		numOfMsgs: 4,
		trades:    trades,
		orders:    orders}
	_, err := marshal(&msg, tradesAndOrdersTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBooksMarshaling(t *testing.T) {
	book := orderBookDelta{"NNB_BNB", []priceLevel{priceLevel{100, 100}}, []priceLevel{priceLevel{100, 100}}}
	msg := books{42, 100, 1, []orderBookDelta{book}}
	_, err := marshal(&msg, booksTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAccountsMarshaling(t *testing.T) {
	accs := []Account{Account{"b-1", []AssetBalance{AssetBalance{Asset: "BNB", Free: 100}}}}
	msg := accounts{42, 2, accs}
	_, err := marshal(&msg, accountsTpe)
	if err != nil {
		t.Fatal(err)
	}
}
