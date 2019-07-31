package swap

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	SwapHashKey      = []byte{0x01}
	SwapFromQueueKey = []byte{0x02}
	SwapToQueueKey   = []byte{0x03}
	SwapTimeKey      = []byte{0x04}
	SwapIndexKey     = []byte{0x05}
)

func GetSwapHashKey(randomNumberHash []byte) []byte {
	return append(SwapHashKey, randomNumberHash...)
}

func GetSwapFromKey(addr sdk.AccAddress, index int64) []byte {
	// prefix + addr + index
	key := make([]byte, 1+sdk.AddrLen+8)
	copy(key[:1], SwapFromQueueKey)
	copy(key[1:1+sdk.AddrLen], addr)
	binary.BigEndian.PutUint64(key[1+sdk.AddrLen:], uint64(index))
	return key
}

func GetSwapFromQueueKey(addr sdk.AccAddress) []byte {
	key := make([]byte, 1+sdk.AddrLen)
	copy(key[:1], SwapFromQueueKey)
	copy(key[1:1+sdk.AddrLen], addr)
	return key
}

func GetSwapToKey(addr sdk.AccAddress, index int64) []byte {
	// prefix + addr + index
	key := make([]byte, 1+sdk.AddrLen+8)
	copy(key[:1], SwapToQueueKey)
	copy(key[1:1+sdk.AddrLen], addr)
	binary.BigEndian.PutUint64(key[1+sdk.AddrLen:], uint64(index))
	return key
}

func GetSwapToQueueKey(addr sdk.AccAddress) []byte {
	key := make([]byte, 1+sdk.AddrLen)
	copy(key[:1], SwapToQueueKey)
	copy(key[1:1+sdk.AddrLen], addr)
	return key
}

func GetTimeKey(unixTime int64, index int64) []byte {
	// prefix + unixTime + index
	key := make([]byte, 1+8+8)
	copy(key[:1], SwapTimeKey)
	binary.BigEndian.PutUint64(key[1:1+8], uint64(unixTime))
	binary.BigEndian.PutUint64(key[1+8:], uint64(index))
	return key
}

func GetTimeQueueKey() []byte {
	return SwapTimeKey
}
