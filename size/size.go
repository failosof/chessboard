package size

import (
	"image"

	"gioui.org/f32"
	"github.com/failosof/chessboard/util"
)

type Union struct {
	F32Pt f32.Point
	Pt    image.Point
	Float float32
	Int   int
}

func FromInt(val int) Union {
	float := float32(val)
	return Union{
		F32Pt: f32.Pt(float, float),
		Pt:    image.Pt(val, val),
		Float: float,
		Int:   val,
	}
}

func FromFloat(val float32) Union {
	round := util.Round(val)
	return Union{
		F32Pt: f32.Pt(val, val),
		Pt:    image.Pt(round, round),
		Float: val,
		Int:   round,
	}
}

func FromMinPt(pt image.Point) Union {
	return FromInt(util.Min(pt.X, pt.Y))
}

func FromMinF32Pt(pt f32.Point) Union {
	return FromFloat(util.Min(pt.X, pt.Y))
}
