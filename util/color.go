package util

import "image/color"

var (
	BlackColor  = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	WhiteColor  = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	GrayColor   = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF}
	RedColor    = color.NRGBA{R: 0xC0, G: 0x40, B: 0x40, A: 0xFF}
	YellowColor = color.NRGBA{R: 0xC0, G: 0xC0, B: 0x40, A: 0xFF}
	GreenColor  = color.NRGBA{R: 0x40, G: 0xC0, B: 0x40, A: 0xFF}
	BlueColor   = color.NRGBA{R: 0x40, G: 0x40, B: 0xC0, A: 0xFF}
)

func Transparentize(c color.NRGBA, percent float32) color.NRGBA {
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	a := float32(c.A)
	a *= percent
	c.A = uint8(a)
	return c
}
