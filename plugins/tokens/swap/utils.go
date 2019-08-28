package swap

import (
	"encoding/binary"

	"github.com/tendermint/tendermint/crypto/tmhash"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	FifteenMinutes = 900
	ThirtyMinutes  = 1800
	OneWeek        = 86400 * 7
)

func CalculateRandomHash(randomNumber []byte, timestamp int64) []byte {
	data := make([]byte, RandomNumberLength+8)
	copy(data[:RandomNumberLength], randomNumber)
	binary.BigEndian.PutUint64(data[RandomNumberLength:], uint64(timestamp))
	return tmhash.Sum(data)
}

func CalculateSwapID(randomNumberHash []byte, sender sdk.AccAddress, senderOtherChain HexData) []byte {
	data := randomNumberHash
	data = append(data, []byte(sender)...)
	data = append(data, []byte(senderOtherChain)...)
	return tmhash.Sum(data)
}
