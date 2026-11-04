package main

import (
	"image/color"
	"math"

	ecolor "github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/twod"
)

func moonlight() ecolor.Color {
	return ecolor.RGB(0.75, 0.8, 1)
}

// newShade is the ambient light of each cell: a clearing at the garden's
// heart in soft moonlight, darkening toward its edges. Smooth blends the
// levels between cells.
func newShade() *twod.TiledPicture {
	return twod.NewTiledPicture(func(tile twod.PictureTile) {
		for y := tile.Area.Min.Y; y < tile.Area.Max.Y; y++ {
			for x := tile.Area.Min.X; x < tile.Area.Max.X; x++ {
				s := moonlight().Scale(level(tile.Cell(x, y))).SRGB()
				tile.Image.SetRGBA(x, y, color.RGBA{uint8(s[0] * 255), uint8(s[1] * 255), uint8(s[2] * 255), 255})
			}
		}
	}, twod.TexelSize(32, 32))
}

// level is the ambient light of cell x, y: 0.4 at the garden's heart,
// cell 10, 6, fading out over eight cells.
func level(x, y int) float32 {
	d := math.Hypot(float64(x-10), float64(y-6))
	return float32(0.4 * max(0, 1-d/8))
}
