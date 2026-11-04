package main

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// Walk moves the hero along the ground, 160 points a second, with the left
// and right arrows, facing where he goes; with no key held, he stops.
type Walk struct {
	hero *twod.Sprite
	keys *input.State
}

func (w *Walk) Update(f world.Frame) {
	dir := w.keys.Axis(input.KeyLeft, input.KeyRight)
	if dir == 0 {
		w.hero.Stop()
		return
	}
	w.hero.Translate(geom.V2(dir*160*f.DT, 0))
	w.hero.SetFlip(dir < 0, false)
	w.hero.Play("walk")
}
