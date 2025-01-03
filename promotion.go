package chessboard

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

var (
	whiteCandidates = []chess.Piece{chess.WhiteQueen, chess.WhiteRook, chess.WhiteBishop, chess.WhiteKnight}
	blackCandidates = []chess.Piece{chess.BlackPawn, chess.BlackRook, chess.BlackBishop, chess.BlackKnight}
)

type Promotion struct {
	Position         union.Point
	SquareSize       union.Size
	Color            chess.Color
	Background       color.NRGBA
	Piece            Piece
	HoveredCandidate chess.Piece
	Flipped          bool
}

func (p Promotion) Layout(gtx layout.Context) layout.Dimensions {
	selectionSize := p.SquareSize.Pt.Add(p.Position.Pt)
	selectionSize.Y *= 4

	selection := image.Rectangle{Min: p.Position.Pt, Max: selectionSize}.Canon()
	util.DrawPane(gtx.Ops, selection, p.Background)

	candidates := whiteCandidates
	if p.Color == chess.Black {
		candidates = blackCandidates
	}

	piecePos := p.Position.Pt
	pieceEventTargets := make([]event.Filter, len(candidates))
	for i, piece := range candidates {
		factor := p.SquareSize.F32.Div(p.Piece.Sizes[piece].Float)
		util.DrawImage(gtx.Ops, p.Piece.Images[piece], piecePos, factor)
		pieceClip := clip.Rect(image.Rectangle{Min: piecePos, Max: piecePos.Add(factor.Round())}).Push(gtx.Ops)
		event.Op(gtx.Ops, piece)
		pieceClip.Pop()
		piecePos.Y += p.SquareSize.Int
		pieceEventTargets[i] = pointer.Filter{
			Target: piece,
			Kinds:  pointer.Move | pointer.Press,
		}
	}

	//defer clip.Rect(selection).Push(gtx.Ops).Pop()
	//event.Op(gtx.Ops, p)

	for {
		ev, ok := gtx.Event(pieceEventTargets...)
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Move:
				pointer.CursorPointer.Add(gtx.Ops)
			case pointer.Press:

			}
		}
	}

	return layout.Dimensions{Size: selectionSize}
}
