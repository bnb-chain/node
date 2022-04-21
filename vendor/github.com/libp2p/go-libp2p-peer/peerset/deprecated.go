// Deprecated: use github.com/libp2p/go-libp2p-core/peer/set instead.
package peerset

import core "github.com/libp2p/go-libp2p-core/peer"

// Deprecated: use github.com/libp2p/go-libp2p-core/peer.Set instead.
type PeerSet = core.Set

// Deprecated: use github.com/libp2p/go-libp2p-core/set.NewSet instead.
func New() *PeerSet {
	return core.NewSet()
}

// Deprecated: use github.com/libp2p/go-libp2p-core/peer/set.NewLimitedSet instead.
func NewLimited(size int) *PeerSet {
	return core.NewLimitedSet(size)
}
