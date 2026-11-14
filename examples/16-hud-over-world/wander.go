package main

import (
	"github.com/lantervale/ember/world"
)

// Wander walks each walker back and forth along its route, 60 points a
// second, playing the clip facing where it goes.
type Wander struct {
	walkers []*walker
}

func (s *Wander) Update(f world.Frame) {
	for _, w := range s.walkers {
		to := w.route[w.toward].Sub(w.sprite.Position())
		step := 60 * f.DT
		if to.Len() <= step {
			w.sprite.SetPosition(w.route[w.toward])
			w.toward = 1 - w.toward
			continue
		}
		dir := to.Normalize()
		w.sprite.Translate(dir.Scale(step))
		switch {
		case abs(dir.X) > abs(dir.Y):
			w.sprite.Play("side")
			w.sprite.SetFlip(dir.X < 0, false)
		case dir.Y < 0:
			w.sprite.Play("up")
		default:
			w.sprite.Play("down")
		}
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
