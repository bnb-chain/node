package blockindex

import (
	"context"

	cmn "github.com/tendermint/tendermint/libs/common"

	"github.com/tendermint/tendermint/types"
)

const (
	subscriber = "BlockIndexerService"
)

// IndexerService connects event bus and block indexer together in order
// to index blocks coming from event bus.
type IndexerService struct {
	cmn.BaseService

	idr      BlockIndexer
	eventBus *types.EventBus

	onIndex func(int64)
}

// NewIndexerService returns a new service instance.
func NewIndexerService(idr BlockIndexer, eventBus *types.EventBus) *IndexerService {
	is := &IndexerService{idr: idr, eventBus: eventBus}
	is.BaseService = *cmn.NewBaseService(nil, "BlockIndexerService", is)
	return is
}

func (is *IndexerService) SetOnIndex(callback func(int64)) {
	is.onIndex = callback
}

// OnStart implements cmn.Service by subscribing for blocks and indexing them by hash.
func (is *IndexerService) OnStart() error {
	blockHeadersSub, err := is.eventBus.SubscribeUnbuffered(context.Background(), subscriber, types.EventQueryNewBlockHeader)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg := <-blockHeadersSub.Out()
			header := msg.Data().(types.EventDataNewBlockHeader).Header

			if err := is.idr.Index(&header); err != nil {
				is.Logger.Error("Failed to index block", "height", header.Height, "err", err)
			} else {
				is.Logger.Info("Indexed block", "height", header.Height, "hash", header.LastBlockID.Hash)
			}
			if is.onIndex != nil {
				is.onIndex(header.Height)
			}
		}
	}()
	return nil
}

// OnStop implements cmn.Service by unsubscribing from blocks.
func (is *IndexerService) OnStop() {
	if is.eventBus.IsRunning() {
		_ = is.eventBus.UnsubscribeAll(context.Background(), subscriber)
	}
}
