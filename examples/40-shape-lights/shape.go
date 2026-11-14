package main

import (
	"image"
	"image/color"
	"math"

	ecolor "github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
)

// newLava is a pool of lava whose own picture glows: Emissive lights the
// grass round it 80 points past its edge, in its own colours, and Unlit
// draws the pool itself bright.
func newLava(at geom.Vec2) *twod.Sprite {
	return twod.NewSprite(lavaPicture(), twod.At(at.X, at.Y), twod.Scale(2, 2), light.Emissive(80), light.Unlit())
}

// newNeon is a tube of light, a shape light along a line 64 points long:
// full on the line, fading to nothing 40 points from it. Like every shape
// light it shines through everything.
func newNeon(at geom.Vec2) *light.Light {
	return light.NewShape(
		light.Line(geom.V2(-32, 0), geom.V2(32, 0)),
		ecolor.RGB(0.4, 1, 0.9),
		40,
		light.At(at.X, at.Y),
	)
}

// newCrystal is a round crystal 12 points across, glowing pale blue 120
// points out. With Shadows, the trees and pillars round it block its light.
func newCrystal(at geom.Vec2) *light.Light {
	return light.NewShape(light.Circle(12), ecolor.RGB(0.8, 0.85, 1), 120, light.At(at.X, at.Y), light.Shadows(true))
}

// newRune is a triangle traced on the ground in golden light.
func newRune(at geom.Vec2) *light.Light {
	return light.NewShape(
		light.Polygon(geom.V2(0, -20), geom.V2(18, 12), geom.V2(-18, 12)),
		ecolor.RGB(1, 0.85, 0.3),
		30,
		light.At(at.X, at.Y),
	)
}

// lavaPicture draws an oval of lava, yellow at its heart and dark red at
// its rim, pixel by pixel.
func lavaPicture() *twod.Texture {
	const w, h = 24, 14
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			dx, dy := (float64(x)+0.5-w/2)/(w/2), (float64(y)+0.5-h/2)/(h/2)
			d := math.Sqrt(dx*dx + dy*dy)
			if d > 1 {
				continue
			}
			heat := 1 - d
			img.Set(x, y, color.RGBA{255, uint8(60 + 180*heat), uint8(20 + 60*heat*heat), 255})
		}
	}
	return twod.NewTexture(img, twod.Pixelated())
}
