package util

import (
	"image"
	"image/color"

	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

func DrawImage(ops *op.Ops, img image.Image, at image.Point) {
	imageOp := paint.NewImageOp(img)
	offset := op.Offset(at).Push(ops)
	imageOp.Filter = paint.FilterLinear
	imageOp.Add(ops)
	paint.PaintOp{}.Add(ops)
	offset.Pop()
}

func DrawPane(ops *op.Ops, rect image.Rectangle, color color.NRGBA) {
	defer clip.Rect(rect).Push(ops).Pop()
	paint.ColorOp{Color: color}.Add(ops)
	paint.PaintOp{}.Add(ops)
}

func DrawRectangle(ops *op.Ops, rect image.Rectangle, width float32, color color.NRGBA) {
	const r = 15
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
