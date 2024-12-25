package util

import "math"

func Round(val float32) int {
	return int(math.Round(float64(val)))
}

func Floor(val float32) int {
	return int(math.Floor(float64(val)))
}

func Min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}
