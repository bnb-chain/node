package blockindex_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/state/blockindex"
	"github.com/tendermint/tendermint/state/blockindex/kv"
	"github.com/tendermint/tendermint/types"
)

func genHeader() (*types.Header, cmn.HexBytes) {
	height := cmn.RandInt64()
	header := types.Header{Height: height, ValidatorsHash: cmn.RandBytes(20)}
	hash := header.Hash()
	return &header, hash
}

func TestIndexerServiceIndexesBlocks(t *testing.T) {
	// event bus
	eventBus := types.NewEventBus()
	eventBus.SetLogger(log.TestingLogger())
	err := eventBus.Start()
	require.NoError(t, err)
	defer eventBus.Stop()

	// block indexer
	store := db.NewMemDB()
	blockIndexer := kv.NewBlockIndex(store)

	service := blockindex.NewIndexerService(blockIndexer, eventBus)
	service.SetLogger(log.TestingLogger())
	err = service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// publish block
	header, hash := genHeader()
	eventBus.PublishEventNewBlockHeader(types.EventDataNewBlockHeader{
		Header: *header,
	})

	time.Sleep(100 * time.Millisecond)

	loadedBlockHeight, err := blockIndexer.Get(hash)
	assert.NoError(t, err)
	assert.Equal(t, loadedBlockHeight, header.Height)
}
