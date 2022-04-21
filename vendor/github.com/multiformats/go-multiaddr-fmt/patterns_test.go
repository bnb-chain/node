package mafmt

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
)

type testVector struct {
	Pattern Pattern
	Good    []string
	Bad     []string
}

var TestVectors = map[string]*testVector{
	"IP": {
		Pattern: IP,
		Good:    []string{"/ip4/0.0.0.0", "/ip6/fc00::"},
		Bad:     []string{"/ip4/0.0.0.0/tcp/555", "/udp/789/ip6/fc00::"},
	},
	"TCP": {
		Pattern: TCP,
		Good:    []string{"/ip4/0.0.7.6/tcp/1234", "/ip6/::/tcp/0"},
		Bad:     []string{"/tcp/12345", "/ip6/fc00::/udp/5523/tcp/9543"},
	},
	"UDP": {
		Pattern: UDP,
		Good:    []string{"/ip4/0.0.7.6/udp/1234", "/ip6/::/udp/0"},
		Bad:     []string{"/udp/12345", "/ip6/fc00::/tcp/5523/udp/9543"},
	},
	"UTP": {
		Pattern: UTP,
		Good:    []string{"/ip4/1.2.3.4/udp/3456/utp", "/ip6/::/udp/0/utp"},
		Bad:     []string{"/ip4/0.0.0.0/tcp/12345/utp", "/ip6/1.2.3.4/ip4/0.0.0.0/udp/1234/utp", "/utp"},
	},
	"QUIC": {
		Pattern: QUIC,
		Good:    []string{"/ip4/1.2.3.4/udp/1234/quic", "/ip6/::/udp/1234/quic"},
		Bad:     []string{"/ip4/0.0.0.0/tcp/12345/quic", "/ip6/1.2.3.4/ip4/0.0.0.0/udp/1234/quic", "/quic"},
	},
	"IPFS": {
		Pattern: IPFS,
		Good: []string{
			"/ip4/1.2.3.4/tcp/1234/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip6/::/tcp/1234/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip6/::/udp/1234/utp/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip4/0.0.0.0/udp/1234/utp/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		},
		Bad: []string{
			"/ip4/1.2.3.4/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip6/::/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/tcp/123/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip6/::/udp/1234/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip6/::/utp/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		},
	},
	"DNS": {
		Pattern: DNS,
		Good:    []string{"/dnsaddr/example.io", "/dns4/example.io", "/dns6/example.io"},
		Bad:     []string{"/ip4/127.0.0.1"},
	},
	"WebRTCDirect": {
		Pattern: WebRTCDirect,
		Good:    []string{"/ip4/1.2.3.4/tcp/3456/http/p2p-webrtc-direct", "/ip6/::/tcp/0/http/p2p-webrtc-direct"},
		Bad:     []string{"/ip4/0.0.0.0", "/ip6/fc00::", "/udp/12345", "/ip6/fc00::/tcp/5523/udp/9543"},
	},
	"HTTP": {
		Pattern: HTTP,
		Good:    []string{"/ip4/1.2.3.4/http", "/dns4/example.io/http", "/dns6/::/tcp/7011/http", "/dnsaddr/example.io/http", "/ip6/fc00::/http"},
		Bad:     []string{"/ip4/1.2.3.4/https", "/ip4/0.0.0.0/tcp/12345/quic", "/ip6/fc00::/tcp/5523"},
	},
	"HTTPS": {
		Pattern: HTTPS,
		Good:    []string{"/ip4/1.2.3.4/https", "/dns4/example.io/https", "/dns6/::/tcp/7011/https", "/ip6/fc00::/https"},
		Bad:     []string{"/ip4/1.2.3.4/http", "/ip4/0.0.0.0/tcp/12345/quic", "/ip6/fc00::/tcp/5523"},
	},
}

func TestProtocolMatching(t *testing.T) {
	for name, tc := range TestVectors {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assertMatches(t, tc.Pattern, tc.Good)

			bad := [][]string{tc.Bad}
			for _, other := range TestVectors {
				if other == tc {
					continue
				}
				bad = append(bad, other.Good)
			}
			assertMismatches(t, tc.Pattern, bad...)
		})
	}
}

func TestReliableGroup(t *testing.T) {
	assertMatches(t, Reliable, TestVectors["UTP"].Good, TestVectors["TCP"].Good, TestVectors["QUIC"].Good)
	assertMismatches(t, Reliable, TestVectors["IP"].Good, TestVectors["UDP"].Good, TestVectors["IPFS"].Good)
}

func TestUnreliableGroup(t *testing.T) {
	assertMatches(t, Unreliable, TestVectors["UDP"].Good)
	assertMismatches(t, Unreliable, TestVectors["IP"].Good, TestVectors["TCP"].Good, TestVectors["UTP"].Good, TestVectors["IPFS"].Good, TestVectors["QUIC"].Good)
}

func assertMatches(t *testing.T, p Pattern, args ...[]string) {
	t.Helper()

	t.Logf("testing assertions for %q", p)
	for _, argset := range args {
		for _, s := range argset {
			addr, err := ma.NewMultiaddr(s)
			if err != nil {
				t.Fatal(err)
			}

			if !p.Matches(addr) {
				t.Fatal("mismatch!", s, p)
			}
		}
	}
}

func assertMismatches(t *testing.T, p Pattern, args ...[]string) {
	t.Helper()

	for _, argset := range args {
		for _, s := range argset {
			addr, err := ma.NewMultiaddr(s)
			if err != nil {
				t.Fatal(err)
			}

			if p.Matches(addr) {
				t.Fatal("incorrect match!", s, p)
			}
		}
	}
}
