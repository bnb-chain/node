package utils

import "time"

const SecondsPerDay int64 = 86400

func Now() time.Time {
	return time.Now().UTC()
}

// timestamp is from time.Unix()
func SameDayInUTC(first, second int64) bool {
	return first/SecondsPerDay == second/SecondsPerDay
}
