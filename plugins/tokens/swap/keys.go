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
)

func GetSwapHashKey(randomNumberHash []byte) []byte {
	return append(SwapHashKey, randomNumberHash...)
}

func GetSwapFromKey(addr sdk.AccAddress, randomNumberHash []byte) []byte {
	key := make([]byte, 1+sdk.AddrLen+RandomNumberHashLength)
	copy(key[:1], SwapFromQueueKey)
	copy(key[1:sdk.AddrLen+1], addr)
	copy(key[sdk.AddrLen+1:], randomNumberHash)
	return key
}

func GetSwapFromQueueKey(addr sdk.AccAddress) []byte {
	key := make([]byte, 1+sdk.AddrLen)
	copy(key[:1], SwapFromQueueKey)
	copy(key[1:sdk.AddrLen+1], addr)
	return key
}

func GetSwapToKey(addr sdk.AccAddress, randomNumberHash []byte) []byte {
	key := make([]byte, 1+sdk.AddrLen+RandomNumberHashLength)
	copy(key[:1], SwapToQueueKey)
	copy(key[1:sdk.AddrLen+1], addr)
	copy(key[sdk.AddrLen+1:], randomNumberHash)
	return key
}

func GetSwapToQueueKey(addr sdk.AccAddress) []byte {
	key := make([]byte, 1+sdk.AddrLen)
	copy(key[:1], SwapToQueueKey)
	copy(key[1:sdk.AddrLen+1], addr)
	return key
}

func GetTimeKey(unixTime int64, randomNumberHash []byte) []byte {
	key := make([]byte, 1+8+RandomNumberHashLength)
	copy(key[:1], SwapTimeKey)
	binary.BigEndian.PutUint64(key[1:1+8], uint64(unixTime))
	copy(key[1+8:], randomNumberHash)
	return key
}

func GetTimeQueueKey() []byte {
	return SwapTimeKey
}
