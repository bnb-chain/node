package utils

import "time"

const SecondsPerDay int64 = 86400

func Now() time.Time {
	return time.Now().UTC()
}

// timestamp is from time.Unix()
func SameDayInUTC(first, second time.Time) bool {
	return first.Unix()/SecondsPerDay == second.Unix()/SecondsPerDay
}

func SamePeriodInUTC(first, second time.Time, period int64) bool {
	return first.Unix()/period == second.Unix()/period
}
