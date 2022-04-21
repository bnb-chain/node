package identify_test

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/libp2p/go-eventbus"
	ic "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"

	blhost "github.com/libp2p/go-libp2p-blankhost"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"

	ma "github.com/multiformats/go-multiaddr"
)

func subtestIDService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1 := blhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	h2 := blhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	h1p := h1.ID()
	h2p := h2.ID()

	ids1 := identify.NewIDService(ctx, h1)
	ids2 := identify.NewIDService(ctx, h2)

	testKnowsAddrs(t, h1, h2p, []ma.Multiaddr{}) // nothing
	testKnowsAddrs(t, h2, h1p, []ma.Multiaddr{}) // nothing

	forgetMe, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")

	h2.Peerstore().AddAddr(h1p, forgetMe, peerstore.RecentlyConnectedAddrTTL)
	time.Sleep(500 * time.Millisecond)

	h2pi := h2.Peerstore().PeerInfo(h2p)
	if err := h1.Connect(ctx, h2pi); err != nil {
		t.Fatal(err)
	}

	h1t2c := h1.Network().ConnsToPeer(h2p)
	if len(h1t2c) == 0 {
		t.Fatal("should have a conn here")
	}

	ids1.IdentifyConn(h1t2c[0])

	// the IDService should be opened automatically, by the network.
	// what we should see now is that both peers know about each others listen addresses.
	t.Log("test peer1 has peer2 addrs correctly")
	testKnowsAddrs(t, h1, h2p, h2.Peerstore().Addrs(h2p)) // has them
	testHasProtocolVersions(t, h1, h2p)
	testHasPublicKey(t, h1, h2p, h2.Peerstore().PubKey(h2p)) // h1 should have h2's public key

	// now, this wait we do have to do. it's the wait for the Listening side
	// to be done identifying the connection.
	c := h2.Network().ConnsToPeer(h1.ID())
	if len(c) < 1 {
		t.Fatal("should have connection by now at least.")
	}
	ids2.IdentifyConn(c[0])

	addrs := h1.Peerstore().Addrs(h1p)
	addrs = append(addrs, forgetMe)

	// and the protocol versions.
	t.Log("test peer2 has peer1 addrs correctly")
	testKnowsAddrs(t, h2, h1p, addrs) // has them
	testHasProtocolVersions(t, h2, h1p)
	testHasPublicKey(t, h2, h1p, h1.Peerstore().PubKey(h1p)) // h1 should have h2's public key

	// Need both sides to actually notice that the connection has been closed.
	h1.Network().ClosePeer(h2p)
	h2.Network().ClosePeer(h1p)
	if len(h2.Network().ConnsToPeer(h1.ID())) != 0 || len(h1.Network().ConnsToPeer(h2.ID())) != 0 {
		t.Fatal("should have no connections")
	}

	testKnowsAddrs(t, h2, h1p, addrs)
	testKnowsAddrs(t, h1, h2p, h2.Peerstore().Addrs(h2p))

	time.Sleep(500 * time.Millisecond)

	// Forget the first one.
	testKnowsAddrs(t, h2, h1p, addrs[:len(addrs)-1])

	time.Sleep(1 * time.Second)

	// Forget the rest.
	testKnowsAddrs(t, h1, h2p, []ma.Multiaddr{})
	testKnowsAddrs(t, h2, h1p, []ma.Multiaddr{})
}

func testKnowsAddrs(t *testing.T, h host.Host, p peer.ID, expected []ma.Multiaddr) {
	t.Helper()

	actual := h.Peerstore().Addrs(p)

	if len(actual) != len(expected) {
		t.Errorf("expected: %s", expected)
		t.Errorf("actual: %s", actual)
		t.Fatal("dont have the same addresses")
	}

	have := map[string]struct{}{}
	for _, addr := range actual {
		have[addr.String()] = struct{}{}
	}
	for _, addr := range expected {
		if _, found := have[addr.String()]; !found {
			t.Errorf("%s did not have addr for %s: %s", h.ID(), p, addr)
		}
	}
}

