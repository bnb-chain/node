package utils

const SecondsPerDay int64 = 86400

// timestamp is from time.Unix()
func SameDayInUTC(first, second int64) bool {
	return first/SecondsPerDay == second/SecondsPerDay
}
