package main

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// newHero is the walker of 28-y-sort, standing in the middle of the window.
func newHero() (*twod.Sprite, error) {
	sheet, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	frames := twod.Strip(sheet, twod.Cell(16, 16))
	clips := twod.NewAnimations().
		Add("down", 8, frames[0:4], twod.Loop()).
		Add("up", 8, frames[4:8], twod.Loop()).
		Add("side", 8, frames[8:12], twod.Loop())
	return twod.NewSprite(sheet,
		twod.Animate(clips),
		twod.At(320, 220),
		twod.Scale(2, 2),
		twod.Pivot(0.5, 1),
	), nil
}

// Walk moves the hero 160 points a second where the four move actions
// point, and plays the clip facing that way.
type Walk struct {
	hero     *twod.Sprite
	in       *input.State
	controls *input.Actions[Act]
}

func (w *Walk) Update(f world.Frame) {
	dir := w.controls.Direction(w.in, Left, Right, Up, Down)
	if dir == (geom.Vec2{}) {
		w.hero.Stop()
		return
	}
	w.hero.Translate(dir.Scale(160 * f.DT))
	switch {
	case dir.X != 0:
		w.hero.Play("side")
		w.hero.SetFlip(dir.X < 0, false)
	case dir.Y < 0:
		w.hero.Play("up")
	default:
		w.hero.Play("down")
	}
}
