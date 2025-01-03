package util

import (
	"gioui.org/f32"
	"github.com/notnil/chess"
)

func PointToSquare(point f32.Point, size float32, flipped bool) chess.Square {
	scaled := point.Div(size)

	var file, rank int
	if flipped {
		file = 7 - Floor(scaled.X)
		rank = Floor(scaled.Y)
	} else {
		file = Floor(scaled.X)
		rank = 7 - Floor(scaled.Y)
	}

	if (0 <= rank && rank < 8) && (0 <= file && file < 8) {
		return chess.NewSquare(chess.File(file), chess.Rank(rank))
	} else {
		return chess.NoSquare
	}
}

func SquareToPoint(square chess.Square, size float32, flipped bool) f32.Point {
	var file, rank float32
	if flipped {
		file = float32(7 - square%8)
		rank = float32(square / 8)
	} else {
		file = float32(square % 8)
		rank = float32(7 - square/8)
	}
	return f32.Pt(file*size, rank*size)
}

func IsPromotionMove(square chess.Square, piece chess.Piece) bool {
	whitePromotes := piece.Color() == chess.White && square.Rank() == chess.Rank8
	blackPromotes := piece.Color() == chess.Black && square.Rank() == chess.Rank1
	return piece.Type() == chess.Pawn && (whitePromotes || blackPromotes)
}

func SquareColor(square chess.Square) chess.Color {
	if ((square / 8) % 2) == (square % 2) {
		return chess.Black
	}
	return chess.White
}
