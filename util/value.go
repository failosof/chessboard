package util

import (
	"image"
	"math"

	"gioui.org/f32"
	"golang.org/x/exp/constraints"
)

func Round(val float32) int {
	return int(math.Round(float64(val)))
}

func Floor(val float32) int {
	return int(math.Floor(float64(val)))
}

func Min[T constraints.Ordered](a, b T) T {
	if a <= b {
		return a
	} else {
		return b
	}
}

func ToF32(pt image.Point) f32.Point {
	return f32.Point{X: float32(pt.X), Y: float32(pt.Y)}
}
