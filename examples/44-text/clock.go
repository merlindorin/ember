package main

import (
	"fmt"

	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// Clock writes the time since the start in its text, every frame.
type Clock struct {
	text *twod.Text
}

func (c *Clock) Update(f world.Frame) {
	c.text.SetText(fmt.Sprintf("%05.1f s", f.Elapsed))
}
