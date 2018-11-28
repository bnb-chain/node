package pub

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
)

type MockMarketDataPublisher struct {
	AccountPublished         []*accounts
	BooksPublished           []*Books
	TradesAndOrdersPublished []*tradesAndOrders
	BlockFeePublished        []BlockFee
}

func (publisher *MockMarketDataPublisher) publish(msg AvroMsg, tpe msgType, height int64, timestamp int64) {
	switch tpe {
	case accountsTpe:
		publisher.AccountPublished = append(publisher.AccountPublished, msg.(*accounts))
	case booksTpe:
		publisher.BooksPublished = append(publisher.BooksPublished, msg.(*Books))
	case tradesAndOrdersTpe:
		publisher.TradesAndOrdersPublished = append(publisher.TradesAndOrdersPublished, msg.(*tradesAndOrders))
	case blockFeeTpe:
		publisher.BlockFeePublished = append(publisher.BlockFeePublished, msg.(BlockFee))
	default:
		panic(fmt.Errorf("does not support type %s", tpe.String()))
	}
}

func (publisher *MockMarketDataPublisher) Stop() {
	publisher.AccountPublished = make([]*accounts, 0)
	publisher.BooksPublished = make([]*Books, 0)
	publisher.TradesAndOrdersPublished = make([]*tradesAndOrders, 0)
}

func NewMockMarketDataPublisher(logger log.Logger, config *config.PublicationConfig) (publisher *MockMarketDataPublisher) {
	publisher = &MockMarketDataPublisher{
		make([]*accounts, 0),
		make([]*Books, 0),
		make([]*tradesAndOrders, 0),
		make([]BlockFee, 0),
	}
	if err := setup(logger, config, publisher); err != nil {
		publisher.Stop()
		logger.Error("Cannot start up market data kafka publisher", "err", err)
		panic(err)
	}
	return publisher
}
