package utils

func Abs(number int64) int64 {
	if number < 0 {
		return -number
	}
	return number
}
