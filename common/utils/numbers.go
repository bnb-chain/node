package utils

import (
	"encoding/binary"
)

func AbsInt(a int64) int64 {
	y := a >> 63
	return (a ^ y) - y
}

func MinInt(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func Int642Bytes(n int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(n))
	return b
}
