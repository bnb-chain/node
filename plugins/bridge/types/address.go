package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SmartChainAddress defines a standard smart chain address
type SmartChainAddress [20]byte

// NewSmartChainAddress is a constructor function for SmartChainAddress
func NewSmartChainAddress(addr string) SmartChainAddress {
	// we don't want to return error here, ethereum also do the same thing here
	hexBytes, _ := sdk.HexDecode(addr)
	var address SmartChainAddress
	address.SetBytes(hexBytes)
	return address
}

func (addr *SmartChainAddress) SetBytes(b []byte) {
	if len(b) > len(addr) {
		b = b[len(b)-20:]
	}
	copy(addr[20-len(b):], b)
}

func (addr SmartChainAddress) IsEmpty() bool {
	addrValue := big.NewInt(0)
	addrValue.SetBytes(addr[:])

	return addrValue.Cmp(big.NewInt(0)) == 0
}

// Route should return the name of the module
func (addr SmartChainAddress) String() string {
	return sdk.HexAddress(addr[:])
}

// MarshalJSON marshals the smart chain address to JSON
func (addr SmartChainAddress) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%v\"", addr.String())), nil
}

// UnmarshalJSON unmarshals an smart chain address
func (addr *SmartChainAddress) UnmarshalJSON(input []byte) error {
	hexBytes, err := sdk.HexDecode(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	addr.SetBytes(hexBytes)
	return nil
}
