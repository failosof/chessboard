package config

import (
	"fmt"
	"image"
	"path/filepath"

	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type PieceCache struct {
	Image image.Image
	Size  int
}

type PiecesStyle struct {
	pieces []image.Image
	cache  []PieceCache
}

func LoadPiecesStyle(dir string) (t PiecesStyle, err error) {
	t.pieces = make([]image.Image, 13)
	t.cache = make([]PieceCache, 13)

	for piece := chess.WhiteKing; piece <= chess.BlackPawn; piece++ {
		fileName := fmt.Sprintf("%s%s.png", piece.Color(), piece.Type())
		filePath := filepath.Join(dir, fileName)

		img, err := util.OpenImage(filePath)
		if err != nil {
			return t, fmt.Errorf("failed to open piece file %q: %w", filePath, err)
		}

		t.pieces[piece] = img
		t.cache[piece] = PieceCache{
			Image: img,
			Size:  img.Bounds().Dx(),
		}
	}

	return
}

func (t *PiecesStyle) ImageFor(piece chess.Piece) (img image.Image) {
	return t.pieces[piece]
}
