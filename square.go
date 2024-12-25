package chessboard

import (
	"image"
	"strconv"

	"gioui.org/f32"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type Square struct {
	Letter int
	Number int
	Origin image.Point
	Center image.Point
}

func NewSquare(position f32.Point, size float32) (s Square) {
	s.Letter = util.Floor(position.X / size)
	s.Number = util.Floor(position.Y / size)
	halfSize := size / 2
	s.Center = image.Pt(
		util.Round(float32(s.Letter)*size+halfSize),
		util.Round(float32(s.Number)*size+halfSize),
	)
	return
}

func (s Square) String() string {
	letter := string(rune('a' + s.Letter))
	number := strconv.Itoa(8 - s.Number)
	return letter + number
}

func (s Square) Equal(prev Square) bool {
	return prev.Letter == s.Letter && prev.Number == s.Number
}

func (s Square) ToChess() chess.Square {
	rank := 7 - s.Number
	file := s.Letter
	return chess.NewSquare(chess.File(file), chess.Rank(rank))
}

func SquareToPosition(square chess.Square, size float32) f32.Point {
	letter := float32(square%8) * size
	number := float32(7-square/8) * size
	return f32.Pt(letter, number)
}
