package chessboard

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type CoordinatesStyle struct {
	Type     Coordinates
	Theme    *material.Theme
	FontSize float32
	Flipped  bool
	Board    layout.Widget
}

func (s CoordinatesStyle) Layout(gtx layout.Context) layout.Dimensions {
	switch s.Type {
	case OutsideCoordinates:
		return s.outside(gtx)
	case InsideCoordinates:
		return s.inside(gtx)
	case EachSquare:
		return s.eachSquare(gtx)
	default:
		return s.Board(gtx)
	}
}

func (s CoordinatesStyle) outside(gtx layout.Context) layout.Dimensions {
	size := union.SizeFromMinPt(gtx.Constraints.Max)
	boardSize := size.Float - s.FontSize*2
	squareSize := float32(boardSize) / 8
	coordPadding := squareSize/2 - s.FontSize/4

	for file := chess.FileA; file <= chess.FileH; file++ {
		i := file
		if s.Flipped {
			i = 7 - file
		}
		centerX := util.Round(s.FontSize + float32(i)*squareSize + coordPadding)
		stack := op.Offset(image.Pt(centerX, 0)).Push(gtx.Ops)
		material.Label(s.Theme, unit.Sp(s.FontSize), file.String()).Layout(gtx)
		stack.Pop()
	}

	for rank := chess.Rank1; rank <= chess.Rank8; rank++ {
		i := rank
		if !s.Flipped {
			i = 7 - rank
		}
		centerY := util.Round(s.FontSize + float32(i)*squareSize + coordPadding)
		stack := op.Offset(image.Pt(0, centerY)).Push(gtx.Ops)
		material.Label(s.Theme, unit.Sp(s.FontSize), rank.String()).Layout(gtx)
		stack.Pop()
	}

	return layout.UniformInset(unit.Dp(s.FontSize)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return s.Board(gtx)
	})
}

func (s CoordinatesStyle) inside(gtx layout.Context) layout.Dimensions {
	return s.Board(gtx)
}

func (s CoordinatesStyle) eachSquare(gtx layout.Context) layout.Dimensions {
	return s.Board(gtx)
}
