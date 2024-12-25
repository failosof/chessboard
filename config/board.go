package config

import (
	"fmt"
	"image"
	"image/color"

	"github.com/failosof/chessboard/util"
)

type Coordinates int8

const (
	NoCoordinates Coordinates = iota
	InsideCoordinates
	OutsideCoordinates
	EachSquare
)

type BoardStyle struct {
	Image       image.Image
	Coordinates Coordinates
	Background  color.Color
	Size        int
	cachedImage image.Image
}

func LoadBoardStyle(filename string) (b BoardStyle, err error) {
	img, err := util.OpenImage(filename)
	if err != nil {
		return b, fmt.Errorf("failed to open board file %q: %v", filename, err)
	}

	b.Image = img
	b.cachedImage = img
	b.Size = img.Bounds().Dx()
	return
}

func (b *BoardStyle) ImageFor(size int) (img image.Image) {
	if b.Size == size {
		return b.cachedImage
	}

	img = util.ResizeImage(b.Image, size, size)
	b.cachedImage = img
	return
}
