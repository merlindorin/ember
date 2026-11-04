package main

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/world"
)

// Reopen shows the settings again on Enter once they are closed, and says
// so. Closing only hides a window, so what was set in it is still there.
type Reopen struct {
	settings *settings
	in       *input.State
}

func (r *Reopen) Update(world.Frame) {
	s := r.settings
	if !s.doc.Open(s.window) && r.in.Pressed(input.KeyEnter) {
		s.doc.Show(s.window)
		s.status.SetText("Settings open again")
	}
}
