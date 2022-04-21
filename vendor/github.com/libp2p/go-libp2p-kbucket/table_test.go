package kbucket

import (
	"math/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/test"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

// Test basic features of the bucket struct
func TestBucket(t *testing.T) {
	b := newBucket()

	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = test.RandPeerIDFatal(t)
		b.PushFront(peers[i])
	}

	local := test.RandPeerIDFatal(t)
	localID := ConvertPeerID(local)

	i := rand.Intn(len(peers))
	if !b.Has(peers[i]) {
		t.Errorf("Failed to find peer: %v", peers[i])
	}

	spl := b.Split(0, ConvertPeerID(local))
	llist := b.list
	for e := llist.Front(); e != nil; e = e.Next() {
		p := ConvertPeerID(e.Value.(peer.ID))
		cpl := CommonPrefixLen(p, localID)
		if cpl > 0 {
			t.Fatalf("Split failed. found id with cpl > 0 in 0 bucket")
		}
	}

	rlist := spl.list
	for e := rlist.Front(); e != nil; e = e.Next() {
		p := ConvertPeerID(e.Value.(peer.ID))
		cpl := CommonPrefixLen(p, localID)
		if cpl == 0 {
			t.Fatalf("Split failed. found id with cpl == 0 in non 0 bucket")
		}
	}
}

func TestTableCallbacks(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rt := NewRoutingTable(10, ConvertPeerID(local), time.Hour, m)

	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = test.RandPeerIDFatal(t)
	}

	pset := make(map[peer.ID]struct{})
	rt.PeerAdded = func(p peer.ID) {
		pset[p] = struct{}{}
	}
	rt.PeerRemoved = func(p peer.ID) {
		delete(pset, p)
	}

	rt.Update(peers[0])
	if _, ok := pset[peers[0]]; !ok {
		t.Fatal("should have this peer")
	}

	rt.Remove(peers[0])
	if _, ok := pset[peers[0]]; ok {
		t.Fatal("should not have this peer")
	}

	for _, p := range peers {
		rt.Update(p)
	}

	out := rt.ListPeers()
	for _, outp := range out {
		if _, ok := pset[outp]; !ok {
			t.Fatal("should have peer in the peerset")
		}
		delete(pset, outp)
	}

	if len(pset) > 0 {
		t.Fatal("have peers in peerset that were not in the table", len(pset))
	}
}

// Right now, this just makes sure that it doesnt hang or crash
func TestTableUpdate(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rt := NewRoutingTable(10, ConvertPeerID(local), time.Hour, m)

	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = test.RandPeerIDFatal(t)
	}

	// Testing Update
	for i := 0; i < 10000; i++ {
		rt.Update(peers[rand.Intn(len(peers))])
	}

	for i := 0; i < 100; i++ {
		id := ConvertPeerID(test.RandPeerIDFatal(t))
		ret := rt.NearestPeers(id, 5)
		if len(ret) == 0 {
			t.Fatal("Failed to find node near ID.")
		}
	}
}

func TestTableFind(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rt := NewRoutingTable(10, ConvertPeerID(local), time.Hour, m)

	peers := make([]peer.ID, 100)
	for i := 0; i < 5; i++ {
		peers[i] = test.RandPeerIDFatal(t)
		rt.Update(peers[i])
	}

	t.Logf("Searching for peer: '%s'", peers[2])
	found := rt.NearestPeer(ConvertPeerID(peers[2]))
	if !(found == peers[2]) {
		t.Fatalf("Failed to lookup known node...")
	}
}

func TestTableEldestPreferred(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rt := NewRoutingTable(10, ConvertPeerID(local), time.Hour, m)

	// generate size + 1 peers to saturate a bucket
	peers := make([]peer.ID, 15)
	for i := 0; i < 15; {
		if p := test.RandPeerIDFatal(t); CommonPrefixLen(ConvertPeerID(local), ConvertPeerID(p)) == 0 {
			peers[i] = p
			i++
		}
	}

	// test 10 first peers are accepted.
	for _, p := range peers[:10] {
		if _, err := rt.Update(p); err != nil {
			t.Errorf("expected all 10 peers to be accepted; instead got: %v", err)
		}
	}

	// test next 5 peers are rejected.
	for _, p := range peers[10:] {
		if _, err := rt.Update(p); err != ErrPeerRejectedNoCapacity {
			t.Errorf("expected extra 5 peers to be rejected; instead got: %v", err)
		}
	}
}

func TestTableFindMultiple(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rt := NewRoutingTable(20, ConvertPeerID(local), time.Hour, m)

	peers := make([]peer.ID, 100)
	for i := 0; i < 18; i++ {
		peers[i] = test.RandPeerIDFatal(t)
		rt.Update(peers[i])
	}

	t.Logf("Searching for peer: '%s'", peers[2])
	found := rt.NearestPeers(ConvertPeerID(peers[2]), 15)
	if len(found) != 15 {
		t.Fatalf("Got back different number of peers than we expected.")
	}
}

// Looks for race conditions in table operations. For a more 'certain'
// test, increase the loop counter from 1000 to a much higher number
// and set GOMAXPROCS above 1
func TestTableMultithreaded(t *testing.T) {
	local := peer.ID("localPeer")
	m := pstore.NewMetrics()
	tab := NewRoutingTable(20, ConvertPeerID(local), time.Hour, m)
	var peers []peer.ID
	for i := 0; i < 500; i++ {
		peers = append(peers, test.RandPeerIDFatal(t))
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Update(peers[n])
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Update(peers[n])
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Find(peers[n])
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	<-done
}

func BenchmarkUpdates(b *testing.B) {
	b.StopTimer()
	local := ConvertKey("localKey")
	m := pstore.NewMetrics()
	tab := NewRoutingTable(20, local, time.Hour, m)

	var peers []peer.ID
	for i := 0; i < b.N; i++ {
		peers = append(peers, test.RandPeerIDFatal(b))
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tab.Update(peers[i])
	}
}

func BenchmarkFinds(b *testing.B) {
	b.StopTimer()
	local := ConvertKey("localKey")
	m := pstore.NewMetrics()
	tab := NewRoutingTable(20, local, time.Hour, m)

	var peers []peer.ID
	for i := 0; i < b.N; i++ {
		peers = append(peers, test.RandPeerIDFatal(b))
		tab.Update(peers[i])
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tab.Find(peers[i])
	}
}
