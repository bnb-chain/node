package pub

import (
	"os"
	"testing"

	"github.com/BiJie/BinanceChain/common/log"
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
		trades: []*Trade{{
			Id: "42-0", Symbol: "NNB_BNB", Price: 100, Qty: 100,
			Sid: "s-1", Bid: "b-1",
			Sfee: "BNB:8;ETH:1", Bfee: "BNB:10;BTC:1"}},
	}
	orders := orders{
		numOfMsgs: 3,
		orders: []*order{
			{"NNB_BNB", orderPkg.Ack, "b-1", "", "b", orderPkg.Side.BUY, orderPkg.OrderType.LIMIT, 100, 100, 0, 0, 0, "", 100, 100, orderPkg.TimeInForce.GTC, orderPkg.NEW, ""},
			{"NNB_BNB", orderPkg.FullyFill, "b-1", "42-0", "b", orderPkg.Side.BUY, orderPkg.OrderType.LIMIT, 100, 100, 100, 100, 100, "BNB:10;BTC:1", 100, 100, orderPkg.TimeInForce.GTC, orderPkg.NEW, ""},
			{"NNB_BNB", orderPkg.FullyFill, "s-1", "42-0", "s", orderPkg.Side.SELL, orderPkg.OrderType.LIMIT, 100, 100, 100, 100, 100, "BNB:8;ETH:1", 99, 99, orderPkg.TimeInForce.GTC, orderPkg.NEW, ""},
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
	book := OrderBookDelta{"NNB_BNB", []PriceLevel{{100, 100}}, []PriceLevel{{100, 100}}}
	msg := Books{42, 100, 1, []OrderBookDelta{book}}
	_, err := marshal(&msg, booksTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAccountsMarshaling(t *testing.T) {
	accs := []Account{{"b-1", []*AssetBalance{{Asset: "BNB", Free: 100}}}}
	msg := accounts{42, 2, accs}
	_, err := marshal(&msg, accountsTpe)
	if err != nil {
		t.Fatal(err)
	}
}
