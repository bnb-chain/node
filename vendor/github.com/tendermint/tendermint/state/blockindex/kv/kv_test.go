package kv

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/types"
)

func genHeader() (*types.Header, cmn.HexBytes) {
	height := cmn.RandInt64()
	header := types.Header{Height: height, ValidatorsHash: cmn.RandBytes(20)}
	hash := header.Hash()
	return &header, hash
}

func TestBlockIndex(t *testing.T) {
	indexer := NewBlockIndex(db.NewMemDB())

	blockHeader, hash := genHeader()

	if err := indexer.Index(blockHeader); err != nil {
		t.Error(err)
	}

	loadedBlockHeight, err := indexer.Get(hash)
	require.NoError(t, err)
	assert.Equal(t, blockHeader.Height, loadedBlockHeight)

	blockHeader2, hash2 := genHeader()

	err = indexer.Index(blockHeader2)
	require.NoError(t, err)

	loadedBlockHeight2, err := indexer.Get(hash2)
	require.NoError(t, err)
	assert.Equal(t, blockHeader2.Height, loadedBlockHeight2)
}

func BenchmarkBlockIndex(b *testing.B) {
	dir, err := ioutil.TempDir("", "block_index_db")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir) // nolint: errcheck

	store := db.NewDB("block_index", "leveldb", dir)
	indexer := NewBlockIndex(store)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		block, _ := genHeader()
		err = indexer.Index(block)
	}
	if err != nil {
		b.Fatal(err)
	}
}
