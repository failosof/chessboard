package util

import (
	"image"

	"gioui.org/op"
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
