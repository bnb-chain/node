package pub

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/app/config"
	"github.com/binance-chain/node/common/log"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
)

// This test ensures schema or AvroOrJsonMsg change are consistent and prevent marshal error in runtime

func TestMain(m *testing.M) {
	Logger = log.With("module", "pub")
	Cfg = &config.PublicationConfig{}
	os.Exit(m.Run())
}

func TestExecutionResultsMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "")
	trades := trades{
		NumOfMsgs: 1,
		Trades: []*Trade{{
			Id: "42-0", Symbol: "NNB_BNB", Price: 100, Qty: 100,
			Sid: "s-1", Bid: "b-1", TickType: 1,
			Sfee: "BNB:8;ETH:1", Bfee: "BNB:10;BTC:1",
			SAddr: "s", BAddr: "b", SSrc: 0, BSrc: 0}},
	}
	orders := Orders{
		NumOfMsgs: 3,
		Orders: []*Order{
			{"NNB_BNB", orderPkg.Ack, "b-1", "", "b", orderPkg.Side.BUY, orderPkg.OrderType.LIMIT, 100, 100, 0, 0, 0, "", 100, 100, orderPkg.TimeInForce.GTE, orderPkg.NEW, ""},
			{"NNB_BNB", orderPkg.FullyFill, "b-1", "42-0", "b", orderPkg.Side.BUY, orderPkg.OrderType.LIMIT, 100, 100, 100, 100, 100, "BNB:10;BTC:1", 100, 100, orderPkg.TimeInForce.GTE, orderPkg.NEW, ""},
			{"NNB_BNB", orderPkg.FullyFill, "s-1", "42-0", "s", orderPkg.Side.SELL, orderPkg.OrderType.LIMIT, 100, 100, 100, 100, 100, "BNB:8;ETH:1", 99, 99, orderPkg.TimeInForce.GTE, orderPkg.NEW, ""},
		},
	}
	proposals := Proposals{
		NumOfMsgs: 3,
		Proposals: []*Proposal{
			{1, Succeed},
			{2, Succeed},
			{3, Failed},
		},
	}

	valAddr, _ := sdk.ValAddressFromBech32("bva1e2y8w2rz957lahwy0y5h3w53sm8d78qexkn3rh")
	delAddr, _ := sdk.AccAddressFromBech32("bnb1e2y8w2rz957lahwy0y5h3w53sm8d78qex2jpan")
	stakeUpdates := StakeUpdates{
		NumOfMsgs: 1,
		CompletedUnbondingDelegations: []*CompletedUnbondingDelegation{
			{
				Validator: valAddr,
				Delegator: delAddr,
				Amount:    Coin{"BNB", 100000000000},
			},
		},
	}
	msg := ExecutionResults{
		Height:       42,
		Timestamp:    100,
		NumOfMsgs:    8,
		Trades:       trades,
		Orders:       orders,
		Proposals:    proposals,
		StakeUpdates: stakeUpdates,
	}
	_, err := publisher.marshal(&msg, executionResultTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBooksMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "")
	book := OrderBookDelta{"NNB_BNB", []PriceLevel{{100, 100}}, []PriceLevel{{100, 100}}}
	msg := Books{42, 100, 1, []OrderBookDelta{book}}
	_, err := publisher.marshal(&msg, booksTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAccountsMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "")
	accs := []Account{{"b-1", "BNB:1000;BTC:10", []*AssetBalance{{Asset: "BNB", Free: 100}}}}
	msg := Accounts{42, 2, accs}
	_, err := publisher.marshal(&msg, accountsTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBlockFeeMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "")
	msg := BlockFee{1, "BNB:1000;BTC:10", []string{"bnc1", "bnc2", "bnc3"}}
	_, err := publisher.marshal(&msg, blockFeeTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransferMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "")
	msg := Transfers{42, 20, 1000, []Transfer{{TxHash: "123456ABCDE", From: "", To: []Receiver{Receiver{"bnc1", []Coin{{"BNB", 100}, {"BTC", 100}}}, Receiver{"bnc2", []Coin{{"BNB", 200}, {"BTC", 200}}}}}}}
	_, err := publisher.marshal(&msg, transferType)
	if err != nil {
		t.Fatal(err)
	}
}
