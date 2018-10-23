package pub

import (
	"fmt"

	"github.com/BiJie/BinanceChain/app/config"
	"github.com/BiJie/BinanceChain/common/log"
)

type MockMarketDataPublisher struct {
	AccountPublished         []*accounts
	BooksPublished           []*Books
	TradesAndOrdersPublished []*tradesAndOrders
}

func (publisher *MockMarketDataPublisher) publish(msg AvroMsg, tpe msgType, height int64, timestamp int64) {
	switch tpe {
	case accountsTpe:
		publisher.AccountPublished = append(publisher.AccountPublished, msg.(*accounts))
	case booksTpe:
		publisher.BooksPublished = append(publisher.BooksPublished, msg.(*Books))
	case tradesAndOrdersTpe:
		publisher.TradesAndOrdersPublished = append(publisher.TradesAndOrdersPublished, msg.(*tradesAndOrders))
	default:
		panic(fmt.Errorf("does not support type %s", tpe.String()))
	}
}

func (publisher *MockMarketDataPublisher) Stop() {
	publisher.AccountPublished = make([]*accounts, 0)
	publisher.BooksPublished = make([]*Books, 0)
	publisher.TradesAndOrdersPublished = make([]*tradesAndOrders, 0)
}

func NewMockMarketDataPublisher(config *config.PublicationConfig) (publisher *MockMarketDataPublisher) {
	publisher = &MockMarketDataPublisher{
		make([]*accounts, 0),
		make([]*Books, 0),
		make([]*tradesAndOrders, 0)}
	if err := setup(config, publisher); err != nil {
		publisher.Stop()
		log.Error("Cannot start up market data kafka publisher", "err", err)
		panic(err)
	}
	return publisher
}
