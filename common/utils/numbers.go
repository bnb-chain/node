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

func Mul64(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	if (c < 0) == ((a < 0) != (b < 0)) {
		if c/b == a {
			return c, true
		}
	}
	return c, false
}
