package relay_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	. "github.com/libp2p/go-libp2p-circuit"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peerstore"

	swarm "github.com/libp2p/go-libp2p-swarm"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	ma "github.com/multiformats/go-multiaddr"
)

const TestProto = "test/relay-transport"

var msg = []byte("relay works!")

func testSetupRelay(t *testing.T, ctx context.Context) []host.Host {
	hosts := getNetHosts(t, ctx, 3)

	err := AddRelayTransport(ctx, hosts[0], swarmt.GenUpgrader(hosts[0].Network().(*swarm.Swarm)), OptDiscovery)
	if err != nil {
		t.Fatal(err)
	}

	err = AddRelayTransport(ctx, hosts[1], swarmt.GenUpgrader(hosts[1].Network().(*swarm.Swarm)), OptHop)
	if err != nil {
		t.Fatal(err)
	}

	err = AddRelayTransport(ctx, hosts[2], swarmt.GenUpgrader(hosts[2].Network().(*swarm.Swarm)))
	if err != nil {
		t.Fatal(err)
	}

	connect(t, hosts[0], hosts[1])
	connect(t, hosts[1], hosts[2])

	time.Sleep(100 * time.Millisecond)

	handler := func(s network.Stream) {
		_, err := s.Write(msg)
		if err != nil {
			t.Error(err)
		}
		s.Close()
	}

	hosts[2].SetStreamHandler(TestProto, handler)

	return hosts
}

func TestFullAddressTransportDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := testSetupRelay(t, ctx)

	addr, err := ma.NewMultiaddr(fmt.Sprintf("%s/ipfs/%s/p2p-circuit/ipfs/%s", hosts[1].Addrs()[0].String(), hosts[1].ID().Pretty(), hosts[2].ID().Pretty()))
	if err != nil {
		t.Fatal(err)
	}

	rctx, rcancel := context.WithTimeout(ctx, time.Second)
	defer rcancel()

	hosts[0].Peerstore().AddAddrs(hosts[2].ID(), []ma.Multiaddr{addr}, peerstore.TempAddrTTL)

	s, err := hosts[0].NewStream(rctx, hosts[2].ID(), TestProto)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, msg) {
		t.Fatal("message was incorrect:", string(data))
	}
}

func TestSpecificRelayTransportDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := testSetupRelay(t, ctx)

	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s/p2p-circuit/ipfs/%s", hosts[1].ID().Pretty(), hosts[2].ID().Pretty()))
	if err != nil {
		t.Fatal(err)
	}

	rctx, rcancel := context.WithTimeout(ctx, time.Second)
	defer rcancel()

	hosts[0].Peerstore().AddAddrs(hosts[2].ID(), []ma.Multiaddr{addr}, peerstore.TempAddrTTL)

	s, err := hosts[0].NewStream(rctx, hosts[2].ID(), TestProto)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, msg) {
		t.Fatal("message was incorrect:", string(data))
	}
}

func TestUnspecificRelayTransportDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := testSetupRelay(t, ctx)

	addr, err := ma.NewMultiaddr(fmt.Sprintf("/p2p-circuit/ipfs/%s", hosts[2].ID().Pretty()))
	if err != nil {
		t.Fatal(err)
	}

	rctx, rcancel := context.WithTimeout(ctx, time.Second)
	defer rcancel()

	hosts[0].Peerstore().AddAddrs(hosts[2].ID(), []ma.Multiaddr{addr}, peerstore.TempAddrTTL)

	s, err := hosts[0].NewStream(rctx, hosts[2].ID(), TestProto)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, msg) {
		t.Fatal("message was incorrect:", string(data))
	}
}
