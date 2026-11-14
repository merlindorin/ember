// Gestures: a crate you pinch to zoom, turn with two fingers and pan by
// moving two fingers together on the trackpad; a swipe, or R, puts it
// back.
//
//	go run ./examples/52-gestures
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed crate.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · gestures"),
		platform.Size(640, 420),
		platform.FixedSize(),
		platform.Background(color.Hex("#1d2129")),
	)
	tex, err := twod.LoadTexture(files, "crate.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	crate := twod.NewSprite(tex, twod.At(320, 200), twod.Scale(8, 8))
	win.Add(
		crate,
		twod.NewText("Pinch, turn or pan with two fingers · swipe or R to reset", twod.At(320, 395), twod.FontSize(14)),
	)
	ember.New().
		AddEntities(win).
		AddSystems(&Handle{crate: crate, in: win.Input()}).
		MustRun()
}
