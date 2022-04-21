package mocknet

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"

	"github.com/jbenet/goprocess"
	goprocessctx "github.com/jbenet/goprocess/context"

	ma "github.com/multiformats/go-multiaddr"
)

// peernet implements network.Network
type peernet struct {
	mocknet *mocknet // parent

	peer peer.ID
	ps   peerstore.Peerstore

	// conns are actual live connections between peers.
	// many conns could run over each link.
	// **conns are NOT shared between peers**
	connsByPeer map[peer.ID]map[*conn]struct{}
	connsByLink map[*link]map[*conn]struct{}

	// implement network.Network
	streamHandler network.StreamHandler
	connHandler   network.ConnHandler

	notifmu sync.Mutex
	notifs  map[network.Notifiee]struct{}

	proc goprocess.Process
	sync.RWMutex
}

// newPeernet constructs a new peernet
func newPeernet(ctx context.Context, m *mocknet, p peer.ID, ps peerstore.Peerstore) (*peernet, error) {

	n := &peernet{
		mocknet: m,
		peer:    p,
		ps:      ps,

		connsByPeer: map[peer.ID]map[*conn]struct{}{},
		connsByLink: map[*link]map[*conn]struct{}{},

		notifs: make(map[network.Notifiee]struct{}),
	}

	n.proc = goprocessctx.WithContextAndTeardown(ctx, n.teardown)
	return n, nil
}

func (pn *peernet) teardown() error {

	// close the connections
	for _, c := range pn.allConns() {
		c.Close()
	}
	return nil
}

// allConns returns all the connections between this peer and others
func (pn *peernet) allConns() []*conn {
	pn.RLock()
	var cs []*conn
	for _, csl := range pn.connsByPeer {
		for c := range csl {
			cs = append(cs, c)
		}
	}
	pn.RUnlock()
	return cs
}

// Close calls the ContextCloser func
func (pn *peernet) Close() error {
	return pn.proc.Close()
}

func (pn *peernet) Peerstore() peerstore.Peerstore {
	return pn.ps
}

func (pn *peernet) String() string {
	return fmt.Sprintf("<mock.peernet %s - %d conns>", pn.peer, len(pn.allConns()))
}

// handleNewStream is an internal function to trigger the client's handler
func (pn *peernet) handleNewStream(s network.Stream) {
	pn.RLock()
	handler := pn.streamHandler
	pn.RUnlock()
	if handler != nil {
		go handler(s)
	}
}

// handleNewConn is an internal function to trigger the client's handler
func (pn *peernet) handleNewConn(c network.Conn) {
	pn.RLock()
	handler := pn.connHandler
	pn.RUnlock()
	if handler != nil {
		go handler(c)
	}
}

// DialPeer attempts to establish a connection to a given peer.
// Respects the context.
func (pn *peernet) DialPeer(ctx context.Context, p peer.ID) (network.Conn, error) {
	return pn.connect(p)
}

func (pn *peernet) connect(p peer.ID) (*conn, error) {
	if p == pn.peer {
		return nil, fmt.Errorf("attempted to dial self %s", p)
	}

	// first, check if we already have live connections
	pn.RLock()
	cs, found := pn.connsByPeer[p]
	if found && len(cs) > 0 {
		var chosen *conn
		for c := range cs { // because cs is a map
			chosen = c // select first
			break
		}
		pn.RUnlock()
		return chosen, nil
	}
	pn.RUnlock()

	log.Debugf("%s (newly) dialing %s", pn.peer, p)

	// ok, must create a new connection. we need a link
	links := pn.mocknet.LinksBetweenPeers(pn.peer, p)
	if len(links) < 1 {
		return nil, fmt.Errorf("%s cannot connect to %s", pn.peer, p)
	}

	// if many links found, how do we select? for now, randomly...
	// this would be an interesting place to test logic that can measure
	// links (network interfaces) and select properly
	l := links[rand.Intn(len(links))]

	log.Debugf("%s dialing %s openingConn", pn.peer, p)
	// create a new connection with link
	c := pn.openConn(p, l.(*link))
	return c, nil
}

func (pn *peernet) openConn(r peer.ID, l *link) *conn {
	lc, rc := l.newConnPair(pn)
	log.Debugf("%s opening connection to %s", pn.LocalPeer(), lc.RemotePeer())
	pn.addConn(lc)
	pn.notifyAll(func(n network.Notifiee) {
		n.Connected(pn, lc)
	})
	rc.net.remoteOpenedConn(rc)
	return lc
}

func (pn *peernet) remoteOpenedConn(c *conn) {
	log.Debugf("%s accepting connection from %s", pn.LocalPeer(), c.RemotePeer())
	pn.addConn(c)
	pn.handleNewConn(c)
	pn.notifyAll(func(n network.Notifiee) {
		n.Connected(pn, c)
	})
}

// addConn constructs and adds a connection
// to given remote peer over given link
func (pn *peernet) addConn(c *conn) {
	pn.Lock()
	defer pn.Unlock()

	_, found := pn.connsByPeer[c.RemotePeer()]
	if !found {
		pn.connsByPeer[c.RemotePeer()] = map[*conn]struct{}{}
	}
	pn.connsByPeer[c.RemotePeer()][c] = struct{}{}

	_, found = pn.connsByLink[c.link]
	if !found {
		pn.connsByLink[c.link] = map[*conn]struct{}{}
	}
	pn.connsByLink[c.link][c] = struct{}{}
}

