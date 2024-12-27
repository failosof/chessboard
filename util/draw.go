package util

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

func DrawImage(ops *op.Ops, img image.Image, at image.Point, factor f32.Point) {
	imageOp := paint.NewImageOp(img)
	offset := op.Offset(at).Push(ops)
	imageOp.Filter = paint.FilterLinear
	imageOp.Add(ops)
	op.Affine(f32.Affine2D{}.Scale(f32.Point{}, factor)).Add(ops)
	paint.PaintOp{}.Add(ops)
	offset.Pop()
}

func DrawPane(ops *op.Ops, rect image.Rectangle, color color.NRGBA) {
	defer clip.Rect(rect).Push(ops).Pop()
	paint.ColorOp{Color: color}.Add(ops)
	paint.PaintOp{}.Add(ops)
}

func DrawRectangle(ops *op.Ops, rect image.Rectangle, width float32, color color.NRGBA) {
	r := Round(width)
	rrect := clip.RRect{Rect: rect, SE: r, SW: r, NW: r, NE: r}
	defer clip.Rect(rect).Push(ops).Pop()
	paint.FillShape(ops, color, clip.Stroke{
		Path:  rrect.Path(ops),
		Width: width,
	}.Op())
}

func DrawEllipse(ops *op.Ops, rect image.Rectangle, color color.NRGBA) {
	defer clip.Ellipse(rect).Push(ops).Pop()
	paint.ColorOp{Color: color}.Add(ops)
	paint.PaintOp{}.Add(ops)
}

func DrawCircle(ops *op.Ops, rect image.Rectangle, width float32, color color.NRGBA) {
	circle := clip.Ellipse(rect)
	defer circle.Push(ops).Pop()
	paint.FillShape(ops, color, clip.Stroke{
		Path:  circle.Path(ops),
		Width: width,
	}.Op())
}

func DrawCross(ops *op.Ops, rect image.Rectangle, width float32, color color.NRGBA) {
	halfWidth := Round(width / 2)

	diag1 := clip.Rect{
		Min: image.Point{X: rect.Min.X, Y: rect.Min.Y + halfWidth},
		Max: image.Point{X: rect.Max.X, Y: rect.Max.Y - halfWidth},
	}
	paint.FillShape(ops, color, clip.Stroke{
		Path:  diag1.Path(),
		Width: width,
	}.Op())

	diag2 := clip.Rect{
		Min: image.Point{X: rect.Min.X, Y: rect.Min.Y - halfWidth},
		Max: image.Point{X: rect.Max.X, Y: rect.Max.Y + halfWidth},
	}
	paint.FillShape(ops, color, clip.Stroke{
		Path:  diag2.Path(),
		Width: width,
	}.Op())
}

func DrawArrow(ops *op.Ops, start, end image.Point, squareSize f32.Point, width float32, color color.NRGBA) {
	const arrowHeadSize = 15

	halfSquareSize := squareSize.Div(2)
	startCenter := ToF32(start).Add(halfSquareSize)
	endCenter := ToF32(end).Add(halfSquareSize)

	vector := endCenter.Sub(startCenter)
	angle := math.Atan2(float64(vector.X), float64(vector.Y))

	headBase := f32.Pt(
		endCenter.X-float32(math.Cos(angle))*arrowHeadSize,
		endCenter.Y-float32(math.Sin(angle))*arrowHeadSize,
	)

	line := clip.Rect{
		Min: startCenter.Round(),
		Max: headBase.Round(),
	}
	paint.FillShape(ops, color, clip.Stroke{
		Path:  line.Path(),
		Width: width,
	}.Op())

	headLeft := f32.Pt(
		headBase.X-float32(math.Cos(angle+math.Pi/2))*arrowHeadSize/2,
		headBase.Y-float32(math.Sin(angle+math.Pi/2))*arrowHeadSize/2,
	)
	headRight := f32.Pt(
		headBase.X-float32(math.Cos(angle-math.Pi/2))*arrowHeadSize/2,
		headBase.Y-float32(math.Sin(angle-math.Pi/2))*arrowHeadSize/2,
	)

	var path clip.Path
	path.Begin(ops)
	path.MoveTo(headLeft)
	path.LineTo(endCenter)
	path.LineTo(headRight)
	path.Close()

	paint.FillShape(ops, color, clip.Outline{Path: path.End()}.Op())
}
