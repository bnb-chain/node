package null

import (
	"errors"

	"github.com/tendermint/tendermint/state/blockindex"
	"github.com/tendermint/tendermint/types"
)

var _ blockindex.BlockIndexer = (*BlockIndex)(nil)

// BlockIndex acts as a /dev/null.
type BlockIndex struct{}

// Get on a BlockIndex is disabled and panics when invoked.
func (bki *BlockIndex) Get(hash []byte) (int64, error) {
	return 0, errors.New(`Indexing is disabled (set 'block_index = "kv"' in config)`)
}

// Index is a noop and always returns nil.
func (bki *BlockIndex) Index(result *types.Header) error {
	return nil
}
