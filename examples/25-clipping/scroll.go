package main

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// Scroll moves the texts left 120 points a second, and starts each one
// again from the right once it has gone by.
type Scroll struct {
	texts []*twod.Text
}

func (s *Scroll) Update(f world.Frame) {
	for _, t := range s.texts {
		t.Translate(geom.V2(-120*f.DT, 0))
		if at := t.Position(); at.X < -t.Measure().X {
			t.SetPosition(geom.V2(640, at.Y))
		}
	}
}
