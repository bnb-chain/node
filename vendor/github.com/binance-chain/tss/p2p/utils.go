package p2p

import (
	"crypto/rand"
	"fmt"
	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	"strings"
	"time"

	"github.com/binance-chain/tss/common"
)

var logger = log.Logger(loggerName)

func DumpDHTRoutine(dht *libp2pdht.IpfsDHT) {
	for {
		dht.RoutingTable().Print()
		time.Sleep(10 * time.Second)
	}
}

func DumpPeersRoutine(host host.Host) {
	for {
		time.Sleep(10 * time.Second)
		builder := strings.Builder{}
		for _, peer := range host.Network().Peers() {
			fmt.Fprintf(&builder, "%s\n", peer)
		}
		logger.Debugf("Dump peers:\n%s", builder.String())
	}
}

func GetMonikerFromExpectedPeers(peer string) string {
	return strings.SplitN(peer, "@", 2)[0]
}

func GetClientIdFromExpectedPeers(peer string) common.TssClientId {
	return common.TssClientId(strings.SplitN(peer, "@", 2)[1])
}

// generate node identifier key
func NewP2pPrivKey() (crypto.PrivKey, peer.ID, error) {
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, "", err
	}

	pid, err := peer.IDFromPublicKey(privKey.GetPublic())
	if err != nil {
		return nil, "", err
	}
	return privKey, pid, nil
}
