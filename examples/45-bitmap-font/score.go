package main

import (
	"fmt"

	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// Score counts up 250 points a second and writes them, six digits wide.
type Score struct {
	text   *twod.BitmapText
	points float32
}

func (s *Score) Update(f world.Frame) {
	s.points += 250 * f.DT
	s.text.SetText(fmt.Sprintf("SCORE %06d", int(s.points)))
}
