package union

import (
	"image"

	"gioui.org/f32"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type Point struct {
	Pt     image.Point
	F32    f32.Point
	Square chess.Square
}

func PointFromIntPx(x, y int) Point {
	return PointFromFloatPx(float32(x), float32(y))
}

func PointFromFloatPx(x, y float32) Point {
	return Point{
		Pt:  image.Pt(util.Round(x), util.Round(y)),
		F32: f32.Pt(x, y),
	}
}

func PointFromF32(pt f32.Point) Point {
	return PointFromFloatPx(pt.X, pt.Y)
}

func (p *Point) Scale(factor float32) {
	p.F32 = p.F32.Mul(factor)
	p.Pt = p.F32.Round()
}
