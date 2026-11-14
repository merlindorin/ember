// Keyboard: the hero walks with WASD or the arrows, keys read by where they
// sit on the keyboard, and runs while Shift is held. Enter names him: what
// the player types is read as text, so the name comes out right on any
// layout, and Ctrl or Cmd with V pastes.
//
//	go run ./examples/47-keyboard
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · keyboard"),
		platform.Size(640, 360),
		platform.FixedSize(),
		platform.Background(color.Hex("#5b9a4a")),
	)
	hero, err := newHero()
	if err != nil {
		log.Fatal(err)
	}
	tag := twod.NewText("Hero", twod.At(0, -40), twod.Pivot(0.5, 1), twod.FontSize(14))
	hero.Add(tag)
	help := twod.NewText(
		"WASD or arrows to walk · Shift to run · Enter to name",
		twod.At(16, 12),
		twod.Pivot(0, 0),
		twod.FontSize(15),
	)

	win.Add(hero, help)
	naming := &Naming{window: win, tag: tag, name: "Hero"}

	ember.New().
		AddEntities(win).
		AddSystems(naming, &Walk{hero: hero, in: win.Input(), naming: naming}).
		MustRun()
}
