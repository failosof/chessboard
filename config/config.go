package config

import (
	"fmt"
	"image/color"
	"time"
)

type Chessboard struct {
	ShowLegalMoves bool
	ShowLastMove   bool
	HintColor      color.NRGBA
	LastMoveColor  color.NRGBA
	AnimationSpeed time.Duration
	BoardStyle     BoardStyle
	PiecesStyle    PiecesStyle
}

func Load(boardFilename string, piecesFolderName string) (c Chessboard, err error) {
	c.BoardStyle, err = LoadBoardStyle(boardFilename)
	if err != nil {
		return c, fmt.Errorf("can't load config: %w", err)
	}

	c.PiecesStyle, err = LoadPiecesStyle(piecesFolderName)
	if err != nil {
		return c, fmt.Errorf("can't load config: %w", err)
	}

	return
}
