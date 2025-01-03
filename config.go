package chessboard

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"time"

	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type Coordinates int8

const (
	NoCoordinates Coordinates = iota
	InsideCoordinates
	OutsideCoordinates
	EachSquare
)

type Color struct {
	Hint     color.NRGBA
	LastMove color.NRGBA
	Primary  color.NRGBA
	Info     color.NRGBA
	Warning  color.NRGBA
	Danger   color.NRGBA
}

var (
	defaultColors = Color{
		Hint:     util.Transparentize(util.GrayColor, 0.7),
		LastMove: util.Transparentize(util.YellowColor, 0.5),
		Primary:  util.Transparentize(util.GreenColor, 0.7),
		Info:     util.Transparentize(util.BlueColor, 0.7),
		Warning:  util.Transparentize(util.YellowColor, 0.7),
		Danger:   util.Transparentize(util.RedColor, 0.7),
	}
)

type Config struct {
	ShowHints       bool
	ShowLastMove    bool
	Color           Color
	AnimationSpeed  time.Duration
	Coordinates     Coordinates
	BoardImage      image.Image
	BoardImageSize  union.Size
	PieceImages     []image.Image
	PieceImageSizes []union.Size
}

func NewConfig(boardFilename string, piecesFolderName string) (c Config, err error) {
	c.BoardImage, err = util.OpenImage(boardFilename)
	if err != nil {
		return c, fmt.Errorf("can't load Board image: %Board", err)
	}

	c.BoardImageSize = union.SizeFromMinPt(c.BoardImage.Bounds().Max)

	c.PieceImages, c.PieceImageSizes, err = loadPieceImages(piecesFolderName)
	if err != nil {
		return c, fmt.Errorf("can't load piece images: %Board", err)
	}

	c.Color = defaultColors

	return
}

func loadPieceImages(dir string) (images []image.Image, sizes []union.Size, err error) {
	images = make([]image.Image, 13)
	sizes = make([]union.Size, 13)

	for piece := chess.WhiteKing; piece <= chess.BlackPawn; piece++ {
		fileName := fmt.Sprintf("%s%s.png", piece.Color(), piece.Type())
		filePath := filepath.Join(dir, fileName)

		images[piece], err = util.OpenImage(filePath)
		if err != nil {
			err = fmt.Errorf("failed to open piece file %q: %Board", filePath, err)
			return
		}

		sizes[piece] = union.SizeFromMinPt(images[piece].Bounds().Max)
	}

	return
}
