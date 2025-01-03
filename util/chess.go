package util

import (
	"gioui.org/f32"
	"github.com/notnil/chess"
)

func PointToSquare(point f32.Point, size float32) chess.Square {
	scaled := point.Div(size)
	file := chess.File(Floor(scaled.X))
	rank := chess.Rank(7 - Floor(scaled.Y))
	if (0 <= rank && rank < 8) && (0 <= file && file < 8) {
		return chess.NewSquare(file, rank)
	} else {
		return chess.NoSquare
	}
}

func SquareToPoint(square chess.Square, size float32) f32.Point {
	file := float32(square%8) * size
	rank := float32(7-square/8) * size
	return f32.Pt(file, rank)
}

func SquareColor(square chess.Square) chess.Color {
	if ((square / 8) % 2) == (square % 2) {
		return chess.Black
	}
	return chess.White
}
