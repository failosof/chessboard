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
	paint.Fill(ops, color)
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
	paint.Fill(ops, color)
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
	offsetPt := f32.Pt(width, width).Mul(0.7)
	var aPath clip.Path
	aPath.Begin(ops)
	aPath.MoveTo(ToF32(rect.Min).Add(offsetPt))
	aPath.LineTo(ToF32(rect.Max).Sub(offsetPt))
	paint.FillShape(ops, color, clip.Stroke{
		Path:  aPath.End(),
		Width: width,
	}.Op())

	offset := offsetPt.Round().X
	var bPath clip.Path
	bPath.Begin(ops)
	bPath.MoveTo(f32.Pt(float32(rect.Max.X-offset), float32(rect.Min.Y+offset)))
	bPath.LineTo(f32.Pt(float32(rect.Min.X+offset), float32(rect.Max.Y-offset)))
	paint.FillShape(ops, color, clip.Stroke{
		Path:  bPath.End(),
		Width: width,
	}.Op())
}

func DrawArrow(ops *op.Ops, start, end image.Point, squareSize f32.Point, width float32, color color.NRGBA) {
	arrowHeadSize := width * 4
	lineStartOffset := arrowHeadSize * 0.8
	lineEndOffset := arrowHeadSize * 0.2

	halfSquareSize := squareSize.Div(2)
	startCenter := ToF32(start).Add(halfSquareSize)
	endCenter := ToF32(end).Add(halfSquareSize)

	vector := endCenter.Sub(startCenter)
	angle := math.Atan2(float64(vector.Y), float64(vector.X))

	lineStart := f32.Pt(
		startCenter.X+float32(math.Cos(angle))*(halfSquareSize.X-lineStartOffset),
		startCenter.Y+float32(math.Sin(angle))*(halfSquareSize.Y-lineStartOffset),
	)
	lineEnd := f32.Pt(
		endCenter.X-float32(math.Cos(angle))*(arrowHeadSize+lineEndOffset),
		endCenter.Y-float32(math.Sin(angle))*(arrowHeadSize+lineEndOffset),
	)

	var linePath clip.Path
	linePath.Begin(ops)
	linePath.MoveTo(lineStart)
	linePath.LineTo(lineEnd)
	paint.FillShape(ops, color, clip.Stroke{
		Path:  linePath.End(),
		Width: width,
	}.Op())

	headBase := f32.Pt(
		endCenter.X-float32(math.Cos(angle))*arrowHeadSize,
		endCenter.Y-float32(math.Sin(angle))*arrowHeadSize,
	)
	headLeft := f32.Pt(
		headBase.X-float32(math.Cos(angle+math.Pi/2))*(arrowHeadSize/2),
		headBase.Y-float32(math.Sin(angle+math.Pi/2))*(arrowHeadSize/2),
	)
	headRight := f32.Pt(
		headBase.X-float32(math.Cos(angle-math.Pi/2))*(arrowHeadSize/2),
		headBase.Y-float32(math.Sin(angle-math.Pi/2))*(arrowHeadSize/2),
	)

	var headPath clip.Path
	headPath.Begin(ops)
	headPath.MoveTo(headLeft)
	headPath.LineTo(endCenter)
	headPath.LineTo(headRight)
	headPath.Close()
	paint.FillShape(ops, color, clip.Outline{Path: headPath.End()}.Op())
}
