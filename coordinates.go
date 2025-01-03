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

type OutsideCoordinatesStyle struct {
	th       *material.Theme
	fontSize float32
	board    layout.Widget
}

func (c OutsideCoordinatesStyle) Layout(gtx layout.Context) layout.Dimensions {
	size := union.SizeFromMinPt(gtx.Constraints.Max)
	boardSize := size.Float - c.fontSize*2
	squareSize := float32(boardSize) / 8
	coordPadding := squareSize/2 - c.fontSize/4

	for file := chess.FileA; file <= chess.FileH; file++ {
		centerX := util.Round(c.fontSize + float32(file)*squareSize + coordPadding)
		stack := op.Offset(image.Pt(centerX, 0)).Push(gtx.Ops)
		material.Label(c.th, unit.Sp(c.fontSize), file.String()).Layout(gtx)
		stack.Pop()
	}

	for rank := chess.Rank1; rank <= chess.Rank8; rank++ {
		centerY := util.Round(c.fontSize + float32(7-rank)*squareSize + coordPadding)
		stack := op.Offset(image.Pt(0, centerY)).Push(gtx.Ops)
		material.Label(c.th, unit.Sp(c.fontSize), rank.String()).Layout(gtx)
		stack.Pop()
	}

	return layout.UniformInset(unit.Dp(c.fontSize)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return c.board(gtx)
	})
}
