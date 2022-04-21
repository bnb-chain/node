package cmd

import (
	"testing"

	"github.com/binance-chain/tss/common"
)

func TestTimestampHexConvertion(t *testing.T) {
	expected := 1564372501
	encoded := common.ConvertTimestampToHex(int64(expected))
	if encoded != "5D3E6E15" {
		t.Fail()
	}
	timestamp := common.ConvertHexToTimestamp(encoded)
	if timestamp != expected {
		t.Fail()
	}
}

func TestMultiAddrStrToNormalAddr(t *testing.T) {
	res, err := convertMultiAddrStrToNormalAddr("/ip4/127.0.0.1/tcp/27148")
	if err != nil {
		t.Fail()
	}
	if res != "127.0.0.1:27148" {
		t.Fail()
	}
}

func TestReplaceIpInMultiAddr(t *testing.T) {
	res := replaceIpInAddr("/ip4/0.0.0.0/tcp/27148", "127.0.0.1")
	if res != "/ip4/127.0.0.1/tcp/27148" {
		t.Fail()
	}
}
