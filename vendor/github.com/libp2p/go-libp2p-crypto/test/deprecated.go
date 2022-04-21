// Deprecated: use github.com/libp2p/go-libp2p-core/test instead.
package testutil

import (
	"github.com/libp2p/go-libp2p-core/test"
	ci "github.com/libp2p/go-libp2p-crypto"
)

// Deprecated: use github.com/libp2p/go-libp2p-core/test.RandTestKeyPair instead.
func RandTestKeyPair(typ, bits int) (ci.PrivKey, ci.PubKey, error) {
	return test.RandTestKeyPair(typ, bits)
}

// Deprecated: use github.com/libp2p/go-libp2p-core/test.SeededTestKeyPair instead.
func SeededTestKeyPair(typ, bits int, seed int64) (ci.PrivKey, ci.PubKey, error) {
	return test.SeededTestKeyPair(typ, bits, seed)
}
