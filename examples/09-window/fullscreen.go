package main

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
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
