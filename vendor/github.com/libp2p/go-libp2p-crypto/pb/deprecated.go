// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb instead.
package crypto_pb

import core "github.com/libp2p/go-libp2p-core/crypto/pb"

const (
	// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.KeyType_RSA instead.
	KeyType_RSA = core.KeyType_RSA
	// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.KeyType_Ed25519 instead.
	KeyType_Ed25519 = core.KeyType_Ed25519
	// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.KeyType_Secp256k1 instead.
	KeyType_Secp256k1 = core.KeyType_Secp256k1
	// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.KeyType_ECDSA instead.
	KeyType_ECDSA = core.KeyType_ECDSA
)

var (
	// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.ErrInvalidLengthCrypto instead.
	ErrInvalidLengthCrypto = core.ErrInvalidLengthCrypto
	// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.ErrIntOverflowCrypto instead.
	ErrIntOverflowCrypto = core.ErrIntOverflowCrypto
)

// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.KeyType_name instead.
var KeyType_name = core.KeyType_name

// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.KeyType_value instead.
var KeyType_value = core.KeyType_value

// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.KeyType instead.
type KeyType = core.KeyType

// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.PublicKey instead.
type PublicKey = core.PublicKey

// Deprecated: use github.com/libp2p/go-libp2p-core/crypto/pb.PrivateKey instead.
type PrivateKey = core.PrivateKey
