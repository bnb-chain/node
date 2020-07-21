package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddress(t *testing.T) {
	addrStr := "0x43121d597656E398473b992f0dF667fF0fc0791C"
	address, err := NewSmartChainAddress(addrStr)
	require.Nil(t, err, "err should be nil")
	convertedAddrStr := address.String()
	require.Equal(t, addrStr, convertedAddrStr, "address should be equal")

	addrStr = "0x43121d597656E398473b992f0dF667fF0fc0791C1"
	_, err = NewSmartChainAddress(addrStr)
	require.NotNil(t, err, "err should not be nil")

	addrStr = "0x43121d597656E398473b992f0dF667fF0fc0791"
	_, err = NewSmartChainAddress(addrStr)
	require.NotNil(t, err, "err should not be nil")
}