// removeConn removes a given conn
func (pn *peernet) removeConn(c *conn) {
	pn.Lock()
	defer pn.Unlock()

	cs, found := pn.connsByLink[c.link]
	if !found || len(cs) < 1 {
		panic(fmt.Sprintf("attempting to remove a conn that doesnt exist %p", c.link))
	}
	delete(cs, c)

	cs, found = pn.connsByPeer[c.remote]
	if !found {
		panic(fmt.Sprintf("attempting to remove a conn that doesnt exist %v", c.remote))
	}
	delete(cs, c)
}

// Process returns the network's Process
func (pn *peernet) Process() goprocess.Process {
	return pn.proc
}

// LocalPeer the network's LocalPeer
func (pn *peernet) LocalPeer() peer.ID {
	return pn.peer
}

// Peers returns the connected peers
func (pn *peernet) Peers() []peer.ID {
	pn.RLock()
	defer pn.RUnlock()

	peers := make([]peer.ID, 0, len(pn.connsByPeer))
	for _, cs := range pn.connsByPeer {
		for c := range cs {
			peers = append(peers, c.remote)
			break
		}
	}
	return peers
}

// Conns returns all the connections of this peer
func (pn *peernet) Conns() []network.Conn {
	pn.RLock()
	defer pn.RUnlock()

	out := make([]network.Conn, 0, len(pn.connsByPeer))
	for _, cs := range pn.connsByPeer {
		for c := range cs {
			out = append(out, c)
		}
	}
	return out
}

func (pn *peernet) ConnsToPeer(p peer.ID) []network.Conn {
	pn.RLock()
	defer pn.RUnlock()

	cs, found := pn.connsByPeer[p]
	if !found || len(cs) == 0 {
		return nil
	}

	var cs2 []network.Conn
	for c := range cs {
		cs2 = append(cs2, c)
	}
	return cs2
}

// ClosePeer connections to peer
func (pn *peernet) ClosePeer(p peer.ID) error {
	pn.RLock()
	cs, found := pn.connsByPeer[p]
	if !found {
		pn.RUnlock()
		return nil
	}

	var conns []*conn
	for c := range cs {
		conns = append(conns, c)
	}
	pn.RUnlock()
	for _, c := range conns {
		c.Close()
	}
	return nil
}

// BandwidthTotals returns the total amount of bandwidth transferred
func (pn *peernet) BandwidthTotals() (in uint64, out uint64) {
	// need to implement this. probably best to do it in swarm this time.
	// need a "metrics" object
	return 0, 0
}

// Listen tells the network to start listening on given multiaddrs.
func (pn *peernet) Listen(addrs ...ma.Multiaddr) error {
	pn.Peerstore().AddAddrs(pn.LocalPeer(), addrs, peerstore.PermanentAddrTTL)
	return nil
}

// ListenAddresses returns a list of addresses at which this network listens.
func (pn *peernet) ListenAddresses() []ma.Multiaddr {
	return pn.Peerstore().Addrs(pn.LocalPeer())
}

// InterfaceListenAddresses returns a list of addresses at which this network
// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces.
func (pn *peernet) InterfaceListenAddresses() ([]ma.Multiaddr, error) {
	return pn.ListenAddresses(), nil
}

// Connectedness returns a state signaling connection capabilities
// For now only returns Connecter || NotConnected. Expand into more later.
func (pn *peernet) Connectedness(p peer.ID) network.Connectedness {
	pn.Lock()
	defer pn.Unlock()

	cs, found := pn.connsByPeer[p]
	if found && len(cs) > 0 {
		return network.Connected
	}
	return network.NotConnected
}

// NewStream returns a new stream to given peer p.
// If there is no connection to p, attempts to create one.
func (pn *peernet) NewStream(ctx context.Context, p peer.ID) (network.Stream, error) {
	c, err := pn.DialPeer(ctx, p)
	if err != nil {
		return nil, err
	}
	return c.NewStream()
}

// SetStreamHandler sets the new stream handler on the Network.
// This operation is threadsafe.
func (pn *peernet) SetStreamHandler(h network.StreamHandler) {
	pn.Lock()
	pn.streamHandler = h
	pn.Unlock()
}

// SetConnHandler sets the new conn handler on the Network.
// This operation is threadsafe.
func (pn *peernet) SetConnHandler(h network.ConnHandler) {
	pn.Lock()
	pn.connHandler = h
	pn.Unlock()
}

// Notify signs up Notifiee to receive signals when events happen
func (pn *peernet) Notify(f network.Notifiee) {
	pn.notifmu.Lock()
	pn.notifs[f] = struct{}{}
	pn.notifmu.Unlock()
}

// StopNotify unregisters Notifiee from receiving signals
func (pn *peernet) StopNotify(f network.Notifiee) {
	pn.notifmu.Lock()
	delete(pn.notifs, f)
	pn.notifmu.Unlock()
}

// notifyAll runs the notification function on all Notifiees
func (pn *peernet) notifyAll(notification func(f network.Notifiee)) {
	pn.notifmu.Lock()
	var wg sync.WaitGroup
	for n := range pn.notifs {
		// make sure we dont block
		// and they dont block each other.
		wg.Add(1)
		go func(n network.Notifiee) {
			defer wg.Done()
			notification(n)
		}(n)
	}
	wg.Wait()
	pn.notifmu.Unlock()
}
