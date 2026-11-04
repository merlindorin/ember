package main

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// groundLine is where the hero's feet are.
const groundLine = 280

// newHero is the walking character of 20-sprite-sheet, standing on the
// ground in the middle of the level.
func newHero() (*twod.Sprite, error) {
	sheet, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	frames := twod.Strip(sheet, twod.Cell(16, 16))
	clips := twod.NewAnimations().Add("walk", 8, frames[8:12], twod.Loop())
	return twod.NewSprite(sheet,
		twod.Animate(clips),
		twod.At(level().W/2, groundLine),
		twod.Scale(4, 4),
		twod.Pivot(0.5, 1),
	), nil
}

// newGround is the earth under the hero, the whole level wide.
func newGround() *twod.Polygon {
	return twod.NewPolygon(
		[]geom.Vec2{
			geom.V2(0, groundLine),
			geom.V2(level().W, groundLine),
			geom.V2(level().W, level().H),
			geom.V2(0, level().H),
		},
		twod.Color(color.Hex("#5f8f4e")),
	)
}

// newHills is a far ridge on the horizon, on a layer that moves at half
// the camera's speed both ways: with the view centred 180 points down, it
// shows 90 points lower, so the hills are drawn that much higher.
func newHills() *twod.Layer {
	base := float32(groundLine - 90)
	hills := twod.NewLayer(-1, twod.Parallax(0.5))
	for i := range 8 {
		x := float32(i) * 220
		hills.Add(twod.NewPolygon(
			[]geom.Vec2{geom.V2(x-60, base), geom.V2(x+90, base-130), geom.V2(x+240, base)},
			twod.Color(color.Hex("#7aa26a")),
		))
	}
	return hills
}
