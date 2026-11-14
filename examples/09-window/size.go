package main

import (
	"fmt"

	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

// ShowSize writes the window's size in its label.
type ShowSize struct {
	window *platform.Window
	label  *ui.Node
}

func (s *ShowSize) Update(world.Frame) {
	size := s.window.Size()
	if text := fmt.Sprintf("%.0f × %.0f", size.X, size.Y); text != s.label.Text() {
		s.label.SetText(text)
	}
}
