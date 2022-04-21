// Deprecated: use github.com/libp2p/go-libp2p-core/routing instead.
package routing

import (
	"context"

	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"

	core "github.com/libp2p/go-libp2p-core/routing"
)

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.ErrNotFound instead.
var ErrNotFound = core.ErrNotFound

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.ErrNotSupported instead.
var ErrNotSupported = core.ErrNotSupported

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.ContentRouting instead.
type ContentRouting = core.ContentRouting

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.PeerRouting instead.
type PeerRouting = core.PeerRouting

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.ValueStore instead.
type ValueStore = core.ValueStore

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.Routing instead.
type IpfsRouting = core.Routing

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.PubKeyFetcher instead.
type PubKeyFetcher = core.PubKeyFetcher

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.KeyForPublicKey instead.
func KeyForPublicKey(id peer.ID) string {
	return core.KeyForPublicKey(id)
}

// Deprecated: use github.com/libp2p/go-libp2p-core/routing.GetPublicKey instead.
func GetPublicKey(r core.ValueStore, ctx context.Context, p peer.ID) (ci.PubKey, error) {
	return core.GetPublicKey(r, ctx, p)
}
