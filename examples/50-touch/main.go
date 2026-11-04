// Touch: each finger on the trackpad is drawn where it is on the pad, with
// its ID and phase, from the frame it begins to the frame it lifts. A
// trackpad touch is a place on the pad, 0 to 1, not on the screen: the
// cursor stays the mouse's.
//
//	go run ./examples/50-touch
package main

import (
	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func main() {
	win := platform.NewWindow(
		platform.Title("ember · touch"),
		platform.Size(640, 420),
		platform.FixedSize(),
		platform.Background(color.Hex("#1d2129")),
	)
	fingers := twod.NewLayer(1)
	win.Add(
		newPad(),
		fingers,
		twod.NewText("Rest one or more fingers on the trackpad", twod.At(320, 390), twod.FontSize(14)),
	)
	ember.New().
		AddEntities(win).
		AddSystems(newFingers(win.Input(), fingers)).
		MustRun()
}
