package state

import (
	"sync"

	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/types"
)

const (
	MaxIndexLag = 100
)

var IndexHeightKey = []byte("IndexHeightKey")

type IndexService interface {
	SetOnIndex(callback func(int64))
}

type IndexHub struct {
	cmn.BaseService
	mtx sync.Mutex

	stateHeight  int64
	expectHeight int64

	// the total registered index service
	numIdxSvc        int
	indexTaskCounter map[int64]int
	indexTaskEvents  chan int64

	stateDB    dbm.DB
	blockStore BlockStore
	eventBus   types.BlockEventPublisher

	metrics *Metrics
}

func NewIndexHub(initialHeight int64, stateDB dbm.DB, blockStore BlockStore, eventBus types.BlockEventPublisher, options ...IndexHubOption) *IndexHub {
	ih := &IndexHub{
		stateHeight:      initialHeight,
		indexTaskCounter: make(map[int64]int, 0),
		indexTaskEvents:  make(chan int64, MaxIndexLag),
		stateDB:          stateDB,
		blockStore:       blockStore,
		eventBus:         eventBus,
		metrics:          NopMetrics(),
	}
	indexedHeight := ih.GetIndexedHeight()
	if indexedHeight < 0 {
		// no indexedHeight found, will do no recover
		ih.expectHeight = ih.stateHeight + 1
	} else {
		ih.expectHeight = indexedHeight + 1
	}
	for _, option := range options {
		option(ih)
	}
	ih.BaseService = *cmn.NewBaseService(nil, "indexHub", ih)
	return ih
}

type IndexHubOption func(*IndexHub)

func IndexHubWithMetrics(metrics *Metrics) IndexHubOption {
	return func(ih *IndexHub) {
		ih.metrics = metrics
	}
}

func (ih *IndexHub) OnStart() error {
	// start listen routine before recovering.
	go ih.commitIndexRoutine()
	ih.recoverIndex()
	return nil
}

func (ih *IndexHub) SetExpectedHeight(height int64) {
	ih.expectHeight = height
}

func (ih *IndexHub) recoverIndex() {
	for h := ih.expectHeight; h <= ih.stateHeight; h++ {
		ih.Logger.Info("try to recover index", "height", h)
		block := ih.blockStore.LoadBlock(h)
		if block == nil {
			ih.Logger.Error("index skip since the the block is missing", "height", h)
		} else {
			abciResponses, err := LoadABCIResponses(ih.stateDB, h)
			if err != nil {
				ih.Logger.Error("failed to load ABCIResponse, will use default")
				abciResponses = NewABCIResponses(block)
				abciResponses.EndBlock = &abci.ResponseEndBlock{}
				abciResponses.BeginBlock = &abci.ResponseBeginBlock{}
			}
			abciValUpdates := abciResponses.EndBlock.ValidatorUpdates
			validatorUpdates, err := types.PB2TM.ValidatorUpdates(abciValUpdates)
			if err != nil {
				ih.Logger.Error("failed to load validatorUpdates, will use nil by default")
			}
			fireEvents(ih.Logger, ih.eventBus, block, abciResponses, validatorUpdates)
		}
	}
}

func (ih *IndexHub) commitIndexRoutine() {
	for {
		select {
		case <-ih.Quit():
			return
		case h := <-ih.indexTaskEvents:
			ih.Logger.Info("finish index", "height", h)
			ih.SetIndexedHeight(h)
			ih.metrics.IndexHeight.Set(float64(h))
		}
	}
}

func (ih *IndexHub) RegisterIndexSvc(idx IndexService) {
	ih.mtx.Lock()
	defer ih.mtx.Unlock()
	if ih.IsRunning() {
		panic("can't RegisterIndexSvc when IndexHub is running")
	}
	idx.SetOnIndex(ih.CountDownAt)
	ih.numIdxSvc++
}

// `CountDownAt` is a callback in index service, keep it simple and fast.
func (ih *IndexHub) CountDownAt(height int64) {
	ih.mtx.Lock()
	defer ih.mtx.Unlock()
	count, exist := ih.indexTaskCounter[height]
	if exist {
		count = count - 1
	} else {
		count = ih.numIdxSvc - 1
	}
	// The higher block won't finish index before lower one.
	if count == 0 && height == ih.expectHeight {
		if exist {
			delete(ih.indexTaskCounter, height)
		}
		ih.expectHeight = ih.expectHeight + 1
		ih.indexTaskEvents <- height
	} else {
		ih.indexTaskCounter[height] = count
	}
}

// set and get won't happen in the same time, won't lock
func (ih *IndexHub) SetIndexedHeight(h int64) {
	rawHeight, err := cdc.MarshalBinaryBare(h)
	if err != nil {
		ih.Logger.Error("failed to MarshalBinaryBare for indexed height", "error", err, "height", h)
	} else {
		ih.stateDB.SetSync(IndexHeightKey, rawHeight)
	}
}

// if never store `IndexHeightKey` in index db, will return -1.
func (ih *IndexHub) GetIndexedHeight() int64 {
	rawHeight := ih.stateDB.Get(IndexHeightKey)
	if rawHeight == nil {
		return -1
	} else {
		var height int64
		err := cdc.UnmarshalBinaryBare(rawHeight, &height)
		if err != nil {
			// should not happen
			panic(err)
		}
		return height
	}
}

// get indexed height from memory to save time for RPC
func (ih *IndexHub) GetHeight() int64 {
	ih.mtx.Lock()
	defer ih.mtx.Unlock()
	return ih.expectHeight - 1
}
