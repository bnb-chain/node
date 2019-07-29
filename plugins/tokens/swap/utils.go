package swap

import (
	"encoding/binary"

	"github.com/tendermint/tendermint/crypto/tmhash"
)

func CalculteRandomHash(randomNumber []byte, timestamp uint64) []byte {
	randomNumberAndTimestamp := make([]byte, RandomNumberLength + 8)
	copy(randomNumberAndTimestamp[:RandomNumberLength], randomNumber)
	binary.BigEndian.PutUint64(randomNumberAndTimestamp[RandomNumberLength:], timestamp)
	return tmhash.Sum(randomNumberAndTimestamp)
}
