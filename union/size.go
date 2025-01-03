package union

import (
	"image"
	"math"

	"gioui.org/f32"
	"github.com/failosof/chessboard/util"
)

type Size struct {
	F32   f32.Point
	Pt    image.Point
	Float float32
	Int   int
	Half  *Size // one level
}

func SizeFromInt(val int) Size {
	float := float32(val)
	half := float / 2
	intHalf := util.Round(half)
	return Size{
		F32:   f32.Pt(float, float),
		Pt:    image.Pt(val, val),
		Float: float,
		Int:   val,
		Half: &Size{
			F32:   f32.Pt(half, half),
			Pt:    image.Pt(intHalf, intHalf),
			Float: half,
			Int:   intHalf,
		},
	}
}

func SizeFromFloat(val float32) Size {
	round := util.Round(val)
	half := val / 2
	intHalf := util.Round(half)
	return Size{
		F32:   f32.Pt(val, val),
		Pt:    image.Pt(round, round),
		Float: val,
		Int:   round,
		Half: &Size{
			F32:   f32.Pt(half, half),
			Pt:    image.Pt(intHalf, intHalf),
			Float: half,
			Int:   intHalf,
		},
	}
}

func SizeFromMinPt(pt image.Point) Size {
	return SizeFromInt(util.Min(pt.X, pt.Y))
}

func SizeFromMinF32(pt f32.Point) Size {
	return SizeFromFloat(util.Min(pt.X, pt.Y))
}

func (s *Size) Scale(factor float32) {
	f := float64(factor)
	if !math.IsNaN(f) && !math.IsInf(f, 0) {
		s.Float *= factor
		s.Int = util.Round(s.Float)
		s.F32.X, s.F32.Y = s.Float, s.Float
		s.Pt = s.F32.Round()
		if s.Half != nil {
			s.Half.Scale(factor)
		}
	}
}

func (s Size) Eq(other Size) bool {
	s.Half = nil
	other.Half = nil
	return s == other
}

func (s Size) IsZero() bool {
	return s.Eq(Size{})
}

func (s Size) String() string {
	return s.F32.String()
}
