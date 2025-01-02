package union

import (
	"image"

	"gioui.org/f32"
	"github.com/failosof/chessboard/util"
)

type Point struct {
	Pt  image.Point
	F32 f32.Point
}

func PointFromInt(x, y int) Point {
	return PointFromFloat(float32(x), float32(y))
}

func PointFromFloat(x, y float32) Point {
	return Point{
		Pt:  image.Pt(util.Round(x), util.Round(y)),
		F32: f32.Pt(x, y),
	}
}

func PointFromF32(pt f32.Point) Point {
	return PointFromFloat(pt.X, pt.Y)
}

func (p *Point) Scale(factor float32) {
	p.F32 = p.F32.Mul(factor)
	p.Pt = p.F32.Round()
}

func (p Point) String() string {
	return p.F32.String()
}
