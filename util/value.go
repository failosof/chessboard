package util

import (
	"image"
	"math"

	"gioui.org/f32"
	"golang.org/x/exp/constraints"
)

type Float interface {
	float32 | float64
}

func Round[F Float](val F) int {
	return int(math.Round(float64(val)))
}

func Floor[F Float](val F) int {
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
