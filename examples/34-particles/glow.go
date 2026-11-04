package main

import (
	"image"
	"image/color"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// glow is a white disc of size pixels, opaque in the middle and fading
// to nothing at its edge: one soft puff of fire or smoke.
func glow(size int) *twod.Texture {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	centre := float32(size) / 2
	for y := range size {
		for x := range size {
			d := geom.V2(float32(x)+0.5, float32(y)+0.5).Dist(geom.V2(centre, centre)) / centre
			a := geom.Clamp01(1 - d)
			img.Set(x, y, color.NRGBA{255, 255, 255, uint8(255 * a * a)})
		}
	}
	return twod.NewTexture(img)
}
