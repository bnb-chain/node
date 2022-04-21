package ping_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ping "github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

func TestPing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h1 := bhost.New(swarmt.GenSwarm(t, ctx))
	h2 := bhost.New(swarmt.GenSwarm(t, ctx))

	err := h1.Connect(ctx, peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	})

	if err != nil {
		t.Fatal(err)
	}

	ps1 := ping.NewPingService(h1)
	ps2 := ping.NewPingService(h2)

	testPing(t, ps1, h2.ID())
	testPing(t, ps2, h1.ID())
}

func testPing(t *testing.T, ps *ping.PingService, p peer.ID) {
	pctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ts := ps.Ping(pctx, p)

	for i := 0; i < 5; i++ {
		select {
		case res := <-ts:
			if res.Error != nil {
				t.Fatal(res.Error)
			}
			t.Log("ping took: ", res.RTT)
		case <-time.After(time.Second * 4):
			t.Fatal("failed to receive ping")
		}
	}

}