func testHasProtocolVersions(t *testing.T, h host.Host, p peer.ID) {
	v, err := h.Peerstore().Get(p, "ProtocolVersion")
	if v == nil {
		t.Error("no protocol version")
		return
	}
	if v.(string) != identify.LibP2PVersion {
		t.Error("protocol mismatch", err)
	}
	v, err = h.Peerstore().Get(p, "AgentVersion")
	if v.(string) != identify.ClientVersion {
		t.Error("agent version mismatch", err)
	}
}

func testHasPublicKey(t *testing.T, h host.Host, p peer.ID, shouldBe ic.PubKey) {
	k := h.Peerstore().PubKey(p)
	if k == nil {
		t.Error("no public key")
		return
	}
	if !k.Equals(shouldBe) {
		t.Error("key mismatch")
		return
	}

	p2, err := peer.IDFromPublicKey(k)
	if err != nil {
		t.Error("could not make key")
	} else if p != p2 {
		t.Error("key does not match peerid")
	}
}

// TestIDServiceWait gives the ID service 1s to finish after dialing
// this is because it used to be concurrent. Now, Dial wait till the
// id service is done.
func TestIDService(t *testing.T) {
	oldTTL := peerstore.RecentlyConnectedAddrTTL
	peerstore.RecentlyConnectedAddrTTL = time.Second
	defer func() {
		peerstore.RecentlyConnectedAddrTTL = oldTTL
	}()

	N := 3
	for i := 0; i < N; i++ {
		subtestIDService(t)
	}
}

func TestProtoMatching(t *testing.T) {
	tcp1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	tcp2, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/2345")
	tcp3, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/4567")
	utp, _ := ma.NewMultiaddr("/ip4/1.2.3.4/udp/1234/utp")

	if !identify.HasConsistentTransport(tcp1, []ma.Multiaddr{tcp2, tcp3, utp}) {
		t.Fatal("expected match")
	}

	if identify.HasConsistentTransport(utp, []ma.Multiaddr{tcp2, tcp3}) {
		t.Fatal("expected mismatch")
	}
}

func TestIdentifyDeltaOnProtocolChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1 := blhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	h2 := blhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	defer h2.Close()
	defer h1.Close()

	h2.SetStreamHandler(protocol.TestingID, func(_ network.Stream) {})

	ids1 := identify.NewIDService(ctx, h1)
	_ = identify.NewIDService(ctx, h2)

	if err := h1.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}); err != nil {
		t.Fatal(err)
	}

	conn := h1.Network().ConnsToPeer(h2.ID())[0]
	ids1.IdentifyConn(conn)
	select {
	case <-ids1.IdentifyWait(conn):
	case <-time.After(5 * time.Second):
		t.Fatal("took over 5 seconds to identify")
	}

	protos, err := h1.Peerstore().GetProtocols(h2.ID())
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(protos)
	if sort.SearchStrings(protos, string(protocol.TestingID)) == len(protos) {
		t.Fatalf("expected peer 1 to know that peer 2 speaks the Test protocol amongst others")
	}

	// set up a subscriber to listen to peer protocol updated events in h1. We expect to receive events from h2
	// as protocols are added and removed.
	sub, err := h1.EventBus().Subscribe(&event.EvtPeerProtocolsUpdated{}, eventbus.BufSize(16))
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	// add two new protocols in h2 and wait for identify to send deltas.
	h2.SetStreamHandler(protocol.ID("foo"), func(_ network.Stream) {})
	h2.SetStreamHandler(protocol.ID("bar"), func(_ network.Stream) {})
	<-time.After(500 * time.Millisecond)

	// check that h1 now knows about h2's new protocols.
	protos, err = h1.Peerstore().GetProtocols(h2.ID())
	if err != nil {
		t.Fatal(err)
	}
	have := make(map[string]struct{}, len(protos))
	for _, p := range protos {
		have[p] = struct{}{}
	}

	if _, ok := have["foo"]; !ok {
		t.Fatalf("expected peer 1 to know that peer 2 now speaks protocol 'foo', known: %v", protos)
	}
	if _, ok := have["bar"]; !ok {
		t.Fatalf("expected peer 1 to know that peer 2 now speaks protocol 'bar', known: %v", protos)
	}

	// remove one of the newly added protocols from h2, and wait for identify to send the delta.
	h2.RemoveStreamHandler(protocol.ID("bar"))
	<-time.After(500 * time.Millisecond)

	// check that h1 now has forgotten about h2's bar protocol.
	protos, err = h1.Peerstore().GetProtocols(h2.ID())
	if err != nil {
		t.Fatal(err)
	}
	have = make(map[string]struct{}, len(protos))
	for _, p := range protos {
		have[p] = struct{}{}
	}
	if _, ok := have["foo"]; !ok {
		t.Fatalf("expected peer 1 to know that peer 2 now speaks protocol 'foo', known: %v", protos)
	}
	if _, ok := have["bar"]; ok {
		t.Fatalf("expected peer 1 to have forgotten that peer 2 spoke protocol 'bar', known: %v", protos)
	}

	// make sure that h1 emitted events in the eventbus for h2's protocol updates.
	evts := make([]event.EvtPeerProtocolsUpdated, 3)
	done := make(chan struct{})
	go func() {
		evts[0] = (<-sub.Out()).(event.EvtPeerProtocolsUpdated)
		evts[1] = (<-sub.Out()).(event.EvtPeerProtocolsUpdated)
		evts[2] = (<-sub.Out()).(event.EvtPeerProtocolsUpdated)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out while consuming events from subscription")
	}

	added := protocol.ConvertToStrings(append(evts[0].Added, append(evts[1].Added, evts[2].Added...)...))
	removed := protocol.ConvertToStrings(append(evts[0].Removed, append(evts[1].Removed, evts[2].Removed...)...))
	sort.Strings(added)
	sort.Strings(removed)

	if !reflect.DeepEqual(added, []string{"bar", "foo"}) {
		t.Fatalf("expected to have received updates for added protos")
	}
	if !reflect.DeepEqual(removed, []string{"bar"}) {
		t.Fatalf("expected to have received updates for removed protos")
	}
}

