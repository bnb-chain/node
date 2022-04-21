package relay

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	_ "github.com/multiformats/go-multiaddr-dns"
)

func TestCleanupAddrs(t *testing.T) {
	// test with no addrsplosion
	addrs := makeAddrList(
		"/ip4/127.0.0.1/tcp/4001",
		"/ip4/127.0.0.1/udp/4002/quic",
		"/ip4/1.2.3.4/tcp/4001",
		"/ip4/1.2.3.4/udp/4002/quic",
		"/dnsaddr/somedomain.com/tcp/4002/ws",
	)
	clean := makeAddrList(
		"/ip4/1.2.3.4/tcp/4001",
		"/ip4/1.2.3.4/udp/4002/quic",
		"/dnsaddr/somedomain.com/tcp/4002/ws",
	)

	r := cleanupAddressSet(addrs)
	if !sameAddrs(clean, r) {
		t.Fatal("cleaned up set doesn't match expected")
	}

	// test with default port addrspolosion
	addrs = makeAddrList(
		"/ip4/127.0.0.1/tcp/4001",
		"/ip4/1.2.3.4/tcp/4001",
		"/ip4/1.2.3.4/tcp/33333",
		"/ip4/1.2.3.4/tcp/33334",
		"/ip4/1.2.3.4/tcp/33335",
		"/ip4/1.2.3.4/udp/4002/quic",
	)
	clean = makeAddrList(
		"/ip4/1.2.3.4/tcp/4001",
		"/ip4/1.2.3.4/udp/4002/quic",
	)
	r = cleanupAddressSet(addrs)
	if !sameAddrs(clean, r) {
		t.Fatal("cleaned up set doesn't match expected")
	}

	// test with default port addrsplosion but no private addrs
	addrs = makeAddrList(
		"/ip4/1.2.3.4/tcp/4001",
		"/ip4/1.2.3.4/tcp/33333",
		"/ip4/1.2.3.4/tcp/33334",
		"/ip4/1.2.3.4/tcp/33335",
		"/ip4/1.2.3.4/udp/4002/quic",
	)
	clean = makeAddrList(
		"/ip4/1.2.3.4/tcp/4001",
		"/ip4/1.2.3.4/udp/4002/quic",
	)
	r = cleanupAddressSet(addrs)
	if !sameAddrs(clean, r) {
		t.Fatal("cleaned up set doesn't match expected")
	}

	// test with non-standard port addrsplosion
	addrs = makeAddrList(
		"/ip4/127.0.0.1/tcp/12345",
		"/ip4/1.2.3.4/tcp/12345",
		"/ip4/1.2.3.4/tcp/33333",
		"/ip4/1.2.3.4/tcp/33334",
		"/ip4/1.2.3.4/tcp/33335",
	)
	clean = makeAddrList(
		"/ip4/1.2.3.4/tcp/12345",
	)
	r = cleanupAddressSet(addrs)
	if !sameAddrs(clean, r) {
		t.Fatal("cleaned up set doesn't match expected")
	}

	// test with a squeaky clean address set
	addrs = makeAddrList(
		"/ip4/1.2.3.4/tcp/4001",
		"/ip4/1.2.3.4/udp/4001/quic",
	)
	clean = addrs
	r = cleanupAddressSet(addrs)
	if !sameAddrs(clean, r) {
		t.Fatal("cleaned up set doesn't match expected")
	}
}

func makeAddrList(strs ...string) []ma.Multiaddr {
	result := make([]ma.Multiaddr, 0, len(strs))
	for _, s := range strs {
		a := ma.StringCast(s)
		result = append(result, a)
	}
	return result
}

func sameAddrs(as, bs []ma.Multiaddr) bool {
	if len(as) != len(bs) {
		return false
	}

	for _, a := range as {
		if !findAddr(a, bs) {
			return false
		}
	}
	return true
}

func findAddr(a ma.Multiaddr, bs []ma.Multiaddr) bool {
	for _, b := range bs {
		if a.Equal(b) {
			return true
		}
	}
	return false
}
