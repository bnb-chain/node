package kv

import (
	"crypto/sha256"
	"fmt"

	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/state/blockindex"
	"github.com/tendermint/tendermint/types"
)

var _ blockindex.BlockIndexer = (*BlockIndex)(nil)

// BlockIndex is the simplest possible indexer, backed by key-value storage (levelDB).
type BlockIndex struct {
	store dbm.DB
}

// NewBlockIndex creates new KV indexer.
func NewBlockIndex(store dbm.DB, options ...func(*BlockIndex)) *BlockIndex {
	bki := &BlockIndex{store: store}
	for _, o := range options {
		o(bki)
	}
	return bki
}

// Get gets block height from the BlockIndex storage and returns it or 0 if the
// block is not found.
func (bki *BlockIndex) Get(hash []byte) (int64, error) {
	if len(hash) != sha256.Size {
		return 0, blockindex.ErrorHashMissLength
	}
	rawBytes := bki.store.Get(hash)
	if rawBytes == nil {
		return 0, nil
	}

	var blockHeight int64
	err := cdc.UnmarshalBinaryBare(rawBytes, &blockHeight)
	if err != nil {
		return 0, fmt.Errorf("Error reading block header: %v", err)
	}

	return blockHeight, nil
}

// Index indexes a single block header
func (bki *BlockIndex) Index(header *types.Header) error {

	hash := header.Hash()
	// index header by hash
	rawBytes, err := cdc.MarshalBinaryBare(header.Height)
	if err != nil {
		return err
	}
	bki.store.SetSync(hash, rawBytes)
	return nil
}