// TestIdentifyDeltaWhileIdentifyingConn tests that the host waits to push delta updates if an identify is ongoing.
func TestIdentifyDeltaWhileIdentifyingConn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1 := blhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	h2 := blhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	defer h2.Close()
	defer h1.Close()

	_ = identify.NewIDService(ctx, h1)
	ids2 := identify.NewIDService(ctx, h2)

	// replace the original identify handler by one that blocks until we close the block channel.
	// this allows us to control how long identify runs.
	block := make(chan struct{})
	h1.RemoveStreamHandler(identify.ID)
	h1.SetStreamHandler(identify.ID, func(s network.Stream) {
		<-block
		go helpers.FullClose(s)
	})

	// from h2 connect to h1.
	if err := h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()}); err != nil {
		t.Fatal(err)
	}

	// from h2, identify h1.
	conn := h2.Network().ConnsToPeer(h1.ID())[0]
	go func() {
		ids2.IdentifyConn(conn)
		<-ids2.IdentifyWait(conn)
	}()

	<-time.After(500 * time.Millisecond)

	// subscribe to events in h1; after identify h1 should receive the delta from h2 and publish an event in the bus.
	sub, err := h1.EventBus().Subscribe(&event.EvtPeerProtocolsUpdated{}, eventbus.BufSize(16))
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	// add a handler in h2; the delta to h1 will queue until we're done identifying h1.
	h2.SetStreamHandler(protocol.TestingID, func(_ network.Stream) {})
	<-time.After(500 * time.Millisecond)

	// make sure we haven't received any events yet.
	if q := len(sub.Out()); q > 0 {
		t.Fatalf("expected no events yet; queued: %d", q)
	}

	close(block)
	select {
	case evt := <-sub.Out():
		e := evt.(event.EvtPeerProtocolsUpdated)
		if e.Peer != h2.ID() || len(e.Added) != 1 || e.Added[0] != protocol.TestingID {
			t.Fatalf("expected an event for protocol changes in h2, with the testing protocol added; instead got: %v", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out while waiting for an event for the protocol changes in h2")
	}
}
