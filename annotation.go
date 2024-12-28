package chessboard

import (
	"image/color"
	"log/slog"

	"gioui.org/layout"
	"gioui.org/op"
	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
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
	Start chess.Square
	End   chess.Square // only for arrows
	Color color.NRGBA
	Width union.Size

	drawOp *op.CallOp
}

func (a *Annotation) Copy() Annotation {
	return Annotation{
		Type:  a.Type,
		Start: a.Start,
		End:   a.End,
		Color: a.Color,
		Width: a.Width,
	}
}

func (a *Annotation) Equal(b *Annotation) bool {
	return a.Type == b.Type && a.Start == b.Start && a.Color == b.Color && (a.Type != ArrowAnno || a.End == b.End)
}

func (a *Annotation) Scale(factor float32) {
	if a.Type != NoAnno {
		a.Width.Scale(factor)
	}
}

func (a *Annotation) Draw(gtx layout.Context, squareOrigins []union.Point, squareSize union.Size, redraw bool) {
	if a.Type != NoAnno {
		if redraw || a.drawOp == nil {
			cache := new(op.Ops)
			annoMacro := op.Record(cache)

			annoRect := util.Rect(squareOrigins[a.Start].Pt, squareSize.Pt)
			switch a.Type {
			case RectAnno:
				util.DrawRectangle(cache, annoRect, a.Width.Float, a.Color)
			case CircleAnno:
				util.DrawCircle(cache, annoRect, a.Width.Float, a.Color)
			case CrossAnno:
				util.DrawCross(cache, annoRect, a.Width.Float, a.Color)
			case ArrowAnno:
				start := squareOrigins[a.Start].Pt
				end := squareOrigins[a.End].Pt
				util.DrawArrow(cache, start, end, squareSize.F32, a.Width.Float, a.Color)
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
