// Gamepads: each pad plugged in is drawn as it is held, its buttons lit
// while down, its triggers filling as they are pulled and its sticks
// leaning past the deadzone, and a second pad shows under the first. The
// hero walks with any pad's left stick or cross, or the keys, through one
// action map. South rumbles the pad that pressed it, and its triggers
// drive its two motors.
//
//	go run ./examples/50-gamepads
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed walker.png
var files embed.FS

// field is where the hero walks, right of the pads.
func field() geom.Rect {
	return geom.Rect{X: 440, Y: 40, W: 300, H: 400}
}

func main() {
	win := platform.NewWindow(
		platform.Title("ember · gamepads"),
		platform.Size(760, 460),
		platform.FixedSize(),
		platform.Background(color.Hex("#1d2129")),
	)
	hero, err := newHero()
	if err != nil {
		log.Fatal(err)
	}
	status := twod.NewText("Plug in a pad, or two", twod.At(16, 436), twod.Pivot(0, 0.5), twod.FontSize(14))

	win.Add(
		newPadView(0, twod.At(210, 130)),
		newPadView(1, twod.At(210, 340)),
		twod.NewPolygon([]geom.Vec2{
			{X: field().X, Y: field().Y}, {X: field().X + field().W, Y: field().Y},
			{X: field().X + field().W, Y: field().Y + field().H}, {X: field().X, Y: field().Y + field().H},
		}, twod.Color(color.Hex("#4f7d43"))),
		hero,
		status,
	)

	ember.New().
		AddEntities(win).
		AddSystems(
			&Walk{hero: hero, in: win.Input(), controls: newControls()},
			&Plugs{status: status},
			&Rumble{in: win.Input()},
		).
		MustRun()
}
