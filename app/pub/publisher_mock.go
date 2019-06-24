package pub

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type MockMarketDataPublisher struct {
	AccountPublished          []*Accounts
	BooksPublished            []*Books
	ExecutionResultsPublished []*ExecutionResults
	BlockFeePublished         []BlockFee
	TransferPublished         []Transfers

	Lock             *sync.Mutex // as mock publisher is only used in testing, its no harm to have this granularity Lock
	MessagePublished uint32      // atomic integer used to determine the published messages
}

func (publisher *MockMarketDataPublisher) publish(msg AvroOrJsonMsg, tpe msgType, height int64, timestamp int64) {
	publisher.Lock.Lock()
	defer publisher.Lock.Unlock()

	switch tpe {
	case accountsTpe:
		publisher.AccountPublished = append(publisher.AccountPublished, msg.(*Accounts))
	case booksTpe:
		publisher.BooksPublished = append(publisher.BooksPublished, msg.(*Books))
	case executionResultTpe:
		publisher.ExecutionResultsPublished = append(publisher.ExecutionResultsPublished, msg.(*ExecutionResults))
	case blockFeeTpe:
		publisher.BlockFeePublished = append(publisher.BlockFeePublished, msg.(BlockFee))
	case transferTpe:
		publisher.TransferPublished = append(publisher.TransferPublished, msg.(Transfers))
	default:
		panic(fmt.Errorf("does not support type %s", tpe.String()))
	}

	atomic.AddUint32(&publisher.MessagePublished, 1)
}

func (publisher *MockMarketDataPublisher) Stop() {
	publisher.Lock.Lock()
	defer publisher.Lock.Unlock()

	publisher.AccountPublished = make([]*Accounts, 0)
	publisher.BooksPublished = make([]*Books, 0)
	publisher.ExecutionResultsPublished = make([]*ExecutionResults, 0)
}

func NewMockMarketDataPublisher() (publisher *MockMarketDataPublisher) {
	publisher = &MockMarketDataPublisher{
		make([]*Accounts, 0),
		make([]*Books, 0),
		make([]*ExecutionResults, 0),
		make([]BlockFee, 0),
		make([]Transfers, 0),
		&sync.Mutex{},
		0,
	}
	return publisher
}
