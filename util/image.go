package util

import (
	"image"
	_ "image/png"
	"os"
)

func OpenImage(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func Rect(origin, size image.Point) image.Rectangle {
	return image.Rectangle{
		Min: origin,
		Max: origin.Add(size),
	}
}
