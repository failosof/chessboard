package chessboard

import (
	"image/color"
	"log/slog"

	"gioui.org/layout"
	"gioui.org/op"
	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
)

type AnnoType int8

const (
	NoAnno AnnoType = iota
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
	if a.Type != NoAnno {
		a.Start.Scale(factor)
		a.End.Scale(factor)
		a.Width.Scale(factor)
	}
}

func (a *Annotation) Draw(gtx layout.Context, squareSize union.Size, redraw bool) {
	if a.Type != NoAnno {
		if redraw || a.drawOp == nil {
			cache := new(op.Ops)
			annoMacro := op.Record(cache)

			annoRect := util.Rect(a.Start.Pt, squareSize.Pt)
			switch a.Type {
			case RectAnno:
				util.DrawRectangle(cache, annoRect, a.Width.Float, a.Color)
			case CircleAnno:
				util.DrawCircle(cache, annoRect, a.Width.Float, a.Color)
			case CrossAnno:
				util.DrawCross(cache, annoRect, a.Width.Float, a.Color)
			case ArrowAnno:
				util.DrawArrow(cache, a.Start.Pt, a.End.Pt, squareSize.F32, a.Width.Float, a.Color)
			default:
				slog.Error("unknown annotation type", "type", a.Type)
			}

			ops := annoMacro.Stop()
			a.drawOp = &ops
		}

		if a.drawOp != nil {
			a.drawOp.Add(gtx.Ops)
		}
	}
}
