package main

import (
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

// ToggleFullscreen switches the window fullscreen and back with F.
type ToggleFullscreen struct {
	window *platform.Window
}

func (t *ToggleFullscreen) Update(world.Frame) {
	if t.window.Input().Pressed(input.KeyF) {
		t.window.SetFullscreen(!t.window.Fullscreen())
	}
}
