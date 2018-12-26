package pub

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/app/config"
)

type MockMarketDataPublisher struct {
	AccountPublished          []*accounts
	BooksPublished            []*Books
	ExecutionResultsPublished []*executionResults
	BlockFeePublished         []BlockFee

	Lock             *sync.Mutex // as mock publisher is only used in testing, its no harm to have this granularity Lock
	MessagePublished uint32      // atomic integer used to determine the published messages
}

func (publisher *MockMarketDataPublisher) publish(msg AvroMsg, tpe msgType, height int64, timestamp int64) {
	publisher.Lock.Lock()
	defer publisher.Lock.Unlock()

	switch tpe {
	case accountsTpe:
		publisher.AccountPublished = append(publisher.AccountPublished, msg.(*accounts))
	case booksTpe:
		publisher.BooksPublished = append(publisher.BooksPublished, msg.(*Books))
	case executionResultTpe:
		publisher.ExecutionResultsPublished = append(publisher.ExecutionResultsPublished, msg.(*executionResults))
	case blockFeeTpe:
		publisher.BlockFeePublished = append(publisher.BlockFeePublished, msg.(BlockFee))
	default:
		panic(fmt.Errorf("does not support type %s", tpe.String()))
	}

	atomic.AddUint32(&publisher.MessagePublished, 1)
}

func (publisher *MockMarketDataPublisher) Stop() {
	publisher.Lock.Lock()
	defer publisher.Lock.Unlock()

	publisher.AccountPublished = make([]*accounts, 0)
	publisher.BooksPublished = make([]*Books, 0)
	publisher.ExecutionResultsPublished = make([]*executionResults, 0)
}

func NewMockMarketDataPublisher(logger log.Logger, config *config.PublicationConfig) (publisher *MockMarketDataPublisher) {
	publisher = &MockMarketDataPublisher{
		make([]*accounts, 0),
		make([]*Books, 0),
		make([]*executionResults, 0),
		make([]BlockFee, 0),
		&sync.Mutex{},
		0,
	}
	if err := setup(logger, config, nil, publisher); err != nil {
		publisher.Stop()
		logger.Error("Cannot start up market data kafka publisher", "err", err)
		panic(err)
	}
	return publisher
}
