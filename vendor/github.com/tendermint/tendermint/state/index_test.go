package state

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/state/blockindex"
	kv2 "github.com/tendermint/tendermint/state/blockindex/kv"
	"github.com/tendermint/tendermint/state/txindex"
	"github.com/tendermint/tendermint/state/txindex/kv"
	"github.com/tendermint/tendermint/types"
)

func TestSetHeight(t *testing.T) {

	indexDb := db.NewMemDB()
	indexHub := NewIndexHub(0, indexDb, nil, nil)
	indexHub.SetLogger(log.TestingLogger())

	realHeightAtFirst := indexHub.GetIndexedHeight()
	assert.Equal(t, int64(-1), realHeightAtFirst)
	height := int64(1024)
	indexHub.SetIndexedHeight(height)
	realHeight := indexHub.GetIndexedHeight()
	assert.Equal(t, height, realHeight)
}

func TestCountDown(t *testing.T) {
	// event bus
	eventBus := types.NewEventBus()
	eventBus.SetLogger(log.TestingLogger())
	err := eventBus.Start()
	require.NoError(t, err)
	defer eventBus.Stop()

	indexDb := db.NewMemDB()

	// start tx index
	txIndexer := kv.NewTxIndex(indexDb, kv.IndexAllTags(), kv.EnableRangeQuery())
	txIndexSvc := txindex.NewIndexerService(txIndexer, eventBus)
	txIndexSvc.SetLogger(log.TestingLogger())
	err = txIndexSvc.Start()
	require.NoError(t, err)
	defer txIndexSvc.Stop()

	// start block index
	blockIndexer := kv2.NewBlockIndex(indexDb)
	blockIndexSvc := blockindex.NewIndexerService(blockIndexer, eventBus)
	blockIndexSvc.SetLogger(log.TestingLogger())
	err = blockIndexSvc.Start()
	require.NoError(t, err)
	defer blockIndexSvc.Stop()

	// start index hub
	indexHub := NewIndexHub(0, indexDb, nil, eventBus)
	indexHub.SetLogger(log.TestingLogger())
	indexHub.RegisterIndexSvc(txIndexSvc)
	indexHub.RegisterIndexSvc(blockIndexSvc)
	err = indexHub.Start()
	assert.NoError(t, err)

	// publish block with txs
	for h := int64(1); h < 10; h++ {
		numTxs := rand.Int63n(5)
		eventBus.PublishEventNewBlockHeader(types.EventDataNewBlockHeader{
			Header: types.Header{Height: h, NumTxs: numTxs},
		})
		for i := int64(0); i < numTxs; i++ {
			txResult := &types.TxResult{
				Height: h,
				Index:  uint32(i),
				Tx:     types.Tx("foo"),
				Result: abci.ResponseDeliverTx{Code: 0},
			}
			eventBus.PublishEventTx(types.EventDataTx{*txResult})
		}
		// In test case, 100ms is far enough for index
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int64(h), indexHub.GetIndexedHeight())
		// test no memory leak
		assert.Equal(t, len(indexHub.indexTaskCounter), 0)
	}
}
