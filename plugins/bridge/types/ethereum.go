package types

import (
	"fmt"
	"math/big"
	"reflect"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// EthereumAddress defines a standard ethereum address
type EthereumAddress gethCommon.Address

// NewEthereumAddress is a constructor function for EthereumAddress
func NewEthereumAddress(address string) EthereumAddress {
	return EthereumAddress(gethCommon.HexToAddress(address))
}

func (ethAddr EthereumAddress) IsEmpty() bool {
	addrValue := big.NewInt(0)
	addrValue.SetBytes(ethAddr[:])

	return addrValue.Cmp(big.NewInt(0)) == 0
}

// Route should return the name of the module
func (ethAddr EthereumAddress) String() string {
	return gethCommon.Address(ethAddr).String()
}

// MarshalJSON marshals the ethereum address to JSON
func (ethAddr EthereumAddress) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%v\"", ethAddr.String())), nil
}

// UnmarshalJSON unmarshals an ethereum address
func (ethAddr *EthereumAddress) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(reflect.TypeOf(gethCommon.Address{}), input, ethAddr[:])
}
