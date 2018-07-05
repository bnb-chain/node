package utils

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
