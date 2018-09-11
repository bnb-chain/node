package utils

import "time"

const SecondsPerDay int64 = 86400

// timestamp is from time.Unix()
func SameDayInUTC(first, second time.Time) bool {
	return first.Unix()/SecondsPerDay == second.Unix()/SecondsPerDay
}
