package swap

import (
	"encoding/binary"

	"github.com/tendermint/tendermint/crypto/tmhash"
)

func CalculateRandomHash(randomNumber []byte, timestamp int64) []byte {
	randomNumberAndTimestamp := make([]byte, RandomNumberLength+8)
	copy(randomNumberAndTimestamp[:RandomNumberLength], randomNumber)
	binary.BigEndian.PutUint64(randomNumberAndTimestamp[RandomNumberLength:], uint64(timestamp))
	return tmhash.Sum(randomNumberAndTimestamp)
}
