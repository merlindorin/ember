package main

import (
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// newHero is the walker standing at its feet on at, with a body as wide as
// its legs and a little deep, so it can pass in front of walls, and opts.
func newHero(at geom.Vec2, opts ...twod.SpriteOption) (*twod.Sprite, error) {
	sheet, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	frames := twod.Strip(sheet, twod.Cell(16, 16))
	clips := twod.NewAnimations().
		Add("down", 8, frames[0:4], twod.Loop()).
		Add("up", 8, frames[4:8], twod.Loop()).
		Add("side", 8, frames[8:12], twod.Loop())
	hero := twod.NewSprite(sheet, append([]twod.SpriteOption{
		twod.Animate(clips),
		twod.At(at.X, at.Y),
		twod.Scale(2, 2),
		twod.Pivot(0.5, 1),
	}, opts...)...)
	hero.Add(collide.NewBody(collide.Box(16, 8), collide.Offset(0, -4)))
	return hero, nil
}
