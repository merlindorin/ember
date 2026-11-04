package main

import (
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// Walk moves the hero 140 points a second with the arrow keys, through the
// collision space, and plays the clip facing where it goes.
type Walk struct {
	hero  *twod.Sprite
	keys  *input.State
	space *collide.Space
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
	w.space.Move(w.hero, dir.Normalize().Scale(140*f.DT))
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
