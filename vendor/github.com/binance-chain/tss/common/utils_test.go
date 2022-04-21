package common

import (
	"testing"
)

func TestReplaceIpInAddr(t *testing.T) {
	result := ReplaceIpInAddr("/ip4/192.168.2.35/tcp/56707", "192.168.2.35")
	if result != "/ip4/192.168.2.35/tcp/56707" {
		common.Panic("ip replacement failed")
	}

	result = ReplaceIpInAddr("/ip4/0.0.0.0/tcp/56707", "192.168.2.35")
	if result != "/ip4/192.168.2.35/tcp/56707" {
		common.Panic("ip replacement failed")
	}

	result = ReplaceIpInAddr("/ip4/127.0.0.1/tcp/56707", "127.0.0.1")
	if result != "/ip4/127.0.0.1/tcp/56707" {
		common.Panic("ip replacement failed")
	}
}
