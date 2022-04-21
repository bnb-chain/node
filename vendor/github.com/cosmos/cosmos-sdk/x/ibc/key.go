package ibc

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	prefixLength          = 1
	srcChainIdLength      = 2
	destChainIDLength     = 2
	channelIDLength       = 1
	sequenceLength        = 8
	totalPackageKeyLength = prefixLength + srcChainIdLength + destChainIDLength + channelIDLength + sequenceLength
)

var (
	PrefixForIbcPackageKey = []byte{0x00}
	PrefixForSequenceKey   = []byte{0x01}
)

func buildIBCPackageKey(srcChainID, destChainID sdk.ChainID, channelID sdk.ChannelID, sequence uint64) []byte {
	key := make([]byte, totalPackageKeyLength)

	copy(key[:prefixLength], PrefixForIbcPackageKey)
	binary.BigEndian.PutUint16(key[prefixLength:srcChainIdLength+prefixLength], uint16(srcChainID))
	binary.BigEndian.PutUint16(key[prefixLength+srcChainIdLength:prefixLength+srcChainIdLength+destChainIDLength], uint16(destChainID))
	copy(key[prefixLength+srcChainIdLength+destChainIDLength:], []byte{byte(channelID)})
	binary.BigEndian.PutUint64(key[prefixLength+srcChainIdLength+destChainIDLength+channelIDLength:], sequence)

	return key
}

func buildIBCPackageKeyPrefix(srcChainID, destChainID sdk.ChainID, channelID sdk.ChannelID) []byte {
	key := make([]byte, totalPackageKeyLength-sequenceLength)

	copy(key[:prefixLength], PrefixForIbcPackageKey)
	binary.BigEndian.PutUint16(key[prefixLength:prefixLength+srcChainIdLength], uint16(srcChainID))
	binary.BigEndian.PutUint16(key[prefixLength+srcChainIdLength:prefixLength+srcChainIdLength+destChainIDLength], uint16(destChainID))
	copy(key[prefixLength+srcChainIdLength+destChainIDLength:], []byte{byte(channelID)})

	return key
}