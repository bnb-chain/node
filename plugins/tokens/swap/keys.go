package swap

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	Int64Size = 8
)

var (
	HashKey                      = []byte{0x01}
	SwapCreatorQueueKey          = []byte{0x02}
	SwapRecipientQueueKey        = []byte{0x03}
	SwapCloseTimeKey             = []byte{0x04}
	SwapIndexKey                 = []byte{0x05}
	LatestProcessedRefundSwapKey = []byte{0x06}
)

func BuildHashKey(randomNumberHash []byte) []byte {
	return append(HashKey, randomNumberHash...)
}

func BuildSwapCreatorKey(addr sdk.AccAddress, index int64) []byte {
	// prefix + addr + index
	key := make([]byte, 1+sdk.AddrLen+Int64Size)
	copy(key[:1], SwapCreatorQueueKey)
	copy(key[1:1+sdk.AddrLen], addr)
	binary.BigEndian.PutUint64(key[1+sdk.AddrLen:], uint64(index))
	return key
}

func BuildSwapCreatorQueueKey(addr sdk.AccAddress) []byte {
	key := make([]byte, 1+sdk.AddrLen)
	copy(key[:1], SwapCreatorQueueKey)
	copy(key[1:1+sdk.AddrLen], addr)
	return key
}

func BuildSwapRecipientKey(addr sdk.AccAddress, index int64) []byte {
	// prefix + addr + index
	key := make([]byte, 1+sdk.AddrLen+Int64Size)
	copy(key[:1], SwapRecipientQueueKey)
	copy(key[1:1+sdk.AddrLen], addr)
	binary.BigEndian.PutUint64(key[1+sdk.AddrLen:], uint64(index))
	return key
}

func BuildSwapRecipientQueueKey(addr sdk.AccAddress) []byte {
	key := make([]byte, 1+sdk.AddrLen)
	copy(key[:1], SwapRecipientQueueKey)
	copy(key[1:1+sdk.AddrLen], addr)
	return key
}

func BuildCloseTimeKey(unixTime int64, index int64) []byte {
	// prefix + unixTime + index
	key := make([]byte, 1+Int64Size+Int64Size)
	copy(key[:1], SwapCloseTimeKey)
	binary.BigEndian.PutUint64(key[1:1+Int64Size], uint64(unixTime))
	binary.BigEndian.PutUint64(key[1+Int64Size:], uint64(index))
	return key
}

func BuildCloseTimeQueueKey() []byte {
	return SwapCloseTimeKey
}
