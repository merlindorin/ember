package look

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

// Fit scales the HUD with the window, as Camera does the board, and turns
// full screen on and off with F or Toggle. Add it to the World's systems.
type Fit struct {
	win     *platform.Window
	hud     *ui.Document
	changed func(on bool)
	zoom    float32
}

// NewFit fits hud to win; changed runs after full screen was turned on or
// off, for the game to keep it.
func NewFit(win *platform.Window, hud *ui.Document, changed func(on bool)) *Fit {
	return &Fit{win: win, hud: hud, changed: changed}
}

func (f *Fit) Update(world.Frame) {
	if f.win.Input().Pressed(input.KeyF) && !f.win.Input().KeyboardCaptured() {
		f.Toggle()
	}
	size := f.win.Size()
	if zoom := min(size.X/width, size.Y/height); zoom > 0 && zoom != f.zoom {
		f.zoom = zoom
		f.hud.SetZoom(zoom)
	}
}

// Toggle turns full screen on or off.
func (f *Fit) Toggle() {
	on := !f.win.Fullscreen()
	f.win.SetFullscreen(on)
	f.changed(on)
}
