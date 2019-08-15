package swap

import (
	"encoding/binary"

	"github.com/tendermint/tendermint/crypto/tmhash"
)

const (
	OneHour  = 3600
	TwoHours = 7200
	OneWeek  = 86400*7
)

func CalculateRandomHash(randomNumber []byte, timestamp int64) []byte {
	randomNumberAndTimestamp := make([]byte, RandomNumberLength+8)
	copy(randomNumberAndTimestamp[:RandomNumberLength], randomNumber)
	binary.BigEndian.PutUint64(randomNumberAndTimestamp[RandomNumberLength:], uint64(timestamp))
	return tmhash.Sum(randomNumberAndTimestamp)
}
