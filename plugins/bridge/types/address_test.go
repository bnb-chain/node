package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddress(t *testing.T) {
	addrStr := "0x43121d597656E398473b992f0dF667fF0fc0791C"
	address := NewSmartChainAddress(addrStr)
	convertedAddrStr := address.String()
	require.Equal(t, addrStr, convertedAddrStr, "address should be equal")
}
