package look

import (
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

// Fit scales the HUD with the window, as Camera does the board, turns full
// screen on and off with F or Toggle, and shows the pad's prompts in
// Controls while a gamepad is what the player used last. Add it to the
// World's systems.
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
	in := f.win.Input()
	if in.Pressed(input.KeyF) && !in.KeyboardCaptured() {
		f.Toggle()
	}
	body := f.hud.Body()
	if pad := in.LastDevice() == input.GamepadDevice; pad != body.HasClass("gamepad") {
		if pad {
			body.AddClass("gamepad")
		} else {
			body.RemoveClass("gamepad")
		}
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
