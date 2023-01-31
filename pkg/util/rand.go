package util

import "math/rand"

func RandBetween(min, max int) int {
	return min + rand.Intn(max-min)
}

func RandBetween64(min, max int64) int64 {
	if min == max {
		return 0
	}
	return min + rand.Int63n(max-min)
}
