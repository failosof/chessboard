package theme

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

type PiecesTheme struct {
	pieces []image.Image
	cache  []PieceCache
}

func LoadPiecesTheme(dir string) (t PiecesTheme, err error) {
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

func (t *PiecesTheme) ImageFor(piece chess.Piece, size int) (img image.Image) {
	cache := t.cache[piece]
	if cache.Size == size {
		return cache.Image
	}

	img = t.pieces[piece]
	img = util.ResizeImage(img, size, size)
	cache.Image = img
	cache.Size = size
	t.cache[piece] = cache

	return
}
