package chessboard

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
)

type AnnoType int8

const (
	SolidAnno AnnoType = iota
	RectAnno
	CircleAnno
	CrossAnno
	ArrowAnno
)

type Annotation struct {
	Type  AnnoType
	Start union.Point
	End   union.Point // only for arrows
	Color color.NRGBA
	Width union.Size

	drawOp *op.CallOp
}

func (a *Annotation) Scale(factor float32) {
	a.Start.Scale(factor)
	a.End.Scale(factor)
	a.Width.Scale(factor)
}

func (a *Annotation) Draw(gtx layout.Context, squareSize union.Size, redraw bool) {
	if redraw || a.drawOp == nil {
		cache := new(op.Ops)
		annoMacro := op.Record(cache)

		annoRect := util.Rect(a.Start.Pt, squareSize.Pt)
		switch a.Type {
		case SolidAnno:
			util.DrawPane(cache, annoRect, a.Color)
		case RectAnno:
			util.DrawRectangle(cache, annoRect, a.Width.Float, a.Color)
		case CircleAnno:
			util.DrawCircle(cache, annoRect, a.Width.Float, a.Color)
		case CrossAnno:
			util.DrawCross(cache, annoRect, a.Width.Float, a.Color)
		case ArrowAnno:
			util.DrawArrow(cache, a.Start.Pt, a.End.Pt, squareSize.F32, a.Width.Float, a.Color)
		}

		ops := annoMacro.Stop()
		a.drawOp = &ops
	}

	if a.drawOp != nil {
		a.drawOp.Add(gtx.Ops)
	}
}
