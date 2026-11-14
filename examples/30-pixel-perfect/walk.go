package main

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// Walk moves the hero 60 pixels a second with the arrow keys and plays the
// clip facing where it goes; with no key held, it stops on the clip's
// first frame.
type Walk struct {
	hero *twod.Sprite
	keys *input.State
}

func (w *Walk) Update(f world.Frame) {
	dir := geom.V2(
		w.keys.Axis(input.KeyLeft, input.KeyRight),
		w.keys.Axis(input.KeyUp, input.KeyDown),
	)
	if dir == (geom.Vec2{}) {
		w.hero.Stop()
		return
	}
	w.hero.Translate(dir.Normalize().Scale(60 * f.DT))
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
