package union

import (
	"image"

	"gioui.org/f32"
	"github.com/failosof/chessboard/util"
)

type Size struct {
	F32   f32.Point
	Pt    image.Point
	Float float32
	Int   int
}

func SizeFromInt(val int) Size {
	float := float32(val)
	return Size{
		F32:   f32.Pt(float, float),
		Pt:    image.Pt(val, val),
		Float: float,
		Int:   val,
	}
}

func SizeFromFloat(val float32) Size {
	round := util.Round(val)
	return Size{
		F32:   f32.Pt(val, val),
		Pt:    image.Pt(round, round),
		Float: val,
		Int:   round,
	}
}

func SizeFromMinPt(pt image.Point) Size {
	return SizeFromInt(util.Min(pt.X, pt.Y))
}

func SizeFromMinF32(pt f32.Point) Size {
	return SizeFromFloat(util.Min(pt.X, pt.Y))
}

func (s *Size) Scale(factor float32) {
	s.Float *= factor
	s.Int = util.Round(s.Float)
	s.F32.X, s.F32.Y = s.Float, s.Float
	s.Pt = s.F32.Round()
}
