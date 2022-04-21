package sidechain

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	prefixLength      = 1
	destChainIDLength = 2
	channelIDLength   = 1
	sequenceLength    = 8
)

var (
	SideChainStorePrefixByIdKey = []byte{0x01} // prefix for each key to a side chain store prefix, by side chain id

	PrefixForSendSequenceKey    = []byte{0xf0}
	PrefixForReceiveSequenceKey = []byte{0xf1}

	PrefixForChannelPermissionKey = []byte{0xc0}
)

func GetSideChainStorePrefixKey(sideChainId string) []byte {
	return append(SideChainStorePrefixByIdKey, []byte(sideChainId)...)
}

func buildChannelSequenceKey(destChainID sdk.ChainID, channelID sdk.ChannelID, prefix []byte) []byte {
	key := make([]byte, prefixLength+destChainIDLength+channelIDLength)

	copy(key[:prefixLength], prefix)
	binary.BigEndian.PutUint16(key[prefixLength:prefixLength+destChainIDLength], uint16(destChainID))
	copy(key[prefixLength+destChainIDLength:], []byte{byte(channelID)})
	return key
}

func buildChannelPermissionKey(destChainID sdk.ChainID, channelID sdk.ChannelID) []byte {
	key := make([]byte, prefixLength+destChainIDLength+channelIDLength)

	copy(key[:prefixLength], PrefixForChannelPermissionKey)
	binary.BigEndian.PutUint16(key[prefixLength:prefixLength+destChainIDLength], uint16(destChainID))
	copy(key[prefixLength+destChainIDLength:], []byte{byte(channelID)})
	return key
}

func buildChannelPermissionsPrefixKey(destChainID sdk.ChainID) []byte {
	key := make([]byte, prefixLength+destChainIDLength)

	copy(key[:prefixLength], PrefixForChannelPermissionKey)
	binary.BigEndian.PutUint16(key[prefixLength:prefixLength+destChainIDLength], uint16(destChainID))
	return key
}
