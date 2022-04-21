package p2p

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
)

type cmNotifee struct {
	t *p2pTransporter
}

var _ network.Notifiee = (*cmNotifee)(nil)

func (nn *cmNotifee) Connected(n network.Network, c network.Conn) {
	logger.Debugf("[Connected] %s (%s) dir: %d", c.RemotePeer().Pretty(), c.RemoteMultiaddr().String(), c.Stat().Direction)
}

func (nn *cmNotifee) Disconnected(n network.Network, c network.Conn) {
	logger.Debugf("[Disconnected] %s (%s) dir: %d", c.RemotePeer().Pretty(), c.RemoteMultiaddr().String(), c.Stat().Direction)
	//nn.t.streams.Delete(c.RemotePeer().Pretty())
	// TODO: trigger reconnect
}

func (nn *cmNotifee) Listen(n network.Network, addr multiaddr.Multiaddr) {}

func (nn *cmNotifee) ListenClose(n network.Network, addr multiaddr.Multiaddr) {}

func (nn *cmNotifee) OpenedStream(n network.Network, s network.Stream) {
	logger.Debugf("[OpenedStream] %s (%s) memory: %p", s.Conn().RemotePeer().Pretty(), s.Protocol(), s)
}

func (nn *cmNotifee) ClosedStream(n network.Network, s network.Stream) {
	logger.Debugf("[ClosedStream] %s (%s) memory: %p", s.Conn().RemotePeer().Pretty(), s.Protocol(), s)
}
